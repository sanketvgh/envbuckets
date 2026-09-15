package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func requireSymlinks(t *testing.T, dir string) {
	t.Helper()
	probe := filepath.Join(dir, ".symlink-probe")
	if err := os.Symlink("target", probe); err != nil {
		t.Skipf("symlinks unavailable here: %v", err)
	}
	_ = os.Remove(probe)
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := WriteFileAtomic(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "two" {
		t.Fatalf("got %q", data)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".envbuckets-") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

func TestSetSymlinkFastPathAndSwap(t *testing.T) {
	dir := t.TempDir()
	requireSymlinks(t, dir)
	stage := filepath.Join(dir, "stage")
	if err := os.Mkdir(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")

	changed, err := SetSymlink(link, "a", stage)
	if err != nil || !changed {
		t.Fatalf("create: %v %v", changed, err)
	}
	changed, err = SetSymlink(link, "a", stage)
	if err != nil || changed {
		t.Fatalf("idempotent: %v %v", changed, err)
	}
	changed, err = SetSymlink(link, "b", stage)
	if err != nil || !changed {
		t.Fatalf("swap: %v %v", changed, err)
	}
	st, err := Inspect(link)
	if err != nil || !st.IsSymlink || st.Target != "b" || !st.Dangling {
		t.Fatalf("inspect: %+v %v", st, err)
	}
	entries, _ := os.ReadDir(stage)
	if len(entries) != 0 {
		t.Fatalf("stage dir not clean: %v", entries)
	}
}

func TestInspectRealAndMissing(t *testing.T) {
	dir := t.TempDir()
	st, err := Inspect(filepath.Join(dir, "nope"))
	if err != nil || st.Exists {
		t.Fatalf("missing: %+v %v", st, err)
	}
	f := filepath.Join(dir, "real")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	st, err = Inspect(f)
	if err != nil || !st.Exists || st.IsSymlink {
		t.Fatalf("real: %+v %v", st, err)
	}
}

func TestCopyFileAtomic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFileAtomic(src, dst, dir, 0o600); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "payload" {
		t.Fatalf("got %q", data)
	}
}
