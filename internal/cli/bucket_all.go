package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/gitx"
)

type bulkResult struct {
	created []string
	present []string
	failed  []string
}

func (r bulkResult) summary() string {
	return fmt.Sprintf("created %d, already present %d, failed %d", len(r.created), len(r.present), len(r.failed))
}

func createInScopes(scopes []scope, name string, step func(string, string, ...any)) bulkResult {
	var res bulkResult
	for _, s := range scopes {
		file := s.display(linkTarget(name))
		created, err := createBucket(s, name)
		switch {
		case err != nil:
			var ee *exitError
			msg := err.Error()
			if errors.As(err, &ee) {
				msg = strings.TrimPrefix(ee.msg, s.Name+": ")
			}
			step("failed", "%s: %s", s.Name, msg)
			res.failed = append(res.failed, file)
		case created:
			step("created", "%s: %s (empty)", s.Name, file)
			res.created = append(res.created, file)
		default:
			step("ok", "%s: %s already exists, left untouched", s.Name, file)
			res.present = append(res.present, file)
		}
	}
	return res
}

func bucketAddAll(env Env, p *project, name string) error {
	if err := config.ValidateName(name); err != nil {
		return blocked("%v", err)
	}
	fmt.Fprintf(env.Stdout, "bucket %s in every scope:\n", name)
	res := createInScopes(p.scopes, name, stepPrinter(env))
	fmt.Fprintln(env.Stdout, res.summary())
	if len(res.failed) > 0 {
		return blocked("bucket %s was not created in %d of %d scopes", name, len(res.failed), len(p.scopes)).
			then("fix the failed scopes above, then re-run envbuckets bucket add %s --all", name)
	}
	if len(res.created) > 0 {
		fmt.Fprintln(env.Stdout, "  next: fill the new files in your editor, then: envbuckets check")
	}
	return nil
}

func stepPrinter(env Env) func(string, string, ...any) {
	return func(tag, format string, a ...any) {
		fmt.Fprintf(env.Stdout, "  [%s] %s\n", tag, fmt.Sprintf(format, a...))
	}
}

func bucketListAll(env Env, p *project) error {
	links, err := gitx.Links(p.root)
	if err != nil {
		return envErr("cannot read branch links: %v", err).then("check git config --local --list")
	}
	refs := map[string][]string{}
	for _, r := range p.cfg.Rules {
		refs[r.Bucket] = append(refs[r.Bucket], "rule "+r.Pattern)
	}
	for _, l := range links {
		if config.ValidateName(l.Bucket) == nil {
			refs[l.Bucket] = append(refs[l.Bucket], "pin "+l.Branch)
		}
	}
	names := map[string]bool{}
	for b := range refs {
		names[b] = true
	}
	states := make([]linkState, len(p.scopes))
	var notes []string
	for i, s := range p.scopes {
		if !s.exists() {
			notes = append(notes, fmt.Sprintf("%s: directory %s is missing", s.Name, s.Path))
			continue
		}
		onDisk, err := s.buckets()
		if err != nil {
			return err
		}
		for _, b := range onDisk {
			names[b] = true
		}
		ls, err := s.linkState()
		if err != nil {
			return err
		}
		states[i] = ls
		if ls.kind == linkBucket {
			names[ls.bucket] = true
		}
		if note := unmanagedNote(s, ls); note != "" {
			notes = append(notes, note)
		}
	}
	if len(names) == 0 {
		fmt.Fprintln(env.Stdout, "no buckets on disk or referenced by rules or pins\n  next: envbuckets bucket add <name> --all")
		return nil
	}
	sorted := make([]string, 0, len(names))
	for b := range names {
		sorted = append(sorted, b)
	}
	sort.Strings(sorted)

	tw := tabwriter.NewWriter(env.Stdout, 0, 0, 2, ' ', 0)
	header := []string{"bucket"}
	for _, s := range p.scopes {
		header = append(header, s.Name)
	}
	fmt.Fprintln(tw, strings.Join(append(header, "used by"), "\t"))
	for _, b := range sorted {
		row := []string{b}
		for i, s := range p.scopes {
			row = append(row, matrixCell(s, states[i], b))
		}
		fmt.Fprintln(tw, strings.Join(append(row, joinOr(refs[b], "-")), "\t"))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	fmt.Fprintln(env.Stdout, "present: file exists, contents not checked | active: .env points here | BROKEN: .env points here, file missing")
	for _, n := range notes {
		fmt.Fprintf(env.Stdout, "note: %s\n", n)
	}
	return nil
}

func matrixCell(s scope, ls linkState, bucket string) string {
	if !s.exists() {
		return "no dir"
	}
	if ls.kind == linkBucket && ls.bucket == bucket {
		if ls.dangling {
			return "BROKEN"
		}
		return "active"
	}
	if s.bucketFileExists(bucket) {
		return "present"
	}
	return "missing"
}

func unmanagedNote(s scope, ls linkState) string {
	switch ls.kind {
	case linkReal:
		return fmt.Sprintf("%s: %s is a real file, not managed", s.Name, s.display(envFile))
	case linkForeign:
		return fmt.Sprintf("%s: %s -> %s, outside %s/, not managed", s.Name, s.display(envFile), ls.target, bucketsDir)
	case linkMissing:
		return fmt.Sprintf("%s: no %s link yet", s.Name, s.display(envFile))
	}
	return ""
}
