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

const banner = `
                 _                _        _
  ___ _ ____   _| |__  _   _  ___| | _____| |_ ___
 / _ \ '_ \ \ / / '_ \| | | |/ __| |/ / _ \ __/ __|
|  __/ | | \ V /| |_) | |_| | (__|   <  __/ |_\__ \
 \___|_| |_|\_/ |_.__/ \__,_|\___|_|\_\___|\__|___/
`

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Printf("envbuckets %s (%s, %s)\n", version, commit, date)
		return
	}

	fmt.Print(banner)
	fmt.Println()
	fmt.Println("Your .env switches branches with you.")
	fmt.Println()
	fmt.Println("Hello Sanket.")
	fmt.Println()
	fmt.Println("WARNING: early alpha, under active development.")
	fmt.Println("The CLI is not implemented yet. Commands and config may change without notice until v1.0.")
	fmt.Printf("\nversion %s\n", version)
}
