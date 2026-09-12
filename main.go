// envbuckets keeps .env files in sync with the current git branch.
package main

import (
	"fmt"
	"os"
)

// Filled by GoReleaser's default ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Printf("envbuckets %s (%s, %s)\n", version, commit, date)
		return
	}
	fmt.Fprintln(os.Stderr, "envbuckets: not implemented yet")
	os.Exit(3)
}
