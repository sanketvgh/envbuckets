package output

import (
	"bytes"
	"testing"
)

func TestStreamsAndHookPrefix(t *testing.T) {
	var stdout, stderr bytes.Buffer
	w := Writer{Out: &stdout, Err: &stderr, Hook: true}
	w.Fatal("cannot link %s", ".env")
	w.Error("invalid configuration")
	w.Warning("bucket is missing")
	w.Hint("run envbuckets init")
	w.Info("Switched to bucket 'dev'")
	w.Would("link .env")
	w.List("On branch main\n")
	if got, want := stdout.String(), "Would link .env\nOn branch main\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "envbuckets: fatal: cannot link .env\nenvbuckets: error: invalid configuration\nenvbuckets: warning: bucket is missing\nenvbuckets: hint: run envbuckets init\nenvbuckets: Switched to bucket 'dev'\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}
