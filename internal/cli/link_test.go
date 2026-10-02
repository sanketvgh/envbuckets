package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/sanketvgh/envbuckets/internal/gitx"
)

func TestMissingLinkWarnsAndApplyRepairsIt(t *testing.T) {
	r := setup(t)
	if err := os.Remove(r.path(".env")); err != nil {
		t.Fatal(err)
	}
	if res := r.hook(); !strings.Contains(res.stderr, "no .env symlink, skipped") {
		t.Fatalf("hook: %s", res.all())
	}
	if res := r.ok("link", "prod"); !strings.Contains(res.stderr, "no .env symlink, skipped") {
		t.Fatalf("link: %s", res.all())
	}
	if res := r.ok("unlink"); !strings.Contains(res.stderr, "no .env symlink, skipped") {
		t.Fatalf("unlink: %s", res.all())
	}
	if r.exists(".env") {
		t.Fatal("hook, link, or unlink created the missing link")
	}
	r.ok("apply")
	if got := r.readlink(".env"); got != ".env.d/staging/.env" {
		t.Fatalf("apply did not repair link: %s", got)
	}
}

func TestLinkOverridesRulesAndSticks(t *testing.T) {
	r := setup(t)
	r.newBranch("feature/x")
	r.hook()
	res := r.ok("link", "prod")
	if !strings.Contains(res.stdout, "Pinned branch feature/x to bucket prod") || !strings.Contains(res.stdout, "dev -> prod - 1 scope switched") {
		t.Fatalf("link:\n%s", res.all())
	}
	if got := r.readlink(".env"); got != ".env.d/prod/.env" {
		t.Fatalf("link should switch now, got %s", got)
	}
	if res := r.hook(); res.all() != "" {
		t.Fatalf("linked bucket already active, hook must be silent:\n%s", res.all())
	}
	st := r.ok("status")
	if !strings.Contains(st.stdout, "Using bucket prod (local pin") || !strings.Contains(st.stdout, ".env -> .env.d/prod/.env") {
		t.Fatalf("status:\n%s", st.stdout)
	}
	ml := r.ok("map", "list")
	if strings.Contains(ml.stdout, "* 3.") || !strings.Contains(ml.stdout, "*    feature/x") {
		t.Fatalf("map list should mark the link, not the rule:\n%s", ml.stdout)
	}

	r.checkoutMain()
	r.hook()
	if got := r.readlink(".env"); got != ".env.d/staging/.env" {
		t.Fatalf("other branches follow rules, got %s", got)
	}
	r.git("checkout", "-q", "feature/x")
	res = r.hook()
	if !strings.Contains(res.stdout, "staging -> prod (link)") || r.readlink(".env") != ".env.d/prod/.env" {
		t.Fatalf("link must survive checkouts:\n%s", res.all())
	}
}

func TestUseOnLinkedBranchIsReportedAsOverride(t *testing.T) {
	r := setup(t)
	r.newBranch("feature/x")
	r.ok("link", "prod")
	r.ok("use", "dev")
	st := r.ok("status")
	if !strings.Contains(st.stdout, ".env -> .env.d/dev/.env; expected .env.d/prod/.env") || !strings.Contains(st.stdout, "Using bucket prod (local pin") {
		t.Fatalf("status:\n%s", st.stdout)
	}
}

func TestUnlinkFallsBackToRule(t *testing.T) {
	r := setup(t)
	r.newBranch("feature/x")
	r.ok("link", "prod")
	res := r.ok("unlink")
	if !strings.Contains(res.stdout, "Removed local pin for branch feature/x") || r.readlink(".env") != ".env.d/dev/.env" {
		t.Fatalf("unlink:\n%s", res.all())
	}
	if res := r.ok("unlink"); !strings.Contains(res.stdout, "nothing to do") {
		t.Fatalf("second unlink:\n%s", res.all())
	}
}

func TestUnlinkWithoutMatchingRuleLeavesEnv(t *testing.T) {
	r := setup(t)
	r.ok("map", "rm", "*")
	r.newBranch("feature/x")
	r.ok("link", "prod")
	res := r.ok("unlink")
	if !strings.Contains(res.stdout, "no rule matches") || r.readlink(".env") != ".env.d/prod/.env" {
		t.Fatalf("unlink:\n%s", res.all())
	}
}

func TestLinkOtherBranchDoesNotSwitch(t *testing.T) {
	r := setup(t)
	r.git("branch", "feature/y")
	r.hook()
	res := r.ok("link", "prod", "--branch", "feature/y")
	if !strings.Contains(res.stdout, "Pinned branch feature/y to bucket prod") || r.readlink(".env") != ".env.d/staging/.env" {
		t.Fatalf("link --branch:\n%s", res.all())
	}
	r.git("checkout", "-q", "feature/y")
	r.hook()
	if got := r.readlink(".env"); got != ".env.d/prod/.env" {
		t.Fatalf("got %s", got)
	}
	r.checkoutMain()
	r.ok("unlink", "--branch", "feature/y")
	if links, _ := gitx.Links(r.root); len(links) != 0 {
		t.Fatalf("links left: %v", links)
	}
}

func TestLinkFollowsBranchRenameAndDelete(t *testing.T) {
	r := setup(t)
	r.newBranch("feature/x")
	r.ok("link", "prod")
	r.git("branch", "-m", "feature/x", "feature/renamed")
	if b, _ := gitx.LinkedBucket(r.root, "feature/renamed"); b != "prod" {
		t.Fatalf("rename should carry the link, got %q", b)
	}
	r.checkoutMain()
	r.git("branch", "-D", "feature/renamed")
	if links, _ := gitx.Links(r.root); len(links) != 0 {
		t.Fatalf("delete should drop the link: %v", links)
	}
}

func TestLinkGuards(t *testing.T) {
	r := setup(t)
	r.newBranch("feature/x")
	if res := r.run("link", "nope"); res.code != ExitBlocked || !strings.Contains(res.stderr, "bucket nope does not exist in any scope") {
		t.Fatalf("unknown bucket: %d %s", res.code, res.all())
	}
	if res := r.run("link", "../x"); res.code != ExitUsage {
		t.Fatalf("invalid name: %d %s", res.code, res.all())
	}
	if res := r.run("link", "prod", "--branch", "missing"); res.code != ExitEnv {
		t.Fatalf("missing branch: %d %s", res.code, res.all())
	}
	r.ok("bucket", "add", "qa")
	r.ok("link", "qa")
	r.ok("use", "dev")
	if res := r.run("bucket", "rm", "qa"); res.code != ExitBlocked || !strings.Contains(res.stderr, "linked by branches: feature/x") {
		t.Fatalf("bucket rm linked: %d %s", res.code, res.all())
	}
	r.git("checkout", "-q", "--detach")
	if res := r.run("link", "prod"); res.code != ExitEnv || !strings.Contains(res.stderr, "detached HEAD") {
		t.Fatalf("detached: %d %s", res.code, res.all())
	}
}

func TestHookIgnoresInvalidLinkValue(t *testing.T) {
	r := setup(t)
	r.newBranch("feature/x")
	r.hook()
	r.git("config", "--local", "branch.feature/x.envbuckets", "../escape")
	res := r.hook()
	if !strings.Contains(res.stderr, "cannot read the branch link") || r.readlink(".env") != ".env.d/dev/.env" {
		t.Fatalf("hook:\n%s", res.all())
	}
	if res := r.run("status"); res.code != ExitEnv {
		t.Fatalf("status: %d %s", res.code, res.all())
	}
}

func TestUninstallKeepsLinksUnlessPurged(t *testing.T) {
	r := setup(t)
	r.newBranch("feature/x")
	r.ok("link", "prod")
	res := r.ok("uninstall")
	if !strings.Contains(res.stdout, "kept: branch links (1 in .git/config)") {
		t.Fatalf("uninstall:\n%s", res.stdout)
	}
	res = r.runIn(r.root, "DELETE\n", "uninstall", "--purge")
	if res.code != ExitOK || !strings.Contains(res.stdout, "removed: branch links") {
		t.Fatalf("purge: %d\n%s", res.code, res.all())
	}
	if links, _ := gitx.Links(r.root); len(links) != 0 {
		t.Fatalf("links left after purge: %v", links)
	}
}
