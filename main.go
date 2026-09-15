// envbuckets keeps .env files in sync with the current git branch.
package main

import (
	"fmt"
	"os"

	"github.com/sanketvgh/envbuckets/internal/cli"
)

// Filled by GoReleaser's default ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "envbuckets: %v\n", err)
		os.Exit(cli.ExitEnv)
	}
	os.Exit(cli.Run(os.Args[1:], cli.Env{
		Cwd:     cwd,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Version: fmt.Sprintf("%s (%s, %s)", version, commit, date),
	}))
}
