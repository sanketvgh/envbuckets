package block

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRemoveHookExactRanges(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		t.Run(map[string]string{"\n": "LF", "\r\n": "CRLF"}[newline], func(t *testing.T) {
			before := "#!/bin/bash\necho 'before'\n\n"
			between := "\t# keep whitespace\necho '# >>> envbuckets >>>'\n"
			after := "echo after\n# >>> envbuckets v2 >>>\nfuture block\n# <<< envbuckets v2 <<<\nlast byte"
			current := Begin + "\ncustom edits inside removed range\n" + End + "\n"
			legacy := "# >>> envbuckets v1 >>>\nold command\n# <<< envbuckets v1 <<<\n"
			content := strings.ReplaceAll(before+current+between+legacy+current+after, "\n", newline)
			want := strings.ReplaceAll(before+between+after, "\n", newline)
			path := filepath.Join(t.TempDir(), "post-checkout")
			if err := os.WriteFile(path, []byte(content), 0o751); err != nil {
				t.Fatal(err)
			}
			// File.Stat captures Windows identity now; Lstat can defer identity
			// lookup until SameFile, when the path already names the new hook.
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			info, statErr := file.Stat()
			closeErr := file.Close()
			if statErr != nil || closeErr != nil {
				t.Fatalf("capture identity: %v %v", statErr, closeErr)
			}
			result, err := RemoveHook(path)
			if err != nil || result != HookWritten {
				t.Fatalf("remove: %v %v", result, err)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != want {
				t.Fatalf("outside bytes changed: %q, want %q; %v", got, want, err)
			}
			remaining, err := os.Lstat(path)
			if err != nil || remaining.Mode().Perm() != info.Mode().Perm() {
				t.Fatal("hook permissions changed")
			}
			if runtime.GOOS != "windows" && remaining.Mode().Perm()&0o111 == 0 {
				t.Fatal("hook no longer executable")
			}
			if os.SameFile(info, remaining) {
				t.Fatal("hook was rewritten in place instead of replaced")
			}
			if result, err := RemoveHook(path); err != nil || result != HookUnchanged {
				t.Fatalf("rerun: %v %v", result, err)
			}
		})
	}
}

func TestRemoveHookDeletesOnlyEmptyRemainder(t *testing.T) {
	for _, remainder := range []string{"", " \n\t\r\n", "#!/bin/sh\n\n\t", "#!/bin/bash\r\n\r\n"} {
		path := filepath.Join(t.TempDir(), "post-checkout")
		if err := os.WriteFile(path, []byte(remainder+"\n"+Begin+"\nx\n"+End), 0o755); err != nil {
			t.Fatal(err)
		}
		if result, err := RemoveHook(path); err != nil || result != HookDeleted {
			t.Fatalf("remainder %q: %v %v", remainder, result, err)
		}
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("empty hook remains: %v", err)
		}
	}
}

func TestRemoveHookMalformedRangesStayUntouched(t *testing.T) {
	for _, content := range []string{
		Begin + "\ntruncated\n",
		End + "\n",
		Begin + "\nx\n# <<< envbuckets v1 <<<\n",
		Begin + "\n" + Begin + "\nx\n" + End + "\n",
		Begin + "\nx\n" + End + "\n" + Begin + "\ntruncated",
	} {
		path := filepath.Join(t.TempDir(), "post-checkout")
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := RemoveHook(path); err == nil {
			t.Fatalf("malformed range accepted: %q", content)
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != content {
			t.Fatal("malformed hook changed")
		}
	}
}

func TestRemoveHookRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "post-checkout")
	target := filepath.Join(dir, "keep")
	content := Begin + "\nSYNTHETIC\n" + End + "\n"
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("keep", path); err != nil {
		t.Skipf("symlink support required: %v", err)
	}
	if _, err := RemoveHook(path); err == nil {
		t.Fatal("hook symlink accepted")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != content {
		t.Fatal("hook symlink target changed")
	}
}
