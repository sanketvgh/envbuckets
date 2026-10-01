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
	scaffold := flags.Bool("scaffold", false, "create empty files for every bucket the shared rules reference")
	rest, err := parseFlags(flags, args)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usage("init: takes no arguments").then("envbuckets init [--into <bucket>] [--scaffold]")
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
	cfg, err := config.Load(root)
	missing := errors.Is(err, config.ErrMissing)
	if err != nil && !missing {
		return configError(err)
	}
	if err := probeSymlinks(root); err != nil {
		return err
	}
	step := stepPrinter(env)
	fmt.Fprintln(env.Stdout, "envbuckets: activating in this project")
	if missing {
		cfg = config.New()
	}
	if err := initSetup(root, cfg, missing, step); err != nil {
		return recoverable(err)
	}
	p := newProject(root, cfg)

	var blockedErr error
	bootstrapped := map[string]bool{}
	for _, s := range p.scopes {
		if err := bootstrapScope(env, s, *into, step); err != nil {
			var ee *exitError
			if errors.As(err, &ee) && ee.code == ExitBlocked {
				step("blocked", "%s: %s", s.Name, ee.msg)
				blockedErr = err
				continue
			}
			return recoverable(err)
		}
		bootstrapped[s.Name] = true
	}
	if *scaffold {
		if err := scaffoldScopes(env, p, bootstrapped, step); err != nil {
			blockedErr = err
		}
	}
	return blockedErr
}

func initSetup(root string, cfg *config.Config, missing bool, step func(string, string, ...any)) error {
	if missing {
		if err := cfg.Save(root); err != nil {
			return err
		}
		step("created", config.FileName)
	} else {
		step("ok", config.FileName)
	}
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
	changed, err := ensureIgnored(root, ignoreLines(newProject(root, cfg).scopes))
	if err != nil {
		return err
	}
	if changed {
		step("created", ".gitignore block")
	} else {
		step("ok", ".gitignore block")
	}
	return nil
}

func recoverable(err error) error {
	var ee *exitError
	if !errors.As(err, &ee) {
		ee = envErr("%v", err)
	}
	if ee.next == "" {
		ee.next = "steps marked [created] above are kept; fix the cause and re-run envbuckets init, it skips finished steps (envbuckets uninstall backs everything out)"
	}
	return ee
}

func probeSymlinks(root string) error {
	f, err := os.CreateTemp(root, ".envbuckets-probe-*")
	if err != nil {
		return envErr("cannot write in %s: %v", root, err).then("check the directory permissions, nothing was changed")
	}
	name := f.Name()
	_ = f.Close()
	if err := os.Remove(name); err != nil {
		return envErr("cannot clean up %s: %v", filepath.Base(name), err).then("delete %s by hand", filepath.Base(name))
	}
	err = os.Symlink(bucketsDir+"/probe", name)
	_ = os.Remove(name)
	if err != nil {
		return envErr("this filesystem does not allow symlinks here: %v", err).
			then("on Windows enable Developer Mode or use WSL, then re-run envbuckets init; nothing was changed")
	}
	return nil
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
		return envErr("link %s: %v (values are safe in %s)", envDisplay, err, s.display(linkTarget(bucket))).
			then("envbuckets use %s --scope %s creates the link", bucket, s.Name)
	}
	step("created", "%s: %s moved to %s, %s -> %s", s.Name, envDisplay, s.display(linkTarget(bucket)), envDisplay, linkTarget(bucket))
	return nil
}
