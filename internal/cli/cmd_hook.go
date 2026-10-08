package cli

// runHook remains a silent no-op until branch switching is implemented.
// Existing installations may still invoke it from post-checkout hooks.
func runHook(_ []string, _ Env) int { return ExitOK }
