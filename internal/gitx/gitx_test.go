package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestLocalFilesIncludesIgnoredFilesButPrunesIgnoredFolders(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...) //nolint:gosec // fixed test executable and argument vector
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, out)
		}
	}
	git("init", "-q")
	for _, name := range []string{".env", "apps/api/.env.local", "node_modules/deep/.env", "submodule/.env", ".env.example", " space dir/.env local", "normal.json"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".env\n.env.local\nnode_modules/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", ".env.example")
	// Git's gitlink index entry stands for a submodule; discovery must not
	// descend into it, even though a local synthetic .env exists there.
	git("update-index", "--add", "--cacheinfo", "160000,0123456789012345678901234567890123456789,submodule")
	files, err := LocalFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(files)
	want := []string{" space dir/.env local", ".env", ".gitignore", "apps/api/.env.local", "normal.json"}
	if !slices.Equal(files, want) {
		t.Fatalf("files=%q want=%q", files, want)
	}
	for name, want := range map[string]bool{".env": true, "node_modules/": true, "normal.json": false} {
		got, err := Ignored(dir, name)
		if err != nil || got != want {
			t.Fatalf("Ignored(%s)=%v %v", name, got, err)
		}
	}
}
