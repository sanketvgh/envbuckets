package cli

import (
	"fmt"

	"github.com/sanketvgh/envbuckets/internal/config"
)

func runUse(args []string, env Env) error {
	flags := newFlags("use")
	scopeName := flags.String("scope", "", "target scope by name")
	rest, err := parseFlags(flags, args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usage("use: expected exactly one bucket name").then("envbuckets bucket list")
	}
	bucket := rest[0]
	if err := config.ValidateName(bucket); err != nil {
		return usage("use: %v", err)
	}
	p, err := openProject(env)
	if err != nil {
		return err
	}
	s, err := p.resolveScope(env, *scopeName)
	if err != nil {
		return err
	}
	if !s.bucketFileExists(bucket) {
		return blocked("%s does not exist", s.display(linkTarget(bucket))).then("envbuckets bucket add %s, then retry", bucket)
	}
	ls, err := s.linkState()
	if err != nil {
		return err
	}
	switch ls.kind {
	case linkReal:
		return blocked("%s is a real file, not a symlink", s.display(envFile)).then("envbuckets init moves it into a bucket safely")
	case linkForeign:
		return blocked("%s is a symlink to %s, outside %s/", s.display(envFile), ls.target, bucketsDir).then("remove that symlink by hand, then retry")
	case linkMissing, linkBucket:
	}
	changed, err := s.pointTo(bucket)
	if err != nil {
		return err
	}
	if !changed {
		fmt.Fprintf(env.Stdout, "%s: %s already -> %s\n", s.Name, s.display(envFile), linkTarget(bucket))
		return nil
	}
	fmt.Fprintf(env.Stdout, "%s: %s -> %s\n", s.Name, s.display(envFile), linkTarget(bucket))
	fmt.Fprintln(env.Stdout, "note: manual override, the next checkout matching a rule repoints it\n  next: restart your dev servers")
	return nil
}
