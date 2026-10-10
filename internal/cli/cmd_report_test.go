package cli

import (
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/sanketvgh/envbuckets/internal/block"
	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/switcher"
)

func expectReport(t *testing.T, r *repo, want string, args ...string) {
	t.Helper()
	before := snapshotRepo(t, r)
	res := r.run(args...)
	if res.code != ExitOK || res.stdout != want || res.stderr != "" {
		t.Fatalf("%v: code=%d\nstdout=%q\nstderr=%q\nwant=%q", args, res.code, res.stdout, res.stderr, want)
	}
	if !reflect.DeepEqual(before, snapshotRepo(t, r)) {
		t.Fatalf("%v changed the repository", args)
	}
}

func reportRepo(t *testing.T) *repo {
	t.Helper()
	r := newRepo(t)
	r.write(".envbuckets.json", switchConfig)
	for _, b := range []string{"dev", "prod"} {
		if err := os.MkdirAll(r.path(".env.d/"+b), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	r.write(".gitignore", ".env.d/\n.env\napps/\n")
	if _, err := block.InstallRepoHook(r.root); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestBranchesVariants(t *testing.T) {
	r := reportRepo(t)
	for _, name := range []string{"feature/login", "release/2.0", "release/1.0", "staging", "漢字", "café"} {
		r.git("branch", name)
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"plain", nil, "* main         dev      (default)\n  release/1.0  prod     (rule 'release/**')\n  release/2.0  prod     (rule 'release/**')\n  staging      staging  (rule 'staging', missing; falls back to 'dev')\n"},
		{"names in input order", []string{"future", "main", "release/9.0"}, "  future       dev   (default)\n* main         dev   (default)\n  release/9.0  prod  (rule 'release/**')\n"},
		{"pattern", []string{"release/**"}, "  release/1.0  prod  (rule 'release/**')\n  release/2.0  prod  (rule 'release/**')\n"},
		{"default bucket includes defaults", []string{"--bucket", "dev"}, "  café           dev  (default)\n  feature/login  dev  (default)\n* main           dev  (default)\n  漢字           dev  (default)\n"},
		{"combined", []string{"--bucket=prod", "main", "release/**", "release/9.0"}, "  release/1.0  prod  (rule 'release/**')\n  release/2.0  prod  (rule 'release/**')\n  release/9.0  prod  (rule 'release/**')\n"},
		{"no matches", []string{"no-match*"}, ""},
		{"missing mapping filter", []string{"--bucket", "staging"}, "  staging  staging  (rule 'staging', missing; falls back to 'dev')\n"},
		{"all and Unicode", []string{"**"}, "  café           dev      (default)\n  feature/login  dev      (default)\n* main           dev      (default)\n  release/1.0    prod     (rule 'release/**')\n  release/2.0    prod     (rule 'release/**')\n  staging        staging  (rule 'staging', missing; falls back to 'dev')\n  漢字           dev      (default)\n"},
		{"deduplicate overlapping selectors", []string{"main", "ma*", "main"}, "* main  dev  (default)\n"},
	} {
		t.Run(tc.name, func(t *testing.T) { expectReport(t, r, tc.want, append([]string{"branches"}, tc.args...)...) })
	}
}

func TestBranchesDetachedAndMissingDefault(t *testing.T) {
	r := reportRepo(t)
	r.git("branch", "staging")
	r.git("checkout", "-q", "--detach")
	expectReport(t, r, "  staging  staging  (rule 'staging', missing; falls back to 'dev')\n", "branches")
	expectReport(t, r, "  main  dev  (default)\n", "branches", "main")
	if err := os.Remove(r.path(".env.d/dev")); err != nil {
		t.Fatal(err)
	}
	expectReport(t, r, "  staging  staging  (rule 'staging', missing; default 'dev' also missing; links stay as they are)\n", "branches", "staging")
	expectReport(t, r, "  main  dev  (default, missing; links stay as they are)\n", "branches", "main")
}

func TestReportFailuresAndAlphaWarning(t *testing.T) {
	r := newRepo(t)
	for _, cmd := range []string{"status", "branches"} {
		res := r.run(cmd)
		if res.code != ExitError || res.stdout != "" || res.stderr != "fatal: .envbuckets.json is missing\nhint: Run \"envbuckets init\" to set up this repository.\n" {
			t.Fatalf("missing %s: %+v", cmd, res)
		}
	}
	r.write(".envbuckets.json", `{"default":"dev","bukcet":"prod"}`)
	for _, cmd := range []string{"status", "branches"} {
		res := r.run(cmd)
		if res.code != ExitError || res.stdout != "" || res.stderr != "fatal: invalid .envbuckets.json: unknown key 'bukcet'\n" {
			t.Fatalf("invalid %s: %+v", cmd, res)
		}
	}
	r = reportRepo(t)
	r.write(".envbuckets.toml", "SYNTHETIC_UNUSED")
	for _, cmd := range []string{"status", "branches"} {
		before := snapshotRepo(t, r)
		res := r.run(cmd)
		if res.code != ExitOK || res.stderr != "warning: .envbuckets.toml is unused and can be deleted\n" || strings.Contains(res.all(), "SYNTHETIC") || !reflect.DeepEqual(before, snapshotRepo(t, r)) {
			t.Fatalf("alpha %s: %+v", cmd, res)
		}
	}
	for _, args := range [][]string{{"status", "-n"}, {"status", "extra"}, {"branches", "-n"}, {"branches", "--all"}, {"branches", "--bucket"}, {"branches", "--bucket", "../bad"}, {"branches", "bad["}, {"branches", ""}} {
		res := r.run(args...)
		if res.code != ExitUsage || res.stdout != "" || !strings.Contains(res.stderr, "usage:") {
			t.Fatalf("usage %v: %+v", args, res)
		}
	}
}

func TestStatusEmptyAndHookProblems(t *testing.T) {
	r := reportRepo(t)
	expectReport(t, r, "On branch main\nUsing bucket 'dev' (default)\n\nall 0 files linked\n", "status")
	for _, hook := range []string{"#!/bin/sh\n# replaced by another tool\n", block.Begin + "\ntruncated\n"} {
		r.write(".git/hooks/post-checkout", hook)
		expectReport(t, r, "On branch main\nUsing bucket 'dev' (default)\n\nHook not installed:\n  (use \"envbuckets init\" to install it)\n\nall 0 files linked\n", "status")
	}
	if err := os.Remove(r.path(".git/hooks/post-checkout")); err != nil {
		t.Fatal(err)
	}
	expectReport(t, r, "On branch main\nUsing bucket 'dev' (default)\n\nHook not installed:\n  (use \"envbuckets init\" to install it)\n\nall 0 files linked\n", "status")
	if _, err := block.InstallRepoHook(r.root); err != nil {
		t.Fatal(err)
	}
	r.write(".env.d/dev/.env", "SYNTHETIC_BUCKET")
	r.write(".env", "SYNTHETIC_REAL")
	expectReport(t, r, "On branch main\nUsing bucket 'dev' (default)\n\nFiles not linked:\n  (to keep a file, move it to the same path under .env.d/dev/, then use \"envbuckets switch\")\n\treal file:   .env\n\n0 of 1 files linked\n", "status")
	if err := os.Remove(r.path(".env")); err != nil {
		t.Fatal(err)
	}
	expectReport(t, r, "On branch main\nUsing bucket 'dev' (default)\n\nFiles not linked:\n  (to keep a file, move it to the same path under .env.d/dev/, then use \"envbuckets switch\")\n\tmissing link:   .env\n\n0 of 1 files linked\n", "status")
}

func TestStatusUnsafeWorkingParent(t *testing.T) {
	r := reportRepo(t)
	r.write(".env.d/dev/apps/key", "SYNTHETIC_BUCKET")
	r.write("apps", "SYNTHETIC_BLOCKING_PARENT")
	expectReport(t, r, "On branch main\nUsing bucket 'dev' (default)\n\nFiles not linked:\n  (to keep a file, move it to the same path under .env.d/dev/, then use \"envbuckets switch\")\n\tunsafe path:   apps/key\n\n0 of 1 files linked\n", "status")
}

func TestStatusLinksAndTemporaryBucket(t *testing.T) {
	r := reportRepo(t)
	requireRepoSymlinks(t, r)
	r.write(".env.d/dev/.env", "SYNTHETIC_DEV")
	r.write(".env.d/prod/.env", "SYNTHETIC_PROD")
	if res := r.run("switch"); res.code != ExitOK {
		t.Fatal(res)
	}
	expectReport(t, r, "On branch main\nUsing bucket 'dev' (default)\n\n1 file linked\n", "status")
	r.write(".env.d/dev/apps/key", "SYNTHETIC_NESTED")
	if res := r.run("switch"); res.code != ExitOK {
		t.Fatal(res)
	}
	expectReport(t, r, "On branch main\nUsing bucket 'dev' (default)\n\nall 2 files linked\n", "status")
	if res := r.run("switch", "prod"); res.code != ExitOK {
		t.Fatal(res)
	}
	expectReport(t, r, "On branch main\nUsing bucket 'prod', but this branch uses 'dev' (default)\n  (use \"envbuckets switch\" to go back to 'dev')\n\n1 file linked\n", "status")
	expectReport(t, r, "* main  dev  (default), using 'prod' for now\n", "branches")
	r.git("switch", "-q", "-c", "release/1.0")
	expectReport(t, r, "On branch release/1.0\nUsing bucket 'prod' (rule 'release/**')\n\n1 file linked\n", "status")
}

func TestStatusFallbackMatchesHookAndBranches(t *testing.T) {
	r := reportRepo(t)
	requireRepoSymlinks(t, r)
	r.write(".env.d/dev/.env", "SYNTHETIC_DEV")
	r.write(".env.d/prod/.env", "SYNTHETIC_PROD")
	if res := r.run("switch", "prod"); res.code != ExitOK {
		t.Fatal(res)
	}
	r.git("switch", "-q", "-c", "staging")
	res := r.run("hook", "old", "new", "1")
	if res.code != ExitOK || res.stderr != "envbuckets: warning: bucket 'staging' does not exist; using 'dev' (default)\nenvbuckets: Switched to bucket 'dev' (default)\nenvbuckets: hint: Create it with \"envbuckets switch -c staging\".\n" {
		t.Fatal(res)
	}
	r.linkTarget(".env", ".env.d/dev/.env")
	expectReport(t, r, "On branch staging\nUsing bucket 'dev' (default), because 'staging' (rule 'staging') does not exist\n  (use \"envbuckets switch -c staging\" to create it)\n\n1 file linked\n", "status")
	expectReport(t, r, "* staging  staging  (rule 'staging', missing; falls back to 'dev')\n", "branches")
	if res := r.run("switch", "prod"); res.code != ExitOK {
		t.Fatal(res)
	}
	expectReport(t, r, "On branch staging\nUsing bucket 'prod', but this branch uses 'dev' (default), because 'staging' (rule 'staging') does not exist\n  (use \"envbuckets switch\" to go back to 'dev')\n\n1 file linked\n", "status")
	expectReport(t, r, "* staging  staging  (rule 'staging', missing; falls back to 'dev'), using 'prod' for now\n", "branches")
}

func TestStatusMixedBrokenForeignAndIgnored(t *testing.T) {
	r := reportRepo(t)
	requireRepoSymlinks(t, r)
	r.write(".env.d/dev/.env", "SYNTHETIC_DEV")
	r.write(".env.d/dev/foreign", "SYNTHETIC_FOREIGN")
	r.write(".env.d/prod/other", "SYNTHETIC_OTHER")
	for name, target := range map[string]string{".env": ".env.d/dev/.env", "other": ".env.d/prod/other", "broken": ".env.d/dev/broken", "foreign": "../outside"} {
		if err := os.Symlink(target, r.path(name)); err != nil {
			t.Fatal(err)
		}
	}
	r.write(".gitignore", ".env.d/\n.env\nbroken\n")
	expectReport(t, r, "On branch main\nUsing a mix of buckets: 'dev', 'prod'\n  (use \"envbuckets switch\" to link one bucket)\n\nFiles not linked:\n  (to keep a file, move it to the same path under .env.d/<bucket>/, then use \"envbuckets switch\")\n\tbroken link:   broken\n\tforeign link:   foreign\n\nNot ignored by Git:\n  (add them to .gitignore)\n\tother\n\n2 of 4 files linked\n", "status")
}

func TestStatusDetachedTagAndCommit(t *testing.T) {
	r := reportRepo(t)
	requireRepoSymlinks(t, r)
	r.write(".env.d/dev/.env", "SYNTHETIC_DEV")
	if res := r.run("switch"); res.code != ExitOK {
		t.Fatal(res)
	}
	r.git("tag", "v1.0.0")
	r.git("checkout", "-q", "--detach")
	expectReport(t, r, "HEAD detached at v1.0.0\nUsing bucket 'dev'\n  (links stay as they are on a detached HEAD; use \"envbuckets switch <bucket>\" to pick one)\n\n1 file linked\n", "status")
	r.git("tag", "-d", "v1.0.0")
	name, err := gitCommitName(r)
	if err != nil {
		t.Fatal(err)
	}
	expectReport(t, r, "HEAD detached at "+name+"\nUsing bucket 'dev'\n  (links stay as they are on a detached HEAD; use \"envbuckets switch <bucket>\" to pick one)\n\n1 file linked\n", "status")
}

func gitCommitName(r *repo) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = r.root
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func TestStatusWithoutLinksDetachedAndMissingBuckets(t *testing.T) {
	r := reportRepo(t)
	r.git("tag", "v1.0.0")
	r.git("checkout", "-q", "--detach")
	expectReport(t, r, "HEAD detached at v1.0.0\nNo bucket is linked\n  (links stay as they are on a detached HEAD; use \"envbuckets switch <bucket>\" to pick one)\n\nall 0 files linked\n", "status")
	r.git("switch", "-q", "-c", "staging")
	expectReport(t, r, "On branch staging\nUsing bucket 'dev' (default), because 'staging' (rule 'staging') does not exist\n  (use \"envbuckets switch -c staging\" to create it)\n\nall 0 files linked\n", "status")
	if err := os.Remove(r.path(".env.d/dev")); err != nil {
		t.Fatal(err)
	}
	expectReport(t, r, "On branch staging\nNo bucket is linked, because default bucket 'dev' does not exist\n  (use \"envbuckets switch -c dev\" to create it)\n\nall 0 files linked\n", "status")
}

func TestStatusSafeHookInspection(t *testing.T) {
	r := reportRepo(t)
	r.git("config", "core.hooksPath", ".hooks")
	expectReport(t, r, "On branch main\nUsing bucket 'dev' (default)\n\nHook not installed:\n  (use \"envbuckets init\" to install it)\n\nall 0 files linked\n", "status")
	if _, err := block.InstallRepoHook(r.root); err != nil {
		t.Fatal(err)
	}
	expectReport(t, r, "On branch main\nUsing bucket 'dev' (default)\n\nall 0 files linked\n", "status")
	requireRepoSymlinks(t, r)
	if err := os.Remove(r.path(".hooks/post-checkout")); err != nil {
		t.Fatal(err)
	}
	r.write("secret", "SYNTHETIC_NOT_HOOK_CONTENT")
	if err := os.Symlink("../secret", r.path(".hooks/post-checkout")); err != nil {
		t.Fatal(err)
	}
	before := snapshotRepo(t, r)
	res := r.run("status")
	if res.code != ExitError || res.stdout != "" || res.stderr != "fatal: path \".hooks/post-checkout\" is a symlink\n" || !reflect.DeepEqual(before, snapshotRepo(t, r)) {
		t.Fatal(res)
	}
}

func TestStatusDeletedBucketsPreserveBrokenLinks(t *testing.T) {
	r := reportRepo(t)
	requireRepoSymlinks(t, r)
	r.write(".env.d/dev/.env", "SYNTHETIC_DEV")
	if res := r.run("switch"); res.code != ExitOK {
		t.Fatal(res)
	}
	if err := os.Rename(r.path(".env.d/dev"), r.path(".env.d/old")); err != nil {
		t.Fatal(err)
	}
	expectReport(t, r, "On branch main\nUsing bucket 'dev', because default bucket 'dev' does not exist\n  (use \"envbuckets switch -c dev\" to create it)\n\nFiles not linked:\n  (to keep a file, move it to the same path under .env.d/dev/, then use \"envbuckets switch\")\n\tbroken link:   .env\n\n0 of 1 files linked\n", "status")
	r.git("switch", "-q", "-c", "staging")
	expectReport(t, r, "On branch staging\nUsing bucket 'dev', because 'staging' (rule 'staging') and default bucket 'dev' do not exist\n  (use \"envbuckets switch -c staging\" to create it)\n\nFiles not linked:\n  (to keep a file, move it to the same path under .env.d/dev/, then use \"envbuckets switch\")\n\tbroken link:   .env\n\n0 of 1 files linked\n", "status")
	r.linkTarget(".env", ".env.d/dev/.env")
}

func TestReportsRejectSymlinkedConfigAndBucket(t *testing.T) {
	for _, name := range []string{".envbuckets.json", ".env.d/dev"} {
		t.Run(name, func(t *testing.T) {
			r := reportRepo(t)
			requireRepoSymlinks(t, r)
			if err := os.Remove(r.path(name)); err != nil {
				t.Fatal(err)
			}
			r.write("secret", "SYNTHETIC_NOT_METADATA")
			target := "secret"
			if name == ".env.d/dev" {
				target = "../secret"
			}
			if err := os.Symlink(target, r.path(name)); err != nil {
				t.Fatal(err)
			}
			for _, cmd := range []string{"status", "branches"} {
				before := snapshotRepo(t, r)
				res := r.run(cmd)
				if res.code != ExitError || res.stdout != "" || strings.Contains(res.all(), "SYNTHETIC") || !reflect.DeepEqual(before, snapshotRepo(t, r)) {
					t.Fatalf("%s: %+v", cmd, res)
				}
			}
		})
	}
}

func TestReportMappingMatchesSwitchPlan(t *testing.T) {
	r := reportRepo(t)
	r.write(".envbuckets.json", `{"default":"dev","rules":[{"branch":"release/*-rc*","bucket":"staging"},{"branch":"release/","bucket":"prod"},{"branch":"**/PAY-[0-9]*","bucket":"prod"}]}`)
	repo, err := fsx.OpenRepo(r.root)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	cfg, err := config.Load(repo.Root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main", "release/1.0-rc1", "release/1.0", "release", "team/PAY-42", "PAY-docs"} {
		resolution, err := switcher.Resolve(repo, cfg, name)
		if err != nil {
			t.Fatal(err)
		}
		plan := switcher.Build(repo, cfg, name, "")
		rows, err := branchRows(repo, cfg, nil, []string{name}, "", "", "")
		if err != nil || len(rows) != 1 || rows[0].bucket != plan.BranchBucket || resolution.Bucket != plan.Bucket {
			t.Fatalf("%s: rows=%+v plan=%+v err=%v", name, rows, plan, err)
		}
	}
}
