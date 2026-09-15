package cli

import (
	"fmt"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/gitx"
	"github.com/sanketvgh/envbuckets/internal/pattern"
)

func runMap(args []string, env Env) error {
	sub, rest, err := subcommand("map", args)
	if err != nil {
		return err
	}
	flags := newFlags("map " + sub)
	rest, err = parseFlags(flags, rest)
	if err != nil {
		return err
	}
	p, err := openProject(env)
	if err != nil {
		return err
	}
	switch sub {
	case "add":
		if len(rest) != 2 {
			return usage("map add: expected <pattern> <bucket>").then("envbuckets map add 'release/*' prod")
		}
		return mapAdd(env, p, rest[0], rest[1])
	case "rm":
		if len(rest) != 1 {
			return usage("map rm: expected <pattern>").then("envbuckets map list")
		}
		return mapRm(env, p, rest[0])
	case "list":
		return mapList(env, p)
	default:
		return usage("map: unknown subcommand %q", sub).then("envbuckets map add|rm|list")
	}
}

func mapAdd(env Env, p *project, pat, bucket string) error {
	if err := config.ValidatePattern(pat); err != nil {
		return blocked("%v", err)
	}
	if err := config.ValidateName(bucket); err != nil {
		return blocked("bucket: %v", err)
	}
	if p.cfg.HasCatchAll() {
		return blocked("a catch-all `*` rule exists and must stay last").then("envbuckets map rm '*', add the new rule, then re-add '*'")
	}
	for _, r := range p.cfg.Rules {
		if r.Pattern == pat {
			return blocked("pattern %q already maps to %s", pat, r.Bucket).then("envbuckets map rm %q first to change it", pat)
		}
	}
	exists := false
	for _, s := range p.scopes {
		if s.bucketFileExists(bucket) {
			exists = true
			break
		}
	}
	if !exists {
		return blocked("bucket %s does not exist in any scope", bucket).then("envbuckets bucket add %s, then retry", bucket)
	}
	p.cfg.Rules = append(p.cfg.Rules, config.Rule{Pattern: pat, Bucket: bucket})
	if err := p.cfg.Save(p.root); err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "added rule %s -> %s (priority %d)\n  next: commit %s, then git checkout a matching branch\n",
		pat, bucket, len(p.cfg.Rules), config.FileName)
	return nil
}

func mapRm(env Env, p *project, pat string) error {
	kept := p.cfg.Rules[:0:0]
	found := false
	for _, r := range p.cfg.Rules {
		if r.Pattern == pat {
			found = true
			continue
		}
		kept = append(kept, r)
	}
	if !found {
		return blocked("no rule with pattern %q", pat).then("envbuckets map list")
	}
	p.cfg.Rules = kept
	if err := p.cfg.Save(p.root); err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "removed rule %s\n", pat)
	return nil
}

func mapList(env Env, p *project) error {
	if len(p.cfg.Rules) == 0 {
		fmt.Fprintln(env.Stdout, "no rules\n  next: envbuckets map add <pattern> <bucket>")
		return nil
	}
	var active *config.Rule
	if branch, _ := gitx.Branch(p.root); branch != "" {
		active = p.cfg.Match(branch, pattern.Match)
	}
	for i, r := range p.cfg.Rules {
		marker := " "
		if active != nil && active.Pattern == r.Pattern {
			marker = "*"
		}
		fmt.Fprintf(env.Stdout, "%s %d. %-24s -> %s\n", marker, i+1, r.Pattern, r.Bucket)
	}
	return nil
}
