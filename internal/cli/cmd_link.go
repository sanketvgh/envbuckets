package cli

import (
	"fmt"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/gitx"
	"github.com/sanketvgh/envbuckets/internal/pattern"
)

func runLink(args []string, env Env) error {
	flags := newFlags("link")
	branchName := flags.String("branch", "", "link this branch instead of the current one")
	rest, err := parseFlags(flags, args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usage("link: expected exactly one bucket name").then("envbuckets link <bucket> [--branch <name>]")
	}
	bucket := rest[0]
	if err := config.ValidateName(bucket); err != nil {
		return usage("link: %v", err)
	}
	p, err := openProject(env)
	if err != nil {
		return err
	}
	branch, current, err := p.linkBranch(*branchName)
	if err != nil {
		return err
	}
	if !p.bucketExists(bucket) {
		return blocked("bucket %s does not exist in any scope", bucket).then("envbuckets bucket add %s, then retry", bucket)
	}
	if err := gitx.SetLink(p.root, branch, bucket); err != nil {
		return envErr("cannot write the link to .git/config: %v", err).then("check git config --local --list")
	}
	fmt.Fprintf(env.Stdout, "linked %s -> %s (local, overrides rules, not committed)\n", branch, bucket)
	if !current {
		fmt.Fprintf(env.Stdout, "  next: git checkout %s\n", branch)
		return nil
	}
	p.applyNow(env, bucket)
	return nil
}

func runUnlink(args []string, env Env) error {
	flags := newFlags("unlink")
	branchName := flags.String("branch", "", "unlink this branch instead of the current one")
	rest, err := parseFlags(flags, args)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usage("unlink: takes no arguments").then("envbuckets unlink [--branch <name>]")
	}
	p, err := openProject(env)
	if err != nil {
		return err
	}
	branch, current, err := p.linkBranch(*branchName)
	if err != nil {
		return err
	}
	removed, err := gitx.Unlink(p.root, branch)
	if err != nil {
		return envErr("cannot remove the link from .git/config: %v", err).then("check git config --local --list")
	}
	if !removed {
		fmt.Fprintf(env.Stdout, "%s has no link, nothing to do\n", branch)
		return nil
	}
	fmt.Fprintf(env.Stdout, "unlinked %s, rules apply again\n", branch)
	if !current {
		return nil
	}
	rule := p.cfg.Match(branch, pattern.Match)
	if rule == nil {
		fmt.Fprintf(env.Stdout, "no rule matches %q, .env left as-is\n  next: envbuckets map add <pattern> <bucket>\n", branch)
		return nil
	}
	p.applyNow(env, rule.Bucket)
	return nil
}

func (p *project) linkBranch(name string) (branch string, current bool, err error) {
	cur, err := gitx.Branch(p.root)
	if err != nil {
		return "", false, envErr("cannot resolve the current branch: %v", err).then("check git status")
	}
	if name == "" {
		if cur == "" {
			return "", false, envErr("detached HEAD, no current branch").then("git checkout <branch>, or pass --branch <name>")
		}
		return cur, true, nil
	}
	if !gitx.BranchExists(p.root, name) {
		return "", false, envErr("no local branch named %q", name).then("git branch --list")
	}
	return name, name == cur, nil
}

func linkedBranches(root, bucket string) ([]string, error) {
	links, err := gitx.Links(root)
	if err != nil {
		return nil, envErr("cannot read branch links: %v", err).then("check git config --local --list")
	}
	var out []string
	for _, l := range links {
		if l.Bucket == bucket {
			out = append(out, l.Branch)
		}
	}
	return out, nil
}

func (p *project) applyNow(env Env, bucket string) {
	sw := p.switchAll(bucket)
	if sw.switched > 0 {
		fmt.Fprintf(env.Stdout, "%s -> %s - %s switched\n  next: restart your dev servers\n",
			joinOr(sw.previous, "?"), bucket, scopeCount(sw.switched))
	}
	for _, w := range sw.warnings {
		fmt.Fprintf(env.Stderr, "warning: %s\n", w)
	}
}
