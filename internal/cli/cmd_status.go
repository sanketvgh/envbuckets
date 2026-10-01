package cli

import (
	"fmt"

	"github.com/sanketvgh/envbuckets/internal/gitx"
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

	var t target
	if branch == "" {
		fmt.Fprintln(env.Stdout, "branch: (detached HEAD), the hook leaves .env untouched\n  next: git checkout <branch>, or envbuckets use <bucket>")
	} else {
		var found bool
		t, found, err = p.target(branch)
		switch {
		case err != nil:
			return envErr("cannot read the branch link: %v", err).then("envbuckets unlink, or git config --local --unset branch.%s.envbuckets", branch)
		case !found:
			fmt.Fprintf(env.Stdout, "branch: %s\nrule:   none, .env is left as-is on checkout\n  next: envbuckets map add <pattern> <bucket>, or envbuckets link <bucket>\n", branch)
		case t.linked:
			fmt.Fprintf(env.Stdout, "branch: %s\nlink:   %s (local, overrides rules; envbuckets unlink to drop)\n", branch, t.bucket)
		default:
			fmt.Fprintf(env.Stdout, "branch: %s\nrule:   %s -> %s\n", branch, t.via, t.bucket)
		}
	}

	fmt.Fprintln(env.Stdout, "scopes:")
	for _, s := range p.scopes {
		fmt.Fprintf(env.Stdout, "  %-12s %s\n", s.Name, scopeStatus(s, t))
	}
	return nil
}

func scopeStatus(s scope, t target) string {
	want := t.bucket
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
		source := "rule"
		if t.linked {
			source = "link"
		}
		return fmt.Sprintf("%s (manual override; %s wants %s)", ls.bucket, source, want)
	}
}
