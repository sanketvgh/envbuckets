package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/block"
	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/gitx"
)

func runUninstall(args []string, env Env) error {
	flags := newFlags("uninstall")
	purge := flags.Bool("purge", false, "also delete .env.d/ dirs, the config, and the .gitignore block (asks for DELETE)")
	if _, err := parseFlags(flags, args); err != nil {
		return err
	}
	root, err := repoRoot(env)
	if err != nil {
		return err
	}
	step := func(tag, format string, a ...any) {
		fmt.Fprintf(env.Stdout, "  [%s] %s\n", tag, fmt.Sprintf(format, a...))
	}
	fmt.Fprintln(env.Stdout, "envbuckets: deactivating in this project")

	found, err := discoverScopes(root)
	if err != nil {
		return err
	}
	for _, s := range found {
		ls, err := s.linkState()
		if err != nil {
			fmt.Fprintf(env.Stderr, "warning: %s: %v\n", s.display(envFile), err)
			continue
		}
		switch {
		case ls.kind == linkMissing:
			step("skipped", "%s (absent)", s.display(envFile))
		case ls.kind == linkReal:
			step("skipped", "%s (already a real file)", s.display(envFile))
		case ls.dangling:
			fmt.Fprintf(env.Stderr, "warning: %s -> %s is BROKEN, left as-is\n  next: fix the target or remove the symlink by hand, then re-run\n", s.display(envFile), ls.target)
			step("skipped", "%s (broken symlink)", s.display(envFile))
		default:
			if err := materializeScope(s, ls); err != nil {
				fmt.Fprintf(env.Stderr, "warning: %s: %v\n", s.display(envFile), err)
				step("skipped", "%s (materialize failed)", s.display(envFile))
				continue
			}
			step("materialized", "%s (was -> %s)", s.display(envFile), ls.target)
		}
	}

	hooksDir, err := gitx.HooksDir(root)
	if err != nil {
		return envErr("cannot locate the git hooks directory: %v", err).then("check git rev-parse --git-path hooks")
	}
	hookPath := filepath.Join(hooksDir, "post-checkout")
	switch res, err := block.RemoveHook(hookPath); {
	case err != nil:
		return err
	case res == block.HookDeleted:
		step("removed", "hook (%s, contained only our block)", relOrAbs(root, hookPath))
	case res == block.HookWritten:
		step("removed", "hook block (%s, custom code preserved)", relOrAbs(root, hookPath))
	default:
		step("skipped", "hook block (not installed)")
	}

	dataDirs, err := discoverBucketDirs(root)
	if err != nil {
		return err
	}
	if !*purge {
		if block.Body(mustRead(gitignorePath(root))) != nil {
			step("kept", ".gitignore block (values remain in %s/ dirs)", bucketsDir)
		} else {
			step("skipped", ".gitignore block (not installed)")
		}
		if _, err := os.Stat(config.Path(root)); err == nil {
			step("kept", config.FileName)
		}
		if len(dataDirs) > 0 {
			step("kept", "%s/ dirs (%d scopes, %d buckets)", bucketsDir, len(dataDirs), countBuckets(dataDirs))
		}
		fmt.Fprintln(env.Stdout, "Data kept (.env.d/ and the config)\n  next: reactivate with envbuckets init, or erase everything with envbuckets uninstall --purge")
		return nil
	}

	if len(dataDirs) > 0 {
		fmt.Fprintln(env.Stdout, "Purge will delete:")
		for _, d := range dataDirs {
			fmt.Fprintf(env.Stdout, "  %s (buckets: %s)\n", relOrAbs(root, d.path), joinOr(d.buckets, "none"))
		}
	}
	if _, err := os.Stat(config.Path(root)); err == nil {
		fmt.Fprintf(env.Stdout, "  %s\n", config.FileName)
	}
	if len(dataDirs) > 0 || fileExists(config.Path(root)) {
		if err := confirmDelete(env, "all bucket data and the config"); err != nil {
			return err
		}
	}
	for _, d := range dataDirs {
		if err := os.RemoveAll(d.path); err != nil {
			return err
		}
		step("removed", "%s/", relOrAbs(root, d.path))
	}
	if err := os.Remove(config.Path(root)); err == nil {
		step("removed", config.FileName)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	switch changed, err := removeIgnoreBlock(root); {
	case err != nil:
		return err
	case changed:
		step("removed", ".gitignore block")
	default:
		step("skipped", ".gitignore block (not installed)")
	}
	return nil
}

func materializeScope(s scope, ls linkState) error {
	target := filepath.Join(s.Dir, filepath.FromSlash(ls.target))
	if filepath.IsAbs(ls.target) {
		target = filepath.FromSlash(ls.target)
	}
	return fsx.CopyFileAtomic(target, s.envPath(), s.bucketsPath(), 0o600)
}

func discoverScopes(root string) ([]scope, error) {
	seen := map[string]bool{}
	var out []scope
	add := func(dir string) {
		if seen[dir] {
			return
		}
		seen[dir] = true
		rel, _ := filepath.Rel(root, dir)
		rel = filepath.ToSlash(rel)
		out = append(out, scope{Name: scopeNameFor(rel), Path: rel, Dir: dir})
	}
	err := walkProject(root, func(p string, d fs.DirEntry) error {
		switch {
		case d.Name() == bucketsDir && d.IsDir():
			add(filepath.Dir(p))
		case d.Name() == envFile && d.Type()&fs.ModeSymlink != 0:
			target, err := os.Readlink(p)
			if err == nil && strings.Contains(filepath.ToSlash(target), bucketsDir+"/") {
				add(filepath.Dir(p))
			}
		}
		return nil
	})
	sortByDepth(out, func(s scope) string { return s.Path })
	return out, err
}

type bucketDir struct {
	path    string
	buckets []string
}

func discoverBucketDirs(root string) ([]bucketDir, error) {
	var out []bucketDir
	err := walkProject(root, func(p string, d fs.DirEntry) error {
		if d.Name() != bucketsDir || !d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, filepath.Dir(p))
		s := scope{Path: filepath.ToSlash(rel), Dir: filepath.Dir(p)}
		names, err := s.buckets()
		if err != nil {
			return err
		}
		out = append(out, bucketDir{path: p, buckets: names})
		return nil
	})
	sortByDepth(out, func(b bucketDir) string { return b.path })
	return out, err
}

func walkProject(root string, visit func(string, fs.DirEntry) error) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if p == root {
			return nil
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if err := visit(p, d); err != nil {
			return err
		}
		if d.IsDir() && d.Name() == bucketsDir {
			return filepath.SkipDir
		}
		return nil
	})
}

func sortByDepth[T any](items []T, key func(T) string) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := key(items[i]), key(items[j])
		da, db := strings.Count(a, "/"), strings.Count(b, "/")
		if a == "." {
			da = -1
		}
		if b == "." {
			db = -1
		}
		if da != db {
			return da < db
		}
		return a < b
	})
}

func scopeNameFor(rel string) string {
	if rel == "." {
		return "root"
	}
	return filepath.Base(rel)
}

func countBuckets(dirs []bucketDir) int {
	n := 0
	for _, d := range dirs {
		n += len(d.buckets)
	}
	return n
}

func mustRead(p string) []byte {
	data, _ := os.ReadFile(p)
	return data
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
