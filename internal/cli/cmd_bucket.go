package cli

import (
	"fmt"
	"os"

	"github.com/sanketvgh/envbuckets/internal/config"
)

func runBucket(args []string, env Env) error {
	sub, rest, err := subcommand("bucket", args)
	if err != nil {
		return err
	}
	flags := newFlags("bucket " + sub)
	scopeName := flags.String("scope", "", "target scope by name")
	purge := flags.Bool("purge", false, "rm: delete a non-empty bucket after typed confirmation")
	rest, err = parseFlags(flags, rest)
	if err != nil {
		return err
	}
	p, err := openProject(env)
	if err != nil {
		return err
	}
	s, err := p.resolveScope(env, *scopeName)
	if err != nil {
		return err
	}
	switch sub {
	case "add":
		if len(rest) != 1 {
			return usage("bucket add: expected exactly one name").then("envbuckets bucket add <name>")
		}
		return bucketAdd(env, s, rest[0])
	case "rm":
		if len(rest) != 1 {
			return usage("bucket rm: expected exactly one name").then("envbuckets bucket rm <name> [--purge]")
		}
		return bucketRm(env, p, s, rest[0], *purge)
	case "list":
		return bucketList(env, p, s)
	default:
		return usage("bucket: unknown subcommand %q", sub).then("envbuckets bucket add|rm|list")
	}
}

func bucketAdd(env Env, s scope, name string) error {
	if err := config.ValidateName(name); err != nil {
		return blocked("%v", err)
	}
	if s.bucketFileExists(name) {
		fmt.Fprintf(env.Stdout, "%s: bucket %s already exists (%s)\n  next: envbuckets use %s\n", s.Name, name, s.display(linkTarget(name)), name)
		return nil
	}
	if err := os.MkdirAll(s.bucketDir(name), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.bucketFile(name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "%s: created bucket %s (empty %s)\n  next: fill %s in your editor, then: envbuckets use %s\n",
		s.Name, name, s.display(linkTarget(name)), s.display(linkTarget(name)), name)
	return nil
}

func bucketRm(env Env, p *project, s scope, name string, purge bool) error {
	if err := config.ValidateName(name); err != nil {
		return blocked("%v", err)
	}
	if _, err := os.Stat(s.bucketDir(name)); os.IsNotExist(err) {
		return blocked("%s: no bucket named %s", s.Name, name).then("envbuckets bucket list")
	}
	if refs := p.cfg.RulesFor(name); len(refs) > 0 {
		return blocked("bucket %s is still referenced by rules: %s", name, joinOr(refs, "")).
			then("envbuckets map rm <pattern> for each, then retry")
	}
	ls, err := s.linkState()
	if err != nil {
		return err
	}
	if ls.kind == linkBucket && ls.bucket == name {
		return blocked("bucket %s is the active bucket in scope %s", name, s.Name).then("envbuckets use <other-bucket>, then retry")
	}
	if s.bucketFileExists(name) {
		empty, err := s.bucketFileEmpty(name)
		if err != nil {
			return err
		}
		if !empty {
			if !purge {
				return blocked("%s is not empty, refusing to delete values", s.display(linkTarget(name))).
					then("envbuckets bucket rm %s --purge (asks for confirmation)", name)
			}
			if err := confirmDelete(env, s.display(bucketsDir+"/"+name)+"/"); err != nil {
				return err
			}
		}
	}
	if err := os.RemoveAll(s.bucketDir(name)); err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "%s: removed bucket %s\n", s.Name, name)
	return nil
}

func bucketList(env Env, p *project, s scope) error {
	names, err := s.buckets()
	if err != nil {
		return err
	}
	ls, err := s.linkState()
	if err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "scope %s (%s):\n", s.Name, s.Path)
	if len(names) == 0 {
		fmt.Fprintln(env.Stdout, "  (none)\n  next: envbuckets bucket add <name>")
		return nil
	}
	for _, n := range names {
		marker := " "
		if ls.kind == linkBucket && ls.bucket == n {
			marker = "*"
		}
		state := ""
		if !s.bucketFileExists(n) {
			state = " (missing .env)"
		}
		fmt.Fprintf(env.Stdout, "  %s %-16s rules: %s%s\n", marker, n, joinOr(p.cfg.RulesFor(n), "none"), state)
	}
	return nil
}
