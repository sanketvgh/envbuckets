package block

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUpsertAppendsAndReplaces(t *testing.T) {
	custom := "#!/bin/sh\necho custom\n"
	out, changed := Upsert([]byte(custom), "one")
	if !changed || !strings.HasPrefix(string(out), custom) {
		t.Fatalf("append failed: %q", out)
	}
	if !strings.HasSuffix(string(out), Begin+"\none\n"+End+"\n") {
		t.Fatalf("block missing: %q", out)
	}

	again, changed := Upsert(out, "one")
	if changed || string(again) != string(out) {
		t.Fatalf("idempotent upsert changed content")
	}

	replaced, changed := Upsert(out, "two")
	if !changed || strings.Contains(string(replaced), "one") || !strings.Contains(string(replaced), "two") {
		t.Fatalf("replace failed: %q", replaced)
	}
	if !strings.HasPrefix(string(replaced), custom) {
		t.Fatalf("custom content lost: %q", replaced)
	}
}

func TestUpsertAddsNewlineBeforeBlock(t *testing.T) {
	out, _ := Upsert([]byte("no trailing newline"), "x")
	if !strings.HasPrefix(string(out), "no trailing newline\n"+Begin) {
		t.Fatalf("got %q", out)
	}
}

func TestRemoveKeepsSurroundingBytes(t *testing.T) {
	before := "#!/bin/sh\necho before\n"
	after := "echo after\n"
	content := before + Begin + "\nbody\n" + End + "\n" + after
	out, found := Remove([]byte(content))
	if !found || string(out) != before+after {
		t.Fatalf("got %q", out)
	}
	same, found := Remove([]byte(before))
	if found || string(same) != before {
		t.Fatalf("remove without block changed content")
	}
}

func TestMarkersMatchByPrefix(t *testing.T) {
	content := "# >>> envbuckets v2 >>>\nfuture\n# <<< envbuckets v2 <<<\n"
	out, changed := Upsert([]byte(content), "now")
	if !changed || strings.Contains(string(out), "v2") {
		t.Fatalf("v2 block not replaced: %q", out)
	}
}

func TestBody(t *testing.T) {
	content := "x\n" + Begin + "\na\nb\n" + End + "\ny\n"
	got := Body([]byte(content))
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %v", got)
	}
	if Body([]byte("nothing")) != nil {
		t.Fatal("expected nil body")
	}
}

func TestInstallAndRemoveHook(t *testing.T) {
	dir := t.TempDir()
	hook := filepath.Join(dir, "post-checkout")

	res, err := InstallHook(hook)
	if err != nil || res != HookWritten {
		t.Fatalf("install: %v %v", res, err)
	}
	data, _ := os.ReadFile(hook)
	if !strings.HasPrefix(string(data), "#!/bin/sh\n"+Begin) {
		t.Fatalf("fresh hook: %q", data)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(hook)
		if info.Mode()&0o111 == 0 {
			t.Fatal("hook not executable")
		}
	}

	res, err = InstallHook(hook)
	if err != nil || res != HookUnchanged {
		t.Fatalf("second install: %v %v", res, err)
	}

	res, err = RemoveHook(hook)
	if err != nil || res != HookDeleted {
		t.Fatalf("remove: %v %v", res, err)
	}
	if _, err := os.Stat(hook); !os.IsNotExist(err) {
		t.Fatal("hook file should be deleted when only our block remained")
	}

	res, err = RemoveHook(hook)
	if err != nil || res != HookUnchanged {
		t.Fatalf("remove twice: %v %v", res, err)
	}
}

func TestRemoveHookPreservesCustomContent(t *testing.T) {
	dir := t.TempDir()
	hook := filepath.Join(dir, "post-checkout")
	custom := "#!/bin/bash\necho custom hook\n"
	if err := os.WriteFile(hook, []byte(custom), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallHook(hook); err != nil {
		t.Fatal(err)
	}
	res, err := RemoveHook(hook)
	if err != nil || res != HookWritten {
		t.Fatalf("remove: %v %v", res, err)
	}
	data, _ := os.ReadFile(hook)
	if string(data) != custom {
		t.Fatalf("custom hook not byte-identical: %q", data)
	}
}

func TestMergeAndDropLines(t *testing.T) {
	merged := MergeLines([]string{".env", "old/.env"}, []string{".env", "apps/api/.env"})
	want := []string{".env", "old/.env", "apps/api/.env"}
	if strings.Join(merged, ",") != strings.Join(want, ",") {
		t.Fatalf("merge: %v", merged)
	}
	dropped := DropLines(merged, []string{"old/.env"})
	if strings.Join(dropped, ",") != ".env,apps/api/.env" {
		t.Fatalf("drop: %v", dropped)
	}
}
