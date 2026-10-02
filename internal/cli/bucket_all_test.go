package cli

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestBucketAddAllIsIdempotentAndPreserves(t *testing.T) {
	r := monorepo(t)
	r.write("apps/api/.env.d/prod/.env", "KEEP=1\n")
	res := r.ok("bucket", "add", "prod", "--all")
	for _, want := range []string{
		"ok: root: .env.d/prod/.env already exists, left untouched",
		"ok: api: apps/api/.env.d/prod/.env already exists",
		"created: web: apps/web/.env.d/prod/.env (empty)",
		"Created 1; 2 already present, 0 failed.",
	} {
		if !strings.Contains(res.stdout, want) {
			t.Fatalf("missing %q:\n%s", want, res.stdout)
		}
	}
	if r.read("apps/api/.env.d/prod/.env") != "KEEP=1\n" || r.read("apps/web/.env.d/prod/.env") != "" {
		t.Fatal("existing file changed or new file not empty")
	}
	res = r.ok("bucket", "add", "prod", "--all")
	if !strings.Contains(res.stdout, "Created 0; 3 already present, 0 failed.") || strings.Contains(res.stdout, "next:") {
		t.Fatalf("rerun:\n%s", res.stdout)
	}
	if r.readlink("apps/web/.env") != ".env.d/dev/.env" {
		t.Fatal("bulk add must not switch links")
	}
}

func TestBucketAddAllPartialFailure(t *testing.T) {
	r := monorepo(t)
	if err := os.RemoveAll(r.path("apps/web")); err != nil {
		t.Fatal(err)
	}
	res := r.run("bucket", "add", "qa", "--all")
	if res.code != ExitBlocked {
		t.Fatalf("exit %d\n%s", res.code, res.all())
	}
	for _, want := range []string{
		"created: root: .env.d/qa/.env",
		"created: api: apps/api/.env.d/qa/.env",
		"failed: web: scope directory apps/web is missing, not created",
		"Created 2; 0 already present, 1 failed.",
	} {
		if !strings.Contains(res.stdout, want) {
			t.Fatalf("missing %q:\n%s", want, res.stdout)
		}
	}
	if !strings.Contains(res.stderr, "envbuckets: bucket qa was not created in 1 of 3 scopes") {
		t.Fatalf("stderr:\n%s", res.stderr)
	}
	if r.exists("apps/web") {
		t.Fatal("missing scope directory was recreated")
	}
}

func TestBucketAddSingleScopeKeepsMissingDirMissing(t *testing.T) {
	r := monorepo(t)
	if err := os.RemoveAll(r.path("apps/web")); err != nil {
		t.Fatal(err)
	}
	res := r.run("bucket", "add", "qa", "--scope", "web")
	if res.code != ExitBlocked || r.exists("apps/web") {
		t.Fatalf("exit %d, dir recreated=%v\n%s", res.code, r.exists("apps/web"), res.all())
	}
}

func TestBucketAllFlagValidation(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.ok("init")
	cases := [][]string{
		{"bucket", "add", "x", "--all", "--scope", "root"},
		{"bucket", "list", "--all", "--scope", "root"},
		{"bucket", "rm", "x", "--all"},
		{"bucket", "add", "x", "--purge"},
		{"bucket", "list", "extra"},
		{"bucket", "add"},
		{"bucket", "add", "a", "b", "--all"},
	}
	for _, args := range cases {
		if res := r.run(args...); res.code != ExitUsage {
			t.Fatalf("%v: exit %d\n%s", args, res.code, res.all())
		}
	}
	if res := r.run("bucket", "add", "../x", "--all"); res.code != ExitBlocked || r.exists("x") {
		t.Fatalf("invalid name: %d", res.code)
	}
}

func TestBucketListAllMatrix(t *testing.T) {
	r := monorepo(t)
	r.ok("bucket", "add", "qa", "--scope", "api")
	r.git("branch", "feature/x")
	r.ok("link", "qa", "--branch", "feature/x")
	if err := os.RemoveAll(r.path("apps/api/.env.d/qa")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(r.path("apps/api/.env.d/dev/.env")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(r.path(".env")); err != nil {
		t.Fatal(err)
	}
	r.write(".env", "REAL=1\n")
	const marker = "SECRET_MARKER_51d"
	r.write("apps/web/.env.d/staging/.env", "K="+marker+"\n")

	res := r.ok("bucket", "list", "--all")
	rows := map[string]string{}
	for _, l := range strings.Split(res.stdout, "\n") {
		if f := strings.Fields(l); len(f) > 0 {
			rows[f[0]] = strings.Join(f, " ")
		}
	}
	want := map[string]string{
		"bucket":  "bucket root api web used by",
		"dev":     "dev present BROKEN active rule *",
		"prod":    "prod present present missing rule release/*",
		"qa":      "qa missing missing missing pin feature/x",
		"staging": "staging present present present rule main",
	}
	for k, v := range want {
		if rows[k] != v {
			t.Fatalf("row %s = %q, want %q\n%s", k, rows[k], v, res.stdout)
		}
	}
	if !strings.Contains(res.stdout, "note: root: .env is a real file, not managed") {
		t.Fatalf("real file note:\n%s", res.stdout)
	}
	if strings.Contains(res.all(), marker) || regexp.MustCompile(`[^\x00-\x7F]`).MatchString(res.all()) {
		t.Fatalf("leaked contents or non-ASCII:\n%s", res.all())
	}
	if r.read(".env") != "REAL=1\n" || r.exists("apps/api/.env.d/qa") {
		t.Fatal("list --all mutated the project")
	}
}

func TestBucketListAllMissingScopeAndEmpty(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.ok("init")
	res := r.ok("bucket", "list", "--all")
	if !strings.Contains(res.stdout, "no buckets on disk or referenced") {
		t.Fatalf("empty:\n%s", res.stdout)
	}

	m := monorepo(t)
	if err := os.RemoveAll(m.path("apps/web")); err != nil {
		t.Fatal(err)
	}
	res = m.ok("bucket", "list", "--all")
	if !strings.Contains(res.stdout, "note: web: directory apps/web is missing") {
		t.Fatalf("missing dir note:\n%s", res.stdout)
	}
	found := false
	for _, l := range strings.Split(res.stdout, "\n") {
		found = found || strings.Join(strings.Fields(l), " ") == "dev active active no dir rule *"
	}
	if !found {
		t.Fatalf("matrix:\n%s", res.stdout)
	}
}
