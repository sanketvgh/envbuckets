package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/block"
	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/switcher"
)

func runUninstall(args []string, env Env) int {
	dry, paths, err := parseDryArgs(args)
	if err != nil || len(paths) != 0 {
		return commandUsage(env, "uninstall [-n]", err)
	}
	repo, err := openCommandRepo(env)
	if err != nil {
		return switchFailure(env, false, err)
	}
	defer repo.Close()
	names, scanErrors := uninstallCandidates(repo)
	code := ExitOK
	for _, err := range scanErrors {
		fmt.Fprintf(env.Stderr, "error: cannot inspect buckets: %s\n", err)
		code = ExitError
	}
	changed := false
	for _, name := range names {
		source, target, err := uninstallSource(repo, name)
		if err != nil {
			fmt.Fprintf(env.Stderr, "error: cannot restore '%s': %s\n", name, err)
			code = ExitError
			continue
		}
		if source == "" {
			continue
		}
		if dry {
			fmt.Fprintf(env.Stdout, "Would move %s to %s\n", source, name)
			continue
		}
		if err := fsx.RestoreBucketFile(repo, name, source, target); err != nil {
			fmt.Fprintf(env.Stderr, "error: cannot restore '%s': %s\n", name, err)
			code = ExitError
			continue
		}
		fmt.Fprintf(env.Stdout, "Moving %s to %s\n", source, name)
		changed = true
	}
	path, result, err := block.RemoveRepoHook(repo.Path, dry)
	if err != nil {
		fmt.Fprintf(env.Stderr, "error: cannot remove envbuckets hook: %s\n", err)
		code = ExitError
	} else if result != block.HookUnchanged {
		if dry {
			fmt.Fprintf(env.Stdout, "Would remove the envbuckets hook from %s\n", path)
		} else {
			fmt.Fprintf(env.Stdout, "Removing the envbuckets hook from %s\n", path)
			changed = true
		}
	}
	if changed {
		fmt.Fprintln(env.Stderr, "hint: Your other buckets are still in .env.d/. Delete it when you no longer need them.")
	}
	return code
}

// Candidate paths come only from bucket entries. No repository walk, config
// read, or target file read is needed, including for a missing active target
// whose path remains in another bucket.
func uninstallCandidates(repo *fsx.Repo) ([]string, []error) {
	if err := fsx.ValidateInternalPath(repo.Root, ".env.d"); err != nil {
		return nil, []error{err}
	}
	info, err := repo.Root.Lstat(".env.d")
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, []error{err}
	}
	if !info.IsDir() {
		return nil, []error{errors.New(".env.d is not a directory")}
	}
	seen := make(map[string]bool)
	var failures []error
	err = fs.WalkDir(repo.Root.FS(), ".env.d", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			failures = append(failures, err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		parts := strings.SplitN(name, "/", 3)
		if len(parts) == 1 {
			return nil
		}
		if !config.ValidBucket(parts[1]) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if len(parts) == 2 {
			if d.Type()&os.ModeSymlink != 0 {
				failures = append(failures, fmt.Errorf("unsafe bucket directory '%s'", name))
			}
			return nil
		}
		if !d.IsDir() {
			seen[parts[2]] = true
		}
		return nil
	})
	if err != nil {
		failures = append(failures, err)
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	slices.Sort(names)
	return names, failures
}

func uninstallSource(repo *fsx.Repo, name string) (source, target string, err error) {
	if err := repo.ValidatePath(filepath.FromSlash(name)); err != nil {
		return "", "", err
	}
	info, err := repo.Root.Lstat(name)
	if os.IsNotExist(err) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return "", "", nil
	}
	target, err = repo.Root.Readlink(name)
	if err != nil {
		return "", "", err
	}
	target = filepath.ToSlash(target)
	bucket := switcher.BucketOf(name, target)
	if bucket == "" {
		return "", "", nil
	}
	source = ".env.d/" + bucket + "/" + name
	if err := fsx.ValidateInternalPath(repo.Root, source); err != nil {
		return "", "", err
	}
	info, err = repo.Root.Lstat(source)
	if os.IsNotExist(err) {
		return "", "", errors.New("bucket target is missing (broken link)")
	}
	if err != nil {
		return "", "", err
	}
	if !info.Mode().IsRegular() {
		return "", "", errors.New("bucket target is not a regular file")
	}
	return source, target, nil
}
