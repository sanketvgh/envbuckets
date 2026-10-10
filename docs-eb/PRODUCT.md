# envbuckets product specification

**Status:** implemented for the next release; [EB-10](tickets/EB-10-pre-pr-gate.md) tracks final verification and [EB-07](tickets/EB-07-release.md) tracks release gates. It replaces the scope-based alpha. The alpha's `.envbuckets.toml` is not read or migrated; delete it and run `envbuckets init`.

envbuckets keeps one set of local files per environment and links the right set into your project when you switch Git branches.

## Problem

Local files like `.env`, service-account JSON, and local settings stay out of Git, so switching branches leaves the previous branch's files in place. Developers swap them by hand and sometimes run against the wrong environment. envbuckets makes those files follow the branch and shows which set is active.

## How it works

```text
repo/
├── .envbuckets.json                  # committed: which branch uses which bucket
├── .env -> .env.d/dev/.env           # symlink, git-ignored
├── apps/api/service-account.json     # symlink to .env.d/dev/apps/api/service-account.json
├── .env.d/                           # git-ignored: the real files
│   ├── dev/
│   │   ├── .env
│   │   └── apps/api/service-account.json
│   └── prod/
│       └── .env
└── .git/hooks/post-checkout          # runs envbuckets after each branch switch
```

- A **bucket** is a folder in `.env.d/`. A file's path inside the bucket is its path in the repo. Bucket names start with a letter or digit and may then use letters, digits, `-`, and `_`.
- Working files are relative symlinks, so editing `.env` edits the active bucket's file.
- envbuckets never copies your files. It moves each one into a bucket once and links it; switching only re-points links. There is no copy mode.
- Buckets can hold different files. Above, `apps/api/service-account.json` exists only while `dev` is active.
- Buckets are plain folders. Copy, rename, or delete them with any tool.

## Features

1. **Files follow the branch.** After `git switch` or `git checkout`, the hook links the bucket chosen for the new branch.
2. **Shared rules with Git globs.** `.envbuckets.json` maps branch patterns to buckets, with a default. Patterns use Git's glob rules (`*`, `**`, `?`, `[0-9]`). The file is committed, so the whole team gets the same mapping.
3. **Any local file.** `init` picks up `.env` files; `add` manages any other file at its normal path.
4. **See which branch uses which bucket.** `branches` lists the branches your rules match, with their bucket and the rule that chose it. Name a branch to check it before it exists.
5. **Switch any time, then go back.** `switch <bucket>` uses another bucket right now; plain `switch` goes back to the branch's bucket. `switch -c <bucket>` creates a bucket with empty files at the current bucket's paths.
6. **Clear status.** `status` shows the active bucket, why it was chosen, and anything that needs attention.
7. **Safe by default.** It never overwrites your real files, never prints file contents, and never blocks a checkout.
8. **Easy exit.** `uninstall` moves the active files back to their paths and keeps every other bucket.
9. **Dry runs.** Add `-n` to see exactly what a command would change before it changes anything.
10. **Git-style output.** Messages look and behave like Git's, with color in a terminal.

## Commands

| Command                                               | What it does                                                                                                                                                                                                                                                                                                                                |
| ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `init`                                                | Create `.envbuckets.json` if missing and the default bucket's folder, add the ignore block, install the hook, and move untracked `.env` and `.env.*` files into the default bucket, leaving links in their place. Safe to rerun; on a fresh clone it only adds what is missing.                                                             |
| `add <file>...`                                       | Move each file into the current bucket, leave a link in its place, and git-ignore it.                                                                                                                                                                                                                                                       |
| `switch`                                              | Go back to the current branch's bucket and repair missing links. Use it after `switch <bucket>`, after editing rules, or after fixing a problem.                                                                                                                                                                                            |
| `switch <bucket>`                                     | Use another bucket right now, for testing or debugging. It lasts until you run `switch` or check out a branch.                                                                                                                                                                                                                              |
| `switch -c <bucket>`                                  | Create a bucket with an empty file at each of the current bucket's paths, then switch to it. Fill in the values for the new environment.                                                                                                                                                                                                    |
| `status`                                              | Show the branch, the bucket in use and why, and any problems, laid out like `git status`.                                                                                                                                                                                                                                                   |
| `branches [--bucket <name>] [<branch or pattern>...]` | List the current branch and every branch a rule matches, with its bucket and why, like `git branch`. Other branches just use `default`, so they are left out. A name is checked even if the branch does not exist yet; a pattern such as `'release/**'` lists matching local branches; `--bucket` keeps only branches that use that bucket. |
| `uninstall`                                           | Move each active file from its bucket back over its link, then remove the hook. Other buckets stay in `.env.d/`; the config and the ignore block stay too.                                                                                                                                                                                  |

`init` skips files Git tracks (such as `.env.example`) and anything under `.git/`, `.env.d/`, or ignored folders like `node_modules/`.

`init`, `add`, `switch`, and `uninstall` accept `-n` (`--dry-run`). A dry run prints every change the command would make, changes nothing, and exits with the code the real run would return. `status` and `branches` never change anything, so they have no dry run.

`add` checks every path before changing anything. If one cannot be added, it stops with `fatal:` and adds none, like `git add` with a bad pathspec. `init` instead skips a file it cannot import, with a `warning:`, and imports the rest.

The **current bucket** is the one the links point to. When there are no links yet, it is the branch's bucket. If links point at more than one bucket, `add` and `switch -c` stop and ask you to run `switch` first.

## Choosing a bucket

```json
{
  "$schema": "https://raw.githubusercontent.com/sanketvgh/envbuckets/main/schema/envbuckets.schema.json",
  "default": "dev",
  "rules": [
    { "branch": "main", "bucket": "staging" },
    { "branch": "release/**", "bucket": "prod" },
    { "branch": "hotfix/[0-9]*", "bucket": "prod" }
  ]
}
```

- Rules are checked top to bottom. The first match wins; otherwise `default` is used. Put specific rules before general ones; order handles exceptions, so there is no `!` negation.
- A pattern must match the whole short branch name (`main`, not `refs/heads/main`). Matching is case-sensitive, like Git branch names.
- Edit the file by hand. Unknown keys, invalid bucket names, and broken patterns (such as an unclosed `[`) are errors, so typos fail loudly.
- `$schema` points your editor at the [JSON Schema](https://json-schema.org/), so VS Code and other editors autocomplete keys, show descriptions, and flag mistakes as you type. `init` adds it to new configs. It is optional, and envbuckets never downloads it: the CLI checks the file itself. The shorter examples below leave it out.
- Check a pattern before committing it: `envbuckets branches release/2.0/rc1` shows which rule wins.
- If a branch's bucket does not exist, the hook and plain `switch` link the default bucket instead and warn, so a missing bucket never leaves another environment, such as `prod`, linked. If the default is missing too, the links stay as they are, with a warning. `switch <bucket>` naming a missing bucket stops with `fatal:`, because you asked for that exact bucket.
- On a detached HEAD (a tag, a commit, a bisect), nothing changes. `switch <bucket>` still works; plain `switch` stops, because there is no branch bucket to go back to.

### Patterns

Patterns follow Git's own glob rules, the ones `.gitignore` and `includeIf "onbranch:..."` in `git config` use:

| Pattern         | Rule                                                                                               | Matches                          | Does not match    |
| --------------- | -------------------------------------------------------------------------------------------------- | -------------------------------- | ----------------- |
| `main`          | No wildcards: exact name                                                                           | `main`                           | `main-2`          |
| `release/*`     | `*` is any text except `/`                                                                         | `release/1.2`                    | `release/1.2/rc1` |
| `release/**`    | `**` crosses `/`                                                                                   | `release/1.2`, `release/1.2/rc1` | `release`         |
| `release/`      | A trailing `/` means everything under it, same as `release/**`                                     | `release/1.2/rc1`                | `release`         |
| `**/hotfix`     | `**/` is zero or more folders                                                                      | `hotfix`, `team/hotfix`          | `hotfix-2`        |
| `v?.?`          | `?` is one character except `/`                                                                    | `v1.2`                           | `v1.10`           |
| `hotfix/[0-9]*` | `[...]` is one character from a set; `[!...]` or `[^...]` negates; classes like `[[:digit:]]` work | `hotfix/42-login`                | `hotfix/login`    |
| `wip\*`         | `\` makes the next character literal                                                               | `wip*`                           | `wip-1`           |

As in Git, there are no `{a,b}` braces; write two rules instead. JSON needs its own escaping, so `wip\*` is written `"wip\\*"`.

### Rule examples

Each example shows a config, then `envbuckets branches <name>...` checking what it does.

#### Git Flow

`main` gets production values, release and hotfix branches get staging, and everything else uses `dev`.

```json
{
  "default": "dev",
  "rules": [
    { "branch": "main", "bucket": "prod" },
    { "branch": "release/", "bucket": "staging" },
    { "branch": "hotfix/", "bucket": "staging" }
  ]
}
```

```console
$ envbuckets branches main develop release/2.4 hotfix/login-crash feature/cart
  main                prod     (rule 'main')
  develop             dev      (default)
  release/2.4         staging  (rule 'release/')
  hotfix/login-crash  staging  (rule 'hotfix/')
  feature/cart        dev      (default)
```

#### Exceptions first

Release candidates use staging; other release branches use prod. The specific rule comes first because the first match wins.

```json
{
  "default": "dev",
  "rules": [
    { "branch": "release/*-rc*", "bucket": "staging" },
    { "branch": "release/**", "bucket": "prod" }
  ]
}
```

```console
$ envbuckets branches release/2.0-rc1 release/2.0
  release/2.0-rc1  staging  (rule 'release/*-rc*')
  release/2.0      prod     (rule 'release/**')
```

With the rules the other way round, `release/**` would win for both.

#### Ticket IDs

Branches for the payments project start with a ticket ID, with or without a `feature/` prefix. `**/` covers both, and `[0-9]` skips names like `PAY-docs`.

```json
{
  "default": "dev",
  "rules": [{ "branch": "**/PAY-[0-9]*", "bucket": "payments" }]
}
```

```console
$ envbuckets branches PAY-142-refunds feature/PAY-7-webhooks PAY-docs
  PAY-142-refunds         payments  (rule '**/PAY-[0-9]*')
  feature/PAY-7-webhooks  payments  (rule '**/PAY-[0-9]*')
  PAY-docs                dev       (default)
```

Matching is case-sensitive, so `pay-142-refunds` would get `dev`.

#### Experiments anywhere

Anyone's `spike-` branch uses a sandbox bucket, at any depth.

```json
{
  "default": "dev",
  "rules": [{ "branch": "**/spike-*", "bucket": "sandbox" }]
}
```

```console
$ envbuckets branches spike-cache alice/spike-cache alice/fix-spikes
  spike-cache        sandbox  (rule '**/spike-*')
  alice/spike-cache  sandbox  (rule '**/spike-*')
  alice/fix-spikes   dev      (default)
```

#### Several names, one bucket

There are no braces, so list each name.

```json
{
  "default": "dev",
  "rules": [
    { "branch": "main", "bucket": "staging" },
    { "branch": "master", "bucket": "staging" },
    { "branch": "trunk", "bucket": "staging" }
  ]
}
```

### Many branches

- **Rules do not grow with branches.** A repo with 1000 branches still needs only a few rules.
- **Branches without a rule use `default`.** Every branch resolves to exactly one bucket, so there is no "no env" state where a branch quietly keeps the previous branch's files.
- **The list shows only what you configured.** Plain `branches` lists your current branch and the branches a rule matches. Every other branch uses `default`, so listing them would add nothing:

  ```console
  $ envbuckets branches
  * feature/login  dev   (default)
    hotfix/42      prod  (rule 'hotfix/[0-9]*')
    release/1.2    prod  (rule 'release/**')
    release/1.3    prod  (rule 'release/**')
  ```

- **Checkouts stay fast.** The hook only looks at the branch being checked out, so the number of branches does not matter. `branches` reads every branch with one Git call.
- **Filters show every match**, including branches that use `default`:
  - `envbuckets branches --bucket prod` shows only the branches that use `prod`.
  - `envbuckets branches 'release/**'` shows only local branches matching the pattern. Quote patterns so your shell does not expand them.
  - `envbuckets branches hotfix/login` checks one name, whether or not the branch exists.
  - `envbuckets branches '**'` lists every local branch, if you ever need the full list.
  - Piped output is plain text, so `| grep` and `| less` work as usual.

Branch names cannot contain `*`, `?`, or `[` (see `git check-ref-format`), so an argument with any of them is always a pattern and an argument without them is always a name.

## What a switch does

- Links every file in the target bucket at its repo path.
- Removes envbuckets links whose path is not in the target bucket. The file stays in its bucket.
- Skips and reports any path where a real file, or a symlink that envbuckets did not create, is in the way.
- Ignores clutter in buckets: `.DS_Store`, `Thumbs.db`, `desktop.ini`, and names ending in `~` or `.swp`.
- Replaces each link with a temporary symlink and a rename. Each rename is atomic on Unix; Windows does not guarantee atomic rename. A multi-file switch is not a transaction. Skipped paths do not stop the others, and running `switch` again finishes the job.

`switch` prints one line, like `git switch`. Links for files the target bucket lacks are removed quietly, the same way Git removes files the target branch does not have; `switch -n` shows them first. Each path that cannot be linked gets an `error:` line. When a switch leaves you on a bucket other than the branch's, a `hint:` line says how to go back.

The hook runs this on branch checkouts only, not `git checkout -- <file>`. It never prompts, prints one line when the bucket changes, and prints an `error:` line for each path it could not link. If the branch's bucket is missing, it links the default bucket and warns. If the config is broken, it changes nothing and prints one warning. If envbuckets is not installed, it does nothing. It always exits 0.

## Git ignore

`init` and `add` keep a marked block in `.gitignore` that lists `.env.d/` and every managed path, so the files are never committed:

```gitignore
# >>> envbuckets >>>
.env.d/
.env
apps/web/.env.local
# <<< envbuckets <<<
```

Paths Git already ignores are not added again. `status` warns about any linked path Git does not ignore. `uninstall` keeps the block because the files are still on disk.

## Safety rules

1. Never overwrite or delete a real file in your working tree.
2. Never overwrite or delete a file inside a bucket. `init` and `add` move files in, `switch -c` creates empty ones, and only `uninstall` moves the active files back out.
3. Never read, copy, print, or parse file contents. envbuckets only moves files, creates empty ones, and creates links.
4. Never block Git. The hook always exits 0.
5. Stay inside the repo. Refuse paths outside it, under `.git/` or `.env.d/`, or through symlinked folders, and refuse files Git tracks. Temporary links go next to the path they replace.
6. Work offline and write nothing outside the repo.

## Requirements

- Git.
- Symlink support. On Windows, turn on Developer Mode. `init` checks this and stops with a clear message if links cannot be created.
- A hooks folder inside the repo. `init` asks Git where hooks live, so a `core.hooksPath` inside the repo works. If it points outside the repo (a folder shared by many repos), `init` stops with `fatal:` instead of changing hooks for all of them.

## Output

Messages follow Git's conventions, so they read like Git.

- **Quiet on success.** Commands that act on paths you name stay silent when they work, like `git add`. Commands that find paths themselves (`init`, `uninstall`) list each one, like `git clean`.
- **Familiar summaries.** `Switched to bucket 'prod'`, `Switched to a new bucket 'prod'`, `Already on bucket 'dev'`, `Initialized envbuckets in /work/shop/.env.d/`.
- **Prefixes carry the severity:**
  - `fatal:` the command stopped and changed nothing.
  - `error:` one path failed; the rest went ahead, and the exit code is 1.
  - `warning:` worth a look; the command still succeeded.
  - `hint:` what to do next, written as a full sentence.
- **Wording.** After `fatal:`, `error:`, or `warning:`, text starts lowercase, has no trailing period, says what went wrong first, and puts names and paths in single quotes: `fatal: cannot add '.env.example': it is tracked by Git`. Commands go in double quotes: `hint: Create it with "envbuckets switch -c prod".`
- **Dry runs** print one `Would ...` line per change and nothing else, like `git clean -n`.
- **`status`** looks like `git status`: the branch, the bucket line, a section per kind of problem with a hint in parentheses, and a one-line summary.
- **Hook output** starts every line with `envbuckets: `, the way Git marks server messages with `remote: `.
- **Streams.** `status`, `branches`, and dry-run lists go to stdout. Everything else, including `Switched to ...`, goes to stderr, as in Git.
- **Usage errors** print `usage: envbuckets <command> ...` and exit 2.

Exit codes: `0` success, `1` failed or only partly done, `2` usage error.

In a terminal, `fatal:` and `error:` are red, `warning:` and `hint:` are yellow, `status` shows the bucket in use in green and problem paths in red, like `git status`, and `branches` shows the current branch in green, like `git branch`. Color is off when output is piped or redirected, or when `NO_COLOR` is set to any non-empty value. The prefixes carry the meaning, so nothing depends on color. Styling uses [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Doing it by hand

- **New empty bucket:** create a folder in `.env.d/`.
- **Rename a bucket:** rename the folder, update `.envbuckets.json`, then run `envbuckets switch`.
- **Delete a bucket:** switch away from it, then delete the folder.
- **Stop managing a file:** move it from the active bucket over its link (`mv .env.d/dev/.env .env` replaces the link, not its target), then delete that path from every other bucket.

## Not in the first release

Add these only when real use asks for them:

- `--json` output for scripts and agents.
- Registering the schema with [SchemaStore](https://www.schemastore.org/), so editors find it even without `$schema`.
- Commands for editing rules or for renaming, deleting, and purging buckets.
- A shell prompt helper, buckets matched by branch name, and path exclusions.
- Per-developer overrides and worktree support.

## Acceptance criteria

1. `init` in a repo with `.env`, `apps/web/.env.local`, and a tracked `.env.example` moves the first two into `dev` unchanged, links them, git-ignores them, and leaves `.env.example` alone.
2. `git switch` to a branch matching a rule links that rule's bucket; any other branch gets `default`.
3. The JSON Schema and the CLI accept the same configs: every config in this document passes both, and both reject unknown keys and invalid bucket names. `init` writes `$schema` into new configs, and no command fetches it.
4. Every row of the Patterns table holds: `*` and `?` stop at `/`, `**` crosses it, a trailing `/` means everything under it, and sets, classes, and `\` escapes work. Broken patterns are config errors.
5. `branches` lists the current branch and every rule-matched branch with its bucket and reason, leaves out other branches that just use the default, marks the current branch and any temporary bucket, and checks names that are not branches yet. Pattern arguments (`'**'` for every branch) and `--bucket` show every match, and a repo with 1000 branches is read with one Git call.
6. Switching to a bucket without a file removes that file's link and keeps the file in its bucket.
7. A real file at a managed path is never touched; other paths still switch, and `status` reports it.
8. A missing bucket, broken config, detached HEAD, file checkout, or missing binary never fails or blocks a checkout. When a branch's bucket is missing, the hook and plain `switch` link the default bucket and warn; coming from `prod`, nothing from `prod` stays linked.
9. `switch -c prod` creates `prod` with an empty file at each of the current bucket's paths and links them. `switch <bucket>` works at any time, and plain `switch` or the next branch checkout goes back to the branch's bucket.
10. `uninstall` moves each active file back over its link, removes only envbuckets' hook lines, and leaves the other buckets in `.env.d/`. Running `init` afterwards imports the same files again without collisions.
11. No command output contains a marker string placed in bucket files, and every command works offline.
12. No command reads file contents: on Linux and macOS, every command still works when the managed files are unreadable (`chmod 000`).
13. `-n` on `init`, `add`, `switch`, and `uninstall` lists the planned changes, leaves the repo byte-identical, and returns the same exit code as the real run.
14. Color appears only in a terminal, is off with `NO_COLOR` or piped output, and every message has the same text with or without it.
15. Messages follow the Output rules: Git's prefixes and wording, silent `add`, `Would ...` dry runs, `git status` and `git branch` layouts, and the stdout/stderr split.

Implementation tickets live in [`tickets/`](tickets/README.md).

## Sample runs

These follow one project with `.env`, `apps/web/.env.local`, and a tracked `.env.example`. Output is shown without color.

### First setup

Preview first, then run it for real. The tracked `.env.example` is left alone.

```console
$ envbuckets init -n
Would create .envbuckets.json
Would create bucket 'dev'
Would add .env to bucket 'dev'
Would add apps/web/.env.local to bucket 'dev'
Would update .gitignore
Would install .git/hooks/post-checkout

$ envbuckets init
Adding .env to bucket 'dev'
Adding apps/web/.env.local to bucket 'dev'
Initialized envbuckets in /work/shop/.env.d/
```

### Create a prod bucket

```console
$ envbuckets switch -c prod
Switched to a new bucket 'prod'
hint: This branch uses 'dev'. Run "envbuckets switch" to go back.

# .env and apps/web/.env.local now point at prod's empty files. Fill in the prod values.

$ envbuckets switch
Switched to bucket 'dev'
```

### Map branches to buckets

Edit `.envbuckets.json` and commit it so the team shares the rules:

```json
{
  "default": "dev",
  "rules": [{ "branch": "release/*", "bucket": "prod" }]
}
```

```console
$ git switch -c release/1.2
Switched to a new branch 'release/1.2'
envbuckets: Switched to bucket 'prod' (rule 'release/*')

$ git switch main
Switched to branch 'main'
envbuckets: Switched to bucket 'dev' (default)

$ git switch -c feature/login
Switched to a new branch 'feature/login'
```

`feature/login` also uses `dev`, so envbuckets prints nothing.

### See which branch uses which bucket

```console
$ envbuckets branches
* feature/login  dev   (default)
  release/1.2    prod  (rule 'release/*')
```

`main` is not listed: it has no rule, so it uses the default like any other branch.

Name branches to check them before they exist. `*` stops at `/`, as in Git, so a nested release branch falls through to the default:

```console
$ envbuckets branches release/2.0 release/2.0/rc1
  release/2.0      prod  (rule 'release/*')
  release/2.0/rc1  dev   (default)
```

Change the rule to `release/**` and check again:

```console
$ envbuckets branches release/2.0/rc1
  release/2.0/rc1  prod  (rule 'release/**')
```

With many branches, narrow the list by bucket or by pattern:

```console
$ envbuckets branches --bucket dev
* feature/login  dev  (default)
  main           dev  (default)

$ envbuckets branches 'release/**'
  release/1.2  prod  (rule 'release/**')
```

### Try another bucket, then go back

Switch whenever you like. Plain `switch`, or the next branch checkout, takes you back to the branch's bucket.

```console
$ envbuckets switch prod
Switched to bucket 'prod'
hint: This branch uses 'dev'. Run "envbuckets switch" to go back.

$ envbuckets status
On branch feature/login
Using bucket 'prod', but this branch uses 'dev' (default)
  (use "envbuckets switch" to go back to 'dev')

all 2 files linked

$ envbuckets branches
* feature/login  dev   (default), using 'prod' for now
  release/1.2    prod  (rule 'release/**')

$ envbuckets switch
Switched to bucket 'dev'
```

### Manage a file that is not `.env`

`add` prints nothing when it works, like `git add`. `prod` has no service account yet, so switching removes that link quietly; the dry run shows it first.

```console
$ envbuckets add apps/api/service-account.json

$ envbuckets switch -n prod
Would link .env to bucket 'prod'
Would remove link apps/api/service-account.json (not in bucket 'prod')
Would link apps/web/.env.local to bucket 'prod'

$ envbuckets switch prod
Switched to bucket 'prod'
hint: This branch uses 'dev'. Run "envbuckets switch" to go back.

$ mv ~/Downloads/prod-service-account.json apps/api/service-account.json
$ envbuckets add apps/api/service-account.json

$ envbuckets switch
Switched to bucket 'dev'
```

### A teammate's fresh clone

The config and the `.gitignore` block are committed, so `init` only adds the local parts.

```console
$ git clone git@github.com:acme/shop.git && cd shop
Cloning into 'shop'...
$ envbuckets init
Initialized envbuckets in /work/shop/.env.d/
hint: No local files found. Create them, then run "envbuckets add <file>".

$ cp .env.example .env    # then fill in your own values
$ envbuckets add .env
```

### A branch whose bucket is missing

The checkout always succeeds. envbuckets falls back to the default bucket and says why. The teammate was already on `dev`, so the links do not change:

```console
$ git switch release/1.2
branch 'release/1.2' set up to track 'origin/release/1.2'.
Switched to a new branch 'release/1.2'
envbuckets: warning: bucket 'prod' does not exist; using 'dev' (default)
envbuckets: hint: Create it with "envbuckets switch -c prod".

$ envbuckets status
On branch release/1.2
Using bucket 'dev' (default), because 'prod' (rule 'release/**') does not exist
  (use "envbuckets switch -c prod" to create it)

1 file linked

$ envbuckets branches
* release/1.2  prod  (rule 'release/**', missing; falls back to 'dev')

$ envbuckets switch -c prod
Switched to a new bucket 'prod'
```

The fallback matters most when you come from `prod`. With the rules from [A team with more rules](#a-team-with-more-rules), someone without a `staging` bucket leaves a release branch:

```console
$ git switch main
Switched to branch 'main'
envbuckets: warning: bucket 'staging' does not exist; using 'dev' (default)
envbuckets: Switched to bucket 'dev' (default)
envbuckets: hint: Create it with "envbuckets switch -c staging".
```

Nothing from `prod` stays linked on `main`.

### A file in the way

Some tools save by replacing the link with a real file. envbuckets never overwrites it; `status` explains how to keep the edit.

```console
$ sed -i 's/LOG_LEVEL=info/LOG_LEVEL=debug/' .env    # GNU sed replaces the link
$ envbuckets status
On branch main
Using bucket 'dev' (default)

Files not linked:
  (to keep a file, move it to the same path under .env.d/dev/, then use "envbuckets switch")
        real file:   .env

2 of 3 files linked

$ mv .env .env.d/dev/.env
$ envbuckets switch
Already on bucket 'dev'

$ envbuckets status
On branch main
Using bucket 'dev' (default)

all 3 files linked
```

### A detached HEAD

Checking out a tag or a commit leaves the links alone. There is no branch, so there is no branch bucket to follow.

```console
$ git checkout v1.4.0
Note: switching to 'v1.4.0'.
...
HEAD is now at 3f2a1c9 Release 1.4.0

$ envbuckets status
HEAD detached at v1.4.0
Using bucket 'dev'
  (links stay as they are on a detached HEAD; use "envbuckets switch <bucket>" to pick one)

all 3 files linked

$ envbuckets switch
fatal: HEAD is detached, so there is no branch bucket to go back to
hint: Name a bucket instead, such as "envbuckets switch prod".

$ envbuckets switch prod
Switched to bucket 'prod'
```

### A broken config

A typo in `.envbuckets.json` never blocks Git. The hook warns and leaves the links alone; other commands stop until the file is fixed.

```console
$ git switch release/1.2
Switched to branch 'release/1.2'
envbuckets: warning: invalid .envbuckets.json: unknown key 'bukcet'; links not changed

$ envbuckets status
fatal: invalid .envbuckets.json: unknown key 'bukcet'
```

### A team with more rules

A bigger project uses these rules:

```json
{
  "default": "dev",
  "rules": [
    { "branch": "main", "bucket": "staging" },
    { "branch": "release/*-rc*", "bucket": "staging" },
    { "branch": "release/**", "bucket": "prod" },
    { "branch": "hotfix/[0-9]*", "bucket": "prod" }
  ]
}
```

```console
$ git switch main
Switched to branch 'main'
envbuckets: Switched to bucket 'staging' (rule 'main')

$ git switch -c release/3.0-rc1
Switched to a new branch 'release/3.0-rc1'

$ git switch -c release/3.0
Switched to a new branch 'release/3.0'
envbuckets: Switched to bucket 'prod' (rule 'release/**')

$ git switch -c hotfix/88-payment-timeout
Switched to a new branch 'hotfix/88-payment-timeout'

$ git switch -c hotfix/docs-typo
Switched to a new branch 'hotfix/docs-typo'
envbuckets: Switched to bucket 'dev' (default)
```

`release/3.0-rc1` stays on `staging` and `hotfix/88-payment-timeout` stays on `prod`, so envbuckets prints nothing for them. `hotfix/docs-typo` has no digit after `hotfix/`, so it falls to the default.

### A repo with 1000 branches

Same rules, 1000 local branches, on `search/fuzzy-match`. Plain `branches` shows only what the rules pick up, plus where you are:

```console
$ envbuckets branches
  hotfix/88-payment-timeout  prod     (rule 'hotfix/[0-9]*')
  main                       staging  (rule 'main')
  release/2.9                prod     (rule 'release/**')
  release/3.0                prod     (rule 'release/**')
  release/3.0-rc1            staging  (rule 'release/*-rc*')
* search/fuzzy-match         dev      (default)

$ envbuckets branches --bucket staging
  main             staging  (rule 'main')
  release/3.0-rc1  staging  (rule 'release/*-rc*')

$ envbuckets branches '**/*payment*'
  bugfix/payment-retry       dev   (default)
  hotfix/88-payment-timeout  prod  (rule 'hotfix/[0-9]*')
```

The last one finds every branch with "payment" in its last part, including one that just uses the default.

### Mistakes

```console
$ envbuckets add .env.example
fatal: cannot add '.env.example': it is tracked by Git
hint: Copy it to an untracked file such as '.env', then add that.

$ envbuckets switch -c prod
fatal: a bucket named 'prod' already exists

$ envbuckets switch staging
fatal: no bucket named 'staging'
hint: Create it with "envbuckets switch -c staging".

$ envbuckets switch -x
error: unknown option '-x'
usage: envbuckets switch [-n] [-c] [<bucket>]
```

### Leave envbuckets

`uninstall` lists each file it touches, like `git clean`.

```console
$ envbuckets uninstall -n
Would move .env.d/dev/.env to .env
Would move .env.d/dev/apps/api/service-account.json to apps/api/service-account.json
Would move .env.d/dev/apps/web/.env.local to apps/web/.env.local
Would remove the envbuckets hook from .git/hooks/post-checkout

$ envbuckets uninstall
Moving .env.d/dev/.env to .env
Moving .env.d/dev/apps/api/service-account.json to apps/api/service-account.json
Moving .env.d/dev/apps/web/.env.local to apps/web/.env.local
Removing the envbuckets hook from .git/hooks/post-checkout
hint: Your other buckets are still in .env.d/. Delete it when you no longer need them.
```
