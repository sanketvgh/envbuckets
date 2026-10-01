package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScopeRmSuggestsWorkingPurge(t *testing.T) {
	r := monorepo(t)
	r.write("apps/web/.env.d/dev/.env", "W=1\n")
	res := r.ok("scope", "rm", "web")
	if !strings.Contains(res.stdout, "next: to erase them: envbuckets scope purge apps/web") {
		t.Fatalf("rm hint:\n%s", res.stdout)
	}
	if res := r.run("scope", "rm", "web", "--purge"); res.code != ExitBlocked {
		t.Fatalf("old suggestion should not resolve: %d", res.code)
	}

	if res := r.runIn(r.root, "no\n", "scope", "purge", "apps/web"); res.code != ExitBlocked {
		t.Fatalf("refused confirmation: %d", res.code)
	}
	if !r.exists("apps/web/.env.d/dev/.env") || r.readlink("apps/web/.env") != ".env.d/dev/.env" || !strings.Contains(r.read(".gitignore"), "apps/web/.env.d/") {
		t.Fatal("refused purge changed something")
	}

	res = r.runIn(r.root, "DELETE\n", "scope", "purge", "apps/web")
	if res.code != ExitOK {
		t.Fatalf("purge: %d\n%s", res.code, res.all())
	}
	for _, want := range []string{"apps/web/.env.d/ (buckets: dev, staging)", "apps/web/.env (symlink -> .env.d/dev/.env, would dangle)", "removed: apps/web/.env.d/, apps/web/.env, its .gitignore lines"} {
		if !strings.Contains(res.stdout, want) {
			t.Fatalf("missing %q:\n%s", want, res.stdout)
		}
	}
	if r.exists("apps/web/.env.d") || r.exists("apps/web/.env") || !r.exists("apps/web") {
		t.Fatal("purge removed the wrong things")
	}
	ignore := r.read(".gitignore")
	if strings.Contains(ignore, "apps/web/") || !strings.Contains(ignore, "apps/api/.env.d/") {
		t.Fatalf("gitignore:\n%s", ignore)
	}
	if res := r.ok("scope", "purge", "apps/web"); !strings.Contains(res.stdout, "nothing to delete") {
		t.Fatalf("rerun:\n%s", res.stdout)
	}
}

func TestScopePurgeKeepsUnmanagedEnv(t *testing.T) {
	r := monorepo(t)
	r.ok("scope", "rm", "web")
	if err := os.Remove(r.path("apps/web/.env")); err != nil {
		t.Fatal(err)
	}
	r.write("apps/web/.env", "REAL=1\n")
	res := r.runIn(r.root, "DELETE\n", "scope", "purge", "apps/web")
	if res.code != ExitOK || !strings.Contains(res.stdout, "kept: apps/web/.env is not managed") {
		t.Fatalf("purge: %d\n%s", res.code, res.all())
	}
	if r.read("apps/web/.env") != "REAL=1\n" || r.exists("apps/web/.env.d") {
		t.Fatal("real .env not preserved or data kept")
	}
	ignore := r.read(".gitignore")
	if !strings.Contains(ignore, "apps/web/.env\n") || strings.Contains(ignore, "apps/web/.env.d/") {
		t.Fatalf("real .env must stay ignored:\n%s", ignore)
	}

	r.ok("scope", "rm", "api")
	if err := os.Remove(r.path("apps/api/.env")); err != nil {
		t.Fatal(err)
	}
	r.write("shared.env", "S=1\n")
	if err := os.Symlink("../../shared.env", r.path("apps/api/.env")); err != nil {
		t.Fatal(err)
	}
	res = r.runIn(r.root, "DELETE\n", "scope", "purge", "apps/api")
	if res.code != ExitOK || r.readlink("apps/api/.env") != "../../shared.env" || r.exists("apps/api/.env.d") {
		t.Fatalf("foreign symlink: %d\n%s", res.code, res.all())
	}
}

func TestScopePurgeRejectsUnsafePaths(t *testing.T) {
	r := monorepo(t)
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outside, bucketsDir, "dev"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, r.path("apps/escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(r.path("apps/linked"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside+"/"+bucketsDir, r.path("apps/linked/"+bucketsDir)); err != nil {
		t.Fatal(err)
	}
	cases := []string{"../x", "apps/../../x", "/tmp", r.path("apps/web"), "apps/escape", "apps/linked", "apps/api/.env.d", ".git", "apps/api", "."}
	for _, p := range cases {
		if res := r.runIn(r.root, "DELETE\n", "scope", "purge", p); res.code == ExitOK {
			t.Fatalf("purge %q succeeded:\n%s", p, res.all())
		}
	}
	if _, err := os.Stat(filepath.Join(outside, bucketsDir, "dev")); err != nil {
		t.Fatal("deleted outside the repo")
	}
	for _, p := range []string{"apps/api/.env.d/dev/.env", ".env.d/dev/.env"} {
		if !r.exists(p) {
			t.Fatalf("%s deleted", p)
		}
	}
	if res := r.run("scope", "purge"); res.code != ExitUsage {
		t.Fatalf("missing arg: %d", res.code)
	}
	if res := r.run("scope", "purge", "a", "--purge"); res.code != ExitUsage {
		t.Fatalf("irrelevant flag: %d", res.code)
	}
}

func TestScopeRmPurgeRemovesManagedLink(t *testing.T) {
	r := monorepo(t)
	res := r.runIn(r.root, "DELETE\n", "scope", "rm", "web", "--purge")
	if res.code != ExitOK || r.exists("apps/web/.env") || r.exists("apps/web/.env.d") {
		t.Fatalf("rm --purge: %d\n%s", res.code, res.all())
	}
	if strings.Contains(r.read(".envbuckets.toml"), "apps/web") {
		t.Fatal("scope still registered")
	}
}

func TestScopePurgeImplicitRootRefused(t *testing.T) {
	r := setup(t)
	if res := r.runIn(r.root, "DELETE\n", "scope", "purge", "."); res.code != ExitBlocked || !r.exists(".env.d/dev/.env") {
		t.Fatalf("implicit root: %d\n%s", res.code, res.all())
	}
}
