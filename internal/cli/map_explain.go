package cli

import (
	"fmt"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/gitx"
	"github.com/sanketvgh/envbuckets/internal/pattern"
)

func mapExplain(env Env, p *project, branch string) error {
	if branch == "" || strings.HasPrefix(branch, "-") || strings.ContainsAny(branch, " \t\r\n") {
		return usage("map explain: %q is not a branch name", branch).then("envbuckets map explain feature/login")
	}
	t, found, err := p.target(branch)
	if err != nil {
		return envErr("cannot read the branch link: %v", err).then("envbuckets unlink --branch %s, or git config --local --unset %s", branch, "branch."+branch+".envbuckets")
	}
	fmt.Fprintf(env.Stdout, "Branch %s", branch)
	if cur, _ := gitx.Branch(p.root); cur == branch {
		fmt.Fprint(env.Stdout, " (current)")
	} else if !gitx.BranchExists(p.root, branch) {
		fmt.Fprint(env.Stdout, " (no local branch, explained as if checked out)")
	}
	fmt.Fprintln(env.Stdout)
	if !found {
		fmt.Fprintln(env.Stdout, "No matching rule or local pin; checkout leaves .env unchanged.")
		return nil
	}
	if t.linked {
		fmt.Fprintf(env.Stdout, "Using bucket %s (local pin; overrides rules)\n", t.bucket)
		if rule := p.cfg.Match(branch, pattern.Match); rule != nil {
			fmt.Fprintf(env.Stdout, "Overridden rule: %d. %s -> %s\n", p.cfg.RuleIndex(rule.Pattern)+1, rule.Pattern, rule.Bucket)
		}
	} else {
		fmt.Fprintf(env.Stdout, "Using bucket %s (rule %s, priority %d)\n", t.bucket, t.via, t.priority)
	}
	fmt.Fprintln(env.Stdout)
	for _, s := range p.scopes {
		line := explainScope(s, t.bucket)
		fmt.Fprintf(env.Stdout, "%-12s %s\n", s.Name, line)
	}
	return nil
}

func explainScope(s scope, bucket string) string {
	if !s.exists() {
		return s.Path + " missing (scope directory)"
	}
	avail := s.display(linkTarget(bucket)) + " exists"
	ok := s.bucketFileExists(bucket)
	if !ok {
		avail = s.display(linkTarget(bucket)) + " missing"
	}
	return avail + "; " + activeDescription(s)
}

func activeDescription(s scope) string {
	ls, err := s.linkState()
	if err != nil {
		return s.display(envFile) + ": " + err.Error()
	}
	switch ls.kind {
	case linkMissing:
		return s.display(envFile) + " missing"
	case linkReal:
		return s.display(envFile) + " is a real file (unmanaged)"
	case linkForeign:
		return s.display(envFile) + " -> " + ls.target + " (unmanaged)"
	}
	if ls.dangling {
		return s.display(envFile) + " -> " + s.display(linkTarget(ls.bucket)) + " (target missing)"
	}
	return s.display(envFile) + " -> " + s.display(linkTarget(ls.bucket))
}
