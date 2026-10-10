package switcher

import (
	"fmt"
	"os"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
)

// Resolution describes the mapping and the bucket a branch checkout can use.
// An empty Bucket means both the requested bucket and the default are missing.
type Resolution struct {
	Requested string
	Rule      string
	Bucket    string
	Missing   bool
}

// Resolve shares first-match and missing-bucket fallback between reports and switches.
// It inspects directory metadata only, without scanning any bucket's files.
func Resolve(repo *fsx.Repo, cfg config.Config, branch string) (Resolution, error) {
	requested, rule := cfg.BucketFor(branch)
	r := Resolution{Requested: requested, Rule: rule, Bucket: requested}
	exists, err := bucketExists(repo, requested)
	if err != nil || exists {
		return r, err
	}
	r.Missing = true
	r.Bucket = cfg.Default
	exists, err = bucketExists(repo, cfg.Default)
	if !exists {
		r.Bucket = ""
	}
	return r, err
}

func bucketExists(repo *fsx.Repo, bucket string) (bool, error) {
	name := ".env.d/" + bucket
	if err := fsx.ValidateInternalPath(repo.Root, name); err != nil {
		return false, err
	}
	info, err := repo.Root.Lstat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("unsafe bucket directory %q", name)
	}
	return true, nil
}
