package fsx

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/config"
)

// Repo confines managed filesystem operations to one opened repository root.
type Repo struct {
	Path string
	Root *os.Root
}

// OpenRepo opens path as the root of managed filesystem operations.
func OpenRepo(path string) (*Repo, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(abs)
	if err != nil {
		return nil, err
	}
	return &Repo{Path: abs, Root: r}, nil
}

// Close releases the operating-system handle held on the repository root.
func (r *Repo) Close() error { return r.Root.Close() }

// rootPath converts host paths to the slash-separated paths required by
// io/fs and os.Root, including on Windows.
func rootPath(name string) string { return filepath.ToSlash(name) }

// ValidatePath checks lexical locality, reserved trees, symlink parents, and
// Git tracking before a managed path is used.
func (r *Repo) ValidatePath(name string) error {
	if !filepath.IsLocal(name) || name == "." {
		return fmt.Errorf("path %q is not local", name)
	}
	p := filepath.ToSlash(filepath.Clean(name))
	parts := strings.Split(p, "/")
	for _, part := range parts {
		if strings.EqualFold(part, ".git") || strings.EqualFold(part, ".env.d") || strings.EqualFold(part, "GIT~1") {
			return fmt.Errorf("path %q is reserved", name)
		}
	}
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[:i], "/")
		info, err := r.Root.Lstat(parent)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path %q has a symlinked parent", name)
		}
	}
	tracked, err := trackedByGit(r.Path, filepath.ToSlash(name))
	if err != nil {
		return err
	}
	if tracked {
		return fmt.Errorf("path %q is tracked by Git", name)
	}
	return nil
}

// ScanBucket returns relative regular-file paths, skips common editor/OS
// clutter, and reports unsafe non-regular entries without following links.
func (r *Repo) ScanBucket(bucket string) ([]string, []string, error) {
	if !config.ValidBucket(bucket) {
		return nil, nil, fmt.Errorf("invalid bucket name %q", bucket)
	}
	base := rootPath(filepath.Join(".env.d", bucket))
	for _, dir := range []string{".env.d", base} {
		info, err := r.Root.Lstat(dir)
		if err != nil {
			return nil, nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, nil, fmt.Errorf("unsafe bucket directory %q", dir)
		}
	}
	var files, unsafe []string
	err := fs.WalkDir(r.Root.FS(), base, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == base {
			return nil
		}
		if clutter(filepath.Base(name)) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(filepath.ToSlash(name), filepath.ToSlash(base)+"/")
		if !info.Mode().IsRegular() {
			unsafe = append(unsafe, rel)
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return files, unsafe, nil
}

func clutter(name string) bool {
	switch strings.ToLower(name) {
	case ".ds_store", "thumbs.db", "desktop.ini":
		return true
	}
	return strings.HasSuffix(name, "~") || strings.HasSuffix(name, ".swp")
}

func validateLinkTarget(root *os.Root, link, target string) error {
	target = filepath.FromSlash(target)
	if target == "" || filepath.IsAbs(target) {
		return fmt.Errorf("link target %q is not relative", target)
	}
	resolved := filepath.Clean(filepath.Join(filepath.Dir(link), target))
	if !filepath.IsLocal(resolved) {
		return fmt.Errorf("link target %q escapes the repository", target)
	}
	return validateRootPath(root, resolved, false)
}

func validateRootPath(root *os.Root, name string, allowFinalSymlink bool) error {
	if !filepath.IsLocal(name) || name == "." {
		return fmt.Errorf("path %q is not local", name)
	}
	clean := rootPath(filepath.Clean(name))
	parts := strings.Split(filepath.ToSlash(clean), "/")
	parent := "."
	for i := 0; i < len(parts)-1; i++ {
		parent = rootPath(filepath.Join(parent, parts[i]))
		info, err := root.Lstat(parent)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("path %q has an unsafe parent %q", name, parent)
		}
	}
	if !allowFinalSymlink {
		if info, err := root.Lstat(clean); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path %q is a symlink", name)
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// LinkFile stages a relative symlink beside its destination and renames it
// into place without opening the target file.
func LinkFile(root *os.Root, link, target string) error {
	link = rootPath(link)
	if err := validateRootPath(root, link, true); err != nil {
		return err
	}
	if err := validateLinkTarget(root, link, target); err != nil {
		return err
	}
	if info, err := root.Lstat(link); err == nil && info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("refusing to replace real path %s with a symlink", link)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	tmp, err := StageSymlinkRoot(root, target, filepath.Dir(link))
	if err != nil {
		return err
	}
	if err := root.Rename(tmp, link); err != nil {
		_ = root.Remove(tmp)
		return fmt.Errorf("install link %s: %w", link, err)
	}
	return nil
}

// MoveFileToBucket keeps source present by hard-linking it first, then
// replacing source with a staged relative symlink. If hard links are
// unavailable, it moves the original into the bucket before creating the link.
// The bool reports whether the hard-link path was used.
func MoveFileToBucket(root *os.Root, source, destination, linkTarget string) (bool, error) {
	source = rootPath(source)
	destination = rootPath(destination)
	if err := validateRootPath(root, source, false); err != nil {
		return false, err
	}
	if err := validateRootPath(root, destination, false); err != nil {
		return false, err
	}
	if err := validateLinkTarget(root, source, linkTarget); err != nil {
		return false, err
	}
	info, err := root.Lstat(source)
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("source %s is not a regular file", source)
	}
	if _, err := root.Lstat(destination); err == nil {
		return false, fmt.Errorf("destination %s already exists", destination)
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := root.Link(source, destination); err != nil {
		if moveErr := root.Rename(source, destination); moveErr != nil {
			return false, fmt.Errorf("move %s into bucket: %w", source, moveErr)
		}
		if linkErr := root.Symlink(filepath.FromSlash(linkTarget), source); linkErr != nil {
			return false, fmt.Errorf("file moved to %s but link repair is needed at %s: %w", destination, source, linkErr)
		}
		return false, nil
	}
	tmp, err := StageSymlinkRoot(root, linkTarget, filepath.Dir(source))
	if err != nil {
		_ = root.Remove(destination)
		return true, err
	}
	if err := root.Rename(tmp, source); err != nil {
		_ = root.Remove(tmp)
		_ = root.Remove(destination)
		return true, fmt.Errorf("replace %s with link: %w", source, err)
	}
	return true, nil
}

func trackedByGit(root, path string) (bool, error) {
	cmd := exec.Command("git", "ls-files", "--error-unmatch", "--", path) //nolint:gosec // fixed executable and argument vector; path is separated from options
	cmd.Dir = root
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("git ls-files: %w", err)
}
