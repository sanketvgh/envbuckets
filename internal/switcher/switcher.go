// Package switcher plans and applies bucket switches without reading file contents.
package switcher

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
)

// Kind identifies a planned filesystem action.
type Kind int

// Action kinds; a preflight error prevents the entire plan from being applied.
const (
	Link Kind = iota
	Remove
	Skip
	PreflightError
)

// Action describes a change or a reason a path cannot change.
type Action struct {
	Kind     Kind
	Path     string
	Target   string
	Previous string
	Reason   string
}

// Plan is the complete result of inspecting a switch before changing anything.
type Plan struct {
	Bucket       string
	BranchBucket string
	Reason       string
	Current      string
	Actions      []Action
	Warnings     []string
	Hint         string
}

// ExitCode returns the status shared by dry runs and successful plan application.
func (p Plan) ExitCode() int {
	for _, a := range p.Actions {
		if a.Kind == Skip || a.Kind == PreflightError {
			return 1
		}
	}
	return 0
}

// Blocked reports whether no changes may be made.
func (p Plan) Blocked() bool {
	for _, a := range p.Actions {
		if a.Kind == PreflightError {
			return true
		}
	}
	return p.Bucket == ""
}

func (p *Plan) fail(reason, hint string) {
	p.Actions = append(p.Actions, Action{Kind: PreflightError, Reason: reason})
	p.Hint = hint
}

// Build inspects the repository and returns all changes and diagnostics.
// An empty explicit bucket means returning to the current branch's bucket.
func Build(repo *fsx.Repo, cfg config.Config, branch, explicit string) Plan {
	p := Plan{Reason: "default"}
	if branch != "" {
		var rule string
		p.BranchBucket, rule = cfg.BucketFor(branch)
		if rule != "" {
			p.Reason = "rule '" + rule + "'"
		}
	}
	if explicit == "" && branch == "" {
		p.fail("HEAD is detached, so there is no branch bucket to go back to", "Name a bucket instead, such as \"envbuckets switch prod\".")
		return p
	}
	p.Bucket = explicit
	if p.Bucket == "" {
		p.Bucket = p.BranchBucket
	}
	if !config.ValidBucket(p.Bucket) {
		p.fail(fmt.Sprintf("invalid bucket name '%s'", p.Bucket), "")
		return p
	}
	if explicit == "" {
		resolution, err := Resolve(repo, cfg, branch)
		if err != nil {
			p.fail(fmt.Sprintf("cannot inspect bucket '%s': %s", p.Bucket, err), "")
			return p
		}
		p.Bucket = resolution.Bucket
		if resolution.Missing {
			missing := resolution.Requested
			p.Reason = "default"
			p.Hint = fmt.Sprintf("Create it with \"envbuckets switch -c %s\".", missing)
			if p.Bucket == "" {
				warning := fmt.Sprintf("default bucket '%s' does not exist; links not changed", cfg.Default)
				if missing != cfg.Default {
					warning = fmt.Sprintf("bucket '%s' does not exist and %s", missing, warning)
				}
				p.Warnings = append(p.Warnings, warning)
				return p
			}
			p.Warnings = append(p.Warnings, fmt.Sprintf("bucket '%s' does not exist; using '%s' (default)", missing, cfg.Default))
		}
	}
	files, unsafe, err := repo.ScanBucket(p.Bucket)
	if explicit != "" && os.IsNotExist(err) {
		p.fail(fmt.Sprintf("no bucket named '%s'", explicit), fmt.Sprintf("Create it with \"envbuckets switch -c %s\".", explicit))
		return p
	}
	if err != nil {
		p.fail(fmt.Sprintf("cannot inspect bucket '%s': %s", p.Bucket, err), "")
		return p
	}
	return planFiles(repo, p, files, unsafe)
}

// BuildFiles plans a switch to a new bucket whose empty files do not exist yet.
// It shares destination validation with Build so dry runs need no bucket writes.
func BuildFiles(repo *fsx.Repo, cfg config.Config, branch, bucket string, files []string) Plan {
	p := Plan{Bucket: bucket, Reason: "default"}
	if branch != "" {
		p.BranchBucket, _ = cfg.BucketFor(branch)
	}
	return planFiles(repo, p, files, nil)
}

func planFiles(repo *fsx.Repo, p Plan, files, unsafe []string) Plan {
	links, err := ManagedLinks(repo)
	if err != nil {
		p.fail(fmt.Sprintf("cannot inspect working tree: %s", err), "")
		return p
	}
	for name, target := range links {
		b := BucketOf(name, target)
		if p.Current == "" {
			p.Current = b
		} else if p.Current != b {
			p.Current = "(mixed)"
		}
	}
	if p.Current == "" {
		p.Current = p.BranchBucket
	}
	wanted := make(map[string]bool, len(files)+len(unsafe))
	for _, name := range unsafe {
		wanted[name] = true
		p.Actions = append(p.Actions, Action{Kind: Skip, Path: name, Reason: "unsafe entry in target bucket"})
	}
	for _, name := range files {
		wanted[name] = true
		a := Action{Kind: Link, Path: name, Previous: links[name]}
		if err := repo.ValidatePath(filepath.FromSlash(name)); err != nil {
			a.Kind, a.Reason = Skip, err.Error()
		} else if err := inspectDestination(repo, name, a.Previous); err != nil {
			a.Kind, a.Reason = Skip, err.Error()
		} else {
			target, err := filepath.Rel(filepath.Dir(filepath.FromSlash(name)), filepath.Join(".env.d", p.Bucket, filepath.FromSlash(name)))
			if err != nil {
				a.Kind, a.Reason = Skip, err.Error()
			} else {
				a.Target = filepath.ToSlash(target)
			}
		}
		if a.Kind == Link && a.Target == a.Previous {
			continue
		}
		p.Actions = append(p.Actions, a)
	}
	for name, previous := range links {
		if wanted[name] {
			continue
		}
		a := Action{Kind: Remove, Path: name, Previous: previous}
		for _, entry := range unsafe {
			if strings.HasPrefix(name, entry+"/") {
				a.Kind, a.Reason = Skip, "unsafe parent in target bucket"
				break
			}
		}
		if err := repo.ValidatePath(filepath.FromSlash(name)); err != nil {
			a.Kind, a.Reason = Skip, err.Error()
		} else if err := inspectDestination(repo, name, previous); err != nil {
			a.Kind, a.Reason = Skip, err.Error()
		}
		p.Actions = append(p.Actions, a)
	}
	slices.SortFunc(p.Actions, func(a, b Action) int { return strings.Compare(a.Path, b.Path) })
	return p
}

// BucketOf recognizes only canonical relative links to the same path inside a bucket.
// It is lexical: dangling links remain recognizable without opening their targets.
func BucketOf(name, target string) string {
	target = filepath.FromSlash(target)
	if filepath.IsAbs(target) || target == "" {
		return ""
	}
	resolved := filepath.Clean(filepath.Join(filepath.Dir(filepath.FromSlash(name)), target))
	if !filepath.IsLocal(resolved) {
		return ""
	}
	parts := strings.SplitN(filepath.ToSlash(resolved), "/", 3)
	if len(parts) != 3 || parts[0] != ".env.d" || !config.ValidBucket(parts[1]) || parts[2] != filepath.ToSlash(name) {
		return ""
	}
	canonical, err := filepath.Rel(filepath.Dir(filepath.FromSlash(name)), resolved)
	if err != nil || filepath.ToSlash(target) != filepath.ToSlash(canonical) {
		return ""
	}
	return parts[1]
}

// ManagedLinks finds canonical bucket links without following their targets.
func ManagedLinks(repo *fsx.Repo) (map[string]string, error) {
	links := make(map[string]string)
	err := fs.WalkDir(repo.Root.FS(), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.EqualFold(d.Name(), ".git") || strings.EqualFold(d.Name(), ".env.d") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink == 0 {
			return nil
		}
		target, err := repo.Root.Readlink(name)
		if err != nil {
			return err
		}
		target = filepath.ToSlash(target)
		if BucketOf(name, target) != "" {
			links[name] = target
		}
		return nil
	})
	return links, err
}

// CurrentBucket uses managed links, or the branch mapping when no links exist.
// A detached HEAD without links and a mixed set of links cannot select a bucket.
func CurrentBucket(repo *fsx.Repo, cfg config.Config, branch string) (string, error) {
	links, err := ManagedLinks(repo)
	if err != nil {
		return "", err
	}
	var current string
	for name, target := range links {
		bucket := BucketOf(name, target)
		if current != "" && current != bucket {
			return "", errors.New("links point to several buckets; run \"envbuckets switch\" first")
		}
		current = bucket
	}
	if current != "" {
		return current, nil
	}
	if branch == "" {
		return "", errors.New("HEAD is detached and no links identify the current bucket; name a bucket with \"envbuckets switch <bucket>\" first")
	}
	current, _ = cfg.BucketFor(branch)
	return current, nil
}

func inspectDestination(repo *fsx.Repo, name, previous string) error {
	info, err := repo.Root.Lstat(name)
	if os.IsNotExist(err) {
		if previous != "" {
			return errors.New("path changed after planning")
		}
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return errors.New("real file or directory is in the way")
	}
	target, err := repo.Root.Readlink(name)
	if err != nil {
		return err
	}
	if previous == "" || filepath.ToSlash(target) != previous {
		return errors.New("foreign symlink or changed path is in the way")
	}
	resolved := filepath.Join(filepath.Dir(filepath.FromSlash(name)), filepath.FromSlash(target))
	if err := fsx.ValidateInternalPath(repo.Root, resolved); err != nil {
		return fmt.Errorf("unsafe symlink target: %w", err)
	}
	return nil
}

// Execute renders the plan or applies it. Diagnostics use Git-style prefixes;
// hooks add the envbuckets prefix and always override the returned code to 0.
func Execute(repo *fsx.Repo, p Plan, dry, hook bool, stdout, stderr io.Writer) int {
	prefix := ""
	if hook {
		prefix = "envbuckets: "
	}
	for _, warning := range p.Warnings {
		fmt.Fprintf(stderr, "%swarning: %s\n", prefix, warning)
	}
	code := p.ExitCode()
	for _, a := range p.Actions {
		if a.Kind == PreflightError {
			fmt.Fprintf(stderr, "%sfatal: %s\n", prefix, a.Reason)
		}
	}
	if !p.Blocked() {
		for _, a := range p.Actions {
			if a.Kind == Skip {
				fmt.Fprintf(stderr, "%serror: cannot switch '%s': %s\n", prefix, a.Path, a.Reason)
				continue
			}
			if dry {
				if a.Kind == Link {
					fmt.Fprintf(stdout, "Would link %s to bucket '%s'\n", a.Path, p.Bucket)
				} else {
					fmt.Fprintf(stdout, "Would remove link %s (not in bucket '%s')\n", a.Path, p.Bucket)
				}
				continue
			}
			if err := apply(repo, a); err != nil {
				fmt.Fprintf(stderr, "%serror: cannot switch '%s': %s\n", prefix, a.Path, err)
				code = 1
			}
		}
		if !dry && code == 0 {
			switch {
			case hook:
				if p.Current != p.Bucket {
					fmt.Fprintf(stderr, "%sSwitched to bucket '%s' (%s)\n", prefix, p.Bucket, p.Reason)
				}
			case p.Current == p.Bucket:
				fmt.Fprintf(stdout, "Already on bucket '%s'\n", p.Bucket)
			default:
				fmt.Fprintf(stdout, "Switched to bucket '%s'\n", p.Bucket)
			}
		}
		if !dry && !hook && p.BranchBucket != "" && p.Bucket != p.BranchBucket {
			fmt.Fprintf(stderr, "hint: This branch uses '%s'. Run \"envbuckets switch\" to go back.\n", p.BranchBucket)
		}
	}
	if p.Hint != "" {
		fmt.Fprintf(stderr, "%shint: %s\n", prefix, p.Hint)
	}
	return code
}

func apply(repo *fsx.Repo, a Action) error {
	if err := repo.ValidatePath(filepath.FromSlash(a.Path)); err != nil {
		return err
	}
	if err := inspectDestination(repo, a.Path, a.Previous); err != nil {
		return err
	}
	if a.Kind == Remove {
		return repo.Root.Remove(a.Path)
	}
	source := filepath.Clean(filepath.Join(filepath.Dir(filepath.FromSlash(a.Path)), filepath.FromSlash(a.Target)))
	if err := fsx.ValidateInternalPath(repo.Root, source); err != nil {
		return err
	}
	info, err := repo.Root.Lstat(filepath.ToSlash(source))
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("target bucket entry is not a regular file")
	}
	if err := repo.Root.MkdirAll(filepath.ToSlash(filepath.Dir(filepath.FromSlash(a.Path))), 0o755); err != nil {
		return err
	}
	return fsx.LinkFile(repo.Root, a.Path, a.Target)
}
