//go:build integration

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
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
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"readlink": cmdReadlink,
			"regular":  cmdRegular,
		},
	})
}

const gitconfig = "[user]\n\tname = envbuckets test\n\temail = test@example.com\n[commit]\n\tgpgsign = false\n[init]\n\tdefaultBranch = main\n"

func sandboxGit(env *testscript.Env) error {
	cfg := filepath.Join(env.WorkDir, ".gitconfig")
	if err := os.WriteFile(cfg, []byte(gitconfig), 0o600); err != nil {
		return err
	}
	env.Vars = append(env.Vars,
		"HOME="+env.WorkDir,
		"USERPROFILE="+env.WorkDir,
		"GIT_CONFIG_GLOBAL="+cfg,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
	)
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
