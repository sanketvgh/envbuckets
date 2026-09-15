package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sanketvgh/envbuckets/internal/config"
)

func runScope(args []string, env Env) error {
	sub, rest, err := subcommand("scope", args)
	if err != nil {
		return err
	}
	flags := newFlags("scope " + sub)
	name := flags.String("name", "", "add: scope name (default: directory name)")
	into := flags.String("into", "", "add: bucket to move an existing real .env into")
	purge := flags.Bool("purge", false, "rm: also delete the scope's .env.d/ and its .gitignore lines (asks for DELETE)")
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
		if len(rest) != 1 {
			return usage("scope add: expected <path>").then("envbuckets scope add apps/api --name api")
		}
		return scopeAdd(env, p, rest[0], *name, *into)
	case "rm":
		if len(rest) != 1 {
			return usage("scope rm: expected <name>").then("envbuckets scope list")
		}
		return scopeRm(env, p, rest[0], *purge)
	case "list":
		return scopeList(env, p)
	default:
		return usage("scope: unknown subcommand %q", sub).then("envbuckets scope add|rm|list")
	}
}

func scopeAdd(env Env, p *project, arg, name, into string) error {
	if filepath.IsAbs(arg) {
		return blocked("scope path must be relative, got %s", arg).then("envbuckets scope add apps/api")
	}
	abs := filepath.Join(env.Cwd, arg)
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return blocked("%s is not an existing directory", arg).then("mkdir it first, or check the path")
	}
	rel, err := relPath(p.root, abs)
	if err != nil {
		return blocked("%s resolves outside the repo", arg).then("use a directory inside the repo")
	}
	if name == "" {
		name = scopeNameFor(rel)
	}
	if err := config.ValidateName(name); err != nil {
		return blocked("%v", err)
	}
	for _, s := range p.cfg.Scopes {
		existing := config.CleanScopePath(s.Path)
		switch {
		case s.Name == name:
			return blocked("scope name %q is already used by %s", name, existing).then("envbuckets scope add %s --name <other>", arg)
		case existing == rel:
			return blocked("path %s is already registered as scope %q", rel, s.Name).then("envbuckets scope list")
		case nested(existing, rel):
			return blocked("%s nests inside or around scope %q (%s), nested scopes are not supported", rel, s.Name, existing).
				then("register the parent or the child directory, not both")
		}
	}
	if into != "" {
		if err := config.ValidateName(into); err != nil {
			return usage("--into: %v", err)
		}
	}

	wasImplicit := len(p.cfg.Scopes) == 0
	p.cfg.Scopes = append(p.cfg.Scopes, config.Scope{Name: name, Path: rel})
	if err := p.cfg.Save(p.root); err != nil {
		return err
	}
	p.reloadScopes()
	if _, err := ensureIgnored(p.root, ignoreLines(p.scopes)); err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "registered scope %s (%s)\n  next: envbuckets status\n", name, rel)
	if wasImplicit && rel != "." {
		fmt.Fprintln(env.Stdout, "note: the repo root is no longer an implicit scope\n  next: if the root has its own .env, run: envbuckets scope add . --name root")
	}
	s, _ := p.scopeByName(name)
	step := func(tag, format string, a ...any) {
		fmt.Fprintf(env.Stdout, "  [%s] %s\n", tag, fmt.Sprintf(format, a...))
	}
	return bootstrapScope(env, s, into, step)
}

func nested(a, b string) bool {
	if a == "." || b == "." {
		return false
	}
	return withinScope(a, b) || withinScope(b, a)
}

func scopeRm(env Env, p *project, name string, purge bool) error {
	idx := -1
	for i, s := range p.cfg.Scopes {
		if s.Name == name {
			idx = i
		}
	}
	if idx < 0 {
		return blocked("no scope named %q", name).then("envbuckets scope list")
	}
	s, _ := p.scopeByName(name)
	if purge {
		if err := confirmDelete(env, s.display(bucketsDir)+"/ for scope "+name); err != nil {
			return err
		}
		if err := os.RemoveAll(s.bucketsPath()); err != nil {
			return err
		}
	}
	p.cfg.Scopes = append(p.cfg.Scopes[:idx], p.cfg.Scopes[idx+1:]...)
	if err := p.cfg.Save(p.root); err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "unregistered scope %s (%s)\n", name, s.Path)
	if !purge {
		fmt.Fprintf(env.Stdout, "kept: %s/ and its .gitignore lines, so values never become git-visible\n  next: to erase them: envbuckets scope rm %s --purge\n", s.display(bucketsDir), name)
		return nil
	}
	if _, err := dropIgnored(p.root, scopeIgnoreLines(s)); err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "removed: %s/ and its .gitignore lines\n", s.display(bucketsDir))
	return nil
}

func scopeList(env Env, p *project) error {
	if len(p.cfg.Scopes) == 0 {
		fmt.Fprintln(env.Stdout, "no scopes declared, the repo root is the implicit scope\n  next: for a monorepo: envbuckets scope add <path> --name <name>")
	}
	for _, s := range p.scopes {
		fmt.Fprintf(env.Stdout, "  %-12s %-20s %s\n", s.Name, s.Path, scopeStatus(s, ""))
	}
	return nil
}
