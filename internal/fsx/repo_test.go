package fsx

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestValidatePathSafety(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	r, err := OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, path := range []string{"../escape", filepath.Join(dir, "absolute"), filepath.Join(".git", "config"), filepath.Join(".GIT", "config"), filepath.Join("GIT~1", "config"), filepath.Join(".env.d", "dev", ".env"), "."} {
		if err := r.ValidatePath(path); err == nil {
			t.Errorf("ValidatePath(%q) unexpectedly succeeded", path)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "tracked.env"), []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "tracked.env")
	if err := r.ValidatePath("tracked.env"); err == nil {
		t.Fatal("tracked path accepted")
	}
	if err := os.Mkdir(filepath.Join(dir, "real"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(dir, "alias")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := r.ValidatePath(filepath.Join("alias", "file.env")); err == nil {
		t.Fatal("symlinked parent accepted")
	}
}

func TestScanBucketSkipsClutterAndReportsUnsafeEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".env.d", "dev", "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	r, err := OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, name := range []string{"a.env", ".DS_Store", "Thumbs.db", "desktop.ini", "backup~", "swap.swp", filepath.Join("nested", "b.env")} {
		if err := os.WriteFile(filepath.Join(dir, ".env.d", "dev", name), []byte("synthetic"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("a.env", filepath.Join(dir, ".env.d", "dev", "link.env")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	files, unsafe, err := r.ScanBucket("dev")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(files, []string{"a.env", "nested/b.env"}) {
		t.Fatalf("files = %v", files)
	}
	if !slices.Equal(unsafe, []string{"link.env"}) {
		t.Fatalf("unsafe = %v", unsafe)
	}
}

func TestScanBucketRejectsSymlinkRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "outside"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("outside", filepath.Join(dir, ".env.d")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	r, err := OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, _, err := r.ScanBucket("dev"); err == nil {
		t.Fatal("symlinked bucket root accepted")
	}
}

func TestScanBucketRejectsSymlinkBucketDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".env.d"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "outside"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "outside"), filepath.Join(dir, ".env.d", "dev")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	r, err := OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, _, err := r.ScanBucket("dev"); err == nil {
		t.Fatal("symlinked bucket directory accepted")
	}
}

func TestRenameBasedFileHelpers(t *testing.T) {
	dir := t.TempDir()
	requireSymlinks(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, ".env.d", "dev"), 0o700); err != nil {
		t.Fatal(err)
	}
	r, err := OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	source := filepath.Join(dir, ".env")
	if err := os.WriteFile(source, []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	usedHardlink, err := MoveFileToBucket(r.Root, ".env", filepath.Join(".env.d", "dev", ".env"), filepath.ToSlash(filepath.Join(".env.d", "dev", ".env")))
	if err != nil {
		t.Fatal(err)
	}
	info, err := r.Root.Lstat(".env")
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("source not replaced by link: %v %v", info, err)
	}
	if usedHardlink {
		if target, err := r.Root.Readlink(".env"); err != nil || filepath.ToSlash(target) != ".env.d/dev/.env" {
			t.Fatalf("link target = %q, %v", target, err)
		}
	}
	if err := LinkFile(r.Root, "another.env", filepath.ToSlash(filepath.Join(".env.d", "dev", ".env"))); err != nil {
		t.Fatal(err)
	}
	if info, err := r.Root.Lstat("another.env"); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("LinkFile result: %v %v", info, err)
	}
}

func TestMoveAndLinkWorkWithoutReadPermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod 000 does not deny reads on Windows")
	}
	dir := t.TempDir()
	requireSymlinks(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, ".env.d", "dev"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, ".env"), 0); err != nil {
		t.Fatal(err)
	}
	r, err := OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err := MoveFileToBucket(r.Root, ".env", filepath.Join(".env.d", "dev", ".env"), ".env.d/dev/.env"); err != nil {
		t.Fatal(err)
	}
	info, err := r.Root.Lstat(".env")
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("chmod 000 move did not install link: %v %v", info, err)
	}
	if err := LinkFile(r.Root, "alias.env", ".env.d/dev/.env"); err != nil {
		t.Fatalf("chmod 000 link target: %v", err)
	}
	info, err = r.Root.Lstat("alias.env")
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("chmod 000 LinkFile did not install link: %v %v", info, err)
	}
}

func TestLinkRepairsCrashAfterMove(t *testing.T) {
	dir := t.TempDir()
	requireSymlinks(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, ".env.d", "dev"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	// Model a crash after the fallback rename and before symlink creation.
	if err := r.Root.Rename(".env", filepath.Join(".env.d", "dev", ".env")); err != nil {
		t.Fatal(err)
	}
	if err := LinkFile(r.Root, ".env", ".env.d/dev/.env"); err != nil {
		t.Fatal(err)
	}
	if err := LinkFile(r.Root, ".env", ".env.d/dev/.env"); err != nil {
		t.Fatalf("repair rerun: %v", err)
	}
	link, err := r.Root.Readlink(".env")
	if err != nil || filepath.ToSlash(link) != ".env.d/dev/.env" {
		t.Fatalf("repaired link = %q, %v", link, err)
	}
}

func TestLinkFileRejectsEscapingTargetsAndSymlinkParents(t *testing.T) {
	dir := t.TempDir()
	requireSymlinks(t, dir)
	if err := os.MkdirAll(filepath.Join(dir, "real"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(dir, "alias")); err != nil {
		t.Fatal(err)
	}
	r, err := OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, target := range []string{"../../../outside", "alias/file"} {
		if err := LinkFile(r.Root, "link", target); err == nil {
			t.Errorf("LinkFile accepted unsafe target %q", target)
		}
	}
}

func TestLinkFileDoesNotReplaceRealFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "target"), []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "link"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := LinkFile(r.Root, "link", "target"); err == nil {
		t.Fatal("LinkFile replaced a real file")
	}
	info, err := r.Root.Lstat("link")
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("real file was changed: %v %v", info, err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...) //nolint:gosec // test helper invokes Git with fixed test arguments
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
