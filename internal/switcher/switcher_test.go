package switcher

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
)

func TestBucketOf(t *testing.T) {
	for _, tc := range []struct{ name, target, want string }{
		{".env", ".env.d/dev/.env", "dev"},
		{".env", "./.env.d/dev/.env", ""},
		{".env", ".env.d/dev/../dev/.env", ""},
		{"apps/api/key", "../../.env.d/dev/apps/./api/key", ""},
		{"apps/api/key", "../../.env.d/prod/apps/api/key", "prod"},
		{".env", ".env.d/dev/other", ""},
		{".env", "../.env.d/dev/.env", ""},
		{".env", ".env.d/../dev/.env", ""},
		{".env", ".env.d/bad.name/.env", ""},
		{".env", "README", ""},
		{".env", "/.env.d/dev/.env", ""},
	} {
		if got := BucketOf(tc.name, tc.target); got != tc.want {
			t.Errorf("BucketOf(%q, %q)=%q, want %q", tc.name, tc.target, got, tc.want)
		}
	}
}

func plannerRepo(t *testing.T) *fsx.Repo {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	r, err := fsx.OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Root.MkdirAll(".env.d/dev", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := r.Root.WriteFile(".env.d/dev/.env", []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestPlanIsReadOnlyAndPreflightBlocksWrites(t *testing.T) {
	r := plannerRepo(t)
	p := Build(r, config.Config{Default: "dev"}, "main", "")
	if len(p.Actions) != 1 || p.Actions[0].Kind != Link || p.ExitCode() != 0 {
		t.Fatalf("plan: %+v", p)
	}
	var out, errout bytes.Buffer
	if code := Execute(r, p, true, false, &out, &errout); code != 0 {
		t.Fatalf("dry code %d", code)
	}
	if _, err := r.Root.Lstat(".env"); !os.IsNotExist(err) {
		t.Fatal("planning/dry run wrote a link")
	}
	p.Actions = append(p.Actions, Action{Kind: PreflightError, Reason: "synthetic preflight failure"})
	out.Reset()
	if code := Execute(r, p, false, false, &out, &errout); code != 1 || out.Len() != 0 {
		t.Fatalf("preflight code=%d output=%q", code, out.String())
	}
	if _, err := r.Root.Lstat(".env"); !os.IsNotExist(err) {
		t.Fatal("preflight failure allowed a write")
	}
}

func TestApplyRefusesDestinationChangedAfterPlanning(t *testing.T) {
	r := plannerRepo(t)
	p := Build(r, config.Config{Default: "dev"}, "main", "")
	if err := r.Root.WriteFile(".env", []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	if code := Execute(r, p, false, false, &out, &errout); code != 1 {
		t.Fatal("changed destination accepted")
	}
	content, err := r.Root.ReadFile(".env")
	if err != nil || string(content) != "keep me" {
		t.Fatal("real file was changed")
	}
}

func TestUnsafeAndTrackedPathsAreSkipped(t *testing.T) {
	r := plannerRepo(t)
	if err := r.Root.WriteFile("tracked", []byte("tracked"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "tracked")
	cmd.Dir = r.Path
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tracked", ".git/config"} {
		path := filepath.ToSlash(filepath.Join(".env.d/dev", name))
		if err := r.Root.MkdirAll(filepath.ToSlash(filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := r.Root.WriteFile(path, []byte("synthetic"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	p := Build(r, config.Config{Default: "dev"}, "main", "")
	skips := 0
	for _, a := range p.Actions {
		if a.Kind == Skip {
			skips++
		}
	}
	if p.ExitCode() != 1 || skips != 2 {
		t.Fatalf("unsafe plan: %+v", p)
	}
}

func TestSymlinkedParentsAndBucketEntries(t *testing.T) {
	r := plannerRepo(t)
	if err := r.Root.MkdirAll(".env.d/dev/apps", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := r.Root.WriteFile(".env.d/dev/apps/key", []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := r.Root.Mkdir("real", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := r.Root.Symlink("real", "apps"); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := r.Root.Symlink(".env", ".env.d/dev/unsafe"); err != nil {
		t.Fatal(err)
	}
	p := Build(r, config.Config{Default: "dev"}, "main", "")
	skips := 0
	for _, a := range p.Actions {
		if a.Kind == Skip {
			skips++
		}
	}
	if skips != 2 || p.ExitCode() != 1 {
		t.Fatalf("unsafe paths: %+v", p)
	}
	var out, errout bytes.Buffer
	Execute(r, p, false, false, &out, &errout)
	if _, err := r.Root.Lstat("real/key"); !os.IsNotExist(err) {
		t.Fatal("wrote through a symlink parent")
	}
}

func TestRerunConvergesAfterPartialApply(t *testing.T) {
	r := plannerRepo(t)
	if err := r.Root.Symlink(".env.d/dev/.env", "probe"); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := r.Root.Remove("probe"); err != nil {
		t.Fatal(err)
	}
	if err := r.Root.WriteFile(".env.d/dev/other", []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	p := Build(r, config.Config{Default: "dev"}, "main", "")
	if err := apply(r, p.Actions[0]); err != nil {
		t.Fatal(err)
	}
	p = Build(r, config.Config{Default: "dev"}, "main", "")
	if len(p.Actions) != 1 || p.Actions[0].Path != "other" {
		t.Fatalf("remaining actions: %+v", p)
	}
	var out, errout bytes.Buffer
	if code := Execute(r, p, false, false, &out, &errout); code != 0 {
		t.Fatalf("resume: %s", errout.String())
	}
	if p := Build(r, config.Config{Default: "dev"}, "main", ""); len(p.Actions) != 0 {
		t.Fatalf("not converged: %+v", p)
	}
	entries, err := os.ReadDir(r.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".envbuckets-link-") {
			t.Fatal("staged link left behind")
		}
	}
}
