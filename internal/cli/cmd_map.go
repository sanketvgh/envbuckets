package cli

import (
	"errors"
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
	var before, after *string
	if sub == "move" {
		before = flags.String("before", "", "place the rule directly before this pattern")
		after = flags.String("after", "", "place the rule directly after this pattern")
	}
	rest, err = parseFlags(flags, rest)
	if err != nil {
		return err
	}
	switch sub {
	case "add", "update":
		if len(rest) != 2 {
			return usage("map %s: expected <pattern> <bucket>", sub).then("envbuckets map %s 'release/*' prod", sub)
		}
	case "explain":
		if len(rest) != 1 {
			return usage("map explain: expected one <branch>").then("envbuckets map explain feature/login")
		}
	case "rm", "move":
		if len(rest) != 1 {
			return usage("map %s: expected one <pattern>", sub).then("envbuckets map list")
		}
	case "list":
		if len(rest) != 0 {
			return usage("map list: takes no arguments").then("envbuckets map list")
		}
	default:
		return usage("map: unknown subcommand %q", sub).then("envbuckets map add|update|move|rm|list|explain")
	}
	if sub == "move" && (*before == "") == (*after == "") {
		return usage("map move: pass exactly one of --before or --after").then("envbuckets map move %q --before <pattern>", rest[0])
	}
	p, err := openProject(env)
	if err != nil {
		return err
	}
	switch sub {
	case "add":
		return mapAdd(env, p, rest[0], rest[1])
	case "update":
		return mapUpdate(env, p, rest[0], rest[1])
	case "move":
		return mapMove(env, p, rest[0], *before, *after)
	case "rm":
		return mapRm(env, p, rest[0])
	case "explain":
		return mapExplain(env, p, rest[0])
	default:
		return mapList(env, p)
	}
}

func ruleErr(err error) error {
	switch {
	case errors.Is(err, config.ErrDuplicateRule):
		return blocked("%v", err).then("envbuckets map update <pattern> <bucket> to change its bucket")
	case errors.Is(err, config.ErrNoRule):
		return blocked("%v", err).then("envbuckets map list")
	case errors.Is(err, config.ErrCatchAllOrder):
		return blocked("the catch-all `*` rule must stay last").then("envbuckets map list")
	case errors.Is(err, config.ErrSelfReference):
		return usage("map move: %v", err).then("envbuckets map move <pattern> --before <other-pattern>")
	default:
		return blocked("%v", err)
	}
}

func requireBucket(p *project, bucket string) error {
	if err := config.ValidateName(bucket); err != nil {
		return blocked("bucket: %v", err)
	}
	if !p.bucketExists(bucket) {
		return blocked("bucket %s does not exist in any scope", bucket).then("envbuckets bucket add %s, then retry", bucket)
	}
	return nil
}

func mapAdd(env Env, p *project, pat, bucket string) error {
	if err := config.ValidatePattern(pat); err != nil {
		return blocked("%v", err)
	}
	if i := p.cfg.RuleIndex(pat); i >= 0 {
		return ruleErr(fmt.Errorf("%w: %q already maps to %s", config.ErrDuplicateRule, pat, p.cfg.Rules[i].Bucket))
	}
	if err := requireBucket(p, bucket); err != nil {
		return err
	}
	at, err := p.cfg.AddRule(config.Rule{Pattern: pat, Bucket: bucket})
	if err != nil {
		return ruleErr(err)
	}
	if err := p.cfg.Save(p.root); err != nil {
		return err
	}
	where := ""
	if pat != config.CatchAll && p.cfg.HasCatchAll() {
		where = ", before the catch-all `*`"
	}
	fmt.Fprintf(env.Stdout, "added rule %s -> %s (priority %d%s)\n  next: commit %s, then git checkout a matching branch\n",
		pat, bucket, at+1, where, config.FileName)
	return nil
}

func mapUpdate(env Env, p *project, pat, bucket string) error {
	if p.cfg.RuleIndex(pat) < 0 {
		return ruleErr(fmt.Errorf("%w: %q", config.ErrNoRule, pat))
	}
	if err := requireBucket(p, bucket); err != nil {
		return err
	}
	prev, at, err := p.cfg.UpdateRule(pat, bucket)
	if err != nil {
		return ruleErr(err)
	}
	if prev == bucket {
		fmt.Fprintf(env.Stdout, "rule %s already -> %s (priority %d), nothing to do\n", pat, bucket, at+1)
		return nil
	}
	if err := p.cfg.Save(p.root); err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "updated rule %s: %s -> %s (priority %d)\n  next: commit %s, then envbuckets apply to switch now\n",
		pat, prev, bucket, at+1, config.FileName)
	return nil
}

func mapMove(env Env, p *project, pat, before, after string) error {
	anchor := before
	if after != "" {
		anchor = after
	}
	changed, err := p.cfg.MoveRule(pat, anchor, after != "")
	if err != nil {
		return ruleErr(err)
	}
	if !changed {
		fmt.Fprintf(env.Stdout, "rule %s is already there, order unchanged\n", pat)
		printRules(env, p.cfg.Rules, pat)
		return nil
	}
	if err := p.cfg.Save(p.root); err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "moved rule %s, new order (first match wins):\n", pat)
	printRules(env, p.cfg.Rules, pat)
	fmt.Fprintf(env.Stdout, "  next: commit %s, then envbuckets apply to switch now\n", config.FileName)
	return nil
}

func printRules(env Env, rules []config.Rule, mark string) {
	for i, r := range rules {
		marker := " "
		if r.Pattern == mark {
			marker = ">"
		}
		fmt.Fprintf(env.Stdout, "%s %d. %-24s -> %s\n", marker, i+1, r.Pattern, r.Bucket)
	}
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
	links, err := gitx.Links(p.root)
	if err != nil {
		return envErr("cannot read branch links: %v", err).then("check git config --local --list")
	}
	branch, _ := gitx.Branch(p.root)
	linked := false
	for _, l := range links {
		linked = linked || l.Branch == branch
	}
	if len(p.cfg.Rules) == 0 {
		fmt.Fprintln(env.Stdout, "no rules\n  next: envbuckets map add <pattern> <bucket>")
	}
	var active *config.Rule
	if branch != "" && !linked {
		active = p.cfg.Match(branch, pattern.Match)
	}
	for i, r := range p.cfg.Rules {
		marker := " "
		if active != nil && active.Pattern == r.Pattern {
			marker = "*"
		}
		fmt.Fprintf(env.Stdout, "%s %d. %-24s -> %s\n", marker, i+1, r.Pattern, r.Bucket)
	}
	if len(links) > 0 {
		fmt.Fprintln(env.Stdout, "links (local, override rules):")
		for _, l := range links {
			marker := " "
			if l.Branch == branch {
				marker = "*"
			}
			fmt.Fprintf(env.Stdout, "%s    %-24s -> %s\n", marker, l.Branch, l.Bucket)
		}
	}
	return nil
}
