package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/sanketvgh/envbuckets/internal/block"
)

func TestInitIsIdempotent(t *testing.T) {
	r := newRepo(t)
	first := r.ok("init")
	if !strings.Contains(first.stdout, "[created] .envbuckets.toml") || !strings.Contains(first.stdout, "[created] hook block") {
		t.Fatalf("first init:\n%s", first.stdout)
	}
	if !r.exists(".envbuckets.toml") || !strings.Contains(r.read(".gitignore"), block.Begin) {
		t.Fatal("structure not created")
	}
	second := r.ok("init")
	if strings.Contains(second.stdout, "[created]") {
		t.Fatalf("second init not idempotent:\n%s", second.stdout)
	}
}

func TestSwitchBranchesAndSilenceWhenUnchanged(t *testing.T) {
	r := setup(t)
	r.hook()
	if got := r.readlink(".env"); got != ".env.d/staging/.env" {
		t.Fatalf("main should map to staging, got %s", got)
	}
	r.newBranch("release/1.0")
	res := r.hook()
	if !strings.Contains(res.stdout, "staging -> prod (release/*)") {
		t.Fatalf("summary line missing:\n%s", res.all())
	}
	if got := r.readlink(".env"); got != ".env.d/prod/.env" {
		t.Fatalf("got %s", got)
	}
	r.newBranch("release/1.1")
	res = r.hook()
	if res.all() != "" {
		t.Fatalf("same bucket must be silent, got:\n%s", res.all())
	}
	r.checkoutMain()
	r.hook()
	if got := r.readlink(".env"); got != ".env.d/staging/.env" {
		t.Fatalf("got %s", got)
	}
}

func TestUnmappedBranchLeavesEnvUntouched(t *testing.T) {
	r := setup(t)
	r.ok("map", "rm", "*")
	r.newBranch("feature/x")
	res := r.hook()
	if !strings.Contains(res.stderr, "no rule matches") || strings.Count(strings.TrimSpace(res.all()), "\n") != 0 {
		t.Fatalf("want one hint line:\n%s", res.all())
	}
	if got := r.readlink(".env"); got != ".env.d/dev/.env" {
		t.Fatalf("env changed: %s", got)
	}
}

func TestCatchAllAndFirstMatchWins(t *testing.T) {
	r := setup(t)
	r.ok("use", "staging")
	r.newBranch("anything/else")
	r.hook()
	if got := r.readlink(".env"); got != ".env.d/dev/.env" {
		t.Fatalf("catch-all: got %s", got)
	}
	r.newBranch("release/1.2/x")
	r.hook()
	if got := r.readlink(".env"); got != ".env.d/prod/.env" {
		t.Fatalf("first match: got %s", got)
	}
}

func TestUseIsATransientOverride(t *testing.T) {
	r := setup(t)
	r.newBranch("feature/a")
	r.hook()
	res := r.ok("use", "prod")
	if !strings.Contains(res.stdout, "manual override") || r.readlink(".env") != ".env.d/prod/.env" {
		t.Fatalf("use:\n%s", res.all())
	}
	st := r.ok("status")
	if !strings.Contains(st.stdout, "manual override") {
		t.Fatalf("status:\n%s", st.stdout)
	}
	r.checkoutMain()
	r.hook()
	if r.readlink(".env") != ".env.d/staging/.env" {
		t.Fatal("next matching checkout should repoint")
	}
}

func TestHandEditsSurviveSwitching(t *testing.T) {
	r := setup(t)
	r.hook()
	r.write(".env", "EDITED=1\n")
	r.newBranch("release/1")
	r.hook()
	r.checkoutMain()
	r.hook()
	if r.read(".env.d/staging/.env") != "EDITED=1\n" {
		t.Fatal("edit lost")
	}
}

func TestMissingBucketFileKeepsOldEnv(t *testing.T) {
	r := setup(t)
	if err := os.RemoveAll(r.path(".env.d/staging")); err != nil {
		t.Fatal(err)
	}
	res := r.hook()
	if !strings.Contains(res.stderr, "missing .env.d/staging/.env") {
		t.Fatalf("want warning:\n%s", res.all())
	}
	if r.readlink(".env") != ".env.d/dev/.env" {
		t.Fatal("old env not kept")
	}
	if _, err := os.Stat(r.path(".env")); err != nil {
		t.Fatal("dangling symlink")
	}
}

func TestStatusReportsBroken(t *testing.T) {
	r := setup(t)
	if err := os.Remove(r.path(".env.d/dev/.env")); err != nil {
		t.Fatal(err)
	}
	res := r.ok("status")
	if !strings.Contains(res.stdout, "BROKEN") || !strings.Contains(res.stdout, "create it") {
		t.Fatalf("status:\n%s", res.stdout)
	}
	if err := os.Remove(r.path(".env.d/staging/.env")); err != nil {
		t.Fatal(err)
	}
	hook := r.hook()
	if !strings.Contains(hook.stderr, "missing .env.d/staging/.env") || r.readlink(".env") != ".env.d/dev/.env" {
		t.Fatalf("hook should keep stale and warn:\n%s", hook.all())
	}
}

func TestRealEnvIsNeverTouchedByHook(t *testing.T) {
	r := setup(t)
	if err := os.Remove(r.path(".env")); err != nil {
		t.Fatal(err)
	}
	r.write(".env", "REAL=1\n")
	res := r.hook()
	if !strings.Contains(res.stderr, "real file") || r.read(".env") != "REAL=1\n" {
		t.Fatalf("hook:\n%s", res.all())
	}
}

func TestFileCheckoutIsSilent(t *testing.T) {
	r := newRepo(t)
	res := r.run("hook", "a", "b", "0")
	if res.code != 0 || res.all() != "" {
		t.Fatalf("got %d %q", res.code, res.all())
	}
}

func TestDetachedHead(t *testing.T) {
	r := newRepo(t)
	r.ok("init")
	r.git("checkout", "-q", "--detach")
	res := r.hook()
	if !strings.Contains(res.stderr, "detached HEAD") {
		t.Fatalf("got %q", res.all())
	}
	st := r.ok("status")
	if !strings.Contains(st.stdout, "detached") {
		t.Fatalf("status:\n%s", st.stdout)
	}
}

func TestCustomHookPreserved(t *testing.T) {
	r := newRepo(t)
	custom := "#!/bin/sh\necho custom\n"
	r.write(".git/hooks/post-checkout", custom)
	r.ok("init")
	got := r.read(".git/hooks/post-checkout")
	if !strings.HasPrefix(got, custom) || !strings.Contains(got, block.Begin) || !strings.Contains(got, `envbuckets hook "$@" || true`) {
		t.Fatalf("hook:\n%s", got)
	}
}

func TestCorruptTomlRefusesButHookSucceeds(t *testing.T) {
	r := setup(t)
	before := r.readlink(".env")
	r.write(".envbuckets.toml", "schema = 1\n[[rules]]\npattern = \"ma")
	hook := r.hook()
	if !strings.Contains(hook.stderr, "dormant") {
		t.Fatalf("hook:\n%s", hook.all())
	}
	for _, args := range [][]string{{"status"}, {"use", "dev"}, {"map", "list"}, {"bucket", "list"}, {"scope", "list"}, {"init"}} {
		res := r.run(args...)
		if res.code != ExitConfig {
			t.Errorf("%v: want exit 2, got %d\n%s", args, res.code, res.all())
		}
	}
	if r.readlink(".env") != before {
		t.Fatal("symlink changed")
	}
	if r.read(".envbuckets.toml") != "schema = 1\n[[rules]]\npattern = \"ma" {
		t.Fatal("corrupt config was rewritten")
	}
}

func TestSchemaTooNewRefuses(t *testing.T) {
	r := newRepo(t)
	r.write(".envbuckets.toml", "schema = 2\n")
	if res := r.run("status"); res.code != ExitConfig {
		t.Fatalf("got %d", res.code)
	}
}

func TestMapAddUnknownBucketBlocked(t *testing.T) {
	r := newRepo(t)
	r.ok("init")
	before := r.read(".envbuckets.toml")
	res := r.run("map", "add", "x", "nope")
	if res.code != ExitBlocked || !strings.Contains(res.stderr, "bucket add nope") {
		t.Fatalf("got %d %s", res.code, res.all())
	}
	if r.read(".envbuckets.toml") != before {
		t.Fatal("TOML changed")
	}
}

func TestBucketRmGuards(t *testing.T) {
	r := setup(t)
	res := r.run("bucket", "rm", "prod")
	if res.code != ExitBlocked || !strings.Contains(res.stderr, "release/*") {
		t.Fatalf("referenced: %d %s", res.code, res.all())
	}
	r.ok("bucket", "add", "scratch")
	r.write(".env.d/scratch/.env", "SECRET=1\n")
	res = r.run("bucket", "rm", "scratch")
	if res.code != ExitBlocked || !strings.Contains(res.stderr, "--purge") {
		t.Fatalf("non-empty: %d %s", res.code, res.all())
	}
	res = r.runIn(r.root, "no\n", "bucket", "rm", "scratch", "--purge")
	if res.code != ExitBlocked || !r.exists(".env.d/scratch/.env") {
		t.Fatalf("wrong confirmation should block: %d", res.code)
	}
	res = r.runIn(r.root, "DELETE\n", "bucket", "rm", "scratch", "--purge")
	if res.code != ExitOK || r.exists(".env.d/scratch") {
		t.Fatalf("purge: %d %s", res.code, res.all())
	}
	res = r.run("bucket", "rm", "dev")
	if res.code != ExitBlocked {
		t.Fatalf("active bucket should be blocked: %d", res.code)
	}
}

func TestDuplicateAndCatchAllBlocked(t *testing.T) {
	r := newRepo(t)
	r.ok("init")
	r.ok("bucket", "add", "dev")
	r.ok("map", "add", "main", "dev")
	if res := r.run("map", "add", "main", "dev"); res.code != ExitBlocked {
		t.Fatalf("duplicate: %d", res.code)
	}
	r.ok("map", "add", "*", "dev")
	if res := r.run("map", "add", "feature/*", "dev"); res.code != ExitBlocked || !strings.Contains(res.stderr, "catch-all") {
		t.Fatalf("catch-all: %d %s", res.code, res.all())
	}
}

func TestValueBlindnessSweep(t *testing.T) {
	r := setup(t)
	const marker = "SUPERSECRET_MARKER_9f2c"
	for _, b := range []string{"dev", "staging", "prod"} {
		r.write(".env.d/"+b+"/.env", "KEY="+marker+"\n")
	}
	r.git("config", "core.hooksPath", ".git/hooks")
	commands := [][]string{
		{"status"},
		{"init"},
		{"use", "prod"},
		{"use", "dev"},
		{"bucket", "list"},
		{"bucket", "add", "x"},
		{"bucket", "rm", "x"},
		{"map", "list"},
		{"map", "rm", "*"},
		{"map", "add", "*", "dev"},
		{"scope", "list"},
		{"scope", "add", "."},
		{"scope", "rm", "root"},
		{"hook", "a", "b", "1"},
		{"version"},
		{"help"},
		{"nope"},
		{"uninstall"},
		{"init", "--into", "dev"},
		{"init", "--into", "restored"},
		{"uninstall", "--purge"},
	}
	for _, args := range commands {
		res := r.runIn(r.root, "DELETE\n", args...)
		if strings.Contains(res.all(), marker) {
			t.Fatalf("%v printed a value:\n%s", args, res.all())
		}
	}
}

func TestCommandsFromSubdirectory(t *testing.T) {
	r := setup(t)
	sub := r.path("src/deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	res := r.runIn(sub, "", "bucket", "list")
	if res.code != ExitOK || !strings.Contains(res.stdout, "scope root") {
		t.Fatalf("%d %s", res.code, res.all())
	}
	res = r.runIn(sub, "", "use", "prod")
	if res.code != ExitOK || r.readlink(".env") != ".env.d/prod/.env" {
		t.Fatalf("%d %s", res.code, res.all())
	}
}

func TestStatusUninitialized(t *testing.T) {
	r := newRepo(t)
	res := r.run("status")
	if res.code != ExitConfig || !strings.Contains(res.stderr, "envbuckets init") {
		t.Fatalf("%d %s", res.code, res.all())
	}
	if res := r.runIn(t.TempDir(), "", "status"); res.code != ExitEnv {
		t.Fatalf("outside repo: want exit 4, got %d", res.code)
	}
}

func TestUsageErrors(t *testing.T) {
	r := newRepo(t)
	r.ok("init")
	for _, args := range [][]string{{"use"}, {"map", "add", "x"}, {"bucket"}, {"scope", "add"}, {"init", "--bogus"}} {
		if res := r.run(args...); res.code != ExitUsage {
			t.Errorf("%v: want exit 3, got %d", args, res.code)
		}
	}
}
