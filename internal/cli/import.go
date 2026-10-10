package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/gitx"
)

type importFile struct {
	path   string
	dest   string
	target string
}

func openCommandRepo(env Env) (*fsx.Repo, error) {
	root, err := gitx.Root(env.Cwd)
	if err != nil {
		return nil, err
	}
	return fsx.OpenRepo(root)
}

func bucketDirectory(repo *fsx.Repo, bucket string) (bool, error) {
	name := ".env.d/" + bucket
	if err := fsx.ValidateInternalPath(repo.Root, name); err != nil {
		return false, err
	}
	info, err := repo.Root.Lstat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("unsafe bucket directory '%s'", name)
	}
	return true, nil
}

func planImport(repo *fsx.Repo, bucket, name string) (importFile, error) {
	f := importFile{path: name, dest: ".env.d/" + bucket + "/" + name}
	if name == ".envbuckets.json" || name == ".envbuckets.toml" || name == ".gitignore" || strings.ContainsAny(name, "\r\n") {
		return f, errors.New("path is repository metadata or cannot be represented in .gitignore")
	}
	if err := repo.ValidatePath(filepath.FromSlash(name)); err != nil {
		return f, err
	}
	info, err := repo.Root.Lstat(name)
	if err != nil {
		return f, err
	}
	if !info.Mode().IsRegular() {
		return f, errors.New("it is not a regular file (directories and links cannot be added)")
	}
	if err := fsx.ValidateInternalPath(repo.Root, f.dest); err != nil {
		return f, err
	}
	if _, err := repo.Root.Lstat(f.dest); err == nil {
		return f, fmt.Errorf("bucket '%s' already contains this path", bucket)
	} else if !os.IsNotExist(err) {
		return f, err
	}
	target, err := filepath.Rel(filepath.Dir(filepath.FromSlash(name)), filepath.FromSlash(f.dest))
	if err != nil {
		return f, err
	}
	f.target = filepath.ToSlash(target)
	return f, nil
}

func (f importFile) apply(repo *fsx.Repo) error {
	if err := repo.ValidatePath(filepath.FromSlash(f.path)); err != nil {
		return err
	}
	if err := fsx.ValidateInternalPath(repo.Root, f.dest); err != nil {
		return err
	}
	if err := repo.Root.MkdirAll(filepath.ToSlash(filepath.Dir(filepath.FromSlash(f.dest))), 0o755); err != nil {
		return err
	}
	_, err := fsx.MoveFileToBucket(repo.Root, f.path, f.dest, f.target)
	return err
}

func loadCurrent(repo *fsx.Repo) (config.Config, string, error) {
	cfg, err := config.Load(repo.Root)
	if err != nil {
		return cfg, "", fmt.Errorf("invalid %w", err)
	}
	branch, err := gitx.Branch(repo.Path)
	return cfg, branch, err
}

func parseDryArgs(args []string) (bool, []string, error) {
	var paths []string
	dry, positional := false, false
	for _, arg := range args {
		switch {
		case !positional && (arg == "-n" || arg == "--dry-run"):
			dry = true
		case !positional && arg == "--":
			positional = true
		case !positional && strings.HasPrefix(arg, "-"):
			return false, nil, fmt.Errorf("unknown option '%s'", arg)
		default:
			paths = append(paths, arg)
		}
	}
	return dry, paths, nil
}

func commandUsage(env Env, command string, err error) int {
	if err != nil {
		env.output(false).Error("%s", err)
	}
	env.output(false).Usage("envbuckets %s", command)
	return ExitUsage
}
