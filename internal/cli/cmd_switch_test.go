package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const switchConfig = `{"default":"dev","rules":[{"branch":"release/**","bucket":"prod"},{"branch":"staging","bucket":"staging"}]}`

func switchRepo(t *testing.T) *repo {
	t.Helper()
	r := newRepo(t)
	r.write(".envbuckets.json", switchConfig)
	r.write(".env.d/dev/.env", "SYNTHETIC_SECRET_DEV\n")
	r.write(".env.d/prod/.env", "SYNTHETIC_SECRET_PROD\n")
	r.write(".env.d/dev/apps/api/key.json", "SYNTHETIC_KEY\n")
	probe := r.path("symlink-probe")
	if err := os.Symlink("README", probe); err != nil {
		t.Skipf("symlink support required: %v", err)
	}
	if err := os.Remove(probe); err != nil {
		t.Fatal(err)
	}
	return r
}

func (r *repo) linkTarget(name, want string) {
	r.t.Helper()
	got, err := os.Readlink(r.path(name))
	if err != nil || filepath.ToSlash(got) != want {
		r.t.Fatalf("%s -> %q, %v; want %q", name, got, err, want)
	}
}

func TestSwitchTemporaryAndRepair(t *testing.T) {
	r := switchRepo(t)
	first := r.run("switch")
	if first.code != ExitOK || first.stdout != "Already on bucket 'dev'\n" {
		t.Fatalf("first: %+v", first)
	}
	r.linkTarget("apps/api/key.json", "../../.env.d/dev/apps/api/key.json")
	dry := r.run("switch", "-n", "prod")
	if dry.code != ExitOK || !strings.Contains(dry.stdout, "Would link .env to bucket 'prod'") || !strings.Contains(dry.stdout, "Would remove link apps/api/key.json") {
		t.Fatalf("dry: %+v", dry)
	}
	r.linkTarget(".env", ".env.d/dev/.env")
	r.linkTarget("apps/api/key.json", "../../.env.d/dev/apps/api/key.json")
	changed := r.run("switch", "prod")
	if changed.code != ExitOK || changed.stdout != "Switched to bucket 'prod'\n" || !strings.Contains(changed.stderr, "This branch uses 'dev'") {
		t.Fatalf("temporary: %+v", changed)
	}
	if _, err := os.Lstat(r.path("apps/api/key.json")); !os.IsNotExist(err) {
		t.Fatalf("obsolete link still exists: %v", err)
	}
	if _, err := os.Lstat(r.path(".env.d/dev/apps/api/key.json")); err != nil {
		t.Fatal("bucket file removed")
	}
	back := r.run("switch")
	if back.code != ExitOK || back.stdout != "Switched to bucket 'dev'\n" {
		t.Fatalf("back: %+v", back)
	}
	if err := os.Remove(r.path(".env")); err != nil {
		t.Fatal(err)
	}
	repaired := r.run("switch")
	if repaired.code != ExitOK || repaired.stdout != "Already on bucket 'dev'\n" {
		t.Fatalf("repair: %+v", repaired)
	}
	r.linkTarget(".env", ".env.d/dev/.env")
	for _, res := range []result{first, dry, changed, back, repaired} {
		if strings.Contains(res.all(), "SYNTHETIC_") {
			t.Fatal("file contents exposed")
		}
	}
}

func TestSwitchBlockedPathsAndDryRunParity(t *testing.T) {
	r := switchRepo(t)
	r.write(".env", "keep real file")
	r.write(".env.d/dev/foreign", "synthetic")
	if err := os.Symlink("README", r.path("foreign")); err != nil {
		t.Fatal(err)
	}
	dry := r.run("switch", "-n")
	if _, err := os.Lstat(r.path("apps/api/key.json")); !os.IsNotExist(err) {
		t.Fatal("dry run wrote a link")
	}
	applied := r.run("switch")
	if dry.code != ExitError || applied.code != dry.code || strings.Count(applied.stderr, "error:") != 2 {
		t.Fatalf("dry=%+v applied=%+v", dry, applied)
	}
	content, err := os.ReadFile(r.path(".env"))
	if err != nil || string(content) != "keep real file" {
		t.Fatal("real file changed")
	}
	r.linkTarget("foreign", "README")
	r.linkTarget("apps/api/key.json", "../../.env.d/dev/apps/api/key.json")
}

func TestSwitchMissingBuckets(t *testing.T) {
	r := switchRepo(t)
	if res := r.run("switch", "prod"); res.code != ExitOK {
		t.Fatalf("prod: %+v", res)
	}
	r.git("switch", "-q", "-c", "staging")
	res := r.run("switch")
	if res.code != ExitOK || !strings.Contains(res.stderr, "bucket 'staging' does not exist; using 'dev' (default)") {
		t.Fatalf("fallback: %+v", res)
	}
	r.linkTarget(".env", ".env.d/dev/.env")
	missing := r.run("switch", "missing")
	if missing.code != ExitError || !strings.Contains(missing.stderr, "fatal: no bucket named 'missing'") {
		t.Fatalf("missing: %+v", missing)
	}
	r.linkTarget(".env", ".env.d/dev/.env")
	// Rename the default away: missing-default fallback must leave dangling links alone.
	if err := os.Rename(r.path(".env.d/dev"), r.path(".env.d/old")); err != nil {
		t.Fatal(err)
	}
	noDefault := r.run("switch")
	if noDefault.code != ExitOK || !strings.Contains(noDefault.stderr, "links not changed") {
		t.Fatalf("missing default: %+v", noDefault)
	}
	r.linkTarget(".env", ".env.d/dev/.env")
}

func TestSwitchDetachedAndHookGuards(t *testing.T) {
	r := switchRepo(t)
	r.git("checkout", "-q", "--detach")
	res := r.run("switch")
	if res.code != ExitError || !strings.Contains(res.stderr, "HEAD is detached") {
		t.Fatalf("detached: %+v", res)
	}
	if res := r.run("switch", "prod"); res.code != ExitOK {
		t.Fatalf("explicit detached: %+v", res)
	}
	for _, args := range [][]string{{"hook"}, {"hook", "old", "new", "0"}, {"hook", "old", "new", "1"}} {
		res := r.run(args...)
		if res.code != ExitOK || res.all() != "" {
			t.Fatalf("hook guard %v: %+v", args, res)
		}
	}
	r.linkTarget(".env", ".env.d/prod/.env")
	r.git("switch", "-q", "main")
	r.write(".envbuckets.json", `{"default":"dev","bukcet":"prod"}`)
	broken := r.run("hook", "old", "new", "1")
	if broken.code != ExitOK || strings.Count(broken.stderr, "envbuckets: warning:") != 1 || !strings.Contains(broken.stderr, "unknown key 'bukcet'") {
		t.Fatalf("broken hook: %+v", broken)
	}
	r.linkTarget(".env", ".env.d/prod/.env")
}

func TestSwitchHookSilenceAndTemporaryReset(t *testing.T) {
	r := switchRepo(t)
	if res := r.run("switch"); res.code != ExitOK {
		t.Fatal(res)
	}
	if res := r.run("hook", "old", "new", "1"); res.code != ExitOK || res.all() != "" {
		t.Fatalf("same bucket: %+v", res)
	}
	if res := r.run("switch", "prod"); res.code != ExitOK {
		t.Fatal(res)
	}
	reset := r.run("hook", "post-checkout", "old", "new", "1")
	if reset.code != ExitOK || reset.stderr != "envbuckets: Switched to bucket 'dev' (default)\n" {
		t.Fatalf("reset: %+v", reset)
	}
	r.linkTarget(".env", ".env.d/dev/.env")
}

func TestSwitchWithoutReadingBucketFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix mode bits required")
	}
	r := switchRepo(t)
	for _, name := range []string{".env.d/dev/.env", ".env.d/prod/.env", ".env.d/dev/apps/api/key.json"} {
		if err := os.Chmod(r.path(name), 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(r.path(name), 0o600) })
	}
	for _, args := range [][]string{{"switch"}, {"switch", "prod"}, {"switch", "--dry-run"}, {"hook", "old", "new", "1"}} {
		res := r.run(args...)
		if res.code != ExitOK || strings.Contains(res.all(), "SYNTHETIC_") {
			t.Fatalf("unreadable %v: %+v", args, res)
		}
	}
}

func TestSwitchUsage(t *testing.T) {
	r := newRepo(t)
	if res := r.run("switch", ""); res.code != ExitError || !strings.Contains(res.stderr, "invalid bucket name") {
		t.Fatalf("empty bucket: %+v", res)
	}
	for _, args := range [][]string{{"switch", "-x"}, {"switch", "dev", "prod"}, {"switch", "-c"}} {
		res := r.run(args...)
		if res.code != ExitUsage || !strings.Contains(res.stderr, "usage:") {
			t.Fatalf("usage %v: %+v", args, res)
		}
	}
}
