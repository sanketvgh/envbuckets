package fsx

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRestoreBucketFileRevalidatesBeforeRename(t *testing.T) {
	for _, change := range []string{"real destination", "changed link", "symlink source", "directory source", "wrong source", "outside source"} {
		t.Run(change, func(t *testing.T) {
			dir := t.TempDir()
			requireSymlinks(t, dir)
			cmd := exec.Command("git", "init", "-q", dir) //nolint:gosec // Fixed executable and arguments; dir is a private synthetic test repository.
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git init: %v %s", err, out)
			}
			repo, err := OpenRepo(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer repo.Close()
			root := repo.Root
			if err := root.MkdirAll(".env.d/dev", 0o700); err != nil {
				t.Fatal(err)
			}
			if err := root.WriteFile(".env.d/dev/file", []byte("SYNTHETIC_BUCKET"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := root.WriteFile("keep", []byte("SYNTHETIC_KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := root.Symlink(".env.d/dev/file", "file"); err != nil {
				t.Fatal(err)
			}
			source := ".env.d/dev/file"
			switch change {
			case "real destination", "changed link":
				if err := root.Remove("file"); err != nil {
					t.Fatal(err)
				}
				if change == "real destination" {
					err = root.WriteFile("file", []byte("SYNTHETIC_REAL"), 0o600)
				} else {
					err = root.Symlink("keep", "file")
				}
			case "symlink source", "directory source":
				if err := root.Remove(source); err != nil {
					t.Fatal(err)
				}
				if change == "symlink source" {
					err = root.Symlink("../../keep", source)
				} else {
					err = root.Mkdir(source, 0o700)
				}
			case "wrong source":
				source = ".env.d/dev/keep"
			case "outside source":
				source = "keep"
			}
			if err != nil {
				t.Fatal(err)
			}
			beforeSource, err := root.Lstat(".env.d/dev/file")
			if err != nil {
				t.Fatal(err)
			}
			beforeDest, err := root.Lstat("file")
			if err != nil {
				t.Fatal(err)
			}
			if err := RestoreBucketFile(repo, "file", source, ".env.d/dev/file"); err == nil {
				t.Fatal("unsafe restoration accepted")
			}
			afterSource, err := root.Lstat(".env.d/dev/file")
			if err != nil || !os.SameFile(beforeSource, afterSource) {
				t.Fatal("source changed")
			}
			afterDest, err := root.Lstat("file")
			if err != nil || !os.SameFile(beforeDest, afterDest) {
				t.Fatal("destination changed")
			}
			keep, err := os.ReadFile(filepath.Join(dir, "keep"))
			if err != nil || string(keep) != "SYNTHETIC_KEEP" {
				t.Fatal("foreign target changed")
			}
		})
	}
}
