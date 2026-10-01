package cli

import "fmt"

func runCheck(args []string, env Env) error {
	flags := newFlags("check")
	scopeName := flags.String("scope", "", "check one scope by name")
	rest, err := parseFlags(flags, args)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usage("check: takes no arguments").then("envbuckets check [--scope <name>]")
	}
	if flagProvided(flags, "scope") && *scopeName == "" {
		return usage("check: --scope requires a name")
	}
	p, err := openProject(env)
	if err != nil {
		return err
	}
	selected, err := p.selectScopes(env, *scopeName, false, true)
	if err != nil {
		return err
	}
	r, err := p.evaluateCurrent(selected)
	if err != nil {
		return err
	}
	printReadiness(env, r)
	if r.healthy() {
		fmt.Fprintf(env.Stdout, "check: %d ready\n", len(r.scopes))
		return nil
	}
	ready := 0
	for _, s := range r.scopes {
		if s.healthy() && r.resolved {
			ready++
		}
	}
	fmt.Fprintf(env.Stdout, "check: %d ready, %d unhealthy or unresolved\n", ready, len(r.scopes)-ready)
	switch {
	case r.branch == "":
		return blocked("expected environment unresolved on detached HEAD").then("git checkout <branch>, or envbuckets use <bucket>")
	case !r.resolved:
		return blocked("no mapping matches branch %q", r.branch).then("envbuckets map add <pattern> <bucket>, or envbuckets link <bucket>")
	default:
		return blocked("selected scopes are not structurally ready").then("repair the reported paths, then envbuckets apply")
	}
}
