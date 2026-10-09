package block

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sanketvgh/envbuckets/internal/fsx"
)

func TestIgnorePlanMergesLegacyAndEscapesLiteralPaths(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	root, err := fsx.OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	existing := "outside.env\n# >>> envbuckets v1 >>>\r\nold.env\r\n# <<< envbuckets v1 <<<\r\n# >>> envbuckets >>>\nother.env\n# <<< envbuckets <<<\n"
	if err := root.Root.WriteFile(".gitignore", []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}
	paths := []string{"outside.env", "literal[1].json", "#secret", "!secret", "apps/a b/key?.json"}
	p, err := PlanIgnore(root, paths)
	if err != nil || !p.Changed || runtime.GOOS != "windows" && p.Mode != 0o600 {
		t.Fatalf("plan=%+v err=%v", p, err)
	}
	if strings.Count(string(p.Content), Begin) != 1 || strings.Contains(string(p.Content), "v1") || strings.Count(string(p.Content), "outside.env") != 1 {
		t.Fatalf("bad block: %s", p.Content)
	}
	if err := p.Apply(root); err != nil {
		t.Fatal(err)
	}
	for _, name := range paths {
		cmd := exec.Command("git", "check-ignore", "-q", "--", name) //nolint:gosec // synthetic filenames are passed as arguments
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("literal path %q not ignored: %v", name, err)
		}
	}
	p, err = PlanIgnore(root, paths)
	if err != nil || p.Changed {
		t.Fatalf("rerun changed block: %+v %v", p, err)
	}
	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil || string(content) != string(p.Content) {
		t.Fatal("unexpected .gitignore contents")
	}
}

func TestIgnorePlanRejectsIncompleteMarkers(t *testing.T) {
	dir := t.TempDir()
	root, err := fsx.OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.Root.WriteFile(".gitignore", []byte(Begin+"\n.env\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanIgnore(root, nil); err == nil {
		t.Fatal("incomplete markers accepted")
	}
}
