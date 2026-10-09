package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/block"
	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/gitx"
	"github.com/sanketvgh/envbuckets/internal/switcher"
)

const initialConfig = "{\n  \"$schema\": \"https://raw.githubusercontent.com/sanketvgh/envbuckets/main/schema/envbuckets.schema.json\",\n  \"default\": \"dev\",\n  \"rules\": []\n}\n"

type initPlan struct {
	bucket       string
	createConfig bool
	createBucket bool
	files        []importFile
	warnings     []string
	ignore       block.IgnorePlan
	hookPath     string
	hookChange   bool
}

func planInit(repo *fsx.Repo) (initPlan, error) {
	p := initPlan{bucket: "dev"}
	if _, err := repo.Root.Lstat(".envbuckets.json"); os.IsNotExist(err) {
		p.createConfig = true
	} else {
		cfg, err := config.Load(repo.Root)
		if err != nil {
			return p, fmt.Errorf("invalid %w", err)
		}
		p.bucket = cfg.Default
	}
	exists, err := bucketDirectory(repo, p.bucket)
	if err != nil {
		return p, err
	}
	p.createBucket = !exists
	if _, err := repo.Root.Lstat(".envbuckets.toml"); err == nil {
		p.warnings = append(p.warnings, ".envbuckets.toml is unused and can be deleted")
	} else if !os.IsNotExist(err) {
		return p, err
	}
	candidates, err := gitx.LocalFiles(repo.Path)
	if err != nil {
		return p, err
	}
	slices.Sort(candidates)
	candidates = slices.Compact(candidates)
	var names []string
	for _, name := range candidates {
		reserved := false
		for part := range strings.SplitSeq(name, "/") {
			if strings.EqualFold(part, ".git") || strings.EqualFold(part, ".env.d") || strings.EqualFold(part, "GIT~1") {
				reserved = true
				break
			}
		}
		if reserved {
			continue
		}
		base := filepath.Base(filepath.FromSlash(name))
		envFile := base == ".env" || strings.HasPrefix(base, ".env.")
		// Git may list existing managed links when their ignore block is absent.
		// Rerunning init must leave all links, including foreign ones, untouched.
		if info, err := repo.Root.Lstat(name); err == nil && info.Mode()&os.ModeSymlink != 0 {
			target, linkErr := repo.Root.Readlink(name)
			if linkErr == nil && switcher.BucketOf(name, filepath.ToSlash(target)) != "" {
				if err := repo.ValidatePath(filepath.FromSlash(name)); err == nil {
					names = append(names, name)
				}
			} else if envFile {
				p.warnings = append(p.warnings, fmt.Sprintf("cannot import '%s': it is a foreign symlink", name))
			}
			continue
		}
		if !envFile {
			continue
		}
		f, err := planImport(repo, p.bucket, name)
		if err != nil {
			p.warnings = append(p.warnings, fmt.Sprintf("cannot import '%s': %s", name, err))
			continue
		}
		p.files, names = append(p.files, f), append(names, name)
	}
	p.ignore, err = block.PlanIgnore(repo, names)
	if err != nil {
		return p, err
	}
	var hook block.HookResult
	p.hookPath, hook, err = block.PreviewRepoHook(repo.Path)
	if err != nil {
		return p, err
	}
	p.hookChange = hook == block.HookWritten
	return p, fsx.ProbeSymlinks(repo.Root)
}

func runInit(args []string, env Env) int {
	dry, paths, err := parseDryArgs(args)
	if err != nil || len(paths) != 0 {
		return commandUsage(env, "init [-n]", err)
	}
	repo, err := openCommandRepo(env)
	if err != nil {
		return switchFailure(env, false, err)
	}
	defer repo.Close()
	p, err := planInit(repo)
	for _, warning := range p.warnings {
		fmt.Fprintf(env.Stderr, "warning: %s\n", warning)
	}
	if err != nil {
		return switchFailure(env, false, err)
	}
	if dry {
		if p.createConfig {
			fmt.Fprintln(env.Stdout, "Would create .envbuckets.json")
		}
		if p.createBucket {
			fmt.Fprintf(env.Stdout, "Would create bucket '%s'\n", p.bucket)
		}
		for _, f := range p.files {
			fmt.Fprintf(env.Stdout, "Would add %s to bucket '%s'\n", f.path, p.bucket)
		}
		if p.ignore.Changed {
			fmt.Fprintln(env.Stdout, "Would update .gitignore")
		}
		if p.hookChange {
			fmt.Fprintf(env.Stdout, "Would install %s\n", p.hookPath)
		}
		return ExitOK
	}
	if p.createConfig {
		// O_EXCL avoids replacing a config that appeared after planning.
		f, err := repo.Root.OpenFile(".envbuckets.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return switchFailure(env, false, err)
		}
		_, writeErr := f.WriteString(initialConfig)
		closeErr := f.Close()
		if writeErr != nil {
			return switchFailure(env, false, writeErr)
		}
		if closeErr != nil {
			return switchFailure(env, false, closeErr)
		}
	}
	if _, err := bucketDirectory(repo, p.bucket); err != nil {
		return switchFailure(env, false, err)
	}
	if err := repo.Root.MkdirAll(".env.d/"+p.bucket, 0o755); err != nil {
		return switchFailure(env, false, err)
	}
	if err := p.ignore.Apply(repo); err != nil {
		return switchFailure(env, false, err)
	}
	if _, err := block.InstallRepoHook(repo.Path); err != nil {
		return switchFailure(env, false, err)
	}
	for _, f := range p.files {
		if err := f.apply(repo); err != nil {
			fmt.Fprintf(env.Stderr, "warning: cannot import '%s': %s\n", f.path, err)
			continue
		}
		fmt.Fprintf(env.Stdout, "Adding %s to bucket '%s'\n", f.path, p.bucket)
	}
	fmt.Fprintf(env.Stdout, "Initialized envbuckets in %s/.env.d/\n", filepath.ToSlash(repo.Path))
	if len(p.files) == 0 {
		fmt.Fprintln(env.Stderr, "hint: No local files found. Create them, then run \"envbuckets add <file>\".")
	}
	return ExitOK
}
