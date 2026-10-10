package cli

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/block"
	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/switcher"
)

func runAdd(args []string, env Env) int {
	dry, paths, err := parseDryArgs(args)
	if err != nil || len(paths) == 0 {
		return commandUsage(env, "add [-n] <file>...", err)
	}
	repo, err := openCommandRepo(env)
	if err != nil {
		return switchFailure(env, false, err)
	}
	defer repo.Close()
	cfg, branch, err := loadCurrent(repo)
	if err != nil {
		return switchFailure(env, false, err)
	}
	bucket, err := switcher.CurrentBucket(repo, cfg, branch)
	if err != nil {
		return switchFailure(env, false, err)
	}
	if exists, err := bucketDirectory(repo, bucket); err != nil {
		return switchFailure(env, false, err)
	} else if !exists {
		return switchFailure(env, false, fmt.Errorf("no bucket named '%s'; create it with \"envbuckets switch -c %s\"", bucket, bucket))
	}
	files := make([]importFile, 0, len(paths))
	names := make([]string, 0, len(paths))
	seen := make(map[string]bool)
	for _, arg := range paths {
		abs := arg
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(env.Cwd, arg)
		}
		name, err := filepath.Rel(repo.Path, abs)
		if err != nil || arg == "" {
			return switchFailure(env, false, fmt.Errorf("cannot add '%s': invalid path", arg))
		}
		name = filepath.ToSlash(name)
		key := name
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		f, err := planImport(repo, bucket, name)
		if err != nil {
			if strings.Contains(err.Error(), "is tracked by Git") {
				env.output(false).Fatal("cannot add '%s': it is tracked by Git", name)
				env.output(false).Hint("Copy it to an untracked file such as '.env', then add that.")
				return ExitError
			}
			return switchFailure(env, false, fmt.Errorf("cannot add '%s': %w", name, err))
		}
		files, names = append(files, f), append(names, name)
	}
	ignore, err := block.PlanIgnore(repo, names)
	if err != nil {
		return switchFailure(env, false, err)
	}
	if err := fsx.ProbeSymlinks(repo.Root); err != nil {
		return switchFailure(env, false, err)
	}
	if dry {
		for _, f := range files {
			env.output(false).List("Would add %s to bucket '%s'\n", f.path, bucket)
		}
		if ignore.Changed {
			env.output(false).Would("update .gitignore")
		}
		return ExitOK
	}
	// Write the ignore block before importing, so a write failure leaves every
	// source intact and an imported file is never exposed to an accidental add.
	if err := ignore.Apply(repo); err != nil {
		return switchFailure(env, false, err)
	}
	for _, f := range files {
		if err := f.apply(repo); err != nil {
			return switchFailure(env, false, fmt.Errorf("cannot add '%s': %w", f.path, err))
		}
	}
	return ExitOK
}
