package cli

import (
	"fmt"
	"strings"
)

func runStatus(args []string, env Env) error {
	flags := newFlags("status")
	rest, err := parseFlags(flags, args)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usage("status: takes no arguments").then("envbuckets status")
	}
	p, err := openProject(env)
	if err != nil {
		return err
	}
	r, err := p.evaluateCurrent(p.scopes)
	if err != nil {
		return err
	}
	printReadiness(env, r)
	return nil
}

func printReadiness(env Env, r readiness) {
	switch {
	case r.branch == "":
		fmt.Fprintln(env.Stdout, "branch: (detached HEAD)\nexpected: unresolved; checkout leaves .env as-is")
	case !r.resolved:
		fmt.Fprintf(env.Stdout, "branch: %s\nrule:   none; checkout leaves .env as-is\n", r.branch)
	case r.target.linked:
		fmt.Fprintf(env.Stdout, "branch: %s\nlink:   %s (local, overrides rules; envbuckets unlink to drop)\n", r.branch, r.target.bucket)
	default:
		fmt.Fprintf(env.Stdout, "branch: %s\nrule:   %s -> %s (priority %d)\n", r.branch, r.target.via, r.target.bucket, r.target.priority)
	}
	fmt.Fprintln(env.Stdout, "scopes:")
	for _, s := range r.scopes {
		fmt.Fprintf(env.Stdout, "  %-12s %s\n", s.scope.Name, scopeReadinessText(s, r.target.bucket, r.resolved))
	}
}

func scopeReadinessText(r scopeReadiness, expected string, resolved bool) string {
	s := r.scope
	if !r.directoryExists {
		return "MISSING directory " + s.Path
	}
	if r.linkErr != nil {
		return "ERROR " + r.linkErr.Error()
	}
	var facts []string
	switch r.link.kind {
	case linkMissing:
		facts = append(facts, "no "+s.display(envFile)+" (managed link missing)")
	case linkReal:
		facts = append(facts, s.display(envFile)+" is a real file, unmanaged")
	case linkForeign:
		facts = append(facts, fmt.Sprintf("%s -> %s (foreign, unmanaged)", s.display(envFile), r.link.target))
	case linkBucket:
		facts = append(facts, "active: "+r.link.bucket)
		if r.link.dangling {
			facts = append(facts, "BROKEN active link")
		}
	}
	if resolved {
		expectedText := "expected: " + expected
		if r.healthy() {
			expectedText += " (ok)"
		}
		facts = append(facts, expectedText)
		if !r.expectedExists {
			facts = append(facts, "MISSING "+s.display(linkTarget(expected))+" (create it)")
		}
	}
	return strings.Join(facts, "; ")
}

func scopeStatus(s scope, t target) string {
	return scopeReadinessText(evaluateScope(s, t.bucket, t.bucket != ""), t.bucket, t.bucket != "")
}
