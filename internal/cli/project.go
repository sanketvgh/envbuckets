package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
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
	return scope{Name: name, Path: rel, Dir: filepath.Join(p.root, filepath.FromSlash(rel))}
}

func (p *project) scopeByName(name string) (scope, bool) {
	for _, s := range p.scopes {
		if s.Name == name {
			return s, true
		}
	}
	return scope{}, false
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
	info, err := os.Stat(s.Dir)
	return err == nil && info.IsDir()
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
	t := path.Clean(filepath.ToSlash(target))
	parts := strings.Split(t, "/")
	if len(parts) != 3 || parts[0] != bucketsDir || parts[2] != envFile {
		return "", false
	}
	return parts[1], true
}

func (s scope) bucketFileExists(name string) bool {
	info, err := os.Stat(s.bucketFile(name))
	return err == nil && info.Mode().IsRegular()
}

func (s scope) bucketFileEmpty(name string) (bool, error) {
	info, err := os.Stat(s.bucketFile(name))
	if err != nil {
		return false, err
	}
	return info.Size() == 0, nil
}

func (s scope) buckets() ([]string, error) {
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
	return fsx.SetSymlink(s.envPath(), linkTarget(bucket), s.bucketsPath())
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
