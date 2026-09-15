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
	rule := cfg.Match(branch, pattern.Match)
	if rule == nil {
		warn("no rule matches %q, .env left as-is, next: envbuckets map add <pattern> <bucket>", branch)
		return ExitOK
	}

	p := newProject(root, cfg)
	var previous, warnings []string
	switched := 0
	for _, s := range p.scopes {
		from, skip, ok := switchScope(s, rule.Bucket)
		if !ok {
			warnings = append(warnings, skip)
			continue
		}
		if from == "" {
			continue
		}
		switched++
		previous = appendUnique(previous, from)
	}
	if switched > 0 {
		unit := "scope"
		if switched != 1 {
			unit = "scopes"
		}
		fmt.Fprintf(env.Stdout, "envbuckets: %s -> %s (%s) - %d %s switched, next: restart your dev servers\n",
			joinOr(previous, "?"), rule.Bucket, rule.Pattern, switched, unit)
	}
	for _, w := range warnings {
		fmt.Fprintf(env.Stderr, "warning: %s\n", w)
	}
	return ExitOK
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
		return "", fmt.Sprintf("%s: no %s symlink, skipped, next: envbuckets init", s.Name, s.display(envFile)), false
	case linkBucket:
		if ls.bucket == bucket && !ls.dangling {
			return "", "", true
		}
	case linkForeign:
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
