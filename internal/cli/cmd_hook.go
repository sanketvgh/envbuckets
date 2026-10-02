package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/gitx"
	"github.com/sanketvgh/envbuckets/internal/pattern"
)

func runHook(args []string, env Env) int {
	if len(args) >= 3 && args[2] == "0" {
		return ExitOK
	}
	warn := func(format string, a ...any) {
		fmt.Fprintf(env.Stderr, "envbuckets: "+format+"\n", a...)
	}

	root, err := gitx.Root(env.Cwd)
	if err != nil {
		return ExitOK
	}
	cfg, err := config.Load(root)
	if errors.Is(err, config.ErrMissing) {
		warn("not initialized here, .env left as-is, next: envbuckets init")
		return ExitOK
	}
	if err != nil {
		warn("acting dormant, .env left as-is: %v, next: restore %s from git or run envbuckets uninstall", err, config.FileName)
		return ExitOK
	}
	branch, err := gitx.Branch(root)
	if err != nil {
		warn("cannot resolve the branch, .env left as-is: %v, next: envbuckets status", err)
		return ExitOK
	}
	if branch == "" {
		warn("detached HEAD, .env left as-is")
		return ExitOK
	}
	p := newProject(root, cfg)
	t, found, err := p.target(branch)
	if err != nil {
		warn("cannot read the branch link, .env left as-is: %v, next: envbuckets status", err)
		return ExitOK
	}
	if !found {
		warn("no rule matches %q, .env left as-is, next: envbuckets map add <pattern> <bucket>", branch)
		return ExitOK
	}

	sw := p.switchAll(t.bucket)
	if sw.switched > 0 {
		fmt.Fprintf(env.Stdout, "envbuckets: %s -> %s (%s) - %s switched, next: restart your dev servers\n",
			joinOr(sw.previous, "?"), t.bucket, t.via, scopeCount(sw.switched))
	}
	for _, w := range sw.warnings {
		fmt.Fprintf(env.Stderr, "warning: %s\n", w)
	}
	return ExitOK
}

type target struct {
	bucket   string
	via      string
	linked   bool
	priority int
}

func (p *project) target(branch string) (target, bool, error) {
	bucket, err := gitx.LinkedBucket(p.root, branch)
	if err != nil {
		return target{}, false, err
	}
	if bucket != "" {
		if err := config.ValidateName(bucket); err != nil {
			return target{}, false, fmt.Errorf("branch.%s.envbuckets in .git/config: %w", branch, err)
		}
		return target{bucket: bucket, via: "link", linked: true}, true, nil
	}
	for i, rule := range p.cfg.Rules {
		if pattern.Match(rule.Pattern, branch) {
			return target{bucket: rule.Bucket, via: rule.Pattern, priority: i + 1}, true, nil
		}
	}
	return target{}, false, nil
}

type switchResult struct {
	previous []string
	warnings []string
	switched int
}

func (p *project) switchAll(bucket string) switchResult {
	var res switchResult
	for _, s := range p.scopes {
		from, skip, ok := switchScope(s, bucket)
		if !ok {
			res.warnings = append(res.warnings, skip)
			continue
		}
		if from == "" {
			continue
		}
		res.switched++
		res.previous = appendUnique(res.previous, from)
	}
	return res
}

func scopeCount(n int) string {
	if n == 1 {
		return "1 scope"
	}
	return fmt.Sprintf("%d scopes", n)
}

func switchScope(s scope, bucket string) (from, skip string, ok bool) {
	if !s.exists() {
		return "", s.Name + ": scope directory missing, skipped, next: envbuckets scope rm " + s.Name, false
	}
	ls, err := s.linkState()
	if err != nil {
		return "", fmt.Sprintf("%s: %v", s.Name, err), false
	}
	switch ls.kind {
	case linkReal:
		return "", fmt.Sprintf("%s: %s is a real file, left untouched, next: envbuckets init", s.Name, s.display(envFile)), false
	case linkMissing:
		return "", fmt.Sprintf("%s: no %s symlink, skipped, next: envbuckets apply", s.Name, s.display(envFile)), false
	case linkBucket:
		if ls.bucket == bucket && !ls.dangling {
			return "", "", true
		}
	case linkForeign:
		return "", fmt.Sprintf("%s: %s is a foreign symlink to %s, left untouched, next: remove it by hand", s.Name, s.display(envFile), ls.target), false
	}
	if !s.bucketFileExists(bucket) {
		return "", fmt.Sprintf("%s: missing %s, kept previous bucket, next: create that file, then envbuckets status", s.Name, s.display(linkTarget(bucket))), false
	}
	changed, err := s.pointTo(bucket)
	if err != nil {
		return "", fmt.Sprintf("%s: %v", s.Name, err), false
	}
	if !changed {
		return "", "", true
	}
	if ls.kind == linkBucket {
		return ls.bucket, "", true
	}
	return strings.TrimSpace(ls.target), "", true
}

func appendUnique(list []string, item string) []string {
	for _, l := range list {
		if l == item {
			return list
		}
	}
	return append(list, item)
}
