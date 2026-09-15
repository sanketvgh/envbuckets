package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/sanketvgh/envbuckets/internal/block"
)

func TestDefaultUninstallKeepsData(t *testing.T) {
	r := setup(t)
	r.write(".env.d/dev/.env", "A=1\n")
	res := r.ok("uninstall")
	if !strings.Contains(res.stdout, "[materialized] .env (was -> .env.d/dev/.env)") {
		t.Fatalf("output:\n%s", res.stdout)
	}
	info, err := os.Lstat(r.path(".env"))
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal(".env should be a real file")
	}
	if r.read(".env") != "A=1\n" {
		t.Fatal("materialized content mismatch")
	}
	if r.exists(".git/hooks/post-checkout") {
		t.Fatal("hook file with only our block should be deleted")
	}
	if !r.exists(".envbuckets.toml") || !r.exists(".env.d/dev/.env") || !strings.Contains(r.read(".gitignore"), block.Begin) {
		t.Fatal("data or gitignore removed by default uninstall")
	}
	if !strings.Contains(res.stdout, "Data kept") {
		t.Fatal("footer missing")
	}
}

func TestUninstallPreservesCustomHook(t *testing.T) {
	r := newRepo(t)
	custom := "#!/bin/sh\necho custom\n"
	r.write(".git/hooks/post-checkout", custom)
	r.ok("init")
	r.ok("uninstall")
	if r.read(".git/hooks/post-checkout") != custom {
		t.Fatalf("custom hook not byte-identical: %q", r.read(".git/hooks/post-checkout"))
	}
}

func TestPurgeRequiresConfirmation(t *testing.T) {
	r := setup(t)
	r.write(".env.d/dev/.env", "A=1\n")
	res := r.runIn(r.root, "", "uninstall", "--purge")
	if res.code != ExitBlocked || !r.exists(".env.d/dev") {
		t.Fatalf("no confirmation must block: %d %s", res.code, res.all())
	}
	res = r.runIn(r.root, "DELETE\n", "uninstall", "--purge")
	if res.code != ExitOK {
		t.Fatalf("purge: %d %s", res.code, res.all())
	}
	if r.exists(".env.d") || r.exists(".envbuckets.toml") || r.exists(".gitignore") {
		t.Fatal("purge left tool data behind")
	}
	if r.read(".env") != "A=1\n" {
		t.Fatal("plain .env should survive purge")
	}
	status := r.git("status", "--porcelain")
	if strings.Contains(status, ".env.d") || strings.Contains(status, ".envbuckets.toml") || strings.Contains(status, ".gitignore") {
		t.Fatalf("tool traces visible to git:\n%s", status)
	}
}

func TestUninstallIgnoresCorruptToml(t *testing.T) {
	r := setup(t)
	r.write(".envbuckets.toml", "schema = 2\n[[rules")
	res := r.run("uninstall")
	if res.code != ExitOK {
		t.Fatalf("got %d %s", res.code, res.all())
	}
}

func TestUninstallIdempotentAndOnFreshRepo(t *testing.T) {
	r := newRepo(t)
	res := r.ok("uninstall")
	if strings.Contains(res.stdout, "[removed]") || strings.Contains(res.stdout, "[materialized]") {
		t.Fatalf("fresh repo:\n%s", res.stdout)
	}
	requireSymlinks(t, r.root)
	r.ok("init")
	r.ok("bucket", "add", "dev")
	r.ok("use", "dev")
	r.ok("uninstall")
	second := r.ok("uninstall")
	if strings.Contains(second.stdout, "[removed]") || strings.Contains(second.stdout, "[materialized]") {
		t.Fatalf("second uninstall:\n%s", second.stdout)
	}
}

func TestUninstallWithBrokenSymlink(t *testing.T) {
	r := setup(t)
	if err := os.Remove(r.path(".env.d/dev/.env")); err != nil {
		t.Fatal(err)
	}
	res := r.ok("uninstall")
	if !strings.Contains(res.stderr, "BROKEN") || !strings.Contains(res.stdout, "[skipped] .env") {
		t.Fatalf("output:\n%s", res.all())
	}
	if r.exists(".git/hooks/post-checkout") {
		t.Fatal("deactivation did not complete")
	}
}

func TestRoundTrip(t *testing.T) {
	r := setup(t)
	r.write(".env.d/dev/.env", "A=1\n")
	r.ok("uninstall")
	r.newBranch("release/9")
	if res := r.hook(); !strings.Contains(res.stderr, "real file") || r.read(".env") != "A=1\n" {
		t.Fatalf("post-uninstall hook touched .env:\n%s", res.all())
	}
	res := r.run("init", "--into", "dev")
	if res.code != ExitBlocked || !strings.Contains(res.stdout, "refusing to clobber") {
		t.Fatalf("clobber guard: %d %s", res.code, res.all())
	}
	res = r.ok("init", "--into", "restored")
	if r.readlink(".env") != ".env.d/restored/.env" || r.read(".env.d/restored/.env") != "A=1\n" {
		t.Fatalf("bootstrap: %s", res.all())
	}
	prompted := r.ok("uninstall")
	if !strings.Contains(prompted.stdout, "[materialized]") {
		t.Fatal("second uninstall should materialize again")
	}
	res = r.runIn(r.root, "fromprompt\n", "init")
	if res.code != ExitOK || r.readlink(".env") != ".env.d/fromprompt/.env" {
		t.Fatalf("prompted bootstrap: %d %s", res.code, res.all())
	}
}

func TestCrashBetweenMaterializeAndHookRemoval(t *testing.T) {
	r := setup(t)
	r.write(".env.d/dev/.env", "A=1\n")
	s := (&project{root: r.root}).newScope("root", ".")
	ls, err := s.linkState()
	if err != nil {
		t.Fatal(err)
	}
	if err := materializeScope(s, ls); err != nil {
		t.Fatal(err)
	}
	if res := r.hook(); !strings.Contains(res.stderr, "real file") {
		t.Fatalf("hook should be inert:\n%s", res.all())
	}
	res := r.ok("uninstall")
	if !strings.Contains(res.stdout, "[skipped] .env (already a real file)") || r.exists(".git/hooks/post-checkout") {
		t.Fatalf("re-run:\n%s", res.stdout)
	}
}
