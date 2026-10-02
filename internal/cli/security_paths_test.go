package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sanketvgh/envbuckets/internal/config"
)

func TestSymlinkedScopeCannotWriteOutsideRepo(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	outside := t.TempDir()
	r.write("apps/README", "synthetic\n")
	if err := os.Symlink(outside, r.path("apps/external")); err != nil {
		t.Fatal(err)
	}
	cfg := config.New()
	cfg.Scopes = []config.Scope{{Name: "external", Path: "apps/external"}}
	cfg.Rules = []config.Rule{{Pattern: "main", Bucket: "qa"}}
	if err := cfg.Save(r.root); err != nil {
		t.Fatal(err)
	}
	if res := r.ok("status"); !strings.Contains(res.stdout, "ERROR scope path") || !strings.Contains(res.stdout, "symlink") {
		t.Fatalf("status hid unsafe scope path: %s", res.all())
	}
	if res := r.run("bucket", "add", "qa", "--scope", "external"); res.code != ExitBlocked {
		t.Fatalf("bucket add followed scope link: %d %s", res.code, res.all())
	}
	if _, err := os.Lstat(filepath.Join(outside, bucketsDir)); !os.IsNotExist(err) {
		t.Fatalf("bucket data created outside repo: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(outside, bucketsDir, "qa"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, bucketsDir, "qa", envFile), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if res := r.run("apply"); res.code != ExitBlocked {
		t.Fatalf("apply followed scope link: %d %s", res.code, res.all())
	}
	if _, err := os.Lstat(filepath.Join(outside, envFile)); !os.IsNotExist(err) {
		t.Fatalf("apply created link outside repo: %v", err)
	}
}

func TestSymlinkedBucketDirectoryCannotWriteOrDeleteOutsideRepo(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	if err := config.New().Save(r.root); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, r.path(bucketsDir)); err != nil {
		t.Fatal(err)
	}
	if res := r.run("bucket", "add", "qa"); res.code != ExitBlocked {
		t.Fatalf("bucket add followed .env.d link: %d %s", res.code, res.all())
	}
	file := filepath.Join(outside, "qa", envFile)
	if _, err := os.Lstat(file); !os.IsNotExist(err) {
		t.Fatalf("bucket created outside repo: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if res := r.runIn(r.root, "DELETE\n", "bucket", "rm", "qa", "--purge"); res.code != ExitBlocked {
		t.Fatalf("bucket rm followed .env.d link: %d %s", res.code, res.all())
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("bucket file outside repo was deleted: %v", err)
	}
}

func TestBucketRmNeedsPurgeForUnexpectedFiles(t *testing.T) {
	r := newRepo(t)
	if err := config.New().Save(r.root); err != nil {
		t.Fatal(err)
	}
	r.write(".env.d/qa/.env", "")
	r.write(".env.d/qa/notes.txt", "synthetic data\n")
	if res := r.run("bucket", "rm", "qa"); res.code != ExitBlocked || !strings.Contains(res.all(), "contains data or non-bucket files") {
		t.Fatalf("bucket rm deleted extra file: %d %s", res.code, res.all())
	}
	if r.read(".env.d/qa/notes.txt") != "synthetic data\n" {
		t.Fatal("extra file changed")
	}
	if res := r.runIn(r.root, "no\n", "bucket", "rm", "qa", "--purge"); res.code != ExitBlocked || !r.exists(".env.d/qa/notes.txt") {
		t.Fatalf("refused purge changed data: %d %s", res.code, res.all())
	}
	if res := r.runIn(r.root, "DELETE\n", "bucket", "rm", "qa", "--purge"); res.code != ExitOK || r.exists(".env.d/qa") {
		t.Fatalf("confirmed purge failed: %d %s", res.code, res.all())
	}
}

func TestSymlinkedBucketFileIsNotAvailable(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	outside := filepath.Join(t.TempDir(), "outside.env")
	if err := os.WriteFile(outside, []byte("synthetic data\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(r.path(".env.d/qa"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, r.path(".env.d/qa/.env")); err != nil {
		t.Fatal(err)
	}
	cfg := config.New()
	cfg.Rules = []config.Rule{{Pattern: "main", Bucket: "qa"}}
	if err := cfg.Save(r.root); err != nil {
		t.Fatal(err)
	}
	if res := r.run("bucket", "add", "qa"); res.code != ExitBlocked {
		t.Fatalf("bucket add accepted file symlink: %d %s", res.code, res.all())
	}
	if res := r.run("check"); res.code != ExitBlocked || !strings.Contains(res.stdout, "MISSING .env.d/qa/.env") {
		t.Fatalf("check accepted file symlink: %d %s", res.code, res.all())
	}
	if res := r.run("apply"); res.code != ExitBlocked || r.exists(envFile) {
		t.Fatalf("apply accepted file symlink: %d %s", res.code, res.all())
	}
	if got, err := os.ReadFile(outside); err != nil || string(got) != "synthetic data\n" {
		t.Fatalf("outside file changed: %v", err)
	}
}

func TestUninstallDoesNotMaterializeOutsideBucketFiles(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	outside := t.TempDir()
	file := filepath.Join(outside, bucketsDir, "qa", envFile)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("synthetic data\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, bucketsDir), r.path(bucketsDir)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(linkTarget("qa"), r.path(envFile)); err != nil {
		t.Fatal(err)
	}
	if res := r.run("uninstall"); res.code != ExitOK || !strings.Contains(res.all(), "unsafe bucket file") {
		t.Fatalf("uninstall followed outside bucket: %d %s", res.code, res.all())
	}
	if got := r.readlink(envFile); got != linkTarget("qa") {
		t.Fatalf("unsafe link was materialized: %s", got)
	}
	if got, err := os.ReadFile(file); err != nil || string(got) != "synthetic data\n" {
		t.Fatalf("outside file changed: %v", err)
	}
	if res := r.runIn(r.root, "DELETE\n", "uninstall", "--purge"); res.code != ExitBlocked || !strings.Contains(res.all(), "purge stopped") {
		t.Fatalf("unsafe link should block purge: %d %s", res.code, res.all())
	}
	if got := r.readlink(envFile); got != linkTarget("qa") {
		t.Fatalf("purge removed unsafe link: %s", got)
	}
	if err := os.Remove(r.path(envFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(file, r.path(envFile)); err != nil {
		t.Fatal(err)
	}
	if res := r.run("uninstall"); res.code != ExitOK || !strings.Contains(res.all(), "foreign symlink") {
		t.Fatalf("uninstall followed foreign link: %d %s", res.code, res.all())
	}
	if got := r.readlink(envFile); got != filepath.ToSlash(file) {
		t.Fatalf("foreign link was materialized: %s", got)
	}
}

func TestInitDoesNotMoveRealEnvThroughBucketDirectorySymlink(t *testing.T) {
	r := newRepo(t)
	requireSymlinks(t, r.root)
	r.write(envFile, "synthetic data\n")
	outside := t.TempDir()
	if err := os.Symlink(outside, r.path(bucketsDir)); err != nil {
		t.Fatal(err)
	}
	if res := r.run("init", "--into", "qa"); res.code != ExitBlocked {
		t.Fatalf("init followed .env.d link: %d %s", res.code, res.all())
	}
	if got := r.read(envFile); got != "synthetic data\n" {
		t.Fatalf("real .env changed: %q", got)
	}
	if _, err := os.Lstat(filepath.Join(outside, "qa")); !os.IsNotExist(err) {
		t.Fatalf("bucket created outside repo: %v", err)
	}
}
