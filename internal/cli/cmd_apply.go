package cli

import "fmt"

type scopePlan struct {
	scope  scope
	bucket string
	action string
	reason string
}

type applyResult struct {
	changed   int
	unchanged int
	failed    int
}

func planScope(r scopeReadiness, bucket string) scopePlan {
	p := scopePlan{scope: r.scope, bucket: bucket}
	switch {
	case !r.directoryExists:
		p.reason = r.problems[0]
	case r.linkErr != nil:
		p.reason = fmt.Sprintf("cannot inspect .env: %v", r.linkErr)
	case r.link.kind == linkReal:
		p.reason = "real .env is unmanaged"
	case r.link.kind == linkForeign:
		p.reason = "foreign .env symlink is unmanaged"
	case !r.expectedExists:
		p.reason = "expected bucket file missing: " + r.scope.display(linkTarget(bucket))
	case r.link.kind == linkBucket && r.link.bucket == bucket && !r.link.dangling:
		p.action = "unchanged"
	default:
		p.action = "change"
	}
	return p
}

func runApply(args []string, env Env) error {
	flags := newFlags("apply")
	scopeName := flags.String("scope", "", "apply one scope by name")
	dryRun := flags.Bool("dry-run", false, "show changes without writing")
	rest, err := parseFlags(flags, args)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usage("apply: takes no arguments").then("envbuckets apply [--scope <name>] [--dry-run]")
	}
	if scopeFlagProvided(flags) && *scopeName == "" {
		return usage("apply: --scope requires a name")
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
	if r.branch == "" {
		return blocked("expected environment unresolved on detached HEAD").then("git checkout <branch>, or envbuckets use <bucket>")
	}
	if !r.resolved {
		return blocked("no mapping matches branch %q", r.branch).then("envbuckets map add <pattern> <bucket>, or envbuckets link <bucket>")
	}
	return executePlans(env, r.scopes, r.target.bucket, *dryRun, "apply")
}

func executePlans(env Env, states []scopeReadiness, bucket string, dryRun bool, command string) error {
	var result applyResult
	view := jsonPlanResult{Bucket: bucket, DryRun: dryRun, Scopes: make([]jsonPlanScope, 0, len(states))}
	for _, state := range states {
		plan := planScope(state, bucket)
		row := jsonPlanScope{Name: plan.scope.Name}
		switch plan.action {
		case "unchanged":
			result.unchanged++
			row.Action = "unchanged"
			fmt.Fprintf(env.Stdout, "%s: unchanged (%s)\n", plan.scope.Name, bucket)
		case "change":
			if dryRun {
				result.changed++
				row.Action = "would_change"
				fmt.Fprintf(env.Stdout, "%s: would point %s -> %s\n", plan.scope.Name, plan.scope.display(envFile), plan.scope.display(linkTarget(bucket)))
				view.Scopes = append(view.Scopes, row)
				continue
			}
			changed, err := plan.scope.pointTo(bucket)
			if err != nil {
				result.failed++
				row.Action, row.Reason = "failed", err.Error()
				fmt.Fprintf(env.Stdout, "%s: failed: %v\n", plan.scope.Name, err)
				view.Scopes = append(view.Scopes, row)
				continue
			}
			if changed {
				result.changed++
				row.Action = "changed"
				fmt.Fprintf(env.Stdout, "%s: %s -> %s\n", plan.scope.Name, plan.scope.display(envFile), plan.scope.display(linkTarget(bucket)))
			} else {
				result.unchanged++
				row.Action = "unchanged"
				fmt.Fprintf(env.Stdout, "%s: unchanged (%s)\n", plan.scope.Name, bucket)
			}
		default:
			result.failed++
			row.Action, row.Reason = "blocked", plan.reason
			fmt.Fprintf(env.Stdout, "%s: blocked: %s\n", plan.scope.Name, plan.reason)
		}
		view.Scopes = append(view.Scopes, row)
	}
	if env.jsonData != nil {
		view.Changed, view.Unchanged, view.Failed = result.changed, result.unchanged, result.failed
		*env.jsonData = view
	}
	label := command
	if dryRun {
		label = "dry-run"
	}
	fmt.Fprintf(env.Stdout, "%s: %d changed, %d unchanged, %d failed\n", label, result.changed, result.unchanged, result.failed)
	if result.failed > 0 {
		return blocked("%d scope(s) could not be applied", result.failed).then("repair the reported paths, then retry")
	}
	if result.changed > 0 && !dryRun {
		fmt.Fprintln(env.Stdout, "  next: restart your dev servers")
	}
	return nil
}
