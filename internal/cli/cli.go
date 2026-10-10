// Package cli implements the envbuckets commands.
package cli

import (
	"io"

	"github.com/sanketvgh/envbuckets/internal/output"
)

// Exit codes returned by Run.
const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

// Env is the process environment a command runs in.
type Env struct {
	Cwd     string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Version string
	Getenv  func(string) string
}

const helpText = `envbuckets - switch project files with Git branches.

Usage:
  envbuckets version
  envbuckets help
  envbuckets init [-n]
  envbuckets add [-n] <file>...
  envbuckets switch [-n] [-c] [<bucket>]
  envbuckets status
  envbuckets branches [--bucket <name>] [<branch or pattern>...]
  envbuckets uninstall [-n]
  envbuckets hook [post-checkout arguments]
`

// Run executes args and returns the process exit code.
func Run(args []string, env Env) int {
	stdout := output.NewStream(env.Stdout, env.Getenv)
	defer stdout.Close()
	stderr := output.NewStream(env.Stderr, env.Getenv)
	defer stderr.Close()
	env.Stdout, env.Stderr = stdout, stderr
	if len(args) == 0 {
		env.output(false).List("%s", helpText)
		return ExitOK
	}
	switch args[0] {
	case "version", "--version", "-v":
		env.output(false).List("envbuckets %s\n", env.Version)
		return ExitOK
	case "help", "--help", "-h":
		env.output(false).List("%s", helpText)
		return ExitOK
	case "hook":
		return runHook(args[1:], env)
	case "switch":
		return runSwitch(args[1:], env)
	case "init":
		return runInit(args[1:], env)
	case "add":
		return runAdd(args[1:], env)
	case "status":
		return runStatus(args[1:], env)
	case "branches":
		return runBranches(args[1:], env)
	case "uninstall":
		return runUninstall(args[1:], env)
	default:
		env.output(false).Usage("envbuckets <command>")
		return ExitUsage
	}
}

func (env Env) output(hook bool) output.Writer {
	return output.Writer{Out: env.Stdout, Err: env.Stderr, Hook: hook}
}
