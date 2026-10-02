package cli

import (
	"fmt"

	"github.com/sanketvgh/envbuckets/internal/config"
)

func ruleBuckets(rules []config.Rule) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range rules {
		if !seen[r.Bucket] {
			seen[r.Bucket] = true
			out = append(out, r.Bucket)
		}
	}
	return out
}

func scaffoldScopes(env Env, p *project, bootstrapped map[string]bool, step func(string, string, ...any)) error {
	buckets := ruleBuckets(p.cfg.Rules)
	if len(buckets) == 0 {
		step("skipped", "scaffold: no shared rules reference a bucket yet")
		return nil
	}
	var total bulkResult
	var waiting []string
	for _, s := range p.scopes {
		switch {
		case !s.exists():
			step("skipped", "%s: scope directory %s missing, not created", s.Name, s.Path)
			waiting = append(waiting, s.Name)
			continue
		case !bootstrapped[s.Name]:
			step("skipped", "%s: scaffold waits until the blocked bootstrap above is resolved", s.Name)
			waiting = append(waiting, s.Name)
			continue
		}
		ls, err := s.linkState()
		if err != nil {
			return err
		}
		if ls.kind == linkReal {
			step("skipped", "%s: %s is still a real file, scaffold waits so it cannot block the move (envbuckets init --scaffold --into <bucket>)", s.Name, s.display(envFile))
			waiting = append(waiting, s.Name)
			continue
		}
		for _, b := range buckets {
			res := createInScopes([]scope{s}, b, step)
			total.created = append(total.created, res.created...)
			total.present = append(total.present, res.present...)
			total.failed = append(total.failed, res.failed...)
		}
	}
	fmt.Fprintf(env.Stdout, "Buckets from shared rules: %s\n", joinOr(buckets, ""))
	fmt.Fprintln(env.Stdout, total.summary())
	if len(waiting) > 0 {
		fmt.Fprintf(env.Stdout, "  Not scaffolded: %s (see skipped scopes above)\n", joinOr(waiting, ""))
	}
	if len(total.failed) > 0 {
		return blocked("scaffold incomplete: %d files not created", len(total.failed)).
			then("fix the [failed] lines above, then re-run envbuckets init --scaffold")
	}
	return nil
}
