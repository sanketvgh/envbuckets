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

Commands:
  init       Set up config, buckets, ignores, and the checkout hook; import .env files.
  add        Move local files into the current bucket and leave relative links.
  switch     Use a named bucket, or return to this branch's bucket without a name.
             -c creates a bucket with empty files at the current bucket's paths.
  status     Show the active bucket, branch mapping, and paths needing attention.
  branches   Show the current and rule-matched branches, or check names/patterns.
  uninstall  Restore active files and remove the hook; keep other buckets.

Options:
  -n, --dry-run  Preview init, add, switch, or uninstall without changing files.
  -h, --help     Show this help.
  -v, --version  Show the version.

Rules live in .envbuckets.json. The first matching Git glob wins; otherwise
default is used. Buckets are folders under .env.d/ with files at their repo paths.
Requires Git and symlink support (Windows Developer Mode or symlink privilege).
Set NO_COLOR to a non-empty value to disable terminal color.

The checkout hook runs envbuckets hook [post-checkout arguments] automatically.
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
