package cli

import (
	"strings"
	"testing"

	"github.com/sanketvgh/envbuckets/internal/pattern"
)

func mapRepo(t *testing.T) *repo {
	t.Helper()
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.ok("init")
	for _, b := range []string{"dev", "staging", "prod"} {
		r.ok("bucket", "add", b)
	}
	r.ok("map", "add", "main", "staging")
	r.ok("map", "add", "release/*", "prod")
	r.ok("map", "add", "*", "dev")
	return r
}

func ruleOrder(t *testing.T, r *repo) string {
	t.Helper()
	var lines []string
	for _, l := range strings.Split(r.ok("map", "list").stdout, "\n") {
		if f := strings.Fields(l); len(f) >= 4 && f[len(f)-2] == "->" {
			lines = append(lines, f[len(f)-3]+"="+f[len(f)-1])
		}
	}
	return strings.Join(lines, " ")
}

func TestMapAddInsertsBeforeCatchAll(t *testing.T) {
	r := mapRepo(t)
	res := r.ok("map", "add", "hotfix/*", "prod")
	if !strings.Contains(res.stdout, "priority 3, before the catch-all") {
		t.Fatalf("add:\n%s", res.stdout)
	}
	if got := ruleOrder(t, r); got != "main=staging release/*=prod hotfix/*=prod *=dev" {
		t.Fatalf("order: %s", got)
	}
	if got := r.git("rev-parse", "--abbrev-ref", "HEAD"); got != "main" {
		t.Fatalf("branch changed: %s", got)
	}
}

func TestMapAddRefusalsLeaveConfigUnchanged(t *testing.T) {
	r := mapRepo(t)
	before := r.read(".envbuckets.toml")
	cases := map[string][]string{
		"duplicate":      {"map", "add", "release/*", "dev"},
		"second *":       {"map", "add", "*", "prod"},
		"missing bucket": {"map", "add", "qa/*", "qa"},
		"bad bucket":     {"map", "add", "qa/*", "../x"},
	}
	for name, args := range cases {
		if res := r.run(args...); res.code != ExitBlocked {
			t.Fatalf("%s: exit %d\n%s", name, res.code, res.all())
		}
	}
	res := r.run("map", "add", "release/*", "dev")
	if !strings.Contains(res.stderr, "already maps to prod") || !strings.Contains(res.stderr, "next: envbuckets map update") {
		t.Fatalf("duplicate hint:\n%s", res.stderr)
	}
	if r.read(".envbuckets.toml") != before {
		t.Fatal("refused add changed the config")
	}
}

func TestMapUpdate(t *testing.T) {
	r := mapRepo(t)
	res := r.ok("map", "update", "release/*", "staging")
	if !strings.Contains(res.stdout, "updated rule release/*: prod -> staging (priority 2)") {
		t.Fatalf("update:\n%s", res.stdout)
	}
	if got := ruleOrder(t, r); got != "main=staging release/*=staging *=dev" {
		t.Fatalf("order: %s", got)
	}
	before := r.read(".envbuckets.toml")
	if res := r.ok("map", "update", "release/*", "staging"); !strings.Contains(res.stdout, "nothing to do") {
		t.Fatalf("no-op:\n%s", res.stdout)
	}
	for _, args := range [][]string{
		{"map", "update", "nope", "dev"},
		{"map", "update", "main", "qa"},
		{"map", "update", "main", "bad/name"},
	} {
		if res := r.run(args...); res.code != ExitBlocked {
			t.Fatalf("%v: exit %d\n%s", args, res.code, res.all())
		}
	}
	if res := r.run("map", "update", "main"); res.code != ExitUsage {
		t.Fatalf("missing bucket arg: %d", res.code)
	}
	if r.read(".envbuckets.toml") != before {
		t.Fatal("refused update changed the config")
	}
}

func TestMapMove(t *testing.T) {
	r := mapRepo(t)
	r.ok("map", "add", "feature/*", "dev")
	res := r.ok("map", "move", "feature/*", "--before", "main")
	if !strings.Contains(res.stdout, "moved rule feature/*") || !strings.Contains(res.stdout, "> 1. feature/*") {
		t.Fatalf("move:\n%s", res.stdout)
	}
	if got := ruleOrder(t, r); got != "feature/*=dev main=staging release/*=prod *=dev" {
		t.Fatalf("order: %s", got)
	}
	r.ok("map", "move", "feature/*", "--after", "release/*")
	if got := ruleOrder(t, r); got != "main=staging release/*=prod feature/*=dev *=dev" {
		t.Fatalf("order: %s", got)
	}
	if res := r.ok("map", "move", "main", "--before", "release/*"); !strings.Contains(res.stdout, "order unchanged") {
		t.Fatalf("no-op move:\n%s", res.stdout)
	}
}

func TestMapMoveRefusals(t *testing.T) {
	r := mapRepo(t)
	before := r.read(".envbuckets.toml")
	cases := []struct {
		args []string
		code int
		want string
	}{
		{[]string{"map", "move", "main", "--after", "*"}, ExitBlocked, "catch-all `*` rule must stay last"},
		{[]string{"map", "move", "*", "--before", "main"}, ExitBlocked, "catch-all `*` rule must stay last"},
		{[]string{"map", "move", "main", "--before", "main"}, ExitUsage, "relative to itself"},
		{[]string{"map", "move", "nope", "--before", "main"}, ExitBlocked, `no such rule: "nope"`},
		{[]string{"map", "move", "main", "--before", "nope"}, ExitBlocked, `no such rule: "nope"`},
		{[]string{"map", "move", "main"}, ExitUsage, "exactly one of --before or --after"},
		{[]string{"map", "move", "main", "--before", "*", "--after", "release/*"}, ExitUsage, "exactly one of --before or --after"},
		{[]string{"map", "move", "--before", "main"}, ExitUsage, "expected one <pattern>"},
		{[]string{"map", "add", "x", "dev", "--before", "main"}, ExitUsage, "flag provided but not defined"},
	}
	for _, tc := range cases {
		res := r.run(tc.args...)
		if res.code != tc.code || !strings.Contains(res.stderr, tc.want) {
			t.Fatalf("%v: exit %d, want %d with %q\n%s", tc.args, res.code, tc.code, tc.want, res.all())
		}
	}
	if r.read(".envbuckets.toml") != before {
		t.Fatal("refused move changed the config")
	}
}

func TestMapOrderDrivesFirstMatch(t *testing.T) {
	r := mapRepo(t)
	r.ok("map", "add", "release/hot*", "dev")
	bucketFor := func(branch string) string {
		t.Helper()
		p, err := openProject(Env{Cwd: r.root})
		if err != nil {
			t.Fatal(err)
		}
		return p.cfg.Match(branch, pattern.Match).Bucket
	}
	if got := bucketFor("release/hotfix"); got != "prod" {
		t.Fatalf("before move: %s", got)
	}
	r.ok("map", "move", "release/hot*", "--before", "release/*")
	if got := bucketFor("release/hotfix"); got != "dev" {
		t.Fatalf("after move: %s", got)
	}
}
