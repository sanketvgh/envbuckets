// Package gitx shells out to git for the repo root, current branch, hooks
// directory, and per-branch bucket links in the local git config.
package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
		if exitCode(err) == 1 {
			return "", nil
		}
		return "", err
	}
	return out, nil
}

// HooksDir returns the hooks directory only when the hook and all existing
// parent components are ordinary paths inside the work tree.
func HooksDir(root string) (string, error) {
	out, err := run(root, "rev-parse", "--git-path", "hooks")
	if err != nil {
		return "", err
	}
	p := filepath.FromSlash(out)
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	p = filepath.Clean(p)
	if err := checkLocalHookPath(root, filepath.Join(p, "post-checkout")); err != nil {
		return "", err
	}
	return p, nil
}

func checkLocalHookPath(root, hook string) error {
	rel, err := filepath.Rel(root, hook)
	if err != nil || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("git hooks path %s is outside the repository", hook)
	}
	current := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect git hook path %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("git hook path %s contains a symlink", current)
		}
	}
	return nil
}

// BranchExists reports whether branch is a local branch.
func BranchExists(root, branch string) bool {
	_, err := run(root, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// BranchLink is a branch pinned to a bucket in the local git config.
type BranchLink struct {
	Branch string
	Bucket string
}

const linkVar = "envbuckets"

func linkKey(branch string) string {
	return "branch." + branch + "." + linkVar
}

// LinkedBucket returns the bucket linked to branch, or "" when none is.
func LinkedBucket(root, branch string) (string, error) {
	out, err := run(root, "config", "--local", "--get", linkKey(branch))
	if exitCode(err) == 1 {
		return "", nil
	}
	return out, err
}

// SetLink links branch to bucket in the local git config.
func SetLink(root, branch, bucket string) error {
	_, err := run(root, "config", "--local", linkKey(branch), bucket)
	return err
}

// Unlink removes the link of branch and reports whether one existed.
func Unlink(root, branch string) (bool, error) {
	_, err := run(root, "config", "--local", "--unset-all", linkKey(branch))
	if exitCode(err) == 5 {
		return false, nil
	}
	return err == nil, err
}

// Links returns every branch link in the local git config, sorted by branch.
func Links(root string) ([]BranchLink, error) {
	out, err := run(root, "config", "--local", "--get-regexp", `^branch\..*\.`+linkVar+`$`)
	if exitCode(err) == 1 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var links []BranchLink
	for _, line := range strings.Split(out, "\n") {
		key, bucket, _ := strings.Cut(line, " ")
		branch := strings.TrimSuffix(strings.TrimPrefix(key, "branch."), "."+linkVar)
		links = append(links, BranchLink{Branch: branch, Bucket: bucket})
	}
	sort.Slice(links, func(i, j int) bool { return links[i].Branch < links[j].Branch })
	return links, nil
}

func exitCode(err error) int {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
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
	return strings.TrimSpace(stdout.String()), nil
}
