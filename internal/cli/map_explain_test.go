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
		"branch: release/2.0 (no local branch, explained as if checked out)",
		"rule:   2. release/* -> prod (first match wins)",
		"bucket: prod",
		"root         available .env.d/prod/.env | active: dev",
		"api          available apps/api/.env.d/prod/.env | active: dev",
		"web          missing apps/web/.env.d/prod/.env | active: dev",
		"next: envbuckets bucket add prod --all",
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
	if !strings.Contains(res.stdout, "branch: main (current)") || !strings.Contains(res.stdout, "1. main -> staging") {
		t.Fatalf("main:\n%s", res.stdout)
	}
	res = r.ok("map", "explain", "feature/x")
	if !strings.Contains(res.stdout, "3. * -> dev") || strings.Contains(res.stdout, "next:") {
		t.Fatalf("catch-all:\n%s", res.stdout)
	}
}

func TestMapExplainPinOverridesRule(t *testing.T) {
	r := monorepo(t)
	r.git("branch", "release/1.0")
	r.ok("link", "staging", "--branch", "release/1.0")
	res := r.ok("map", "explain", "release/1.0")
	for _, want := range []string{
		"link:   staging (local pin, overrides rules)",
		"rule:   2. release/* -> prod (overridden by the pin)",
		"bucket: staging",
		"web          available apps/web/.env.d/staging/.env",
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
	for _, want := range []string{"rule:   none matches", "bucket: none", "next: envbuckets map add"} {
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
	if !strings.Contains(res.stdout, "web          MISSING directory apps/web") {
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
