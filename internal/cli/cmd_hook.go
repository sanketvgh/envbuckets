package cli

// runHook ignores file checkouts and malformed calls, and never blocks Git.
func runHook(args []string, env Env) int {
	if len(args) == 4 && args[0] == "post-checkout" {
		args = args[1:]
	}
	if len(args) != 3 || args[2] != "1" {
		return ExitOK
	}
	return switchBucket(env, "", false, true)
}
