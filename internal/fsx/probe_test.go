package fsx

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
)

func TestSymlinkProbeErrorClassification(t *testing.T) {
	privilege := &os.LinkError{Op: "symlink", Old: "target", New: "link", Err: syscall.Errno(1314)}
	for _, tc := range []struct {
		err     error
		windows bool
		want    bool
	}{
		{privilege, true, true},
		{fmt.Errorf("wrapped: %w", privilege), true, true},
		{privilege, false, false},
		{&os.LinkError{Op: "symlink", Err: os.ErrPermission}, true, false},
		{os.ErrPermission, true, false},
	} {
		err := symlinkProbeError(tc.err, tc.windows)
		if strings.Contains(err.Error(), "Developer Mode") != tc.want || !errors.Is(err, tc.err) {
			t.Fatalf("classification: %v", err)
		}
	}
}

func TestProbeCleansUpOnSuccessAndFailure(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	_ = ProbeSymlinks(root) // Either platform outcome must leave no temporary link.
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("probe left entries: %v %v", entries, err)
	}
}
