package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpWithoutRepository(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"--help"}, {"-h"}} {
		var stdout, stderr bytes.Buffer
		code := Run(args, Env{Cwd: t.TempDir(), Stdout: &stdout, Stderr: &stderr})
		if code != ExitOK || stderr.Len() != 0 || stdout.String() != helpText {
			t.Fatalf("help %v: code=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
	}
}

func TestVersionAndHook(t *testing.T) {
	r := newRepo(t)
	version := r.run("version")
	if version.code != ExitOK || !strings.Contains(version.stdout, "envbuckets test") {
		t.Fatalf("version: %+v", version)
	}
	hook := r.run("hook", "old", "new", "0")
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
