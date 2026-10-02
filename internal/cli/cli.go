// Package cli implements the envbuckets commands. Run is the sole entry
// point and takes an Env so tests can drive it with a fake cwd and IO.
package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

// Exit codes.
const (
	ExitOK      = 0
	ExitBlocked = 1
	ExitConfig  = 2
	ExitUsage   = 3
	ExitEnv     = 4
)

// Env is the process environment a command runs in.
type Env struct {
	Cwd     string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Version string
}

// exitError is a failure with a stable class (exit code), a one-line
// description of what happened, and an optional next step.
type exitError struct {
	code int
	msg  string
	next string
}

func (e *exitError) Error() string { return e.msg }

func (e *exitError) then(format string, a ...any) *exitError {
	e.next = fmt.Sprintf(format, a...)
	return e
}

func blocked(format string, a ...any) *exitError {
	return &exitError{code: ExitBlocked, msg: fmt.Sprintf(format, a...)}
}

func configErr(format string, a ...any) *exitError {
	return &exitError{code: ExitConfig, msg: fmt.Sprintf(format, a...)}
}

func usage(format string, a ...any) *exitError {
	return &exitError{code: ExitUsage, msg: fmt.Sprintf(format, a...)}
}

func envErr(format string, a ...any) *exitError {
	return &exitError{code: ExitEnv, msg: fmt.Sprintf(format, a...)}
}

func className(code int) string {
	switch code {
	case ExitBlocked:
		return "blocked"
	case ExitConfig:
		return "config"
	case ExitUsage:
		return "usage"
	default:
		return "environment"
	}
}

const helpText = `envbuckets - your .env switches branches with you.

Usage:
  envbuckets init [--into <bucket>] [--scaffold]  hook + gitignore + bootstrap; optional rule buckets
  envbuckets status                      branch -> rule -> per-scope bucket + symlink health
  envbuckets check [--scope s]           verify structural readiness for the current branch
  envbuckets apply [--scope s] [--dry-run] apply the current branch mapping
  envbuckets use <bucket> [--scope s|--all] repoint scope(s) (manual override)
  envbuckets link <bucket> [--branch b]  pin a branch to a bucket, overrides rules (local)
  envbuckets unlink [--branch b]         remove the pin, rules apply again
  envbuckets uninstall [--purge]         deactivate in this project; data kept by default

  envbuckets bucket add <name> [--scope s|--all]  create empty bucket file(s)
  envbuckets bucket rm <name> [--purge]  refuse if referenced or non-empty
  envbuckets bucket list [--scope s|--all]       buckets in scope, or a scope matrix

  envbuckets map add <pattern> <bucket>  add a rule before a catch-all *
  envbuckets map update <pattern> <bucket>  change bucket, keep priority
  envbuckets map move <pattern> --before|--after <pattern>
  envbuckets map rm <pattern>
  envbuckets map list
  envbuckets map explain <branch>        explain a pin or matching rule

  envbuckets scope add <path> [--name n] register a scope directory
  envbuckets scope rm <name> [--purge]   unregister; data kept unless --purge
  envbuckets scope purge <path>          erase data left by an unregistered scope
  envbuckets scope list

  envbuckets version
`

// Run executes args and returns the process exit code.
func Run(args []string, env Env) int {
	if len(args) == 0 {
		fmt.Fprint(env.Stdout, helpText)
		return ExitOK
	}
	if env.Stdin == nil {
		env.Stdin = strings.NewReader("")
	}
	env.Stdin = bufio.NewReader(env.Stdin)
	cmd, rest := args[0], args[1:]
	if commandHelpRequested(rest) {
		if help, ok := groupHelp(cmd, rest); ok {
			fmt.Fprint(env.Stdout, help)
			return ExitOK
		}
		if help, ok := commandHelp(cmd); ok {
			fmt.Fprint(env.Stdout, help)
			return ExitOK
		}
	}
	var err error
	switch cmd {
	case "version", "--version", "-v":
		fmt.Fprintf(env.Stdout, "envbuckets %s\n", env.Version)
		return ExitOK
	case "help", "--help", "-h":
		fmt.Fprint(env.Stdout, helpText)
		return ExitOK
	case "hook":
		return runHook(rest, env)
	case "init":
		err = runInit(rest, env)
	case "status":
		err = runStatus(rest, env)
	case "check":
		err = runCheck(rest, env)
	case "apply":
		err = runApply(rest, env)
	case "use":
		err = runUse(rest, env)
	case "link":
		err = runLink(rest, env)
	case "unlink":
		err = runUnlink(rest, env)
	case "uninstall":
		err = runUninstall(rest, env)
	case "bucket":
		err = runBucket(rest, env)
	case "map":
		err = runMap(rest, env)
	case "scope":
		err = runScope(rest, env)
	default:
		err = usage("unknown command %q", cmd).then("envbuckets help")
	}
	return report(err, env)
}

func commandHelpRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func commandHelp(cmd string) (string, bool) {
	switch cmd {
	case "status":
		return "Usage: envbuckets status\n\nShow current branch resolution and each scope's actual .env state.\n", true
	case "check":
		return "Usage: envbuckets check [--scope <name>]\n\nCheck structural readiness for the current branch. Defaults to all scopes.\n", true
	case "apply":
		return "Usage: envbuckets apply [--scope <name>] [--dry-run]\n\nApply the current branch's pin or rule to all scopes by default. --dry-run plans without writing.\n", true
	case "use":
		return "Usage: envbuckets use <bucket> [--scope <name> | --all]\n\nTemporarily point one scope, or all scopes with --all, to a bucket.\n", true
	case "link":
		return "Usage: envbuckets link <bucket> [--branch <name>]\n\nPin a local branch to a bucket. Defaults to the current branch.\n", true
	case "unlink":
		return "Usage: envbuckets unlink [--branch <name>]\n\nRemove a local branch pin. Defaults to the current branch.\n", true
	case "uninstall":
		return "Usage: envbuckets uninstall [--purge]\n\nRemove the hook and restore managed .env files in this project. By default, bucket data and config are kept. --purge also removes bucket data, config, and local pins after confirmation.\n", true
	}
	return "", false
}

func report(err error, env Env) int {
	if err == nil {
		return ExitOK
	}
	var ee *exitError
	if !errors.As(err, &ee) {
		ee = envErr("%v", err)
	}
	fmt.Fprintf(env.Stderr, "envbuckets: %s: %s\n", className(ee.code), ee.msg)
	if ee.next != "" {
		fmt.Fprintf(env.Stderr, "  next: %s\n", ee.next)
	}
	return ee.code
}

func newFlags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	var flagArgs, positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}
		flagArgs = append(flagArgs, arg)
		name, _, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		f := fs.Lookup(name)
		if f == nil || hasValue || isBoolFlag(f) || i+1 >= len(args) {
			continue
		}
		i++
		flagArgs = append(flagArgs, args[i])
	}
	if err := fs.Parse(flagArgs); err != nil {
		return nil, usage("%s: %v", fs.Name(), err).then("envbuckets help")
	}
	return positional, nil
}

func isBoolFlag(f *flag.Flag) bool {
	b, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && b.IsBoolFlag()
}

func scopeFlagProvided(fs *flag.FlagSet) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		found = found || f.Name == "scope"
	})
	return found
}

func subcommand(group string, args []string) (string, []string, error) {
	if len(args) == 0 {
		choices := "add|rm|list"
		switch group {
		case "map":
			choices = "add|update|move|rm|list|explain"
		case "scope":
			choices = "add|rm|purge|list"
		}
		return "", nil, usage("%s: missing subcommand", group).then("envbuckets %s %s", group, choices)
	}
	return args[0], args[1:], nil
}

func joinOr(items []string, empty string) string {
	if len(items) == 0 {
		return empty
	}
	return strings.Join(items, ", ")
}
