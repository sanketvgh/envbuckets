package cli

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/sanketvgh/envbuckets/internal/block"
	"github.com/sanketvgh/envbuckets/internal/fsx"
)

const uninstallHint = "hint: Your other buckets are still in .env.d/. Delete it when you no longer need them.\n"

func uninstallLink(t *testing.T, r *repo, name, target string) {
	t.Helper()
	if err := os.Symlink(target, r.path(name)); err != nil {
		t.Fatal(err)
	}
}

func TestUninstallRestoresFilesAndReinitializes(t *testing.T) {
	r := newRepo(t)
	requireRepoSymlinks(t, r)
	r.write(".env", "SYNTHETIC_ENV")
	r.write("apps/api/.env.local", "SYNTHETIC_NESTED")
	if res := r.run("init"); res.code != ExitOK {
		t.Fatalf("init: %+v", res)
	}
	r.write(".env.d/prod/.env", "SYNTHETIC_OTHER")
	r.write(".env.d/prod/unlinked", "SYNTHETIC_UNLINKED")
	before := snapshotRepo(t, r)
	// Capture identity from a handle before the move, including on Windows.
	file, err := os.Open(r.path(".env.d/dev/.env"))
	if err != nil {
		t.Fatal(err)
	}
	original, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil || closeErr != nil {
		t.Fatalf("capture identity: %v %v", statErr, closeErr)
	}
	want := "Would move .env.d/dev/.env to .env\nWould move .env.d/dev/apps/api/.env.local to apps/api/.env.local\nWould remove the envbuckets hook from .git/hooks/post-checkout\n"
	if res := r.runIn(r.path("apps/api"), "uninstall", "--dry-run"); res.code != ExitOK || res.stdout != want || res.stderr != "" {
		t.Fatalf("dry: %+v", res)
	}
	if !reflect.DeepEqual(before, snapshotRepo(t, r)) {
		t.Fatal("dry run changed repository")
	}
	want = "Moving .env.d/dev/.env to .env\nMoving .env.d/dev/apps/api/.env.local to apps/api/.env.local\nRemoving the envbuckets hook from .git/hooks/post-checkout\n"
	if res := r.run("uninstall"); res.code != ExitOK || res.stdout != want || res.stderr != uninstallHint {
		t.Fatalf("uninstall: %+v", res)
	}
	restored, err := os.Lstat(r.path(".env"))
	if err != nil || !restored.Mode().IsRegular() || !os.SameFile(original, restored) {
		t.Fatal("bucket file was not moved over the link")
	}
	after := snapshotRepo(t, r)
	for _, name := range []string{".envbuckets.json", ".gitignore", ".env.d/prod/.env", ".env.d/prod/unlinked"} {
		if before[name] != after[name] {
			t.Fatalf("%s changed", name)
		}
	}
	for _, name := range []string{".env.d/dev/.env", ".env.d/dev/apps/api/.env.local", ".git/hooks/post-checkout"} {
		if _, exists := after[name]; exists {
			t.Fatalf("%s remains", name)
		}
	}
	if res := r.run("uninstall"); res.code != ExitOK || res.all() != "" {
		t.Fatalf("second uninstall: %+v", res)
	}
	if !reflect.DeepEqual(after, snapshotRepo(t, r)) {
		t.Fatal("second uninstall changed repository")
	}
	if res := r.run("init"); res.code != ExitOK || res.stderr != "" {
		t.Fatalf("reinit: %+v", res)
	}
	r.linkTarget(".env", ".env.d/dev/.env")
	r.linkTarget("apps/api/.env.local", "../../.env.d/dev/apps/api/.env.local")
}

func TestUninstallMissingAndBrokenConfigMixedBuckets(t *testing.T) {
	for _, cfg := range []string{"", "invalid config"} {
		t.Run(map[string]string{"": "missing", "invalid config": "broken"}[cfg], func(t *testing.T) {
			r := newRepo(t)
			requireRepoSymlinks(t, r)
			if cfg != "" {
				r.write(".envbuckets.json", cfg)
			}
			r.write(".env.d/prod/.env", "SYNTHETIC_PROD")
			r.write(".env.d/dev/key.json", "SYNTHETIC_DEV")
			uninstallLink(t, r, ".env", ".env.d/prod/.env")
			uninstallLink(t, r, "key.json", ".env.d/dev/key.json")
			want := "Moving .env.d/prod/.env to .env\nMoving .env.d/dev/key.json to key.json\n"
			if res := r.run("uninstall"); res.code != ExitOK || res.stdout != want || res.stderr != uninstallHint {
				t.Fatalf("uninstall: %+v", res)
			}
			if cfg != "" {
				got, err := os.ReadFile(r.path(".envbuckets.json"))
				if err != nil || string(got) != cfg {
					t.Fatal("config changed")
				}
			}
		})
	}
}

func TestUninstallBrokenForeignAndRealPaths(t *testing.T) {
	r := newRepo(t)
	requireRepoSymlinks(t, r)
	for _, name := range []string{"broken", "foreign", "real", "good"} {
		r.write(".env.d/dev/"+name, "SYNTHETIC_DEV")
	}
	r.write(".env.d/prod/good", "SYNTHETIC_PROD")
	r.write("outside-target", "SYNTHETIC_FOREIGN")
	r.write("real", "SYNTHETIC_REAL")
	uninstallLink(t, r, "broken", ".env.d/deleted/broken")
	uninstallLink(t, r, "foreign", "outside-target")
	uninstallLink(t, r, "good", ".env.d/prod/good")
	r.write(".git/hooks/post-checkout", "#!/bin/sh\n# custom\n"+block.Begin+"\nx\n"+block.End+"\n")
	before := snapshotRepo(t, r)
	wantErr := "error: cannot restore 'broken': bucket target is missing (broken link)\n"
	if res := r.run("uninstall", "-n"); res.code != ExitError || res.stdout != "Would move .env.d/prod/good to good\nWould remove the envbuckets hook from .git/hooks/post-checkout\n" || res.stderr != wantErr {
		t.Fatalf("dry: %+v", res)
	}
	if !reflect.DeepEqual(before, snapshotRepo(t, r)) {
		t.Fatal("failed dry run changed repository")
	}
	if res := r.run("uninstall"); res.code != ExitError || res.stdout != "Moving .env.d/prod/good to good\nRemoving the envbuckets hook from .git/hooks/post-checkout\n" || res.stderr != wantErr+uninstallHint {
		t.Fatalf("partial: %+v", res)
	}
	after := snapshotRepo(t, r)
	for _, name := range []string{"broken", "foreign", "outside-target", "real", ".env.d/dev/broken", ".env.d/dev/foreign", ".env.d/dev/real", ".env.d/dev/good"} {
		if before[name] != after[name] {
			t.Fatalf("preserved path %s changed", name)
		}
	}
	if res := r.run("uninstall"); res.code != ExitError || res.stdout != "" || res.stderr != wantErr {
		t.Fatalf("partial rerun: %+v", res)
	}
}

func TestUninstallResumesAfterFirstMove(t *testing.T) {
	r := newRepo(t)
	requireRepoSymlinks(t, r)
	for _, name := range []string{"first", "second"} {
		r.write(".env.d/dev/"+name, "SYNTHETIC_"+name)
		uninstallLink(t, r, name, ".env.d/dev/"+name)
	}
	root, err := fsx.OpenRepo(r.root)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := fsx.RestoreBucketFile(root, "first", ".env.d/dev/first", ".env.d/dev/first"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(r.path("first"))
	if err != nil || string(content) != "SYNTHETIC_first" {
		t.Fatal("first path is not usable after partial completion")
	}
	content, err = os.ReadFile(r.path("second"))
	if err != nil || string(content) != "SYNTHETIC_second" {
		t.Fatal("remaining link is not usable after partial completion")
	}
	if res := r.run("uninstall"); res.code != ExitOK || res.stdout != "Moving .env.d/dev/second to second\n" {
		t.Fatalf("resume: %+v", res)
	}
}

func TestUninstallNoSetupHookOnlyAndUsage(t *testing.T) {
	r := newRepo(t)
	for _, cfg := range []string{"", "broken"} {
		if cfg != "" {
			r.write(".envbuckets.json", cfg)
		}
		before := snapshotRepo(t, r)
		for _, args := range [][]string{{"uninstall"}, {"uninstall", "-n"}} {
			if res := r.run(args...); res.code != ExitOK || res.all() != "" {
				t.Fatalf("no setup: %+v", res)
			}
		}
		if !reflect.DeepEqual(before, snapshotRepo(t, r)) {
			t.Fatal("no-op changed repository")
		}
	}
	r.git("config", "core.hooksPath", ".hooks")
	r.write(".hooks/post-checkout", "#!/bin/sh\n# >>> envbuckets v1 >>>\nold\n# <<< envbuckets v1 <<<\n")
	before := snapshotRepo(t, r)
	if res := r.run("uninstall", "-n"); res.code != ExitOK || res.stdout != "Would remove the envbuckets hook from .hooks/post-checkout\n" || res.stderr != "" {
		t.Fatalf("hook dry: %+v", res)
	}
	if !reflect.DeepEqual(before, snapshotRepo(t, r)) {
		t.Fatal("hook dry run changed repository")
	}
	if res := r.run("uninstall"); res.code != ExitOK || res.stdout != "Removing the envbuckets hook from .hooks/post-checkout\n" || res.stderr != uninstallHint {
		t.Fatalf("hook removal: %+v", res)
	}
	if _, err := os.Lstat(r.path(".hooks/post-checkout")); !os.IsNotExist(err) {
		t.Fatal("hook remains")
	}
	for _, args := range [][]string{{"uninstall", "extra"}, {"uninstall", "-x"}} {
		if res := r.run(args...); res.code != ExitUsage || !strings.Contains(res.stderr, "usage: envbuckets uninstall [-n]") {
			t.Fatalf("usage: %+v", res)
		}
	}
}

func TestUninstallMalformedHookAndUnsafeDirectory(t *testing.T) {
	r := newRepo(t)
	r.write(".git/hooks/post-checkout", block.Begin+"\ntruncated\n")
	before := snapshotRepo(t, r)
	for _, args := range [][]string{{"uninstall"}, {"uninstall", "-n"}} {
		if res := r.run(args...); res.code != ExitError || res.stdout != "" || !strings.Contains(res.stderr, "incomplete envbuckets hook block") {
			t.Fatalf("malformed hook: %+v", res)
		}
	}
	if !reflect.DeepEqual(before, snapshotRepo(t, r)) {
		t.Fatal("malformed hook changed")
	}
	if err := os.Remove(r.path(".git/hooks/post-checkout")); err != nil {
		t.Fatal(err)
	}
	r.write(".env.d", "SYNTHETIC_FILE")
	if res := r.run("uninstall"); res.code != ExitError || res.stderr != "error: cannot inspect buckets: .env.d is not a directory\n" {
		t.Fatalf("unsafe directory: %+v", res)
	}
}

func TestUninstallUnsafePathsRemainUnchanged(t *testing.T) {
	for _, scenario := range []string{"bucket root symlink", "bucket symlink", "source symlink", "working parent symlink", "bucket parent symlink", "tracked path", "hook symlink", "external hooks", "bucket hooks"} {
		t.Run(scenario, func(t *testing.T) {
			r := newRepo(t)
			requireRepoSymlinks(t, r)
			r.write("keep", "SYNTHETIC_KEEP")
			switch scenario {
			case "bucket root symlink":
				r.write("data/dev/file", "SYNTHETIC_BUCKET")
				uninstallLink(t, r, ".env.d", "data")
			case "bucket symlink":
				r.write(".env.d/prod/file", "SYNTHETIC_BUCKET")
				uninstallLink(t, r, ".env.d/dev", "prod")
			case "source symlink":
				r.write(".env.d/prod/file", "SYNTHETIC_BUCKET")
				uninstallLink(t, r, ".env.d/prod/unsafe", "../../keep")
				uninstallLink(t, r, "unsafe", ".env.d/prod/unsafe")
			case "working parent symlink":
				r.write(".env.d/dev/app/file", "SYNTHETIC_BUCKET")
				r.write("other/file", "SYNTHETIC_REAL")
				uninstallLink(t, r, "app", "other")
			case "bucket parent symlink":
				r.write(".env.d/prod/app/file", "SYNTHETIC_BUCKET")
				r.write(".env.d/dev/keep", "SYNTHETIC_DEV")
				r.write("app/keep", "SYNTHETIC_REAL")
				uninstallLink(t, r, ".env.d/dev/app", "../prod/app")
				uninstallLink(t, r, "app/file", "../.env.d/dev/app/file")
			case "tracked path":
				r.write(".env.d/dev/file", "SYNTHETIC_BUCKET")
				uninstallLink(t, r, "file", ".env.d/dev/file")
				r.git("add", "file")
			case "hook symlink":
				uninstallLink(t, r, ".git/hooks/post-checkout", "../../keep")
			case "external hooks":
				r.git("config", "core.hooksPath", "../shared-hooks")
			case "bucket hooks":
				r.git("config", "core.hooksPath", ".env.d/dev")
				r.write(".env.d/dev/post-checkout", "SYNTHETIC_BUCKET")
			}
			before := snapshotRepo(t, r)
			for _, args := range [][]string{{"uninstall", "-n"}, {"uninstall"}} {
				res := r.run(args...)
				if res.code != ExitError || res.stdout != "" || !strings.HasPrefix(res.stderr, "error:") {
					t.Fatalf("unsafe scenario: %+v", res)
				}
				if !reflect.DeepEqual(before, snapshotRepo(t, r)) {
					t.Fatal("unsafe scenario changed repository")
				}
			}
		})
	}
}
