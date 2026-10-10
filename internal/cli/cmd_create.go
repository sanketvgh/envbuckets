package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/switcher"
)

func createBucket(env Env, bucket string, dry bool) int {
	if !config.ValidBucket(bucket) {
		return switchFailure(env, false, fmt.Errorf("invalid bucket name '%s'", bucket))
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
	base := ".env.d/" + bucket
	// Existing symlinks and files also count as existing buckets.
	if _, err := repo.Root.Lstat(base); err == nil {
		return switchFailure(env, false, fmt.Errorf("a bucket named '%s' already exists", bucket))
	} else if !os.IsNotExist(err) {
		return switchFailure(env, false, err)
	}
	if err := fsx.ValidateInternalPath(repo.Root, base); err != nil {
		return switchFailure(env, false, err)
	}
	current, err := switcher.CurrentBucket(repo, cfg, branch)
	if err != nil {
		return switchFailure(env, false, err)
	}
	files, unsafe, err := repo.ScanBucket(current)
	if err != nil {
		return switchFailure(env, false, err)
	}
	if len(unsafe) != 0 {
		return switchFailure(env, false, fmt.Errorf("unsafe entry in current bucket: %s", unsafe[0]))
	}
	// Use a virtual target to get the exact switch preflight without writing it.
	p := switcher.BuildFiles(repo, cfg, branch, bucket, files)
	if p.ExitCode() != ExitOK {
		return switcher.Execute(repo, p, true, false, env.Stdout, env.Stderr)
	}
	if err := fsx.ProbeSymlinks(repo.Root); err != nil {
		return switchFailure(env, false, err)
	}
	if dry {
		env.output(false).List("Would create bucket '%s'\n", bucket)
		for _, name := range files {
			env.output(false).List("Would create empty %s/%s\n", base, name)
		}
		return switcher.Execute(repo, p, true, false, env.Stdout, env.Stderr)
	}
	if err := repo.Root.Mkdir(base, 0o755); err != nil {
		return switchFailure(env, false, err)
	}
	for _, name := range files {
		path := base + "/" + name
		if err := fsx.ValidateInternalPath(repo.Root, path); err != nil {
			return switchFailure(env, false, err)
		}
		if err := repo.Root.MkdirAll(filepath.ToSlash(filepath.Dir(filepath.FromSlash(path))), 0o755); err != nil {
			return switchFailure(env, false, err)
		}
		f, err := repo.Root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return switchFailure(env, false, err)
		}
		if err := f.Close(); err != nil {
			return switchFailure(env, false, err)
		}
	}
	// Execute the checked plan while replacing its standard success line.
	code := switcher.Execute(repo, p, false, false, io.Discard, env.Stderr)
	if code == ExitOK {
		env.output(false).List("Switched to a new bucket '%s'\n", bucket)
	}
	return code
}
