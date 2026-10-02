package cli

import (
	"os"
	"strings"
	"testing"
)

const clonedConfig = `schema = 1

[[rules]]
pattern = "main"
bucket = "staging"

[[rules]]
pattern = "release/*"
bucket = "prod"

[[rules]]
pattern = "*"
bucket = "dev"

[[scopes]]
name = "api"
path = "apps/api"

[[scopes]]
name = "web"
path = "apps/web"
`

func freshClone(t *testing.T) *repo {
	t.Helper()
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.write(".envbuckets.toml", clonedConfig)
	r.write("apps/api/main.go", "package main\n")
	r.git("config", "--local", "branch.feature.envbuckets", "qa")
	return r
}

func TestInitScaffoldFreshClone(t *testing.T) {
	r := freshClone(t)
	res := r.ok("init", "--scaffold")
	for _, want := range []string{
		"created: api: apps/api/.env.d/staging/.env (empty)",
		"created: api: apps/api/.env.d/prod/.env (empty)",
		"created: api: apps/api/.env.d/dev/.env (empty)",
		"skipped: web: scope directory apps/web missing, not created",
		"Buckets from shared rules: staging, prod, dev",
		"Created 3; 0 already present, 0 failed.",
		"Not scaffolded: web",
	} {
		if !strings.Contains(res.stdout, want) {
			t.Fatalf("missing %q:\n%s", want, res.stdout)
		}
	}
	if r.exists("apps/web") || r.exists("apps/api/.env.d/qa") {
		t.Fatal("created a missing app dir or a pin-only bucket")
	}
	if r.exists("apps/api/.env") {
		t.Fatal("scaffold activated an empty bucket")
	}
	if r.read("apps/api/.env.d/dev/.env") != "" {
		t.Fatal("scaffolded file not empty")
	}

	r.write("apps/api/.env.d/dev/.env", "FILLED=1\n")
	res = r.ok("init", "--scaffold")
	if !strings.Contains(res.stdout, "Created 0; 3 already present, 0 failed.") || strings.Contains(res.stdout, "fill the new") {
		t.Fatalf("rerun:\n%s", res.stdout)
	}
	if r.read("apps/api/.env.d/dev/.env") != "FILLED=1\n" {
		t.Fatal("rerun overwrote a filled bucket")
	}
}

func TestInitWithoutScaffoldCreatesNoBuckets(t *testing.T) {
	r := freshClone(t)
	r.ok("init")
	if r.exists("apps/api/.env.d") {
		t.Fatal("plain init scaffolded")
	}
}

func TestInitScaffoldWithInto(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.write(".envbuckets.toml", "schema = 1\n\n[[rules]]\npattern = \"main\"\nbucket = \"staging\"\n\n[[rules]]\npattern = \"*\"\nbucket = \"dev\"\n")
	r.write(".env", "A=1\n")
	res := r.ok("init", "--scaffold", "--into", "dev")
	if !strings.Contains(res.stdout, "ok: root: .env.d/dev/.env already exists") || !strings.Contains(res.stdout, "created: root: .env.d/staging/.env (empty)") {
		t.Fatalf("scaffold+into:\n%s", res.stdout)
	}
	if r.readlink(".env") != ".env.d/dev/.env" || r.read(".env.d/dev/.env") != "A=1\n" {
		t.Fatal("bootstrap did not move the real .env into dev")
	}
}

func TestInitScaffoldWaitsForRealEnv(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.write(".envbuckets.toml", "schema = 1\n\n[[rules]]\npattern = \"*\"\nbucket = \"dev\"\n")
	r.write(".env", "A=1\n")
	res := r.ok("init", "--scaffold")
	if !strings.Contains(res.stdout, "is still a real file, scaffold waits") || r.exists(".env.d/dev") {
		t.Fatalf("scaffold must not pre-create the bootstrap target:\n%s", res.stdout)
	}
	r.ok("init", "--scaffold", "--into", "dev")
	if r.readlink(".env") != ".env.d/dev/.env" || r.read(".env.d/dev/.env") != "A=1\n" {
		t.Fatal("later bootstrap blocked")
	}
}

func TestInitScaffoldIntoExistingTargetBlocks(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.write(".envbuckets.toml", "schema = 1\n\n[[rules]]\npattern = \"*\"\nbucket = \"dev\"\n")
	r.write(".env", "A=1\n")
	r.write(".env.d/dev/.env", "B=2\n")
	res := r.run("init", "--scaffold", "--into", "dev")
	if res.code != ExitBlocked || !strings.Contains(res.stdout, "scaffold waits until the blocked bootstrap") {
		t.Fatalf("exit %d\n%s", res.code, res.all())
	}
	if r.read(".env") != "A=1\n" || r.read(".env.d/dev/.env") != "B=2\n" {
		t.Fatal("files changed")
	}
}

func TestInitScaffoldWithoutRules(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	res := r.ok("init", "--scaffold")
	if !strings.Contains(res.stdout, "scaffold: no shared rules reference a bucket yet") {
		t.Fatalf("no rules:\n%s", res.stdout)
	}
}

func TestInitScaffoldPartialFailure(t *testing.T) {
	r := freshClone(t)
	if err := os.MkdirAll(r.path("apps/api/.env.d/prod/.env"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := r.run("init", "--scaffold")
	if res.code != ExitBlocked || !strings.Contains(res.stdout, "failed: api: apps/api/.env.d/prod/.env exists but is not a regular file") {
		t.Fatalf("exit %d\n%s", res.code, res.all())
	}
	if !strings.Contains(res.stdout, "Created 2; 0 already present, 1 failed.") || !strings.Contains(res.stderr, "scaffold incomplete") {
		t.Fatalf("summary:\n%s", res.all())
	}
}
