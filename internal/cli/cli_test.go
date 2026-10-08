package cli

import (
	"strings"
	"testing"
)

func TestVersionAndHook(t *testing.T) {
	r := newRepo(t)
	version := r.run("version")
	if version.code != ExitOK || !strings.Contains(version.stdout, "envbuckets test") {
		t.Fatalf("version: %+v", version)
	}
	hook := r.run("hook", "old", "new", "1")
	if hook.code != ExitOK || hook.all() != "" {
		t.Fatalf("hook must be a silent success: %+v", hook)
	}
}

func TestUnknownCommand(t *testing.T) {
	r := newRepo(t)
	res := r.run("removed-command")
	if res.code != ExitUsage || !strings.HasPrefix(res.stderr, "usage:") {
		t.Fatalf("unknown command: %+v", res)
	}
}
