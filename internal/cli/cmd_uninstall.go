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
	hooksDir, err := gitx.HooksDir(root)
	if err != nil {
		return envErr("cannot use the git hooks directory: %v", err)
	}
	ignoreData, err := readGitignore(root)
	if err != nil {
		return envErr("cannot use .gitignore: %v", err)
	}
	step := func(tag, format string, a ...any) {
		fmt.Fprintf(env.Stdout, "  %s: %s\n", tag, fmt.Sprintf(format, a...))
	}
	fmt.Fprintln(env.Stdout, "Uninstalling envbuckets in this project")

	found, err := discoverScopes(root)
	if err != nil {
		return err
	}
	if *purge {
		if err := confirmUninstallPurge(env, root, found); err != nil {
			return err
		}
	}
	retainedLink := false
	for _, s := range found {
		ls, err := s.linkState()
		if err != nil {
			warnf(env, "%s: %v", s.display(envFile), err)
			retainedLink = true
			continue
		}
		switch {
		case ls.kind == linkMissing:
			step("skipped", "%s (absent)", s.display(envFile))
		case ls.kind == linkReal:
			step("skipped", "%s (already a real file)", s.display(envFile))
		case ls.kind == linkForeign:
			warnf(env, "%s -> %s is a foreign symlink, left as-is", s.display(envFile), ls.target)
			step("skipped", "%s (foreign symlink)", s.display(envFile))
			retainedLink = true
		case ls.dangling:
			warnf(env, "%s -> %s is BROKEN, left as-is", s.display(envFile), ls.target)
			step("skipped", "%s (broken symlink)", s.display(envFile))
			retainedLink = true
		default:
			if !s.bucketFileExists(ls.bucket) {
				warnf(env, "%s does not point to a regular in-repo bucket file, left as-is", s.display(envFile))
				step("skipped", "%s (unsafe bucket file)", s.display(envFile))
				retainedLink = true
				continue
			}
			if err := materializeScope(s, ls); err != nil {
				warnf(env, "%s: %v", s.display(envFile), err)
				step("skipped", "%s (materialize failed)", s.display(envFile))
				retainedLink = true
				continue
			}
			step("materialized", "%s (was -> %s)", s.display(envFile), ls.target)
		}
	}
	if *purge && retainedLink {
		return blocked("some .env symlinks could not be materialized; purge stopped so their ignore protection stays in place").
			then("fix the warnings above, then re-run envbuckets uninstall --purge")
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
	links, err := gitx.Links(root)
	if err != nil {
		return envErr("cannot read branch links: %v", err).then("check git config --local --list")
	}
	if !*purge {
		if block.Body(ignoreData) != nil {
			step("kept", ".gitignore block (values remain in %s/ dirs)", bucketsDir)
		} else {
			step("skipped", ".gitignore block (not installed)")
		}
		if _, err := os.Stat(config.Path(root)); err == nil {
			step("kept", config.FileName)
		}
		if len(links) > 0 {
			step("kept", "branch links (%d in .git/config)", len(links))
		}
		if len(dataDirs) > 0 {
			step("kept", "%s/ dirs (%d scopes, %d bucket directories)", bucketsDir, len(dataDirs), countBuckets(dataDirs))
		}
		fmt.Fprintln(env.Stdout, "Data kept (active .env, remaining .env.d/ files, and config)")
		return nil
	}

	repo, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer repo.Close()
	for _, d := range dataDirs {
		rel, err := filepath.Rel(root, d.path)
		if err != nil {
			return err
		}
		if err := repo.RemoveAll(rel); err != nil {
			return err
		}
		step("removed", "%s/", relOrAbs(root, d.path))
	}
	if err := os.Remove(config.Path(root)); err == nil {
		step("removed", config.FileName)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, l := range links {
		if _, err := gitx.Unlink(root, l.Branch); err != nil {
			return envErr("cannot remove the link of %s: %v", l.Branch, err).then("git config --local --unset branch.%s.envbuckets", l.Branch)
		}
	}
	if len(links) > 0 {
		step("removed", "branch links (%d in .git/config)", len(links))
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

func confirmUninstallPurge(env Env, root string, scopes []scope) error {
	for _, s := range scopes {
		ls, err := s.linkState()
		if err != nil {
			return blocked("cannot inspect %s before purge: %v", s.display(envFile), err)
		}
		if ls.kind == linkForeign || ls.dangling || (ls.kind == linkBucket && !s.bucketFileExists(ls.bucket)) {
			return blocked("%s cannot be safely restored; purge stopped", s.display(envFile))
		}
	}
	dataDirs, err := discoverBucketDirs(root)
	if err != nil {
		return err
	}
	links, err := gitx.Links(root)
	if err != nil {
		return envErr("cannot read branch links: %v", err)
	}
	fmt.Fprintln(env.Stdout, "Purge will delete:")
	for _, d := range dataDirs {
		fmt.Fprintf(env.Stdout, "  %s (buckets: %s)\n", relOrAbs(root, d.path), joinOr(d.buckets, "none"))
	}
	if fileExists(config.Path(root)) {
		fmt.Fprintf(env.Stdout, "  %s\n", config.FileName)
	}
	for _, l := range links {
		fmt.Fprintf(env.Stdout, "  link %s -> %s (.git/config)\n", l.Branch, l.Bucket)
	}
	return confirmDelete(env, "all bucket data and the config")
}

func materializeScope(s scope, ls linkState) error {
	if err := s.checkBucketParents(ls.bucket); err != nil {
		return err
	}
	root, err := os.OpenRoot(s.Root)
	if err != nil {
		return err
	}
	defer root.Close()
	dst := filepath.Join(filepath.FromSlash(s.Path), envFile)
	info, err := root.Lstat(dst)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return blocked("%s changed from a managed symlink, left untouched", s.display(envFile))
	}
	target, err := root.Readlink(dst)
	if err != nil {
		return err
	}
	if filepath.ToSlash(target) != ls.target {
		return blocked("%s changed from its inspected target, left untouched", s.display(envFile))
	}
	return root.Rename(filepath.Join(s.bucketRel(ls.bucket), envFile), dst)
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
		out = append(out, scope{Name: scopeNameFor(rel), Path: rel, Dir: dir, Root: root})
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
		s := scope{Path: filepath.ToSlash(rel), Dir: filepath.Dir(p), Root: root}
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

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
