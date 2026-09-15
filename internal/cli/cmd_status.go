package cli

import (
	"fmt"

	"github.com/sanketvgh/envbuckets/internal/gitx"
	"github.com/sanketvgh/envbuckets/internal/pattern"
)

func runStatus(args []string, env Env) error {
	flags := newFlags("status")
	if _, err := parseFlags(flags, args); err != nil {
		return err
	}
	p, err := openProject(env)
	if err != nil {
		return err
	}
	branch, err := gitx.Branch(p.root)
	if err != nil {
		return envErr("cannot resolve the current branch: %v", err).then("check git status")
	}

	want := ""
	if branch == "" {
		fmt.Fprintln(env.Stdout, "branch: (detached HEAD), the hook leaves .env untouched\n  next: git checkout <branch>, or envbuckets use <bucket>")
	} else if rule := p.cfg.Match(branch, pattern.Match); rule == nil {
		fmt.Fprintf(env.Stdout, "branch: %s\nrule:   none, .env is left as-is on checkout\n  next: envbuckets map add <pattern> <bucket>\n", branch)
	} else {
		want = rule.Bucket
		fmt.Fprintf(env.Stdout, "branch: %s\nrule:   %s -> %s\n", branch, rule.Pattern, rule.Bucket)
	}

	fmt.Fprintln(env.Stdout, "scopes:")
	for _, s := range p.scopes {
		fmt.Fprintf(env.Stdout, "  %-12s %s\n", s.Name, scopeStatus(s, want))
	}
	return nil
}

func scopeStatus(s scope, want string) string {
	if !s.exists() {
		return "MISSING directory " + s.Path
	}
	ls, err := s.linkState()
	if err != nil {
		return "ERROR " + err.Error()
	}
	envDisplay := s.display(envFile)
	switch ls.kind {
	case linkMissing:
		return fmt.Sprintf("no %s (envbuckets use <bucket>)", envDisplay)
	case linkReal:
		return envDisplay + " is a real file, not managed (envbuckets init to bootstrap)"
	case linkForeign:
		return fmt.Sprintf("%s -> %s (outside %s/, not managed)", envDisplay, ls.target, bucketsDir)
	}
	if ls.dangling {
		return fmt.Sprintf("%s BROKEN - %s is missing (create it, or envbuckets use <bucket>)", ls.bucket, s.display(linkTarget(ls.bucket)))
	}
	switch {
	case want == "":
		return ls.bucket
	case ls.bucket == want:
		return ls.bucket + " (ok)"
	default:
		return fmt.Sprintf("%s (manual override; rule wants %s)", ls.bucket, want)
	}
}
