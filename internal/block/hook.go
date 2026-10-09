package block

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/gitx"
)

// InstallRepoHook installs into Git's actual hooks directory. Linked worktrees
// may use their shared Git metadata directory; unrelated shared hooks are refused.
func InstallRepoHook(repoPath string) (HookResult, error) {
	hooks, err := gitx.HooksDir(repoPath)
	if err != nil {
		return HookUnchanged, err
	}
	base := repoPath
	rel, err := filepath.Rel(base, hooks)
	if err != nil || !filepath.IsLocal(rel) {
		common, err := gitx.CommonDir(repoPath)
		if err != nil {
			return HookUnchanged, err
		}
		rel, err = filepath.Rel(common, hooks)
		if err != nil || !filepath.IsLocal(rel) {
			return HookUnchanged, fmt.Errorf("hooks directory is outside the repository; use a core.hooksPath inside the repository")
		}
		base = common
	}
	info, err := os.Lstat(base)
	if err != nil {
		return HookUnchanged, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return HookUnchanged, fmt.Errorf("unsafe repository hooks root")
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return HookUnchanged, err
	}
	defer root.Close()
	name := filepath.ToSlash(filepath.Join(rel, "post-checkout"))
	if bucketPath(name) {
		return HookUnchanged, fmt.Errorf("hooks directory is inside a bucket tree")
	}
	if err := fsx.ValidateInternalPath(root, name); err != nil {
		return HookUnchanged, err
	}
	if err := root.MkdirAll(filepath.ToSlash(rel), 0o755); err != nil {
		return HookUnchanged, err
	}
	return installHookRoot(root, name)
}

func bucketPath(name string) bool {
	for _, part := range strings.Split(filepath.ToSlash(filepath.Clean(name)), "/") {
		if strings.EqualFold(part, ".env.d") {
			return true
		}
	}
	return false
}

func installHookRoot(root *os.Root, name string) (HookResult, error) {
	if err := fsx.ValidateInternalPath(root, name); err != nil {
		return HookUnchanged, err
	}
	if info, err := root.Lstat(name); err == nil && !info.Mode().IsRegular() {
		return HookUnchanged, fmt.Errorf("hook is not a regular file")
	} else if err != nil && !os.IsNotExist(err) {
		return HookUnchanged, err
	}
	existing, err := root.ReadFile(name)
	if err != nil && !os.IsNotExist(err) {
		return HookUnchanged, err
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
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, beginPrefix) || strings.HasPrefix(line, endPrefix) {
			return HookUnchanged, fmt.Errorf("incomplete envbuckets hook block; repair its markers before reinstalling")
		}
	}
	if len(content) == 0 {
		content = []byte("#!/bin/sh\n")
	}
	out, _ := Upsert(content, HookBody)
	if bytes.Equal(existing, out) {
		file, err := root.Open(name)
		if err != nil {
			return HookUnchanged, err
		}
		defer file.Close()
		return HookUnchanged, file.Chmod(0o755) //nolint:gosec // git hooks must be executable
	}
	if err := fsx.WriteFileAtomicRoot(root, name, out, 0o755); err != nil {
		return HookUnchanged, err
	}
	return HookWritten, nil
}
