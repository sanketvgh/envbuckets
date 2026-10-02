package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sanketvgh/envbuckets/internal/config"
)

func runBucket(args []string, env Env) error {
	sub, rest, err := subcommand("bucket", args)
	if err != nil {
		return err
	}
	flags := newFlags("bucket " + sub)
	scopeName := flags.String("scope", "", "target scope by name")
	var purge, all *bool
	switch sub {
	case "rm":
		purge = flags.Bool("purge", false, "delete a non-empty bucket after typed confirmation")
	case "add", "list":
		all = flags.Bool("all", false, "every configured scope")
	}
	rest, err = parseFlags(flags, rest)
	if err != nil {
		return err
	}
	p, err := openProject(env)
	if err != nil {
		return err
	}
	switch sub {
	case "add", "rm":
		if len(rest) != 1 {
			return usage("bucket %s: expected exactly one name", sub).then("envbuckets bucket %s <name>", sub)
		}
	case "list":
		if len(rest) != 0 {
			return usage("bucket list: takes no arguments").then("envbuckets bucket list [--all]")
		}
	default:
		return usage("bucket: unknown subcommand %q", sub).then("envbuckets bucket add|rm|list")
	}
	if all != nil && *all && *scopeName != "" {
		return usage("bucket %s: --all and --scope cannot be used together", sub).then("envbuckets bucket %s --all", sub)
	}
	if all != nil && *all {
		if sub == "add" {
			return bucketAddAll(env, p, rest[0])
		}
		return bucketListAll(env, p)
	}
	s, err := p.resolveScope(env, *scopeName)
	if err != nil {
		return err
	}
	switch sub {
	case "add":
		return bucketAdd(env, s, rest[0])
	case "rm":
		return bucketRm(env, p, s, rest[0], *purge)
	default:
		return bucketList(env, p, s)
	}
}

func bucketAdd(env Env, s scope, name string) error {
	if err := config.ValidateName(name); err != nil {
		return blocked("%v", err)
	}
	created, err := createBucket(s, name)
	if err != nil {
		return err
	}
	if !created {
		fmt.Fprintf(env.Stdout, "%s: bucket %s already exists (%s)\n  next: envbuckets use %s\n", s.Name, name, s.display(linkTarget(name)), name)
		return nil
	}
	fmt.Fprintf(env.Stdout, "%s: created bucket %s (empty %s)\n  next: fill %s in your editor, then: envbuckets use %s\n",
		s.Name, name, s.display(linkTarget(name)), s.display(linkTarget(name)), name)
	return nil
}

func createBucket(s scope, name string) (bool, error) {
	if err := s.checkScopeDir(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, blocked("%s: scope directory %s is missing, not created", s.Name, s.Path).
				then("restore %s, or envbuckets scope rm %s", s.Path, s.Name)
		}
		return false, err
	}
	if err := s.checkBucketParents(name); err != nil {
		return false, err
	}
	if s.bucketFileExists(name) {
		return false, nil
	}
	root, err := os.OpenRoot(s.Root)
	if err != nil {
		return false, err
	}
	defer root.Close()
	if err := root.MkdirAll(s.bucketRel(name), 0o755); err != nil {
		return false, err
	}
	if err := s.checkBucketParents(name); err != nil {
		return false, err
	}
	f, err := root.OpenFile(filepath.Join(s.bucketRel(name), envFile), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, os.ErrExist) {
		return false, blocked("%s exists but is not a regular file, left untouched", s.display(linkTarget(name))).
			then("inspect %s by hand", s.display(linkTarget(name)))
	}
	if err != nil {
		return false, err
	}
	return true, f.Close()
}

func bucketRm(env Env, p *project, s scope, name string, purge bool) error {
	if err := config.ValidateName(name); err != nil {
		return blocked("%v", err)
	}
	if err := s.checkBucketParents(name); err != nil {
		return err
	}
	root, err := os.OpenRoot(s.Root)
	if err != nil {
		return err
	}
	defer root.Close()
	dir := s.bucketRel(name)
	if _, err := root.Lstat(dir); errors.Is(err, os.ErrNotExist) {
		return blocked("%s: no bucket named %s", s.Name, name).then("envbuckets bucket list")
	} else if err != nil {
		return err
	}
	if refs := p.cfg.RulesFor(name); len(refs) > 0 {
		return blocked("bucket %s is still referenced by rules: %s", name, joinOr(refs, "")).
			then("envbuckets map rm <pattern> for each, then retry")
	}
	if branches, err := linkedBranches(p.root, name); err != nil {
		return err
	} else if len(branches) > 0 {
		return blocked("bucket %s is linked by branches: %s", name, joinOr(branches, "")).
			then("envbuckets unlink --branch <name> for each, then retry")
	}
	ls, err := s.linkState()
	if err != nil {
		return err
	}
	if ls.kind == linkBucket && ls.bucket == name {
		return blocked("bucket %s is the active bucket in scope %s", name, s.Name).then("envbuckets use <other-bucket>, then retry")
	}
	needsConfirmation, err := bucketHasData(root, dir)
	if err != nil {
		return err
	}
	if needsConfirmation {
		if !purge {
			return blocked("%s contains data or non-bucket files, refusing to delete it", s.display(bucketsDir+"/"+name)).
				then("envbuckets bucket rm %s --purge (asks for confirmation)", name)
		}
		if err := confirmDelete(env, s.display(bucketsDir+"/"+name)+"/"); err != nil {
			return err
		}
	}
	if err := root.RemoveAll(dir); err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "%s: removed bucket %s\n", s.Name, name)
	return nil
}

func bucketHasData(root *os.Root, dir string) (bool, error) {
	f, err := root.Open(dir)
	if err != nil {
		return false, err
	}
	entries, readErr := f.ReadDir(-1)
	closeErr := f.Close()
	if readErr != nil {
		return false, readErr
	}
	if closeErr != nil {
		return false, closeErr
	}
	for _, entry := range entries {
		if entry.Name() != envFile {
			return true, nil
		}
		info, err := root.Lstat(filepath.Join(dir, envFile))
		if err != nil {
			return false, err
		}
		if !info.Mode().IsRegular() || info.Size() != 0 {
			return true, nil
		}
	}
	return false, nil
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
