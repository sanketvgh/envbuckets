package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/block"
	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/gitx"
	"github.com/sanketvgh/envbuckets/internal/switcher"
)

type statusReport struct {
	head, bucket, hint string
	fileHint           string
	problems, ignored  []string
	linked, total      int
	hookMissing        bool
}

func runStatus(args []string, env Env) int {
	if len(args) != 0 {
		return commandUsage(env, "status", nil)
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
	report, err := inspectStatus(repo, cfg, branch)
	if err != nil {
		return switchFailure(env, false, err)
	}
	renderStatus(report, env)
	return ExitOK
}

func inspectStatus(repo *fsx.Repo, cfg config.Config, branch string) (statusReport, error) {
	p := statusReport{head: "On branch " + branch}
	if branch == "" {
		name, err := gitx.DetachedName(repo.Path)
		if err != nil {
			return p, err
		}
		p.head = "HEAD detached at " + name
	}
	links, err := switcher.ManagedLinks(repo)
	if err != nil {
		return p, err
	}
	buckets := linkBuckets(links)
	resolution := switcher.Resolution{}
	if branch != "" {
		resolution, err = switcher.Resolve(repo, cfg, branch)
		if err != nil {
			return p, err
		}
		if len(buckets) == 0 && resolution.Bucket != "" {
			buckets = []string{resolution.Bucket}
		}
	}
	p.bucket, p.hint = statusBucket(buckets, resolution, cfg.Default, branch == "")
	p.fileHint = "to keep a file, move it to the same path under .env.d/<bucket>/, then use \"envbuckets switch\""
	if len(buckets) == 1 {
		p.fileHint = "to keep a file, move it to the same path under .env.d/" + buckets[0] + "/, then use \"envbuckets switch\""
	}
	wanted := make(map[string]bool)
	for _, bucket := range buckets {
		files, unsafe, err := repo.ScanBucket(bucket)
		if os.IsNotExist(err) {
			continue // Dangling links still appear below, even if a bucket was deleted.
		}
		if err != nil {
			return p, err
		}
		for _, name := range append(files, unsafe...) {
			wanted[name] = true
		}
	}
	for name := range links {
		wanted[name] = true
	}
	names := make([]string, 0, len(wanted))
	for name := range wanted {
		names = append(names, name)
	}
	slices.Sort(names)
	p.total = len(names)
	for _, name := range names {
		if links[name] != "" {
			ignored, err := gitx.Ignored(repo.Path, name)
			if err != nil {
				return p, err
			}
			if !ignored {
				p.ignored = append(p.ignored, name)
			}
		}
		label, err := statusPath(repo, name, links[name])
		if err != nil {
			return p, err
		}
		if label != "" {
			p.problems = append(p.problems, label+":   "+name)
			continue
		}
		p.linked++
	}
	installed, err := block.RepoHookInstalled(repo.Path)
	p.hookMissing = !installed
	return p, err
}

func statusPath(repo *fsx.Repo, name, target string) (string, error) {
	// Validate parents before touching the working path. Do not inspect a foreign
	// link's target: even a harmless-looking link may escape the repository.
	if err := repo.ValidatePath(filepath.FromSlash(name)); err != nil {
		return "unsafe path", nil
	}
	info, err := repo.Root.Lstat(name)
	if os.IsNotExist(err) {
		return "missing link", nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return "real file", nil
	}
	if target == "" {
		return "foreign link", nil
	}
	source := filepath.Clean(filepath.Join(filepath.Dir(filepath.FromSlash(name)), filepath.FromSlash(target)))
	if err := fsx.ValidateInternalPath(repo.Root, source); err != nil {
		return "broken link", nil
	}
	info, err = repo.Root.Lstat(filepath.ToSlash(source))
	if os.IsNotExist(err) {
		return "broken link", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "broken link", nil
	}
	return "", nil
}

func statusBucket(buckets []string, r switcher.Resolution, defaultBucket string, detached bool) (string, string) {
	if len(buckets) > 1 {
		return "Using a mix of buckets: '" + strings.Join(buckets, "', '") + "'", "use \"envbuckets switch\" to link one bucket"
	}
	if len(buckets) == 0 {
		if detached {
			return "No bucket is linked", "links stay as they are on a detached HEAD; use \"envbuckets switch <bucket>\" to pick one"
		}
		return fmt.Sprintf("No bucket is linked, because default bucket '%s' does not exist", defaultBucket),
			fmt.Sprintf("use \"envbuckets switch -c %s\" to create it", defaultBucket)
	}
	current := buckets[0]
	line := "Using bucket '" + current + "'"
	if detached {
		return line, "links stay as they are on a detached HEAD; use \"envbuckets switch <bucket>\" to pick one"
	}
	if r.Missing && current == defaultBucket && r.Bucket != "" {
		return fmt.Sprintf("%s (default), because '%s' (%s) does not exist", line, r.Requested, mappingReason(r.Rule)),
			fmt.Sprintf("use \"envbuckets switch -c %s\" to create it", r.Requested)
	}
	if r.Missing && r.Bucket != "" {
		return fmt.Sprintf("%s, but this branch uses '%s' (default), because '%s' (%s) does not exist", line, defaultBucket, r.Requested, mappingReason(r.Rule)),
			fmt.Sprintf("use \"envbuckets switch\" to go back to '%s'", defaultBucket)
	}
	if r.Missing && r.Bucket == "" {
		if r.Requested == defaultBucket {
			return fmt.Sprintf("%s, because default bucket '%s' does not exist", line, defaultBucket),
				fmt.Sprintf("use \"envbuckets switch -c %s\" to create it", defaultBucket)
		}
		return fmt.Sprintf("%s, because '%s' (%s) and default bucket '%s' do not exist", line, r.Requested, mappingReason(r.Rule), defaultBucket),
			fmt.Sprintf("use \"envbuckets switch -c %s\" to create it", r.Requested)
	}
	if current != r.Requested {
		return fmt.Sprintf("%s, but this branch uses '%s' (%s)", line, r.Requested, mappingReason(r.Rule)),
			fmt.Sprintf("use \"envbuckets switch\" to go back to '%s'", r.Requested)
	}
	return line + " (" + mappingReason(r.Rule) + ")", ""
}

func renderStatus(p statusReport, env Env) {
	fmt.Fprintf(env.Stdout, "%s\n%s\n", p.head, p.bucket)
	if p.hint != "" {
		fmt.Fprintf(env.Stdout, "  (%s)\n", p.hint)
	}
	fmt.Fprintln(env.Stdout)
	if len(p.problems) > 0 {
		fmt.Fprintln(env.Stdout, "Files not linked:")
		fmt.Fprintf(env.Stdout, "  (%s)\n", p.fileHint)
		for _, problem := range p.problems {
			fmt.Fprintf(env.Stdout, "\t%s\n", problem)
		}
		fmt.Fprintln(env.Stdout)
	}
	if len(p.ignored) > 0 {
		fmt.Fprintln(env.Stdout, "Not ignored by Git:\n  (add them to .gitignore)")
		for _, name := range p.ignored {
			fmt.Fprintf(env.Stdout, "\t%s\n", name)
		}
		fmt.Fprintln(env.Stdout)
	}
	if p.hookMissing {
		fmt.Fprint(env.Stdout, "Hook not installed:\n  (use \"envbuckets init\" to install it)\n\n")
	}
	switch {
	case p.linked != p.total:
		fmt.Fprintf(env.Stdout, "%d of %d files linked\n", p.linked, p.total)
	case p.total == 1:
		fmt.Fprintln(env.Stdout, "1 file linked")
	default:
		fmt.Fprintf(env.Stdout, "all %d files linked\n", p.total)
	}
}
