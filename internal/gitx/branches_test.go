package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestBranchesReadsShortLocalRefsOnly(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...) //nolint:gosec // Fixed Git executable with synthetic test arguments, no shell.
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, out)
		}
	}
	git("init", "-q", "-b", "main")
	names, err := Branches(dir)
	if err != nil || len(names) != 0 {
		t.Fatalf("unborn: %q %v", names, err)
	}
	// A commit object can be created without files, signing, or host identity.
	cmd := exec.Command("git", "hash-object", "-t", "commit", "-w", "--stdin")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor Test <test@example.com> 0 +0000\ncommitter Test <test@example.com> 0 +0000\n\nfixture\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.TrimSpace(string(out))
	for _, ref := range []string{"refs/heads/main", "refs/heads/release/1.0", "refs/heads/café", "refs/heads/漢字\u00a0", "refs/tags/main", "refs/remotes/origin/remote"} {
		git("update-ref", ref, sha)
	}
	tracePath := filepath.Join(dir, "git.trace")
	t.Setenv("GIT_TRACE", tracePath)
	names, err = Branches(dir)
	want := []string{"café", "main", "release/1.0", "漢字\u00a0"}
	if err != nil || !slices.Equal(names, want) {
		t.Fatalf("branches=%q err=%v want=%q", names, err, want)
	}
	trace, err := os.ReadFile(tracePath)
	if err != nil || strings.Count(string(trace), "built-in: git for-each-ref") != 1 {
		t.Fatalf("expected one branch enumeration: %q %v", trace, err)
	}
	branch, err := Branch(dir)
	if err != nil || branch != "main" {
		t.Fatalf("current branch with ambiguous tag: %q %v", branch, err)
	}
	git("symbolic-ref", "HEAD", "refs/heads/漢字\u00a0")
	branch, err = Branch(dir)
	if err != nil || branch != "漢字\u00a0" {
		t.Fatalf("current branch lost Unicode whitespace: %q %v", branch, err)
	}
}
