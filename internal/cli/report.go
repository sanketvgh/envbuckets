package cli

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/switcher"
)

func reportConfig(repo *fsx.Repo, env Env) (config.Config, bool) {
	cfg, err := config.Load(repo.Root)
	if errors.Is(err, os.ErrNotExist) {
		env.output(false).Fatal(".envbuckets.json is missing")
		env.output(false).Hint("Run \"envbuckets init\" to set up this repository.")
		return cfg, false
	}
	if err != nil {
		switchFailure(env, false, fmt.Errorf("invalid %w", err))
		return cfg, false
	}
	if _, err := repo.Root.Lstat(".envbuckets.toml"); err == nil {
		env.output(false).Warning(".envbuckets.toml is unused and can be deleted")
	} else if !os.IsNotExist(err) {
		switchFailure(env, false, err)
		return cfg, false
	}
	return cfg, true
}

func linkBuckets(links map[string]string) []string {
	buckets := make([]string, 0, len(links))
	for name, target := range links {
		buckets = append(buckets, switcher.BucketOf(name, target))
	}
	slices.Sort(buckets)
	return slices.Compact(buckets)
}

func mappingReason(rule string) string {
	if rule == "" {
		return "default"
	}
	return "rule '" + rule + "'"
}
