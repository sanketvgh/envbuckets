package block

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallHookMovesAndDeduplicatesLegacyBlocks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "post-checkout")
	content := "#!/bin/sh\necho before\n# >>> envbuckets v1 >>>\nold command\n# <<< envbuckets v1 <<<\necho after\n" + Begin + "\nduplicate\n" + End + "\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallHook(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "#!/bin/sh\necho before\necho after\n" + string(render(HookBody))
	if string(got) != want {
		t.Fatalf("hook=%q, want=%q", got, want)
	}
	if res, err := InstallHook(path); err != nil || res != HookUnchanged {
		t.Fatalf("rerun: %v %v", res, err)
	}
}

func TestInstallHookRejectsIncompleteBlockAndSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "post-checkout")
	content := "#!/bin/sh\n" + Begin + "\ntruncated\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallHook(path); err == nil {
		t.Fatal("incomplete block accepted")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != content {
		t.Fatal("incomplete hook changed")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "real")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := InstallHook(path); err == nil {
		t.Fatal("hook symlink accepted")
	}
	got, err = os.ReadFile(target)
	if err != nil || string(got) != "keep" {
		t.Fatal("hook symlink target changed")
	}
}

func TestMarkersMustBeWholeLines(t *testing.T) {
	content := "echo '# >>> envbuckets v1 >>>'\necho '# <<< envbuckets v1 <<<'\n"
	out, changed := Upsert([]byte(content), HookBody)
	if !changed || !strings.HasPrefix(string(out), content) {
		t.Fatalf("marker-like text altered: %q", out)
	}
}

func TestHookReturnsZeroWhenBinaryFailsOrIsMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix executable shell fixtures required; real Git hook is covered by testscript")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh unavailable")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "post-checkout")
	if _, err := InstallHook(path); err != nil {
		t.Fatal(err)
	}
	for _, binary := range []bool{false, true} {
		if binary {
			if err := os.WriteFile(filepath.Join(dir, "envbuckets"), []byte("#!/bin/sh\nexit 42\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(sh, path, "old", "new", "1") //nolint:gosec // shell executes only this synthetic test fixture
		cmd.Env = append(os.Environ(), "PATH="+dir)
		out, err := cmd.CombinedOutput()
		if err != nil || len(out) != 0 {
			t.Fatalf("binary=%v: err=%v output=%q", binary, err, out)
		}
	}
}
