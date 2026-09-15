package cli

import (
	"os"
	"strings"
	"testing"
)

// monorepo builds root + apps/api + apps/web scopes; web lacks the prod
// bucket on purpose.
func monorepo(t *testing.T) *repo {
	t.Helper()
	r := newRepo(t)
	requireSymlinks(t, r.root)
	for _, d := range []string{"apps/api", "apps/web"} {
		if err := os.MkdirAll(r.path(d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	r.ok("init")
	r.ok("scope", "add", ".", "--name", "root")
	r.ok("scope", "add", "apps/api")
	r.ok("scope", "add", "apps/web")
	for _, s := range []string{"root", "api", "web"} {
		for _, b := range []string{"dev", "staging"} {
			r.ok("bucket", "add", b, "--scope", s)
		}
		r.ok("use", "dev", "--scope", s)
	}
	r.ok("bucket", "add", "prod", "--scope", "root")
	r.ok("bucket", "add", "prod", "--scope", "api")
	r.ok("map", "add", "main", "staging")
	r.ok("map", "add", "release/*", "prod")
	r.ok("map", "add", "*", "dev")
	return r
}

func TestImplicitRootScope(t *testing.T) {
	r := setup(t)
	if strings.Contains(r.read(".envbuckets.toml"), "[[scopes]]") {
		t.Fatal("single repo must not write scopes")
	}
	res := r.ok("scope", "list")
	if !strings.Contains(res.stdout, "implicit") || !strings.Contains(res.stdout, "root") {
		t.Fatalf("scope list:\n%s", res.stdout)
	}
}

func TestMonorepoSwitchAndStatus(t *testing.T) {
	r := monorepo(t)
	res := r.hook()
	if !strings.Contains(res.stdout, "dev -> staging (main) - 3 scopes switched") {
		t.Fatalf("summary:\n%s", res.all())
	}
	for _, p := range []string{".env", "apps/api/.env", "apps/web/.env"} {
		if r.readlink(p) != ".env.d/staging/.env" {
			t.Fatalf("%s not switched", p)
		}
	}
	st := r.ok("status")
	if strings.Count(st.stdout, "staging (ok)") != 3 || strings.Count(st.stdout, "rule:") != 1 {
		t.Fatalf("status:\n%s", st.stdout)
	}
	ignore := r.read(".gitignore")
	for _, line := range []string{"\n.env\n", "\n.env.d/\n", "\napps/api/.env\n", "\napps/api/.env.d/\n", "\napps/web/.env.d/\n"} {
		if !strings.Contains(ignore, line) {
			t.Fatalf("gitignore missing %q:\n%s", line, ignore)
		}
	}
	if strings.Contains(ignore, "**") {
		t.Fatal("gitignore must not use a blanket glob")
	}
	for _, line := range strings.Split(r.git("status", "--porcelain"), "\n") {
		if strings.HasSuffix(line, "/.env") || strings.HasSuffix(line, " .env") || strings.Contains(line, ".env.d") {
			t.Fatalf("env file visible to git: %s", line)
		}
	}
}

func TestPerScopeFailSafe(t *testing.T) {
	r := monorepo(t)
	r.newBranch("release/1")
	res := r.hook()
	if !strings.Contains(res.stdout, "2 scopes switched") || !strings.Contains(res.stderr, "warning: web: missing apps/web/.env.d/prod/.env") {
		t.Fatalf("output:\n%s", res.all())
	}
	if r.readlink("apps/web/.env") != ".env.d/dev/.env" || r.readlink("apps/api/.env") != ".env.d/prod/.env" {
		t.Fatal("per-scope outcome wrong")
	}

	if err := os.RemoveAll(r.path("apps/web")); err != nil {
		t.Fatal(err)
	}
	r.checkoutMain()
	res = r.hook()
	if !strings.Contains(res.stderr, "web: scope directory missing") || r.readlink("apps/api/.env") != ".env.d/staging/.env" {
		t.Fatalf("sparse checkout:\n%s", res.all())
	}
}

func TestScopeContext(t *testing.T) {
	r := monorepo(t)
	res := r.runIn(r.path("apps/api"), "", "bucket", "list")
	if res.code != ExitOK || !strings.Contains(res.stdout, "scope api") {
		t.Fatalf("nearest scope: %d %s", res.code, res.all())
	}
	res = r.ok("bucket", "list", "--scope", "web")
	if !strings.Contains(res.stdout, "scope web") {
		t.Fatalf("--scope: %s", res.stdout)
	}
	r.ok("scope", "rm", "root")
	res = r.run("bucket", "list")
	if res.code != ExitEnv || !strings.Contains(res.stderr, "--scope") {
		t.Fatalf("cwd outside scopes: %d %s", res.code, res.all())
	}
}

func TestScopeAddValidation(t *testing.T) {
	r := newRepo(t)
	r.ok("init")
	if err := os.MkdirAll(r.path("apps/api/sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.ok("scope", "add", "apps/api")
	for _, p := range []string{"apps/api/sub", "apps", r.path("apps/api"), "../", "missing", "apps/api"} {
		if res := r.run("scope", "add", p); res.code != ExitBlocked {
			t.Errorf("scope add %q: want exit 1, got %d\n%s", p, res.code, res.all())
		}
	}
	if res := r.run("scope", "add", "apps/api/sub", "--name", "api"); res.code != ExitBlocked {
		t.Error("duplicate name should be blocked")
	}
}

func TestScopeRmKeepsValuesUnlessPurged(t *testing.T) {
	r := monorepo(t)
	r.write("apps/web/.env.d/dev/.env", "W=1\n")
	res := r.ok("scope", "rm", "web")
	if !strings.Contains(res.stdout, "kept") || !r.exists("apps/web/.env.d/dev/.env") {
		t.Fatalf("rm:\n%s", res.stdout)
	}
	if !strings.Contains(r.read(".gitignore"), "apps/web/.env.d/") {
		t.Fatal("gitignore lines removed while values still on disk")
	}
	r.ok("scope", "add", "apps/web")
	if res := r.runIn(r.root, "", "scope", "rm", "web", "--purge"); res.code != ExitBlocked {
		t.Fatalf("purge without DELETE: %d", res.code)
	}
	res = r.runIn(r.root, "DELETE\n", "scope", "rm", "web", "--purge")
	if res.code != ExitOK || r.exists("apps/web/.env.d") || strings.Contains(r.read(".gitignore"), "apps/web/") {
		t.Fatalf("purge: %d %s\n%s", res.code, res.all(), r.read(".gitignore"))
	}
	if !strings.Contains(r.read(".gitignore"), "apps/api/.env.d/") {
		t.Fatal("other scope lines must survive")
	}
}

func TestScopeAddBootstrapsRealEnv(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.ok("init")
	r.write("svc/.env", "S=1\n")
	res := r.ok("scope", "add", "svc", "--into", "dev")
	if !strings.Contains(res.stdout, "[created]") || r.readlink("svc/.env") != ".env.d/dev/.env" || r.read("svc/.env.d/dev/.env") != "S=1\n" {
		t.Fatalf("bootstrap:\n%s", res.stdout)
	}
}
