//go:build integration

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/sanketvgh/envbuckets/internal/block"
	"github.com/sanketvgh/envbuckets/internal/gitx"
)

// TestMain exposes the CLI as an executable named envbuckets on PATH, so
// scripts can exec it and the installed post-checkout hook can find it
// when git runs it. Pattern from mvdan/git-picked main_test.go (BSD-3).
func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"envbuckets": main,
	})
}

// TestScript runs the txtar scripts in testdata/script against real git.
// Each script gets a private HOME and global gitconfig so the host's git
// settings (hooksPath, signing, default branch) cannot leak in.
func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:                 filepath.Join("testdata", "script"),
		RequireExplicitExec: true,
		Setup:               sandboxGit,
		UpdateScripts:       os.Getenv("TESTSCRIPT_UPDATE") == "1",
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"readlink":     cmdReadlink,
			"regular":      cmdRegular,
			"install-hook": cmdInstallHook,
			"sample":       cmdSample,
			"branches1000": cmdBranches1000,
		},
	})
}

// install-hook exercises the installer that EB-03 init will call. Keeping this
// test helper out of the CLI avoids exposing a second installation command.
func cmdInstallHook(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 0 {
		ts.Fatalf("usage: install-hook")
	}
	root, err := gitx.Root(ts.MkAbs("."))
	if err == nil {
		_, err = block.InstallRepoHook(root)
	}
	if neg {
		if err == nil {
			ts.Fatalf("hook installation unexpectedly succeeded")
		}
		ts.Logf("expected hook installation refusal: %v", err)
		return
	}
	if err != nil {
		ts.Fatalf("install hook: %v", err)
	}
}

const gitconfig = "[user]\n\tname = envbuckets test\n\temail = test@example.com\n[commit]\n\tgpgsign = false\n[init]\n\tdefaultBranch = main\n[maintenance]\n\tautoDetach = false\n[core]\n\tautocrlf = false\n[color]\n\tui = false\n"

func sandboxGit(env *testscript.Env) error {
	git, err := exec.LookPath("git")
	if err != nil {
		return err
	}
	cfg := filepath.Join(env.WorkDir, ".gitconfig")
	if err := os.WriteFile(cfg, []byte(gitconfig), 0o600); err != nil {
		return err
	}
	env.Vars = append(env.Vars,
		"HOME="+env.WorkDir,
		"USERPROFILE="+env.WorkDir,
		"XDG_CONFIG_HOME="+filepath.Join(env.WorkDir, "xdg"),
		"GIT_CONFIG_GLOBAL="+cfg,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_EXE="+git,
		"GIT_AUTHOR_NAME=envbuckets test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=envbuckets test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_AUTHOR_DATE=2000-01-01T00:00:00+0000",
		"GIT_COMMITTER_DATE=2000-01-01T00:00:00+0000",
	)
	samples, err := acceptanceSamples()
	if err != nil {
		return err
	}
	env.Values["acceptance-samples"] = samples
	return nil
}

// readlink <path> <target> asserts path is a symlink pointing at target,
// compared with forward slashes so scripts read the same on every OS.
func cmdReadlink(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		ts.Fatalf("usage: readlink path target")
	}
	got, err := os.Readlink(ts.MkAbs(args[0]))
	if neg {
		if err == nil && filepath.ToSlash(got) == args[1] {
			ts.Fatalf("%s unexpectedly -> %s", args[0], args[1])
		}
		return
	}
	if err != nil {
		ts.Fatalf("readlink %s: %v", args[0], err)
	}
	if filepath.ToSlash(got) != args[1] {
		ts.Fatalf("%s -> %s, want %s", args[0], filepath.ToSlash(got), args[1])
	}
}

// regular <path> asserts path exists and is a regular file, not a symlink.
func cmdRegular(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 1 {
		ts.Fatalf("usage: regular path")
	}
	info, err := os.Lstat(ts.MkAbs(args[0]))
	isRegular := err == nil && info.Mode().IsRegular()
	if neg {
		if isRegular {
			ts.Fatalf("%s is a regular file", args[0])
		}
		return
	}
	if err != nil {
		ts.Fatalf("lstat %s: %v", args[0], err)
	}
	if !isRegular {
		ts.Fatalf("%s is not a regular file (%s)", args[0], info.Mode())
	}
}
