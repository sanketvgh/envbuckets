package cli

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/sanketvgh/envbuckets/internal/block"
	"github.com/sanketvgh/envbuckets/internal/config"
)

func requireRepoSymlinks(t *testing.T, r *repo) {
	t.Helper()
	if err := os.Symlink("README", r.path("probe")); err != nil {
		t.Skipf("symlink support required: %v", err)
	}
	if err := os.Remove(r.path("probe")); err != nil {
		t.Fatal(err)
	}
}

// Snapshots read only synthetic test repositories, never the user's worktree.
func snapshotRepo(t *testing.T, r *repo) map[string]string {
	t.Helper()
	root, err := os.OpenRoot(r.root)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	snapshot := make(map[string]string)
	err = fs.WalkDir(root.FS(), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		value := info.Mode().String()
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := root.Readlink(name)
			if err != nil {
				return err
			}
			value += target
		case info.Mode().IsRegular():
			content, err := root.ReadFile(name)
			if err != nil {
				return err
			}
			value += fmt.Sprintf("%x", sha256.Sum256(content))
		}
		snapshot[name] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestInitImportsAndReruns(t *testing.T) {
	r := newRepo(t)
	requireRepoSymlinks(t, r)
	r.write(".gitignore", ".env\nnode_modules/\n")
	r.write(".env", "SYNTHETIC_SECRET")
	r.write("apps/web/.env.local", "SYNTHETIC_NESTED")
	r.write("apps/web/.env.example", "SYNTHETIC_TRACKED")
	r.git("add", "apps/web/.env.example")
	r.write("node_modules/.env", "SYNTHETIC_IGNORED_FOLDER")
	before := snapshotRepo(t, r)
	dry := r.run("init", "-n")
	want := "Would create .envbuckets.json\nWould create bucket 'dev'\nWould add .env to bucket 'dev'\nWould add apps/web/.env.local to bucket 'dev'\nWould update .gitignore\nWould install .git/hooks/post-checkout\n"
	if dry.code != ExitOK || dry.stdout != want || dry.stderr != "" {
		t.Fatalf("dry: %+v", dry)
	}
	if !reflect.DeepEqual(before, snapshotRepo(t, r)) {
		t.Fatal("init dry run changed the repository")
	}
	res := r.run("init")
	want = "Adding .env to bucket 'dev'\nAdding apps/web/.env.local to bucket 'dev'\nInitialized envbuckets in " + filepath.ToSlash(r.root) + "/.env.d/\n"
	if res.code != ExitOK || res.stdout != want || res.stderr != "" {
		t.Fatalf("init: %+v", res)
	}
	r.linkTarget(".env", ".env.d/dev/.env")
	r.linkTarget("apps/web/.env.local", "../../.env.d/dev/apps/web/.env.local")
	for _, name := range []string{"apps/web/.env.example", "node_modules/.env"} {
		info, err := os.Lstat(r.path(name))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("skipped %s changed: %v", name, err)
		}
	}
	content, err := os.ReadFile(r.path(".envbuckets.json"))
	if err != nil || string(content) != initialConfig {
		t.Fatalf("initial config: %s %v", content, err)
	}
	if _, err := config.Parse(content); err != nil {
		t.Fatal(err)
	}
	before = snapshotRepo(t, r)
	if res := r.run("init"); res.code != ExitOK {
		t.Fatalf("rerun: %+v", res)
	}
	if !reflect.DeepEqual(before, snapshotRepo(t, r)) {
		t.Fatal("init rerun changed the repository")
	}
	if res := r.run("init", "-n"); res.code != ExitOK || res.stdout != "" {
		t.Fatalf("rerun dry: %+v", res)
	}
}

func TestInitAlphaCollisionAndFreshClone(t *testing.T) {
	r := newRepo(t)
	requireRepoSymlinks(t, r)
	r.write(".envbuckets.json", `{"default":"local","rules":[]}`)
	r.write(".envbuckets.toml", "unused synthetic config")
	r.write(".gitignore", "# keep\n# >>> envbuckets v1 >>>\nold.env\n# <<< envbuckets v1 <<<\n# >>> envbuckets >>>\nother.env\n# <<< envbuckets <<<\n")
	r.write(".git/hooks/post-checkout", "#!/bin/sh\n# custom\n# >>> envbuckets v1 >>>\nold\n# <<< envbuckets v1 <<<\n")
	r.write(".env.d/local/.env", "SYNTHETIC_BUCKET")
	r.write(".env", "SYNTHETIC_COLLISION")
	r.write(".env.local", "SYNTHETIC_SAFE")
	res := r.run("init")
	if res.code != ExitOK || !strings.Contains(res.stderr, "cannot import '.env'") || !strings.Contains(res.stderr, ".envbuckets.toml is unused") {
		t.Fatalf("alpha: %+v", res)
	}
	for _, name := range []string{".gitignore", ".git/hooks/post-checkout"} {
		content, err := os.ReadFile(r.path(name))
		if err != nil || strings.Count(string(content), block.Begin) != 1 || strings.Count(string(content), block.End) != 1 || strings.Contains(string(content), "envbuckets v1") {
			t.Fatalf("duplicate/legacy block in %s: %s %v", name, content, err)
		}
	}
	content, err := os.ReadFile(r.path(".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "SYNTHETIC_COLLISION" {
		t.Fatal("collision source changed")
	}
	r.linkTarget(".env.local", ".env.d/local/.env.local")
	clone := newRepo(t)
	requireRepoSymlinks(t, clone)
	clone.write(".envbuckets.json", `{"default":"local","rules":[]}`)
	clone.write(".gitignore", block.Begin+"\n.env.d/\n.env\n"+block.End+"\n")
	beforeConfig, err := os.ReadFile(clone.path(".envbuckets.json"))
	if err != nil {
		t.Fatal(err)
	}
	beforeIgnore, err := os.ReadFile(clone.path(".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	res = clone.run("init")
	if res.code != ExitOK || !strings.Contains(res.stderr, "No local files found") {
		t.Fatalf("clone: %+v", res)
	}
	gotConfig, err := os.ReadFile(clone.path(".envbuckets.json"))
	if err != nil {
		t.Fatal(err)
	}
	gotIgnore, err := os.ReadFile(clone.path(".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotConfig) != string(beforeConfig) || string(gotIgnore) != string(beforeIgnore) {
		t.Fatal("fresh clone metadata changed")
	}
}

func TestAddPreflightsEveryPath(t *testing.T) {
	r := newRepo(t)
	r.write(".envbuckets.json", `{"default":"dev","rules":[]}`)
	r.write(".env.d/dev/existing", "SYNTHETIC_BUCKET")
	r.write("good", "SYNTHETIC_SOURCE")
	r.write("existing", "SYNTHETIC_COLLISION")
	r.write(".env.example", "SYNTHETIC_TRACKED")
	r.git("add", ".env.example")
	for _, bad := range []string{"missing", ".env.example", "existing", "../escape", ".git/config", ".env.d/dev/existing", ".gitignore", "."} {
		before := snapshotRepo(t, r)
		for _, flag := range []string{"-n", "--"} {
			res := r.run("add", flag, "good", bad)
			if res.code != ExitError || !strings.HasPrefix(res.stderr, "fatal:") {
				t.Fatalf("%s %s: %+v", flag, bad, res)
			}
			if !reflect.DeepEqual(before, snapshotRepo(t, r)) {
				t.Fatalf("failed add changed repository for %s", bad)
			}
		}
	}
}

func TestAddCurrentBucketSubdirectoryAndDryRun(t *testing.T) {
	r := switchRepo(t)
	if res := r.run("switch", "prod"); res.code != ExitOK {
		t.Fatal(res)
	}
	r.write("apps/api/key.json", "SYNTHETIC_ADD")
	r.write("literal[1].json", "SYNTHETIC_LITERAL")
	before := snapshotRepo(t, r)
	dry := r.runIn(r.path("apps/api"), "", "add", "-n", "key.json", "key.json")
	if dry.code != ExitOK || dry.stdout != "Would add apps/api/key.json to bucket 'prod'\nWould update .gitignore\n" || !reflect.DeepEqual(before, snapshotRepo(t, r)) {
		t.Fatalf("dry: %+v", dry)
	}
	res := r.runIn(r.path("apps/api"), "", "add", "key.json")
	if res.code != ExitOK || res.all() != "" {
		t.Fatalf("add: %+v", res)
	}
	r.linkTarget("apps/api/key.json", "../../.env.d/prod/apps/api/key.json")
	if res := r.run("add", "literal[1].json"); res.code != ExitOK || res.all() != "" {
		t.Fatalf("literal: %+v", res)
	}
	r.git("check-ignore", "literal[1].json")
	before = snapshotRepo(t, r)
	if res := r.run("add", "apps/api/key.json"); res.code != ExitError || !reflect.DeepEqual(before, snapshotRepo(t, r)) {
		t.Fatalf("managed link accepted: %+v", res)
	}
}

func TestMixedLinksAndDetachedWithoutLinks(t *testing.T) {
	r := switchRepo(t)
	if err := os.Symlink(".env.d/dev/.env", r.path(".env")); err != nil {
		t.Fatal(err)
	}
	r.write(".env.d/prod/other", "SYNTHETIC_OTHER")
	if err := os.Symlink(".env.d/prod/other", r.path("other")); err != nil {
		t.Fatal(err)
	}
	r.write("file", "SYNTHETIC_ADD")
	before := snapshotRepo(t, r)
	for _, args := range [][]string{{"add", "file"}, {"add", "-n", "file"}, {"switch", "-c", "new"}, {"switch", "-n", "-c", "new"}} {
		res := r.run(args...)
		if res.code != ExitError || !strings.Contains(res.stderr, "run \"envbuckets switch\" first") || !reflect.DeepEqual(before, snapshotRepo(t, r)) {
			t.Fatalf("mixed %v: %+v", args, res)
		}
	}
	for _, name := range []string{".env", "other"} {
		if err := os.Remove(r.path(name)); err != nil {
			t.Fatal(err)
		}
	}
	r.git("checkout", "-q", "--detach")
	if res := r.run("add", "file"); res.code != ExitError || !strings.Contains(res.stderr, "HEAD is detached") {
		t.Fatalf("detached: %+v", res)
	}
}

func TestCreateBucketAndDryRun(t *testing.T) {
	r := switchRepo(t)
	// No links yet: use the branch's bucket, including unlinked bucket files.
	before := snapshotRepo(t, r)
	dry := r.run("switch", "-n", "-c", "new")
	if dry.code != ExitOK || !strings.Contains(dry.stdout, "Would create empty .env.d/new/apps/api/key.json") || !reflect.DeepEqual(before, snapshotRepo(t, r)) {
		t.Fatalf("dry: %+v", dry)
	}
	res := r.run("switch", "-c", "new")
	if res.code != ExitOK || res.stdout != "Switched to a new bucket 'new'\n" || !strings.Contains(res.stderr, "This branch uses 'dev'") {
		t.Fatalf("create: %+v", res)
	}
	for _, name := range []string{".env", "apps/api/key.json"} {
		info, err := os.Stat(r.path(".env.d/new/" + name))
		if err != nil || info.Size() != 0 {
			t.Fatalf("new file not empty: %s %v", name, err)
		}
		content, err := os.ReadFile(r.path(".env.d/dev/" + name))
		if err != nil || !strings.Contains(string(content), "SYNTHETIC_") {
			t.Fatal("current bucket changed")
		}
	}
	r.linkTarget(".env", ".env.d/new/.env")
	before = snapshotRepo(t, r)
	for _, args := range [][]string{{"switch", "-c", "new"}, {"switch", "-n", "-c", "new"}, {"switch", "-c", "../bad"}} {
		res := r.run(args...)
		if res.code != ExitError || !reflect.DeepEqual(before, snapshotRepo(t, r)) {
			t.Fatalf("existing/invalid %v: %+v", args, res)
		}
	}
}

func TestCreatePreflightsRealFiles(t *testing.T) {
	r := switchRepo(t)
	r.write(".env", "SYNTHETIC_REAL")
	before := snapshotRepo(t, r)
	for _, args := range [][]string{{"switch", "-n", "-c", "new"}, {"switch", "-c", "new"}} {
		res := r.run(args...)
		if res.code != ExitError || !strings.Contains(res.stderr, "real file") || !reflect.DeepEqual(before, snapshotRepo(t, r)) {
			t.Fatalf("real %v: %+v", args, res)
		}
	}
}

func TestInitAndAddRefuseUnsafeMetadata(t *testing.T) {
	for _, name := range []string{".gitignore", ".env.d", ".env.d/dev"} {
		t.Run(name, func(t *testing.T) {
			r := newRepo(t)
			requireRepoSymlinks(t, r)
			r.write(".envbuckets.json", `{"default":"dev","rules":[]}`)
			r.write(".env", "SYNTHETIC_SOURCE")
			r.write("other/.env", "SYNTHETIC_TARGET")
			if name == ".gitignore" {
				if err := os.MkdirAll(r.path(".env.d/dev"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.MkdirAll(filepath.Dir(r.path(name)), 0o755); err != nil {
				t.Fatal(err)
			}
			target := "other"
			if name == ".gitignore" {
				target = "other/.env"
			}
			if name == ".env.d/dev" {
				target = "../other"
			}
			if err := os.Symlink(target, r.path(name)); err != nil {
				t.Fatal(err)
			}
			before := snapshotRepo(t, r)
			for _, args := range [][]string{{"init", "-n"}, {"init"}, {"add", "-n", ".env"}, {"add", ".env"}} {
				res := r.run(args...)
				after := snapshotRepo(t, r)
				if res.code != ExitError || !reflect.DeepEqual(before, after) {
					for path, value := range before {
						if after[path] != value {
							t.Logf("changed or removed path: %s", path)
						}
					}
					for path := range after {
						if _, exists := before[path]; !exists {
							t.Logf("added path: %s", path)
						}
					}
					t.Fatalf("unsafe %v: %+v", args, res)
				}
			}
		})
	}
}

func TestInitAndAddMoveUnreadableFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permissions required")
	}
	r := newRepo(t)
	requireRepoSymlinks(t, r)
	for _, name := range []string{".env", "key.json"} {
		r.write(name, "SYNTHETIC_UNREADABLE")
		if err := os.Chmod(r.path(name), 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(r.path(".env.d/dev/"+name), 0o600) })
	}
	for _, args := range [][]string{{"init"}, {"add", "key.json"}} {
		res := r.run(args...)
		if res.code != ExitOK || strings.Contains(res.all(), "SYNTHETIC_") {
			t.Fatalf("unreadable %v: %+v", args, res)
		}
	}
	for _, name := range []string{".env", "key.json"} {
		r.linkTarget(name, ".env.d/dev/"+name)
	}
}

func TestInitSymlinkPrivilegePreflight(t *testing.T) {
	r := newRepo(t)
	probeErr := os.Symlink("README", r.path("probe"))
	if probeErr == nil {
		_ = os.Remove(r.path("probe"))
		t.Skip("this test exercises a machine without symlink privilege")
	}
	before := snapshotRepo(t, r)
	for _, args := range [][]string{{"init", "-n"}, {"init"}} {
		res := r.run(args...)
		if res.code != ExitError || !strings.Contains(res.stderr, "cannot create symlinks") && !strings.Contains(res.stderr, "Developer Mode") || !reflect.DeepEqual(before, snapshotRepo(t, r)) {
			t.Fatalf("probe %v: %+v", args, res)
		}
	}
	// A valid add must also fail before writing its ignore block or moving its
	// source when the operating system denies symlink creation.
	r.write(".envbuckets.json", `{"default":"dev","rules":[]}`)
	r.write(".env.d/dev/existing", "SYNTHETIC_BUCKET")
	r.write("key.json", "SYNTHETIC_SOURCE")
	before = snapshotRepo(t, r)
	for _, args := range [][]string{{"add", "-n", "key.json"}, {"add", "key.json"}} {
		res := r.run(args...)
		if res.code != ExitError || !reflect.DeepEqual(before, snapshotRepo(t, r)) {
			t.Fatalf("add privilege failure %v: %+v", args, res)
		}
	}
}

func TestImportUsage(t *testing.T) {
	r := newRepo(t)
	for _, args := range [][]string{{"init", "extra"}, {"init", "-x"}, {"add"}, {"add", "-x"}, {"switch", "-c"}} {
		if res := r.run(args...); res.code != ExitUsage || !strings.Contains(res.stderr, "usage:") {
			t.Fatalf("usage %v: %+v", args, res)
		}
	}
}
