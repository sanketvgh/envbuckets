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
	fmt.Fprintf(env.Stdout, "branch: %s", branch)
	if cur, _ := gitx.Branch(p.root); cur == branch {
		fmt.Fprint(env.Stdout, " (current)")
	} else if !gitx.BranchExists(p.root, branch) {
		fmt.Fprint(env.Stdout, " (no local branch, explained as if checked out)")
	}
	fmt.Fprintln(env.Stdout)
	if !found {
		fmt.Fprintln(env.Stdout, "rule:   none matches, checkout leaves .env as-is")
		fmt.Fprintln(env.Stdout, "bucket: none")
		return nil
	}
	if t.linked {
		fmt.Fprintf(env.Stdout, "link:   %s (local pin, overrides rules)\n", t.bucket)
		if rule := p.cfg.Match(branch, pattern.Match); rule != nil {
			fmt.Fprintf(env.Stdout, "rule:   %d. %s -> %s (overridden by the pin)\n", p.cfg.RuleIndex(rule.Pattern)+1, rule.Pattern, rule.Bucket)
		}
	} else {
		fmt.Fprintf(env.Stdout, "rule:   %d. %s -> %s (first match wins)\n", t.priority, t.via, t.bucket)
	}
	fmt.Fprintf(env.Stdout, "bucket: %s\nscopes:\n", t.bucket)
	for _, s := range p.scopes {
		line := explainScope(s, t.bucket)
		fmt.Fprintf(env.Stdout, "  %-12s %s\n", s.Name, line)
	}
	return nil
}

func explainScope(s scope, bucket string) string {
	if !s.exists() {
		return "MISSING directory " + s.Path
	}
	avail := "available " + s.display(linkTarget(bucket))
	ok := s.bucketFileExists(bucket)
	if !ok {
		avail = "missing " + s.display(linkTarget(bucket))
	}
	return avail + " | " + activeDescription(s)
}

func activeDescription(s scope) string {
	ls, err := s.linkState()
	if err != nil {
		return "active: unreadable (" + err.Error() + ")"
	}
	switch ls.kind {
	case linkMissing:
		return "active: none (no " + envFile + ")"
	case linkReal:
		return "active: none (" + envFile + " is a real file, not managed)"
	case linkForeign:
		return "active: none (" + envFile + " -> " + ls.target + ", not managed)"
	}
	if ls.dangling {
		return "active: " + ls.bucket + " (BROKEN, file missing)"
	}
	return "active: " + ls.bucket
}
