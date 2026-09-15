// Package gitx shells out to git for the repo root, current branch, and
// hooks directory.
package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotRepo is returned when dir is not inside a git work tree.
var ErrNotRepo = errors.New("not a git repository")

// Root returns the absolute work tree root containing dir.
func Root(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", ErrNotRepo
	}
	return filepath.Clean(filepath.FromSlash(out)), nil
}

// Branch returns the checked-out branch name, or "" on detached HEAD.
func Branch(root string) (string, error) {
	out, err := run(root, "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}
	return out, nil
}

// HooksDir returns the absolute hooks directory, honoring core.hooksPath.
func HooksDir(root string) (string, error) {
	out, err := run(root, "rev-parse", "--git-path", "hooks")
	if err != nil {
		return "", err
	}
	p := filepath.FromSlash(out)
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	return filepath.Clean(p), nil
}

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...) //nolint:gosec // args are fixed by the callers above, never user input
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}
