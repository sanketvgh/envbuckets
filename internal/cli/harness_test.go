package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type repo struct {
	t    testing.TB
	root string
}

func newRepo(t testing.TB) *repo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	r := &repo{t: t, root: root}
	r.git("init", "-q", "-b", "main")
	r.git("config", "user.email", "t@example.com")
	r.git("config", "user.name", "t")
	r.git("config", "commit.gpgsign", "false")
	// Git's detached maintenance can remove its lock after a snapshot is taken.
	// Finish fixture maintenance before returning from the initial commit.
	r.git("config", "maintenance.autoDetach", "false")
	r.write("README", "x\n")
	r.git("add", "README")
	r.git("commit", "-q", "-m", "init")
	return r
}

func (r *repo) git(args ...string) {
	r.t.Helper()
	cmd := exec.Command("git", args...) //nolint:gosec // test helper, fixed args
	cmd.Dir = r.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func (r *repo) path(rel string) string {
	return filepath.Join(r.root, filepath.FromSlash(rel))
}

func (r *repo) write(rel, content string) {
	r.t.Helper()
	if err := os.MkdirAll(filepath.Dir(r.path(rel)), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(r.path(rel), []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

type result struct {
	code   int
	stdout string
	stderr string
}

func (res result) all() string { return res.stdout + res.stderr }

func (r *repo) runIn(cwd, stdin string, args ...string) result {
	r.t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, Env{Cwd: cwd, Stdin: strings.NewReader(stdin), Stdout: &out, Stderr: &errb, Version: "test"})
	return result{code: code, stdout: out.String(), stderr: errb.String()}
}

func (r *repo) run(args ...string) result {
	r.t.Helper()
	return r.runIn(r.root, "", args...)
}
