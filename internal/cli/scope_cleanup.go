package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/config"
)

func scopePurge(env Env, p *project, arg string) error {
	rel, err := purgeRelPath(arg)
	if err != nil {
		return err
	}
	for _, s := range p.scopes {
		if s.Path == rel {
			return blocked("%s is still registered as scope %q", rel, s.Name).
				then("envbuckets scope rm %s --purge, or envbuckets scope rm %s first", s.Name, s.Name)
		}
	}
	s := p.newScope(scopeNameFor(rel), rel)
	if err := checkPurgeTarget(p.root, s); err != nil {
		return err
	}
	return purgeScopeData(env, p.root, s)
}

func purgeRelPath(arg string) (string, error) {
	if arg == "" || filepath.IsAbs(arg) || filepath.VolumeName(arg) != "" || strings.HasPrefix(filepath.ToSlash(arg), "/") {
		return "", blocked("scope purge: %q is not a repo-relative path", arg).then("envbuckets scope purge apps/web")
	}
	rel := config.CleanScopePath(arg)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", blocked("scope purge: %s resolves outside the repo", arg).then("pass a path relative to the repo root")
	}
	for _, part := range strings.Split(rel, "/") {
		if part == bucketsDir || part == ".git" {
			return "", blocked("scope purge: %s points inside %s, pass the scope directory that contains %s/", arg, part, bucketsDir).
				then("envbuckets scope purge apps/web")
		}
	}
	return rel, nil
}

func checkPurgeTarget(root string, s scope) error {
	resolved, err := filepath.EvalSymlinks(s.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if got, err := relPath(root, resolved); err != nil || got != s.Path {
		return blocked("%s resolves through a symlink to %s, refusing to delete there", s.Path, filepath.ToSlash(resolved)).
			then("remove %s by hand if it is really meant to go", s.display(bucketsDir)+"/")
	}
	info, err := os.Lstat(s.bucketsPath())
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return err
	case info.Mode()&os.ModeSymlink != 0:
		return blocked("%s/ is a symlink, refusing to delete through it", s.display(bucketsDir)).then("remove that symlink by hand")
	case !info.IsDir():
		return blocked("%s is not a directory, refusing to delete it", s.display(bucketsDir)).then("inspect it by hand")
	}
	return nil
}

func purgeScopeData(env Env, root string, s scope) error {
	_, err := os.Lstat(s.bucketsPath())
	hasData := err == nil
	ls, err := s.linkState()
	if err != nil {
		return err
	}
	managed := ls.kind == linkBucket
	envDisplay := s.display(envFile)
	keepsEnv := ls.kind == linkReal || ls.kind == linkForeign
	lines := purgeIgnoreLines(s, keepsEnv)

	if !hasData && !managed {
		changed, err := dropIgnored(root, lines)
		if err != nil {
			return err
		}
		fmt.Fprintf(env.Stdout, "nothing to delete under %s\n", s.Path)
		if changed {
			fmt.Fprintf(env.Stdout, "removed: its stale .gitignore lines\n")
		}
		return nil
	}

	fmt.Fprintln(env.Stdout, "Purge will delete:")
	if hasData {
		names, err := s.buckets()
		if err != nil {
			return err
		}
		fmt.Fprintf(env.Stdout, "  %s/ (buckets: %s)\n", s.display(bucketsDir), joinOr(names, "none"))
	}
	if managed {
		fmt.Fprintf(env.Stdout, "  %s (symlink -> %s, would dangle)\n", envDisplay, ls.target)
	}
	if keepsEnv {
		fmt.Fprintf(env.Stdout, "kept: %s is not managed by envbuckets, left untouched and still ignored\n", envDisplay)
	}
	if err := confirmDelete(env, "the files listed above"); err != nil {
		return err
	}
	if err := checkPurgeTarget(root, s); err != nil {
		return err
	}
	repo, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer repo.Close()
	if managed {
		if now, err := s.linkState(); err == nil && now.kind == linkBucket {
			if err := repo.Remove(filepath.Join(filepath.FromSlash(s.Path), envFile)); err != nil {
				return err
			}
		}
	}
	if hasData {
		if err := repo.RemoveAll(filepath.Join(filepath.FromSlash(s.Path), bucketsDir)); err != nil {
			return fmt.Errorf("delete %s/: %w (rerun envbuckets scope purge %s)", s.display(bucketsDir), err, s.Path)
		}
	}
	if _, err := dropIgnored(root, lines); err != nil {
		return err
	}
	fmt.Fprintf(env.Stdout, "removed: %s\n", strings.Join(removedItems(s, hasData, managed, len(lines) > 0), ", "))
	return nil
}

func purgeIgnoreLines(s scope, keepsEnv bool) []string {
	var out []string
	for _, l := range scopeIgnoreLines(s) {
		if keepsEnv && l == s.Path+"/"+envFile {
			continue
		}
		out = append(out, l)
	}
	return out
}

func removedItems(s scope, hasData, managed, lines bool) []string {
	var out []string
	if hasData {
		out = append(out, s.display(bucketsDir)+"/")
	}
	if managed {
		out = append(out, s.display(envFile))
	}
	if lines {
		out = append(out, "its .gitignore lines")
	}
	return out
}
