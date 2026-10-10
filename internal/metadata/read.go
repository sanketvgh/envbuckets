// Package metadata reads regular project metadata without following a replaced
// directory entry into a managed file.
package metadata

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrNotRegular identifies a symlink, directory, or changed metadata file.
var ErrNotRegular = errors.New("not a regular file")

// ReadRegular verifies the opened file against its directory entry before
// reading any bytes. All path operations stay inside root.
func ReadRegular(root *os.Root, name string) ([]byte, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: %w", name, ErrNotRegular)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	after, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !after.Mode().IsRegular() || !os.SameFile(before, opened) || !os.SameFile(after, opened) {
		return nil, fmt.Errorf("%s changed while opening: %w", name, ErrNotRegular)
	}
	return io.ReadAll(file)
}
