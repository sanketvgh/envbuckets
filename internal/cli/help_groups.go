package cli

var groupHelpText = map[string]string{
	"init": `Usage: envbuckets init [--into <bucket>] [--scaffold]

Activate envbuckets here: config, post-checkout hook, .gitignore block, and
a per-scope bootstrap that moves a real .env into a bucket. Safe to re-run.
Checks symlink support first and changes nothing if it is unavailable.

Flags:
  --into <bucket>  bucket to move an existing real .env into (skips the prompt)
  --scaffold       create empty files for every bucket the shared rules reference,
                   in existing scope directories; never overwrites, never activates

Examples:
  envbuckets init --scaffold          fresh clone: hook + empty bucket files
  envbuckets init --into dev          move ./.env into .env.d/dev/.env
`,
	"bucket": `Usage: envbuckets bucket add|rm|list ...

  bucket add <name> [--scope <s> | --all]   create an empty <scope>/.env.d/<name>/.env
  bucket rm <name> [--scope <s>] [--purge]  remove; refuses if referenced or non-empty
  bucket list [--scope <s> | --all]         buckets in a scope, or a bucket x scope matrix

Without --scope or --all, the scope containing the current directory is used.
`,
	"bucket add": `Usage: envbuckets bucket add <name> [--scope <name> | --all]

Create an empty bucket file. Existing files are never overwritten and
missing scope directories are never created. --all covers every scope,
continues past failures, and exits 1 if any scope was not created.

Example:
  envbuckets bucket add staging --all
`,
	"bucket rm": `Usage: envbuckets bucket rm <name> [--scope <name>] [--purge]

Remove a bucket from one scope. Refuses while a rule or pin references it,
while it is active, or while its file is non-empty unless --purge is given
(asks you to type DELETE).
`,
	"bucket list": `Usage: envbuckets bucket list [--scope <name> | --all]

List buckets in one scope, or with --all a matrix of every bucket found on
disk or referenced by rules and local pins, per scope:
  present  file exists (contents are not checked)
  missing  no file
  active   .env points here
  BROKEN   .env points here but the file is missing
`,
	"map": `Usage: envbuckets map add|update|move|rm|list|explain ...

  map add <pattern> <bucket>                       add a rule (before a catch-all *)
  map update <pattern> <bucket>                    change a rule's bucket, keep its priority
  map move <pattern> --before|--after <pattern>    reorder, keep the mapping
  map rm <pattern>                                 remove a rule
  map list                                         rules in priority order, and local pins
  map explain <branch>                             which pin or rule a branch resolves to

Rules are checked top to bottom; the first match wins. A local pin
(envbuckets link) overrides every rule.
`,
	"map add": `Usage: envbuckets map add <pattern> <bucket>

Add a rule. It goes last, or directly before an existing catch-all * so the
catch-all stays last. The bucket must exist in at least one scope.

Example:
  envbuckets map add 'release/*' prod
`,
	"map update": `Usage: envbuckets map update <pattern> <bucket>

Point an existing rule at another bucket. Its priority does not change.

Example:
  envbuckets map update 'release/*' staging
`,
	"map move": `Usage: envbuckets map move <pattern> (--before <pattern> | --after <pattern>)

Move a rule directly before or after another one. Exactly one of --before
or --after is required. The catch-all * must stay last.

Example:
  envbuckets map move 'release/hotfix-*' --before 'release/*'
`,
	"map rm": `Usage: envbuckets map rm <pattern>

Remove a rule.
`,
	"map list": `Usage: envbuckets map list

List rules in priority order (* marks the current branch's match) and the
local branch pins that override them.
`,
	"map explain": `Usage: envbuckets map explain <branch>

Show how a branch resolves without checking it out: its local pin, or the
first matching rule and its priority, the selected bucket, and whether that
bucket's file exists in each scope. The branch does not need to exist.
Read-only.

Example:
  envbuckets map explain feature/login
`,
	"scope": `Usage: envbuckets scope add|rm|purge|list ...

  scope add <path> [--name <n>] [--into <bucket>]  register a directory with its own .env
  scope rm <name> [--purge]                        unregister; data kept unless --purge
  scope purge <path>                               delete data left by an unregistered scope
  scope list                                       registered scopes and their .env state
`,
	"scope add": `Usage: envbuckets scope add <path> [--name <name>] [--into <bucket>]

Register an existing directory as a scope. The name defaults to the
directory name. --into moves an existing real .env into that bucket.
`,
	"scope rm": `Usage: envbuckets scope rm <name> [--purge]

Unregister a scope. Its .env.d/ and .gitignore lines are kept so values
never become visible to git. --purge also deletes them (asks for DELETE).
To erase the data later, use: envbuckets scope purge <path>
`,
	"scope purge": `Usage: envbuckets scope purge <path>

Delete <path>/.env.d/ left behind by an unregistered scope, after you type
DELETE. <path> is relative to the repo root. A managed .env link there is
removed with it; a real .env or a symlink elsewhere is kept and stays
ignored. Paths outside the repo, through symlinks, or still registered
are refused.

Example:
  envbuckets scope purge apps/web
`,
	"scope list": `Usage: envbuckets scope list

List registered scopes with their path and .env state.
`,
}

// groupHelp returns help for init and the bucket, map, and scope groups,
// specific to the subcommand when args names one.
func groupHelp(cmd string, args []string) (string, bool) {
	if len(args) > 0 {
		if text, ok := groupHelpText[cmd+" "+args[0]]; ok {
			return text, true
		}
	}
	text, ok := groupHelpText[cmd]
	return text, ok
}
