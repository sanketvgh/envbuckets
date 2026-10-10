package metadata

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadRegular(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config"), []byte("synthetic config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	content, err := ReadRegular(root, "config")
	if err != nil || string(content) != "synthetic config\n" {
		t.Fatalf("regular metadata: %q, %v", content, err)
	}
	if _, err := ReadRegular(root, "missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing metadata: %v", err)
	}
	if _, err := ReadRegular(root, "."); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("directory metadata: %v", err)
	}
	if err := root.Symlink("config", "link"); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if content, err := ReadRegular(root, "link"); !errors.Is(err, ErrNotRegular) || len(content) != 0 {
		t.Fatalf("symlink metadata was read: %q, %v", content, err)
	}
}

func TestReadRegularRejectsOutsideRoot(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if content, err := ReadRegular(root, "../outside"); err == nil || len(content) != 0 {
		t.Fatalf("outside metadata was accepted: %q, %v", content, err)
	}
}
