// Package fsx provides symlink inspection, atomic symlink swaps, and
// atomic file writes staged inside the project.
package fsx

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Status describes a path without following it.
type Status struct {
	Exists    bool
	IsSymlink bool
	IsDir     bool
	Target    string
	Dangling  bool
}

// Inspect describes path without following it and reports whether a
// symlink target resolves.
func Inspect(path string) (Status, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Status{}, nil
	}
	if err != nil {
		return Status{}, fmt.Errorf("lstat %s: %w", path, err)
	}
	s := Status{Exists: true, IsDir: info.IsDir(), IsSymlink: info.Mode()&os.ModeSymlink != 0}
	if !s.IsSymlink {
		return s, nil
	}
	target, err := os.Readlink(path)
	if err != nil {
		return Status{}, fmt.Errorf("readlink %s: %w", path, err)
	}
	s.Target = filepath.ToSlash(target)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		s.Dangling = true
	}
	return s, nil
}

// SetSymlink atomically makes link point at target, staging a temporary
// link in stageDir and renaming it into place so link never disappears.
// It reports whether the link changed.
func SetSymlink(link, target, stageDir string) (bool, error) {
	err := os.Symlink(filepath.FromSlash(target), link)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return false, fmt.Errorf("symlink %s: %w", link, err)
	}
	if current, rerr := os.Readlink(link); rerr == nil && filepath.ToSlash(current) == target {
		return false, nil
	}
	tmp, err := reserveTempName(stageDir, ".envbuckets-link-")
	if err != nil {
		return false, err
	}
	if err := os.Symlink(filepath.FromSlash(target), tmp); err != nil {
		return false, fmt.Errorf("symlink %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, link); err != nil {
		_ = os.Remove(tmp)
		return false, fmt.Errorf("swap %s: %w", link, err)
	}
	return true, nil
}

// WriteFileAtomic writes data to path via a temp file in the same
// directory, fsynced and renamed into place.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	return writeAtomic(path, filepath.Dir(path), perm, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
}

// CopyFileAtomic streams src into a temp file in stageDir and renames it
// over dst.
func CopyFileAtomic(src, dst, stageDir string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	return writeAtomic(dst, stageDir, perm, func(w io.Writer) error {
		_, err := io.Copy(w, in)
		return err
	})
}

func writeAtomic(path, stageDir string, perm os.FileMode, fill func(io.Writer) error) error {
	f, err := os.CreateTemp(stageDir, ".envbuckets-write-*")
	if err != nil {
		return fmt.Errorf("stage in %s: %w", stageDir, err)
	}
	tmp := f.Name()
	cleanup := func(err error) error {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := fill(f); err != nil {
		return cleanup(fmt.Errorf("write %s: %w", path, err))
	}
	if err := f.Sync(); err != nil {
		return cleanup(fmt.Errorf("sync %s: %w", path, err))
	}
	if err := f.Chmod(perm); err != nil {
		return cleanup(fmt.Errorf("chmod %s: %w", path, err))
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

func reserveTempName(dir, prefix string) (string, error) {
	f, err := os.CreateTemp(dir, prefix+"*")
	if err != nil {
		return "", fmt.Errorf("stage in %s: %w", dir, err)
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		return "", err
	}
	if err := os.Remove(name); err != nil {
		return "", err
	}
	return name, nil
}
