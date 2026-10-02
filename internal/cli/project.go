package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/gitx"
)

const (
	envFile    = ".env"
	bucketsDir = ".env.d"
)

type scope struct {
	Name string
	Path string
	Dir  string
	Root string
}

type project struct {
	root   string
	cfg    *config.Config
	scopes []scope
}

func repoRoot(env Env) (string, error) {
	root, err := gitx.Root(env.Cwd)
	if err != nil {
		return "", envErr("%s is not inside a git repository", env.Cwd).then("cd into the repo, or git init")
	}
	return root, nil
}

func openProject(env Env) (*project, error) {
	root, err := repoRoot(env)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return nil, configError(err)
	}
	return newProject(root, cfg), nil
}

func configError(err error) error {
	if errors.Is(err, config.ErrMissing) {
		return configErr("envbuckets is not initialized in this repo").then("envbuckets init")
	}
	return configErr("%s is unreadable, refusing to act: %v", config.FileName, err).
		then("restore %s from git (git checkout -- %s), or escape with: envbuckets uninstall", config.FileName, config.FileName)
}

func newProject(root string, cfg *config.Config) *project {
	p := &project{root: root, cfg: cfg}
	p.reloadScopes()
	return p
}

func (p *project) reloadScopes() {
	p.scopes = p.scopes[:0]
	for _, s := range p.cfg.EffectiveScopes() {
		p.scopes = append(p.scopes, p.newScope(s.Name, s.Path))
	}
}

func (p *project) newScope(name, rel string) scope {
	return scope{Name: name, Path: rel, Dir: filepath.Join(p.root, filepath.FromSlash(rel)), Root: p.root}
}

func (p *project) scopeByName(name string) (scope, bool) {
	for _, s := range p.scopes {
		if s.Name == name {
			return s, true
		}
	}
	return scope{}, false
}

func (p *project) bucketExists(name string) bool {
	for _, s := range p.scopes {
		if s.bucketFileExists(name) {
			return true
		}
	}
	return false
}

func (p *project) resolveScope(env Env, name string) (scope, error) {
	if name != "" {
		s, ok := p.scopeByName(name)
		if !ok {
			return scope{}, envErr("no scope named %q", name).then("envbuckets scope list")
		}
		return s, nil
	}
	rel, err := relPath(p.root, env.Cwd)
	if err != nil {
		return scope{}, envErr("current directory is outside the repo").then("cd into the repo")
	}
	best, found := scope{}, false
	for _, s := range p.scopes {
		if !withinScope(s.Path, rel) {
			continue
		}
		if !found || len(s.Path) > len(best.Path) {
			best, found = s, true
		}
	}
	if !found {
		return scope{}, envErr("current directory is not inside any scope").then("pass --scope <name>, see: envbuckets scope list")
	}
	return best, nil
}

func (p *project) selectScopes(env Env, name string, all, defaultAll bool) ([]scope, error) {
	if all && name != "" {
		return nil, usage("--all and --scope cannot be used together")
	}
	if all || (defaultAll && name == "") {
		return append([]scope(nil), p.scopes...), nil
	}
	s, err := p.resolveScope(env, name)
	if err != nil {
		return nil, err
	}
	return []scope{s}, nil
}

func withinScope(scopePath, rel string) bool {
	if scopePath == "." {
		return true
	}
	return rel == scopePath || strings.HasPrefix(rel, scopePath+"/")
}

func relPath(root, dir string) (string, error) {
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		rootReal = root
	}
	dirReal, err := filepath.EvalSymlinks(dir)
	if err != nil {
		dirReal = dir
	}
	rel, err := filepath.Rel(rootReal, dirReal)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", errors.New("outside root")
	}
	return rel, nil
}

func (s scope) exists() bool {
	return s.checkScopeDir() == nil
}

// checkScopeDir rejects links in a registered scope path, including links in
// parent components. A configured scope must not redirect operations elsewhere.
func (s scope) checkScopeDir() error {
	rel, err := filepath.Rel(s.Root, s.Dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return blocked("scope %s resolves outside the repo", s.Path)
	}
	current := s.Root
	if rel == "." {
		info, err := os.Stat(current)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return blocked("repo root is not a directory")
		}
		return nil
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return blocked("scope path %s contains a symlink, refusing to follow it", s.Path)
		}
		if !info.IsDir() {
			return blocked("scope path %s is not a directory", s.Path)
		}
	}
	return nil
}

func (s scope) checkBucketParents(name string) error {
	if err := s.checkScopeDir(); err != nil {
		return err
	}
	paths := []string{s.bucketsPath()}
	if name != "" {
		paths = append(paths, s.bucketDir(name))
	}
	for _, p := range paths {
		info, err := os.Lstat(p)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return blocked("%s is a symlink, refusing to follow it", filepath.ToSlash(p))
		}
		if !info.IsDir() {
			return blocked("%s is not a directory", filepath.ToSlash(p))
		}
	}
	return nil
}

func (s scope) bucketRel(name string) string {
	return filepath.Join(filepath.FromSlash(s.Path), bucketsDir, name)
}

func (s scope) envPath() string {
	return filepath.Join(s.Dir, envFile)
}

func (s scope) bucketsPath() string {
	return filepath.Join(s.Dir, bucketsDir)
}

func (s scope) bucketDir(name string) string {
	return filepath.Join(s.Dir, bucketsDir, name)
}

func (s scope) bucketFile(name string) string {
	return filepath.Join(s.bucketDir(name), envFile)
}

func (s scope) display(name string) string {
	if s.Path == "." {
		return name
	}
	return s.Path + "/" + name
}

func linkTarget(bucket string) string {
	return bucketsDir + "/" + bucket + "/" + envFile
}

func bucketFromTarget(target string) (string, bool) {
	parts := strings.Split(filepath.ToSlash(target), "/")
	if len(parts) != 3 || parts[0] != bucketsDir || parts[2] != envFile || config.ValidateName(parts[1]) != nil {
		return "", false
	}
	return parts[1], true
}

func (s scope) bucketFileExists(name string) bool {
	if err := s.checkBucketParents(name); err != nil {
		return false
	}
	info, err := os.Lstat(s.bucketFile(name))
	return err == nil && info.Mode().IsRegular()
}

func (s scope) buckets() ([]string, error) {
	if err := s.checkBucketParents(""); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.bucketsPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && config.ValidateName(e.Name()) == nil {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

type linkKind int

const (
	linkMissing linkKind = iota
	linkReal
	linkForeign
	linkBucket
)

type linkState struct {
	kind     linkKind
	bucket   string
	target   string
	dangling bool
}

func (s scope) linkState() (linkState, error) {
	if err := s.checkScopeDir(); err != nil {
		return linkState{}, err
	}
	st, err := fsx.Inspect(s.envPath())
	if err != nil {
		return linkState{}, err
	}
	switch {
	case !st.Exists:
		return linkState{kind: linkMissing}, nil
	case !st.IsSymlink:
		return linkState{kind: linkReal}, nil
	}
	ls := linkState{kind: linkForeign, target: st.Target, dangling: st.Dangling}
	if b, ok := bucketFromTarget(st.Target); ok {
		ls.kind, ls.bucket = linkBucket, b
	}
	return ls, nil
}

func (s scope) pointTo(bucket string) (bool, error) {
	if err := s.checkBucketParents(bucket); err != nil {
		return false, err
	}
	if !s.bucketFileExists(bucket) {
		return false, blocked("%s is not a regular bucket file", s.display(linkTarget(bucket)))
	}
	root, err := os.OpenRoot(s.Root)
	if err != nil {
		return false, err
	}
	defer root.Close()
	return fsx.SetSymlinkRoot(root, filepath.Join(filepath.FromSlash(s.Path), envFile), linkTarget(bucket), s.bucketRel(""))
}

func readLine(env Env) (string, bool) {
	r, ok := env.Stdin.(*bufio.Reader)
	if !ok {
		r = bufio.NewReader(env.Stdin)
	}
	line, err := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil && line == "" {
		return "", false
	}
	return line, true
}

func confirmDelete(env Env, what string) error {
	fmt.Fprintf(env.Stdout, "This permanently deletes %s. Type DELETE to confirm: ", what)
	line, ok := readLine(env)
	if !ok || line != "DELETE" {
		return blocked("confirmation not received, nothing deleted").then("re-run and type DELETE to confirm")
	}
	return nil
}
