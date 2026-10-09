// Package cli implements the envbuckets commands.
package cli

import (
	"fmt"
	"io"
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
}

const helpText = `envbuckets - switch project files with Git branches.

Usage:
  envbuckets version
  envbuckets help
  envbuckets init [-n]
  envbuckets add [-n] <file>...
  envbuckets switch [-n] [-c] [<bucket>]
  envbuckets hook [post-checkout arguments]
`

// Run executes args and returns the process exit code.
func Run(args []string, env Env) int {
	if len(args) == 0 {
		fmt.Fprint(env.Stdout, helpText)
		return ExitOK
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintf(env.Stdout, "envbuckets %s\n", env.Version)
		return ExitOK
	case "help", "--help", "-h":
		fmt.Fprint(env.Stdout, helpText)
		return ExitOK
	case "hook":
		return runHook(args[1:], env)
	case "switch":
		return runSwitch(args[1:], env)
	case "init":
		return runInit(args[1:], env)
	case "add":
		return runAdd(args[1:], env)
	default:
		fmt.Fprintln(env.Stderr, "usage: envbuckets <command>")
		return ExitUsage
	}
}
