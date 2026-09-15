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
	t    *testing.T
	root string
}

func newRepo(t *testing.T) *repo {
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
	r.write("README", "x\n")
	r.git("add", "README")
	r.git("commit", "-q", "-m", "init")
	return r
}

func (r *repo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...) //nolint:gosec // test helper, fixed args
	cmd.Dir = r.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
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

func (r *repo) read(rel string) string {
	r.t.Helper()
	data, err := os.ReadFile(r.path(rel))
	if err != nil {
		r.t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func (r *repo) exists(rel string) bool {
	_, err := os.Lstat(r.path(rel))
	return err == nil
}

func (r *repo) readlink(rel string) string {
	r.t.Helper()
	target, err := os.Readlink(r.path(rel))
	if err != nil {
		r.t.Fatalf("readlink %s: %v", rel, err)
	}
	return filepath.ToSlash(target)
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

func (r *repo) ok(args ...string) result {
	r.t.Helper()
	res := r.run(args...)
	if res.code != ExitOK {
		r.t.Fatalf("envbuckets %v: exit %d\n%s", args, res.code, res.all())
	}
	return res
}

func (r *repo) checkoutMain() {
	r.t.Helper()
	r.git("checkout", "-q", "main")
}

func (r *repo) newBranch(branch string) {
	r.t.Helper()
	r.git("checkout", "-q", "-b", branch)
}

func (r *repo) hook() result {
	r.t.Helper()
	res := r.run("hook", "0000", "0000", "1")
	if res.code != ExitOK {
		r.t.Fatalf("hook must exit 0, got %d\n%s", res.code, res.all())
	}
	return res
}

func requireSymlinks(t *testing.T, dir string) {
	t.Helper()
	probe := filepath.Join(dir, ".symlink-probe")
	if err := os.Symlink("probe-target", probe); err != nil {
		t.Skipf("symlinks unavailable here: %v", err)
	}
	_ = os.Remove(probe)
}

// setup initializes a single-scope repo with dev/staging/prod buckets and
// rules main->staging, release/*->prod, *->dev, with .env pointing at dev.
func setup(t *testing.T) *repo {
	t.Helper()
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.ok("init")
	for _, b := range []string{"dev", "staging", "prod"} {
		r.ok("bucket", "add", b)
	}
	r.ok("map", "add", "main", "staging")
	r.ok("map", "add", "release/*", "prod")
	r.ok("map", "add", "*", "dev")
	r.ok("use", "dev")
	return r
}
