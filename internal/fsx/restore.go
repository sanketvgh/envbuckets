package fsx

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/config"
)

// RestoreBucketFile renames an active regular bucket file over its working
// symlink in one operation, without removing the link or reading either file.
// Revalidation leaves replaced working files and changed links untouched.
func RestoreBucketFile(repo *Repo, name, source, target string) error {
	if err := repo.ValidatePath(filepath.FromSlash(name)); err != nil {
		return err
	}
	parts := strings.SplitN(source, "/", 3)
	if len(parts) != 3 || parts[0] != ".env.d" || !config.ValidBucket(parts[1]) || parts[2] != name {
		return errors.New("source is not the same path inside a bucket")
	}
	canonical, err := filepath.Rel(filepath.Dir(filepath.FromSlash(name)), filepath.FromSlash(source))
	if err != nil || filepath.ToSlash(canonical) != target {
		return errors.New("source does not match the expected link target")
	}
	if err := ValidateInternalPath(repo.Root, source); err != nil {
		return err
	}
	info, err := repo.Root.Lstat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("bucket target is not a regular file")
	}
	current, err := repo.Root.Readlink(name)
	if err != nil || filepath.ToSlash(current) != target {
		return errors.New("working link changed before restoration")
	}
	if err := repo.Root.Rename(source, name); err != nil {
		return fmt.Errorf("restore %s: %w", name, err)
	}
	return nil
}
