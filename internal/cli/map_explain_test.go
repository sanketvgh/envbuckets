package cli

import (
	"os"
	"strings"
	"testing"
)

func TestMapExplainHypotheticalBranch(t *testing.T) {
	r := monorepo(t)
	before := r.read(".envbuckets.toml")
	res := r.ok("map", "explain", "release/2.0")
	for _, want := range []string{
		"Branch release/2.0 (no local branch, explained as if checked out)",
		"Using bucket prod (rule release/*, priority 2)",
		"root         .env.d/prod/.env exists; .env -> .env.d/dev/.env",
		"api          apps/api/.env.d/prod/.env exists; apps/api/.env -> apps/api/.env.d/dev/.env",
		"web          apps/web/.env.d/prod/.env missing; apps/web/.env -> apps/web/.env.d/dev/.env",
	} {
		if !strings.Contains(res.stdout, want) {
			t.Fatalf("missing %q:\n%s", want, res.stdout)
		}
	}
	if r.read(".envbuckets.toml") != before || r.readlink("apps/api/.env") != ".env.d/dev/.env" {
		t.Fatal("explain mutated the project")
	}
	if r.exists("apps/web/.env.d/prod") {
		t.Fatal("explain created a bucket")
	}
}

func TestMapExplainCurrentBranchAndCatchAll(t *testing.T) {
	r := monorepo(t)
	res := r.ok("map", "explain", "main")
	if !strings.Contains(res.stdout, "Branch main (current)") || !strings.Contains(res.stdout, "Using bucket staging (rule main, priority 1)") {
		t.Fatalf("main:\n%s", res.stdout)
	}
	res = r.ok("map", "explain", "feature/x")
	if !strings.Contains(res.stdout, "Using bucket dev (rule *, priority 3)") || strings.Contains(res.stdout, "next:") {
		t.Fatalf("catch-all:\n%s", res.stdout)
	}
}

func TestMapExplainPinOverridesRule(t *testing.T) {
	r := monorepo(t)
	r.git("branch", "release/1.0")
	r.ok("link", "staging", "--branch", "release/1.0")
	res := r.ok("map", "explain", "release/1.0")
	for _, want := range []string{
		"Using bucket staging (local pin; overrides rules)",
		"Overridden rule: 2. release/* -> prod",
		"web          apps/web/.env.d/staging/.env exists",
	} {
		if !strings.Contains(res.stdout, want) {
			t.Fatalf("missing %q:\n%s", want, res.stdout)
		}
	}
}

func TestMapExplainNoMatch(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.ok("init")
	r.ok("bucket", "add", "prod")
	r.ok("map", "add", "release/*", "prod")
	res := r.ok("map", "explain", "feature/x")
	for _, want := range []string{"No matching rule or local pin; checkout leaves .env unchanged."} {
		if !strings.Contains(res.stdout, want) {
			t.Fatalf("missing %q:\n%s", want, res.stdout)
		}
	}
	if strings.Contains(res.stdout, "scopes:") {
		t.Fatalf("no-match must not list a guessed bucket:\n%s", res.stdout)
	}
}

func TestMapExplainMissingScopeDirectory(t *testing.T) {
	r := monorepo(t)
	if err := os.RemoveAll(r.path("apps/web")); err != nil {
		t.Fatal(err)
	}
	res := r.ok("map", "explain", "main")
	if !strings.Contains(res.stdout, "web          apps/web missing (scope directory)") {
		t.Fatalf("missing dir:\n%s", res.stdout)
	}
}

func TestMapExplainErrors(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.ok("init")
	cases := []struct {
		args []string
		code int
	}{
		{[]string{"map", "explain"}, ExitUsage},
		{[]string{"map", "explain", "a", "b"}, ExitUsage},
		{[]string{"map", "explain", "a", "--before", "b"}, ExitUsage},
	}
	for _, tc := range cases {
		if res := r.run(tc.args...); res.code != tc.code {
			t.Fatalf("%v: exit %d\n%s", tc.args, res.code, res.all())
		}
	}
	r.git("config", "--local", "branch.bad.envbuckets", "../x")
	if res := r.run("map", "explain", "bad"); res.code != ExitEnv || !strings.Contains(res.stderr, "branch link") {
		t.Fatalf("malformed pin: %d\n%s", res.code, res.all())
	}
}
