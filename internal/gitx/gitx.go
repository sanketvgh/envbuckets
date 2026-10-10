// Package gitx shells out to git for the repo root, current branch, and hooks
// directory.
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
	out, err := run(root, "symbolic-ref", "-q", "HEAD")
	if err != nil {
		if exitCode(err) == 1 {
			return "", nil
		}
		return "", err
	}
	return strings.TrimPrefix(out, "refs/heads/"), nil
}

// Branches reads all short local branch names in sorted order with one Git call.
func Branches(root string) ([]string, error) {
	out, err := run(root, "for-each-ref", "--sort=refname", "--format=%(refname:strip=2)", "refs/heads/")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n"), nil
}

// DetachedName describes a detached HEAD using an exact tag or a short commit.
func DetachedName(root string) (string, error) {
	if tag, err := run(root, "describe", "--tags", "--exact-match", "HEAD"); err == nil {
		return tag, nil
	}
	return run(root, "rev-parse", "--short", "HEAD")
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

// CommonDir returns Git's shared metadata directory, including in a linked worktree.
func CommonDir(root string) (string, error) {
	out, err := run(root, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	p := filepath.FromSlash(out)
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	return filepath.Clean(p), nil
}

// LocalFiles lists untracked files, including ignored files but never descending
// into wholly ignored directories. Directory entries and nested repositories
// from Git's --directory output are intentionally omitted.
func LocalFiles(root string) ([]string, error) {
	var files []string
	for _, extra := range [][]string{nil, {"--ignored", "--directory"}} {
		args := append([]string{"ls-files", "--others", "--exclude-standard", "-z"}, extra...)
		out, err := run(root, args...)
		if err != nil {
			return nil, err
		}
		for name := range strings.SplitSeq(out, "\x00") {
			if name != "" && !strings.HasSuffix(name, "/") {
				files = append(files, name)
			}
		}
	}
	return files, nil
}

// Ignored reports whether Git ignores a literal repository-relative path.
func Ignored(root, name string) (bool, error) {
	_, err := run(root, "check-ignore", "--no-index", "-q", "--", name)
	if err == nil {
		return true, nil
	}
	if exitCode(err) == 1 {
		return false, nil
	}
	return false, err
}

func exitCode(err error) int {
	if exit, ok := errors.AsType[*exec.ExitError](err); ok {
		return exit.ExitCode()
	}
	return 0
}

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...) //nolint:gosec // argv only, no shell; names are validated by the callers
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
	if len(args) > 0 && args[0] == "ls-files" {
		return stdout.String(), nil // NUL-delimited filenames may contain whitespace.
	}
	return strings.TrimRight(stdout.String(), "\r\n"), nil
}
