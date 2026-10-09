package block

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/gitx"
)

// InstallRepoHook installs into Git's actual hooks directory. Linked worktrees
// may use their shared Git metadata directory; unrelated shared hooks are refused.
func InstallRepoHook(repoPath string) (HookResult, error) {
	_, result, err := repoHook(repoPath, false)
	return result, err
}

// PreviewRepoHook performs installation preflight without writing anything.
// The path is relative to the worktree when possible, for dry-run output.
func PreviewRepoHook(repoPath string) (string, HookResult, error) {
	return repoHook(repoPath, true)
}

func repoHook(repoPath string, dry bool) (string, HookResult, error) {
	hooks, err := gitx.HooksDir(repoPath)
	if err != nil {
		return "", HookUnchanged, err
	}
	base := repoPath
	rel, err := filepath.Rel(base, hooks)
	if err != nil || !filepath.IsLocal(rel) {
		common, err := gitx.CommonDir(repoPath)
		if err != nil {
			return "", HookUnchanged, err
		}
		rel, err = filepath.Rel(common, hooks)
		if err != nil || !filepath.IsLocal(rel) {
			return "", HookUnchanged, errors.New("hooks directory is outside the repository; use a core.hooksPath inside the repository")
		}
		base = common
	}
	info, err := os.Lstat(base)
	if err != nil {
		return "", HookUnchanged, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", HookUnchanged, errors.New("unsafe repository hooks root")
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return "", HookUnchanged, err
	}
	defer root.Close()
	name := filepath.ToSlash(filepath.Join(rel, "post-checkout"))
	if bucketPath(name) {
		return "", HookUnchanged, errors.New("hooks directory is inside a bucket tree")
	}
	if err := fsx.ValidateInternalPath(root, name); err != nil {
		return "", HookUnchanged, err
	}
	path, err := filepath.Rel(repoPath, filepath.Join(hooks, "post-checkout"))
	if err != nil {
		return "", HookUnchanged, err
	}
	if dry {
		existing, out, err := hookContent(root, name)
		result := HookUnchanged
		if !bytes.Equal(existing, out) {
			result = HookWritten
		}
		if info, statErr := root.Lstat(name); runtime.GOOS != "windows" && statErr == nil && info.Mode().Perm()&0o111 == 0 {
			result = HookWritten
		}
		return filepath.ToSlash(path), result, err
	}
	if err := root.MkdirAll(filepath.ToSlash(rel), 0o755); err != nil {
		return "", HookUnchanged, err
	}
	result, err := installHookRoot(root, name)
	return filepath.ToSlash(path), result, err
}

func bucketPath(name string) bool {
	for part := range strings.SplitSeq(filepath.ToSlash(filepath.Clean(name)), "/") {
		if strings.EqualFold(part, ".env.d") {
			return true
		}
	}
	return false
}

func installHookRoot(root *os.Root, name string) (HookResult, error) {
	existing, out, err := hookContent(root, name)
	if err != nil {
		return HookUnchanged, err
	}
	if bytes.Equal(existing, out) {
		file, err := root.Open(name)
		if err != nil {
			return HookUnchanged, err
		}
		defer file.Close()
		return HookUnchanged, file.Chmod(0o755)
	}
	if err := fsx.WriteFileAtomicRoot(root, name, out, 0o755); err != nil {
		return HookUnchanged, err
	}
	return HookWritten, nil
}

func hookContent(root *os.Root, name string) ([]byte, []byte, error) {
	if err := fsx.ValidateInternalPath(root, name); err != nil {
		return nil, nil, err
	}
	if info, err := root.Lstat(name); err == nil && !info.Mode().IsRegular() {
		return nil, nil, errors.New("hook is not a regular file")
	} else if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}
	existing, err := root.ReadFile(name)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}
	content := existing
	for {
		var removed bool
		content, removed = Remove(content)
		if !removed {
			break
		}
	}
	// Do not append a second block when a truncated marker cannot be replaced.
	for line := range strings.SplitSeq(string(content), "\n") {
		if strings.HasPrefix(line, beginPrefix) || strings.HasPrefix(line, endPrefix) {
			return nil, nil, errors.New("incomplete envbuckets hook block; repair its markers before reinstalling")
		}
	}
	if len(content) == 0 {
		content = []byte("#!/bin/sh\n")
	}
	out, _ := Upsert(content, HookBody)
	return existing, out, nil
}
