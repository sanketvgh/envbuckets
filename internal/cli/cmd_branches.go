package cli

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/width"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/gitx"
	"github.com/sanketvgh/envbuckets/internal/pattern"
	"github.com/sanketvgh/envbuckets/internal/switcher"
)

const branchesUsage = "branches [--bucket <name>] [<branch or pattern>...]"

func runBranches(args []string, env Env) int {
	bucket, filters, err := parseBranchArgs(args)
	if err != nil {
		return commandUsage(env, branchesUsage, err)
	}
	repo, err := openCommandRepo(env)
	if err != nil {
		return switchFailure(env, false, err)
	}
	defer repo.Close()
	cfg, ok := reportConfig(repo, env)
	if !ok {
		return ExitError
	}
	branch, err := gitx.Branch(repo.Path)
	if err != nil {
		return switchFailure(env, false, err)
	}
	names, err := gitx.Branches(repo.Path)
	if err != nil {
		return switchFailure(env, false, err)
	}
	if branch != "" && !slices.Contains(names, branch) {
		names = append(names, branch) // An unborn current branch has no ref yet.
		slices.Sort(names)
	}
	links, err := switcher.ManagedLinks(repo)
	if err != nil {
		return switchFailure(env, false, err)
	}
	buckets := linkBuckets(links)
	current := ""
	if len(buckets) == 1 {
		current = buckets[0]
	}
	rows, err := branchRows(repo, cfg, names, filters, branch, current, bucket)
	if err != nil {
		return switchFailure(env, false, err)
	}
	branchWidth, bucketWidth := 0, 0
	for _, row := range rows {
		branchWidth = max(branchWidth, displayWidth(row.name))
		bucketWidth = max(bucketWidth, displayWidth(row.bucket))
	}
	for _, row := range rows {
		marker := " "
		if row.name == branch {
			marker = "*"
		}
		fmt.Fprintf(env.Stdout, "%s %s%s  %s%s  %s\n", marker, row.name,
			strings.Repeat(" ", branchWidth-displayWidth(row.name)), row.bucket,
			strings.Repeat(" ", bucketWidth-displayWidth(row.bucket)), row.reason)
	}
	return ExitOK
}

func parseBranchArgs(args []string) (string, []string, error) {
	var bucket string
	var filters []string
	positional := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case !positional && arg == "--":
			positional = true
		case !positional && arg == "--bucket":
			i++
			if i == len(args) {
				return "", nil, errors.New("option '--bucket' requires a bucket name")
			}
			bucket = args[i]
			if !config.ValidBucket(bucket) {
				return "", nil, fmt.Errorf("invalid bucket name '%s'", bucket)
			}
		case !positional && strings.HasPrefix(arg, "--bucket="):
			bucket = strings.TrimPrefix(arg, "--bucket=")
			if !config.ValidBucket(bucket) {
				return "", nil, fmt.Errorf("invalid bucket name '%s'", bucket)
			}
		case !positional && strings.HasPrefix(arg, "-"):
			return "", nil, fmt.Errorf("unknown option '%s'", arg)
		default:
			if arg == "" || strings.ContainsAny(arg, "\r\n") {
				return "", nil, fmt.Errorf("invalid branch name '%s'", arg)
			}
			if isBranchPattern(arg) {
				if err := pattern.Validate(arg); err != nil {
					return "", nil, fmt.Errorf("invalid branch pattern '%s': %w", arg, err)
				}
			}
			filters = append(filters, arg)
		}
	}
	return bucket, filters, nil
}

func isBranchPattern(arg string) bool { return strings.ContainsAny(arg, "*?[") }

type branchRow struct{ name, bucket, reason string }

func branchRows(repo *fsx.Repo, cfg config.Config, names, filters []string, branch, current, bucket string) ([]branchRow, error) {
	selected := names
	if len(filters) > 0 {
		selected = nil
		for _, filter := range filters {
			if !isBranchPattern(filter) {
				selected = append(selected, filter)
				continue
			}
			for _, name := range names {
				if pattern.Match(filter, name) {
					selected = append(selected, name)
				}
			}
		}
	}
	rows := make([]branchRow, 0, len(selected))
	seen := make(map[string]bool, len(selected))
	for _, name := range selected {
		if seen[name] {
			continue
		}
		seen[name] = true
		requested, rule := cfg.BucketFor(name)
		if bucket != "" && requested != bucket {
			continue
		}
		if len(filters) == 0 && bucket == "" && name != branch && rule == "" {
			continue
		}
		resolution, err := switcher.Resolve(repo, cfg, name)
		if err != nil {
			return nil, err
		}
		reason := mappingReason(rule)
		if resolution.Missing {
			if resolution.Bucket == "" {
				if requested == cfg.Default {
					reason += ", missing; links stay as they are"
				} else {
					reason += ", missing; default '" + cfg.Default + "' also missing; links stay as they are"
				}
			} else {
				reason += ", missing; falls back to '" + resolution.Bucket + "'"
			}
		}
		reason = "(" + reason + ")"
		if name == branch && current != "" && current != resolution.Bucket {
			reason += ", using '" + current + "' for now"
		}
		rows = append(rows, branchRow{name, requested, reason})
	}
	return rows, nil
}

// displayWidth keeps CJK and combining characters aligned in plain output.
// EB-06 will use the same terminal-width policy when styling these columns.
func displayWidth(s string) int {
	n := 0
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		switch width.LookupRune(r).Kind() {
		case width.EastAsianWide, width.EastAsianFullwidth:
			n += 2
		default:
			n++
		}
	}
	return n
}
