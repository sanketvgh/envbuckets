package cli

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func symlinksWork(dir string) bool {
	probe := filepath.Join(dir, ".capability-check")
	err := os.Symlink("x", probe)
	_ = os.Remove(probe)
	return err == nil
}

func TestInitPreflightLeavesNothingBehind(t *testing.T) {
	r := newRepo(t)
	if !symlinksWork(r.root) {
		res := r.run("init")
		if res.code != ExitEnv || !strings.Contains(res.stderr, "does not allow symlinks") || !strings.Contains(res.stderr, "nothing was changed") {
			t.Fatalf("no symlinks: exit %d\n%s", res.code, res.all())
		}
		if r.exists(".envbuckets.toml") || r.exists(".gitignore") || r.exists(".git/hooks/post-checkout") {
			t.Fatal("init wrote files before failing the preflight")
		}
		assertNoProbe(t, r)
		return
	}
	r.ok("init")
	assertNoProbe(t, r)
}

func TestInitPreflightUnwritableRepo(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions as a non-root user")
	}
	r := newRepo(t)
	if err := os.Chmod(r.root, 0o555); err != nil { //nolint:gosec // test dir made read-only on purpose
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(r.root, 0o755) }) //nolint:gosec // restore so TempDir cleanup works
	res := r.run("init")
	if res.code != ExitEnv || !strings.Contains(res.stderr, "nothing was changed") || r.exists(".envbuckets.toml") {
		t.Fatalf("exit %d\n%s", res.code, res.all())
	}
}

func TestInitRejectsPositionalArgs(t *testing.T) {
	r := newRepo(t)
	if res := r.run("init", "extra"); res.code != ExitUsage || r.exists(".envbuckets.toml") {
		t.Fatalf("exit %d", res.code)
	}
}

func TestInitPreflightReportsProbeLeftBehind(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	calls := 0
	removeProbe = func(name string) error {
		calls++
		if calls == 2 {
			return os.ErrPermission
		}
		return os.Remove(name)
	}
	t.Cleanup(func() { removeProbe = os.Remove })
	res := r.run("init")
	if res.code != ExitEnv || !strings.Contains(res.stderr, "cannot remove the symlink probe .envbuckets-probe-") ||
		!strings.Contains(res.stderr, "nothing else was changed") {
		t.Fatalf("exit %d\n%s", res.code, res.all())
	}
	if r.exists(".envbuckets.toml") || r.exists(".gitignore") {
		t.Fatal("init continued after the probe was left behind")
	}
	entries, err := os.ReadDir(r.root)
	if err != nil {
		t.Fatal(err)
	}
	left := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".envbuckets-probe-") {
			left++
			if !strings.Contains(res.stderr, e.Name()) {
				t.Fatalf("error does not name %s:\n%s", e.Name(), res.stderr)
			}
		}
	}
	if left != 1 {
		t.Fatalf("want the one leftover probe reported, found %d", left)
	}
}

func TestRecoverableAddsRerunHint(t *testing.T) {
	var ee *exitError
	if !errors.As(recoverable(errors.New("disk full")), &ee) || ee.code != ExitEnv || !strings.Contains(ee.next, "re-run envbuckets init") {
		t.Fatalf("plain error: %+v", ee)
	}
	if strings.Contains(ee.next, "backs everything out") || !strings.Contains(ee.next, "nothing was rolled back") ||
		!strings.Contains(ee.next, "keeps the config and .env.d/") {
		t.Fatalf("hint overpromises: %s", ee.next)
	}
	if !errors.As(recoverable(blocked("x").then("keep me")), &ee) || ee.next != "keep me" {
		t.Fatalf("existing hint replaced: %+v", ee)
	}
}

func assertNoProbe(t *testing.T, r *repo) {
	t.Helper()
	entries, err := os.ReadDir(r.root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".envbuckets-probe-") {
			t.Fatalf("probe left behind: %s", e.Name())
		}
	}
}
