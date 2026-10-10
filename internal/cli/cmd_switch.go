package cli

import (
	"fmt"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/config"
	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/gitx"
	"github.com/sanketvgh/envbuckets/internal/switcher"
)

func runSwitch(args []string, env Env) int {
	var bucket string
	dry, positional, create := false, false, false
	for _, arg := range args {
		switch {
		case !positional && (arg == "-n" || arg == "--dry-run"):
			dry = true
		case !positional && arg == "--":
			positional = true
		case !positional && arg == "-c":
			create = true
		case !positional && strings.HasPrefix(arg, "-"):
			env.output(false).Error("unknown option '%s'", arg)
			env.output(false).Usage("envbuckets switch [-n] [-c] [<bucket>]")
			return ExitUsage
		case bucket == "":
			if arg == "" {
				env.output(false).Fatal("invalid bucket name ''")
				return ExitError
			}
			bucket = arg
		default:
			env.output(false).Usage("envbuckets switch [-n] [-c] [<bucket>]")
			return ExitUsage
		}
	}
	if create {
		if bucket == "" {
			return commandUsage(env, "switch [-n] [-c] [<bucket>]", nil)
		}
		return createBucket(env, bucket, dry)
	}
	return switchBucket(env, bucket, dry, false)
}

func switchBucket(env Env, bucket string, dry, hook bool) int {
	root, err := gitx.Root(env.Cwd)
	if err != nil {
		return switchFailure(env, hook, err)
	}
	repo, err := fsx.OpenRepo(root)
	if err != nil {
		return switchFailure(env, hook, err)
	}
	defer repo.Close()
	branch, err := gitx.Branch(root)
	if err != nil {
		return switchFailure(env, hook, err)
	}
	if hook && branch == "" {
		return ExitOK
	}
	cfg, err := config.Load(repo.Root)
	if err != nil {
		return switchFailure(env, hook, fmt.Errorf("invalid %w", err))
	}
	p := switcher.Build(repo, cfg, branch, bucket)
	code := switcher.Execute(repo, p, dry, hook, env.Stdout, env.Stderr)
	if hook {
		return ExitOK
	}
	return code
}

func switchFailure(env Env, hook bool, err error) int {
	if hook {
		env.output(true).Warning("%s; links not changed", err)
		return ExitOK
	}
	env.output(false).Fatal("%s", err)
	return ExitError
}
