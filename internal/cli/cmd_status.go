package cli

import (
	"errors"
	"fmt"
	"os"
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
	if env.jsonData != nil {
		*env.jsonData = readinessJSON(r)
	}
	printReadiness(env, r)
	return nil
}

func printReadiness(env Env, r readiness) {
	switch {
	case r.branch == "":
		fmt.Fprintln(env.Stdout, "HEAD detached")
		fmt.Fprintln(env.Stdout, "No bucket selected; checkout leaves .env unchanged.")
	case !r.resolved:
		fmt.Fprintf(env.Stdout, "On branch %s\n", r.branch)
		fmt.Fprintln(env.Stdout, "No matching rule or local pin; checkout leaves .env unchanged.")
	case r.target.linked:
		fmt.Fprintf(env.Stdout, "On branch %s\n", r.branch)
		fmt.Fprintf(env.Stdout, "Using bucket %s (local pin; overrides rules)\n", r.target.bucket)
	default:
		fmt.Fprintf(env.Stdout, "On branch %s\n", r.branch)
		fmt.Fprintf(env.Stdout, "Using bucket %s (rule %s, priority %d)\n", r.target.bucket, r.target.via, r.target.priority)
	}
	fmt.Fprintln(env.Stdout)
	for _, s := range r.scopes {
		fmt.Fprintf(env.Stdout, "%-12s %s\n", s.scope.Name, scopeReadinessText(s, r.target.bucket, r.resolved))
	}
}

func scopeReadinessText(r scopeReadiness, expected string, resolved bool) string {
	s := r.scope
	if !r.directoryExists {
		if !errors.Is(r.directoryErr, os.ErrNotExist) {
			return "cannot access " + s.Path + ": " + r.directoryErr.Error()
		}
		return s.Path + " missing (scope directory)"
	}
	if r.linkErr != nil {
		return "cannot inspect " + s.display(envFile) + ": " + r.linkErr.Error()
	}
	var facts []string
	switch r.link.kind {
	case linkMissing:
		facts = append(facts, s.display(envFile)+" missing")
	case linkReal:
		facts = append(facts, s.display(envFile)+" is a real file, unmanaged")
	case linkForeign:
		facts = append(facts, fmt.Sprintf("%s -> %s (foreign, unmanaged)", s.display(envFile), r.link.target))
	case linkBucket:
		facts = append(facts, s.display(envFile)+" -> "+s.display(linkTarget(r.link.bucket)))
		if r.link.dangling {
			facts = append(facts, "target missing")
		}
	}
	if resolved {
		if !r.healthy() && r.expectedExists {
			facts = append(facts, "expected "+s.display(linkTarget(expected)))
		}
		if !r.expectedExists {
			facts = append(facts, s.display(linkTarget(expected))+" missing")
		}
	}
	return strings.Join(facts, "; ")
}

func scopeStatus(s scope, t target) string {
	return scopeReadinessText(evaluateScope(s, t.bucket, t.bucket != ""), t.bucket, t.bucket != "")
}
