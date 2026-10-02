package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/sanketvgh/envbuckets/internal/config"
)

func TestApplyRepairsMissingLinkAndDryRun(t *testing.T) {
	r := setup(t)
	if err := os.Remove(r.path(".env")); err != nil {
		t.Fatal(err)
	}
	res := r.ok("apply", "--dry-run")
	if !strings.Contains(res.stdout, "would point") || r.exists(".env") {
		t.Fatalf("dry-run changed link: %s", res.all())
	}
	res = r.ok("apply")
	if !strings.Contains(res.stdout, "1 changed, 0 unchanged, 0 failed") || r.readlink(".env") != ".env.d/staging/.env" {
		t.Fatalf("apply: %s", res.all())
	}
	if res := r.ok("check"); !strings.Contains(res.stdout, "1 ready") {
		t.Fatalf("check: %s", res.all())
	}
}

func TestApplyPreservesUnmanagedAndReportsPartial(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.ok("init")
	r.write("apps/api/README", "api\n")
	r.write("apps/web/README", "web\n")
	r.ok("scope", "add", "apps/api", "--name", "api")
	r.ok("scope", "add", "apps/web", "--name", "web")
	for _, name := range []string{"api", "web"} {
		r.ok("bucket", "add", "dev", "--scope", name)
	}
	r.ok("map", "add", "main", "dev")
	r.write("apps/api/.env", "KEEP=1\n")
	res := r.run("apply", "--dry-run")
	if res.code != ExitBlocked || !strings.Contains(res.stdout, "1 changed, 0 unchanged, 1 failed") || r.exists("apps/web/.env") {
		t.Fatalf("dry partial: %d %s", res.code, res.all())
	}
	res = r.run("apply")
	if res.code != ExitBlocked || !strings.Contains(res.stdout, "1 changed, 0 unchanged, 1 failed") {
		t.Fatalf("partial: %d %s", res.code, res.all())
	}
	if r.read("apps/api/.env") != "KEEP=1\n" || r.readlink("apps/web/.env") != ".env.d/dev/.env" {
		t.Fatal("apply did not preserve the real file and switch independent scope")
	}
	if res := r.run("apply", "--scope", "web"); res.code != ExitOK || !strings.Contains(res.stdout, "0 changed, 1 unchanged, 0 failed") {
		t.Fatalf("selected apply: %d %s", res.code, res.all())
	}
	if res := r.run("check", "--scope", "web"); res.code != ExitOK || !strings.Contains(res.stdout, "1 scope ready.") {
		t.Fatalf("selected check: %d %s", res.code, res.all())
	}
	if res := r.run("check"); res.code != ExitBlocked {
		t.Fatalf("all-scope check: %d %s", res.code, res.all())
	}
}

func TestUseAllIsTransientAndRejectsConflictingSelectors(t *testing.T) {
	r := setup(t)
	if res := r.run("use", "prod", "--all", "--scope", "root"); res.code != ExitUsage {
		t.Fatalf("selector conflict: %d %s", res.code, res.all())
	}
	r.ok("use", "prod", "--all")
	if got := r.readlink(".env"); got != ".env.d/prod/.env" {
		t.Fatalf("use --all: %s", got)
	}
	if res := r.run("check"); res.code != ExitBlocked || !strings.Contains(res.stdout, "expected .env.d/staging/.env") {
		t.Fatalf("manual use changed mapping: %d %s", res.code, res.all())
	}
}

func TestHookAndApplyPreserveForeignSymlink(t *testing.T) {
	r := setup(t)
	if err := os.Remove(r.path(".env")); err != nil {
		t.Fatal(err)
	}
	r.write("other.env", "KEEP=1\n")
	if err := os.Symlink("other.env", r.path(".env")); err != nil {
		t.Fatal(err)
	}
	if res := r.run("apply"); res.code != ExitBlocked || !strings.Contains(res.stdout, "foreign") {
		t.Fatalf("apply: %d %s", res.code, res.all())
	}
	if res := r.hook(); !strings.Contains(res.stderr, "foreign symlink") {
		t.Fatalf("hook: %s", res.all())
	}
	if r.readlink(".env") != "other.env" || r.read("other.env") != "KEEP=1\n" {
		t.Fatal("foreign symlink or its target changed")
	}
}

func TestApplyNoMatchAndDetachedDoNotMutate(t *testing.T) {
	r := newRepo(t)
	if err := config.New().Save(r.root); err != nil {
		t.Fatal(err)
	}
	if res := r.run("apply", "--dry-run"); res.code != ExitBlocked || !strings.Contains(res.all(), "no mapping") || r.exists(".env") {
		t.Fatalf("no match: %d %s", res.code, res.all())
	}
	r.git("checkout", "-q", "--detach")
	if res := r.run("apply"); res.code != ExitBlocked || !strings.Contains(res.all(), "detached HEAD") || r.exists(".env") {
		t.Fatalf("detached: %d %s", res.code, res.all())
	}
}

func TestApplyMissingTargetPreservesActiveLink(t *testing.T) {
	r := setup(t)
	if err := os.Remove(r.path(".env.d/staging/.env")); err != nil {
		t.Fatal(err)
	}
	before := r.readlink(".env")
	if res := r.run("apply"); res.code != ExitBlocked || !strings.Contains(res.stdout, "expected bucket file missing") {
		t.Fatalf("missing target: %d %s", res.code, res.all())
	}
	if got := r.readlink(".env"); got != before {
		t.Fatalf("active link changed: %s", got)
	}
}
