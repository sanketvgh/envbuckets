package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMoveFailureRestoresSource(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "hardlink", true: "rename"}[fallback], func(t *testing.T) {
			dir := t.TempDir()
			requireSymlinks(t, dir)
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if err := root.MkdirAll(".env.d/dev", 0o700); err != nil {
				t.Fatal(err)
			}
			if err := root.WriteFile(".env", []byte("SYNTHETIC_SOURCE"), 0o600); err != nil {
				t.Fatal(err)
			}
			hardlink := root.Link
			if fallback {
				hardlink = func(_, _ string) error { return errors.New("hard links unavailable") }
			}
			failure := errors.New("injected link installation failure")
			rename := func(oldname, newname string) error {
				if strings.HasPrefix(filepath.Base(oldname), ".envbuckets-link-") {
					return failure
				}
				return root.Rename(oldname, newname)
			}
			_, err = moveFileToBucket(root, ".env", ".env.d/dev/.env", ".env.d/dev/.env", hardlink, rename)
			if !errors.Is(err, failure) {
				t.Fatalf("move: %v", err)
			}
			info, err := root.Lstat(".env")
			if err != nil || !info.Mode().IsRegular() {
				t.Fatalf("source not restored: %v %v", info, err)
			}
			content, err := root.ReadFile(".env")
			if err != nil || string(content) != "SYNTHETIC_SOURCE" {
				t.Fatal("source changed")
			}
			if _, err := root.Lstat(".env.d/dev/.env"); !os.IsNotExist(err) {
				t.Fatalf("failed move left destination: %v", err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".envbuckets-link-") {
					t.Fatal("staged symlink left behind")
				}
			}
		})
	}
}
