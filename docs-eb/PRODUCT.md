# envbuckets product specification

**Status:** plan for the next release. It replaces the scope-based alpha. The alpha's `.envbuckets.toml` is not read or migrated; delete it and run `envbuckets init`.

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

- A **bucket** is a folder in `.env.d/`. A file's path inside the bucket is its path in the repo. Bucket names use letters, digits, `-`, and `_`.
- Working files are relative symlinks, so editing `.env` edits the active bucket's copy.
- Buckets can hold different files. Above, `apps/api/service-account.json` exists only while `dev` is active.
- Buckets are plain folders. Copy, rename, or delete them with any tool.

## Features

1. **Files follow the branch.** After `git switch` or `git checkout`, the hook links the bucket chosen for the new branch.
2. **Shared rules.** `.envbuckets.json` maps branch patterns to buckets, with a default. It is committed, so the whole team gets the same mapping.
3. **Any local file.** `init` picks up `.env` files; `add` manages any other file at its normal path.
4. **Manual switching.** `switch <bucket>` tries another bucket; `switch -c <bucket>` creates one from the current bucket.
5. **Clear status.** `status` shows the active bucket, why it was chosen, and anything that needs attention.
6. **Safe by default.** It never overwrites your real files, never prints file contents, and never blocks a checkout.
7. **Easy exit.** `uninstall` turns links back into real files and keeps your data.
8. **Dry runs.** Add `-n` to see exactly what a command would change before it changes anything.
9. **Git-style output.** Messages look and behave like Git's, with color in a terminal.

## Commands

| Command              | What it does                                                                                                                                                                                                                                               |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `init`               | Create `.envbuckets.json` if missing and the default bucket's folder, add the ignore block, install the hook, and move untracked `.env` and `.env.*` files into the default bucket as links. Safe to rerun; on a fresh clone it only adds what is missing. |
| `add <file>...`      | Copy each file into the current bucket, replace it with a link, and git-ignore it.                                                                                                                                                                         |
| `switch`             | Link the current branch's bucket now and repair missing links. Run it after editing rules or fixing a problem.                                                                                                                                             |
| `switch <bucket>`    | Link another bucket now. The next branch switch follows the rules again.                                                                                                                                                                                   |
| `switch -c <bucket>` | Create a bucket by copying the current bucket's files, then switch to it. Edit the copies for the new environment.                                                                                                                                         |
| `status`             | Show the branch, the bucket in use and why, and any problems, laid out like `git status`.                                                                                                                                                                  |
| `uninstall`          | Remove the hook and replace each link with a real copy of its file. Keep `.env.d/`, the config, and the ignore block.                                                                                                                                      |

`init` skips files Git tracks (such as `.env.example`) and anything under `.git/`, `.env.d/`, or ignored folders like `node_modules/`.

`init`, `add`, `switch`, and `uninstall` accept `-n` (`--dry-run`). A dry run prints every change the command would make, changes nothing, and exits with the code the real run would return. `status` never changes anything, so it has no dry run.

`add` checks every path before changing anything. If one cannot be added, it stops with `fatal:` and adds none, like `git add` with a bad pathspec. `init` instead skips a file it cannot import, with a `warning:`, and imports the rest.

The **current bucket** is the one the links point to. When there are no links yet, it is the branch's bucket. If links point at more than one bucket, `add` and `switch -c` stop and ask you to run `switch` first.

## Choosing a bucket

```json
{
  "default": "dev",
  "rules": [
    { "branch": "main", "bucket": "staging" },
    { "branch": "release/*", "bucket": "prod" }
  ]
}
```

- Rules are checked top to bottom. The first match wins; otherwise `default` is used.
- `*` matches any text, including `/`. Everything else matches exactly.
- Edit the file by hand. Unknown keys and invalid values are errors, so typos fail loudly.
- If the chosen bucket folder does not exist, nothing changes and you get a warning.
- On a detached HEAD (rebase, bisect), nothing changes.

## What a switch does

- Links every file in the target bucket at its repo path.
- Removes envbuckets links whose path is not in the target bucket. The file stays in its bucket.
- Skips and reports any path where a real file, or a symlink that envbuckets did not create, is in the way.
- Ignores clutter in buckets: `.DS_Store`, `Thumbs.db`, `desktop.ini`, and names ending in `~` or `.swp`.
- Replaces each link atomically. Skipped paths do not stop the others, and running `switch` again finishes the job.

`switch` prints one line, like `git switch`. Links for files the target bucket lacks are removed quietly, the same way Git removes files the target branch does not have; `switch -n` shows them first. Each path that cannot be linked gets an `error:` line.

The hook runs this on branch checkouts only, not `git checkout -- <file>`. It never prompts, prints one line when the bucket changes, and prints an `error:` line for each path it could not link. It does nothing if envbuckets is not installed or the config is broken, and it always exits 0.

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
2. Never overwrite or delete a file inside a bucket. `init`, `add`, and `switch -c` only create new bucket files.
3. Never print, log, or parse file contents. Bytes are copied only by `init`, `add`, `switch -c`, and `uninstall`.
4. Never block Git. The hook always exits 0.
5. Stay inside the repo. Refuse paths outside it, under `.git/` or `.env.d/`, or through symlinked folders, and refuse files Git tracks. Temporary files go next to the file they replace.
6. Work offline and write nothing outside the repo.

## Requirements

- Git.
- Symlink support. On Windows, turn on Developer Mode. `init` checks this and stops with a clear message if links cannot be created.

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
- **Streams.** `status` and dry-run lists go to stdout. Everything else, including `Switched to ...`, goes to stderr, as in Git.
- **Usage errors** print `usage: envbuckets <command> ...` and exit 2.

Exit codes: `0` success, `1` failed or only partly done, `2` usage error.

In a terminal, `fatal:` and `error:` are red, `warning:` and `hint:` are yellow, and `status` shows the bucket in use in green and problem paths in red, like `git branch` and `git status`. Color is off when output is piped or redirected, or when `NO_COLOR` is set. The prefixes carry the meaning, so nothing depends on color.

## Doing it by hand

- **New empty bucket:** create a folder in `.env.d/`.
- **Rename a bucket:** rename the folder, update `.envbuckets.json`, then run `envbuckets switch`.
- **Delete a bucket:** switch away from it, then delete the folder.
- **Stop managing a file:** replace the link with a copy of the file, then delete the file from every bucket.

## Not in the first release

Add these only when real use asks for them:

- Copy mode for machines without symlinks.
- `--json` output for scripts and agents.
- Commands for editing rules or for renaming, deleting, and purging buckets.
- A shell prompt helper, buckets matched by branch name, and path exclusions.
- Per-developer overrides and worktree support.

## Acceptance criteria

1. `init` in a repo with `.env`, `apps/web/.env.local`, and a tracked `.env.example` moves the first two into `dev`, links them with identical bytes, git-ignores them, and leaves `.env.example` alone.
2. `git switch` to a branch matching a rule links that rule's bucket; any other branch gets `default`.
3. Switching to a bucket without a file removes that file's link and keeps the file in its bucket.
4. A real file at a managed path is never touched; other paths still switch, and `status` reports it.
5. A missing bucket, broken config, detached HEAD, file checkout, or missing binary never fails or blocks a checkout.
6. `switch -c prod` creates `prod` from the current bucket's files and links it. `switch <bucket>` lasts until the next branch switch.
7. `uninstall` leaves real files with the active bucket's bytes, removes only envbuckets' hook lines, and keeps `.env.d/`.
8. No command output contains a marker string placed in bucket files, and every command works offline.
9. `-n` on `init`, `add`, `switch`, and `uninstall` lists the planned changes, leaves the repo byte-identical, and returns the same exit code as the real run.
10. Color appears only in a terminal, is off with `NO_COLOR` or piped output, and every message has the same text with or without it.
11. Messages follow the Output rules: Git's prefixes and wording, silent `add`, `Would ...` dry runs, `git status` layout, and the stdout/stderr split.

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

# .env and apps/web/.env.local are now prod's copies of dev's files. Replace the values.

$ envbuckets switch dev
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

### Manage a file that is not `.env`

`add` prints nothing when it works, like `git add`. `prod` has no service account yet, so switching removes that link quietly; the dry run shows it first.

```console
$ envbuckets add apps/api/service-account.json

$ envbuckets switch -n prod
Would link .env to bucket 'prod'
Would link apps/web/.env.local to bucket 'prod'
Would remove link apps/api/service-account.json (not in bucket 'prod')

$ envbuckets switch prod
Switched to bucket 'prod'

$ cp ~/Downloads/prod-service-account.json apps/api/service-account.json
$ envbuckets add apps/api/service-account.json
```

### Check where you are

```console
$ envbuckets status
On branch feature/login
Using bucket 'prod', but this branch uses 'dev' (default)
  (use "envbuckets switch" to switch to 'dev')

all 3 files linked

$ envbuckets switch
Switched to bucket 'dev'
```

The next branch switch would also have gone back to the rules.

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

The checkout always succeeds. envbuckets leaves the links alone and says why.

```console
$ git switch release/1.2
branch 'release/1.2' set up to track 'origin/release/1.2'.
Switched to a new branch 'release/1.2'
envbuckets: warning: bucket 'prod' does not exist; links not changed
envbuckets: hint: Create it with "envbuckets switch -c prod".

$ envbuckets switch -c prod
Switched to a new bucket 'prod'
```

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
Would replace link .env with a real file
Would replace link apps/web/.env.local with a real file
Would replace link apps/api/service-account.json with a real file
Would remove the envbuckets hook from .git/hooks/post-checkout

$ envbuckets uninstall
Replacing link .env with a real file
Replacing link apps/web/.env.local with a real file
Replacing link apps/api/service-account.json with a real file
Removing the envbuckets hook from .git/hooks/post-checkout
hint: Your buckets are still in .env.d/. Delete it when you no longer need them.
```
