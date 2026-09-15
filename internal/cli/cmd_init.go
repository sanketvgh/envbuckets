package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sanketvgh/envbuckets/internal/block"
	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/gitx"
)

func runInit(args []string, env Env) error {
	flags := newFlags("init")
	into := flags.String("into", "", "bucket to move an existing real .env into (skips the prompt)")
	if _, err := parseFlags(flags, args); err != nil {
		return err
	}
	if *into != "" {
		if err := config.ValidateName(*into); err != nil {
			return usage("--into: %v", err).then("envbuckets init --into <bucket-name>")
		}
	}

	root, err := repoRoot(env)
	if err != nil {
		return err
	}
	step := func(tag, format string, a ...any) {
		fmt.Fprintf(env.Stdout, "  [%s] %s\n", tag, fmt.Sprintf(format, a...))
	}
	fmt.Fprintln(env.Stdout, "envbuckets: activating in this project")

	cfg, err := config.Load(root)
	switch {
	case errors.Is(err, config.ErrMissing):
		cfg = config.New()
		if err := cfg.Save(root); err != nil {
			return err
		}
		step("created", config.FileName)
	case err != nil:
		return configError(err)
	default:
		step("ok", config.FileName)
	}
	p := newProject(root, cfg)

	hooksDir, err := gitx.HooksDir(root)
	if err != nil {
		return envErr("cannot locate the git hooks directory: %v", err).then("check git rev-parse --git-path hooks")
	}
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}
	hookPath := filepath.Join(hooksDir, "post-checkout")
	res, err := block.InstallHook(hookPath)
	if err != nil {
		return err
	}
	if res == block.HookWritten {
		step("created", "hook block (%s)", relOrAbs(root, hookPath))
	} else {
		step("ok", "hook block (%s)", relOrAbs(root, hookPath))
	}

	changed, err := ensureIgnored(root, ignoreLines(p.scopes))
	if err != nil {
		return err
	}
	if changed {
		step("created", ".gitignore block")
	} else {
		step("ok", ".gitignore block")
	}

	var blockedErr error
	for _, s := range p.scopes {
		if err := bootstrapScope(env, s, *into, step); err != nil {
			var ee *exitError
			if errors.As(err, &ee) && ee.code == ExitBlocked {
				step("blocked", "%s: %s", s.Name, ee.msg)
				blockedErr = err
				continue
			}
			return err
		}
	}
	return blockedErr
}

func relOrAbs(root, p string) string {
	if rel, err := filepath.Rel(root, p); err == nil {
		return filepath.ToSlash(rel)
	}
	return p
}

func bootstrapScope(env Env, s scope, into string, step func(string, string, ...any)) error {
	if !s.exists() {
		step("skipped", "%s: scope directory missing", s.Name)
		return nil
	}
	ls, err := s.linkState()
	if err != nil {
		return err
	}
	envDisplay := s.display(envFile)
	switch ls.kind {
	case linkMissing:
		step("skipped", "%s: no %s yet (envbuckets bucket add <name>, then envbuckets use <name>)", s.Name, envDisplay)
		return nil
	case linkBucket:
		if ls.dangling {
			step("skipped", "%s: %s -> %s is BROKEN (create the file, or envbuckets use <bucket>)", s.Name, envDisplay, ls.target)
			return nil
		}
		step("ok", "%s: %s -> %s", s.Name, envDisplay, ls.target)
		return nil
	case linkForeign:
		step("skipped", "%s: %s is a symlink outside %s/ (%s), left untouched", s.Name, envDisplay, bucketsDir, ls.target)
		return nil
	case linkReal:
	}

	bucket := into
	if bucket == "" {
		fmt.Fprintf(env.Stdout, "%s: %s is a real file. Move it into which bucket? (empty to skip): ", s.Name, envDisplay)
		line, ok := readLine(env)
		if !ok || line == "" {
			step("skipped", "%s: %s left as a real file (re-run with --into <bucket>)", s.Name, envDisplay)
			return nil
		}
		if err := config.ValidateName(line); err != nil {
			return blocked("%v", err)
		}
		bucket = line
	}
	if s.bucketFileExists(bucket) {
		return blocked("%s already exists, refusing to clobber it", s.display(linkTarget(bucket))).then("envbuckets init --into <another-bucket>")
	}
	if err := os.MkdirAll(s.bucketDir(bucket), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.bucketsPath(), ".envbuckets-link-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Remove(tmpName); err != nil {
		return err
	}
	if err := os.Symlink(filepath.FromSlash(linkTarget(bucket)), tmpName); err != nil {
		return fmt.Errorf("symlink: %w", err)
	}
	if err := os.Rename(s.envPath(), s.bucketFile(bucket)); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("move %s: %w", envDisplay, err)
	}
	if err := os.Rename(tmpName, s.envPath()); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("link %s: %w (values are safe in %s)", envDisplay, err, s.display(linkTarget(bucket)))
	}
	step("created", "%s: %s moved to %s, %s -> %s", s.Name, envDisplay, s.display(linkTarget(bucket)), envDisplay, linkTarget(bucket))
	return nil
}
