package fsx

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
)

// ProbeSymlinks checks actual symlink support even for dry runs. The temporary
// link stays inside root, points to a nonexistent file, and is removed at once.
func ProbeSymlinks(root *os.Root) error {
	name, err := StageSymlinkRoot(root, ".envbuckets-probe-target", ".")
	if err != nil {
		return symlinkProbeError(err, runtime.GOOS == "windows")
	}
	if err := root.Remove(name); err != nil {
		return fmt.Errorf("remove symlink probe: %w", err)
	}
	return nil
}

func symlinkProbeError(err error, windows bool) error {
	if link, ok := errors.AsType[*os.LinkError](err); windows && ok && errors.Is(link.Err, syscall.Errno(1314)) {
		return fmt.Errorf("symlink privilege is unavailable; enable Windows Developer Mode, then rerun init: %w", err)
	}
	return fmt.Errorf("cannot create symlinks in this repository: %w", err)
}
