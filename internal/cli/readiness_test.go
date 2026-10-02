package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/sanketvgh/envbuckets/internal/config"
)

func TestCheckUnresolvedAndUnmanaged(t *testing.T) {
	r := newRepo(t)
	if err := config.New().Save(r.root); err != nil {
		t.Fatal(err)
	}
	if res := r.run("check"); res.code != ExitBlocked || !strings.Contains(res.all(), "no mapping") {
		t.Fatalf("no match: %d %s", res.code, res.all())
	}
	r.ok("bucket", "add", "dev")
	r.ok("map", "add", "main", "dev")
	if res := r.run("check"); res.code != ExitBlocked || !strings.Contains(res.stdout, "managed link missing") {
		t.Fatalf("missing link: %d %s", res.code, res.all())
	}
	r.write(".env", "SECRET=untouched\n")
	if res := r.run("check"); res.code != ExitBlocked || !strings.Contains(res.stdout, "real file, unmanaged") {
		t.Fatalf("real file: %d %s", res.code, res.all())
	}
	if got := r.read(".env"); got != "SECRET=untouched\n" {
		t.Fatalf("check changed .env: %q", got)
	}
	r.git("checkout", "-q", "--detach")
	if res := r.run("check"); res.code != ExitBlocked || !strings.Contains(res.all(), "detached HEAD") {
		t.Fatalf("detached: %d %s", res.code, res.all())
	}
}

func TestCheckPinAndBrokenLink(t *testing.T) {
	r := setup(t)
	r.ok("link", "dev")
	if res := r.run("check"); res.code != ExitOK || !strings.Contains(res.stdout, "link:   dev") {
		t.Fatalf("pin readiness: %d %s", res.code, res.all())
	}
	if err := os.Remove(r.path(".env.d/dev/.env")); err != nil {
		t.Fatal(err)
	}
	if res := r.run("check"); res.code != ExitBlocked || !strings.Contains(res.stdout, "BROKEN") || !strings.Contains(res.stdout, "MISSING .env.d/dev/.env") {
		t.Fatalf("broken: %d %s", res.code, res.all())
	}
	if res := r.run("status"); res.code != ExitOK || !strings.Contains(res.stdout, "BROKEN") {
		t.Fatalf("status: %d %s", res.code, res.all())
	}
}

func TestStatusDescribesMismatchWithoutCause(t *testing.T) {
	r := setup(t)
	res := r.ok("status")
	if !strings.Contains(res.stdout, "active: dev; expected: staging") || strings.Contains(res.stdout, "manual override") {
		t.Fatalf("status: %s", res.stdout)
	}
	if res := r.run("check"); res.code != ExitBlocked {
		t.Fatalf("mismatch: %d %s", res.code, res.all())
	}
}

func TestStatusSuggestsNextStepWhenBranchIsUnresolved(t *testing.T) {
	r := newRepo(t)
	if err := config.New().Save(r.root); err != nil {
		t.Fatal(err)
	}
	if res := r.ok("status"); !strings.Contains(res.stdout, "next: envbuckets map add <pattern> <bucket>, or envbuckets link <bucket>") {
		t.Fatalf("unmatched branch: %s", res.all())
	}
	r.git("checkout", "-q", "--detach")
	if res := r.ok("status"); !strings.Contains(res.stdout, "next: git checkout <branch>, or envbuckets use <bucket>") {
		t.Fatalf("detached HEAD: %s", res.all())
	}
}

func TestCheckReportsMissingScopeDirectory(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.ok("init")
	r.write("apps/api/README", "api\n")
	r.ok("scope", "add", "apps/api", "--name", "api")
	r.ok("bucket", "add", "dev", "--scope", "api")
	r.ok("map", "add", "main", "dev")
	if err := os.Rename(r.path("apps/api"), r.path("apps/moved")); err != nil {
		t.Fatal(err)
	}
	res := r.run("check")
	if res.code != ExitBlocked || !strings.Contains(res.stdout, "MISSING directory apps/api") {
		t.Fatalf("missing scope: %d %s", res.code, res.all())
	}
	if st := r.ok("status"); !strings.Contains(st.stdout, "MISSING directory apps/api") {
		t.Fatalf("status: %s", st.all())
	}
}
