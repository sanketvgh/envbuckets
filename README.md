# envbuckets

[![ci](https://github.com/sanketvgh/envbuckets/actions/workflows/ci.yml/badge.svg)](https://github.com/sanketvgh/envbuckets/actions/workflows/ci.yml)
[![npm](https://img.shields.io/npm/v/envbuckets?label=npm)](https://www.npmjs.com/package/envbuckets)
[![license](https://img.shields.io/github/license/sanketvgh/envbuckets)](https://github.com/sanketvgh/envbuckets/blob/main/LICENSE)
[![website](https://img.shields.io/badge/heysanket.com-333?logo=googlechrome&logoColor=white)](https://heysanket.com)
[![x](https://img.shields.io/badge/x-@sanketvgh-000?logo=x&logoColor=white)](https://x.com/sanketvgh)
[![linkedin](https://img.shields.io/badge/linkedin-sanketvgh-0A66C2?logo=linkedin&logoColor=white)](https://www.linkedin.com/in/sanketvgh)

Keep one set of local files per environment. envbuckets links the right set into
your project when you switch Git branches. It works with `.env` files, service
accounts, and other local settings without reading or printing their contents.

## Guarantees

- **Your values stay private.** envbuckets never reads, parses, copies, or prints
  the contents of managed files.
- **Switching protects your files.** It leaves ordinary files, tracked paths,
  and links it doesn't manage alone. It reports the problem so you can fix it.
- **Bucket files stay in their buckets when you switch.** Only working links
  change. Imports refuse to overwrite a file already in a bucket.
- **The CLI works offline.** It never fetches the editor schema or uploads your
  files. File operations stay inside the repository and unsafe paths are refused.
- **envbuckets' hook never blocks checkout.** Its hook block exits 0, even when
  config is broken or a file can't be switched.
- **A dry run changes nothing.** `-n` shows the plan and known problems without
  moving files or changing links.

## Install

```sh
npm install -g envbuckets
envbuckets --version
```

The npm package selects a prebuilt binary for Linux, macOS, or Windows on x64 or
arm64. Node.js 18 or newer and Git are required. You can also download a binary
from [GitHub releases](https://github.com/sanketvgh/envbuckets/releases), or build
with the Go version in `go.mod`:

```sh
go install github.com/sanketvgh/envbuckets@latest
```

On Windows, enable Developer Mode or use an account with symlink privilege.
`init` checks support before moving files and explains what to enable if needed.

## Start with your local files

From a Git repository with an untracked `.env`:

```sh
envbuckets init -n            # preview setup
envbuckets init               # import .env and .env.* files; install the hook
envbuckets switch -c prod     # create empty files at the same paths in prod
# Fill in the new files using your editor.
envbuckets switch            # return to this branch's bucket
envbuckets add apps/api/service-account.json
```

`init` moves local environment files into `.env.d/dev/` and leaves relative
symlinks at their working paths. It skips tracked files such as `.env.example`
and ignored directories such as `node_modules/`. Editing a linked file edits
the active bucket's file. `add` manages any other local file.

For an existing config, `init` imports into its `default` bucket. `add` uses
the bucket your working links currently point to. Create the file at its normal
project path before adding it. See [setup and adding files](https://github.com/sanketvgh/envbuckets/blob/main/docs/setup.md)
for fresh clones, multiple files, and import problems.

```text
your-project/
|-- .envbuckets.json            shared rules; commit this
|-- .gitignore                  ignore rules; commit this
|-- .env ----------------------> .env.d/dev/.env
`-- .env.d/                     private files; ignored by Git
    |-- dev/
    |   `-- .env                real file; edited through .env
    `-- prod/
        `-- .env                a separate file for prod
```

Commit `.envbuckets.json` and `.gitignore`. Keep `.env.d/` out of Git. Teammates
run `envbuckets init` after cloning, create their own local files, then add them.
Rerunning `init` preserves existing setup and imports new environment files.

In a monorepo, files keep their paths inside each bucket: `apps/api/.env` becomes
`.env.d/dev/apps/api/.env`. One branch mapping chooses the bucket for the whole
repository. See the [monorepo guide](https://github.com/sanketvgh/envbuckets/blob/main/docs/monorepos.md)
for a complete layout and setup.

Older `.envbuckets.toml` configs are unused. Remove that config and run `init`;
resolve any reported path collisions before importing files. There is no
automatic migration.

## Map branches to buckets

Edit `.envbuckets.json`:

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

The first matching rule wins; otherwise `default` is used. Patterns match the
whole branch name and are case-sensitive. `*` and `?` stop at `/`, `**` crosses
it, and `release/` means `release/**`. Sets, character classes, and backslash
escapes follow Git glob rules. List separate rules for separate names.

When the mapped bucket exists, checking out a branch updates the working links:

```text
+-----------------------------+
| git switch release/2.0      |
+--------------+--------------+
               |
               v
+-----------------------------+
| Checkout hook reads rules   |
| release/** -> prod          |
+--------------+--------------+
               |
               v
+-----------------------------+
| .env -> .env.d/prod/.env    |
+-----------------------------+
```

Your files stay in their buckets. The hook changes which files the working
links point to.

Check a mapping before the branch exists:

```sh
envbuckets branches release/2.0/rc1
envbuckets branches 'release/**'       # quote patterns to avoid shell expansion
envbuckets branches --bucket prod
```

`init` writes `$schema` into new configs. VS Code and other JSON Schema editors
use that URL for descriptions, completion, and validation. If your editor does
not automatically recognize it, associate `.envbuckets.json` with the same URL
in its JSON Schema settings. The CLI validates config locally and never fetches
the schema. The URL follows `main`, which remains the published schema branch.

## Commands

| Command | Behavior |
| --- | --- |
| `init [-n]` | Set up config, the default bucket, ignores, and the hook; import local environment files. |
| `add [-n] <file>...` | Move files into the current bucket and leave links; validate every path before changing any. |
| `switch [-n] [<bucket>]` | Use a bucket now, or return to the branch's bucket without a name. |
| `switch [-n] -c <bucket>` | Create empty files at the current bucket's paths, then switch. |
| `status` | Show the active bucket, mapping, and paths needing attention. |
| `branches [--bucket <name>] [<branch or pattern>...]` | List current and rule-matched branches, check future names, or filter local branches. Use `'**'` for all. |
| `uninstall [-n]` | Move active files back to their working paths and remove the hook; keep other buckets. |
| `help`, `--version` | Show command help or the installed version. |

`-n` / `--dry-run` lists planned changes without applying them. Reports write
to stdout; warnings, hints, and errors use stderr. Output is plain when piped.
Set `NO_COLOR` to any non-empty value to disable terminal color. Exit codes are
0 for success, 1 for an operation error, and 2 for invalid usage.

## Switching and recovery

The checkout hook follows branch rules automatically. A manual `switch prod`
lasts until plain `switch` or the next branch checkout. Files absent from the
target bucket lose their working links; their bucket files stay in place.

If a branch bucket is missing, the hook and plain `switch` use the default and
warn. If the default is also missing, links stay as they are. A detached HEAD
leaves links alone; choose a bucket explicitly if needed. The envbuckets hook
never blocks checkout. Hooks configured outside the repository are refused.

Existing hooks are preserved, with envbuckets appended in a marked block. An
earlier `exit` in another hook can prevent that block from running. If another
tool replaces the hook, `status` reports it; rerun `init` to reinstall the block.

Real files, foreign symlinks, tracked paths, and unsafe paths are preserved.
`status` reports problems. Resolve the blocking path, then rerun `switch` to
repair links and finish an interrupted switch. On Windows, Go uses one
`MoveFileEx(MOVEFILE_REPLACE_EXISTING)` call for rename, without a delete step
in envbuckets. Microsoft does not document it as atomic, so rerunning is the
recovery mechanism. See [Go's implementation](https://go.dev/src/internal/syscall/windows/syscall_windows.go)
and [Microsoft's API documentation](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw).

For common setup errors and blocked links, see the
[troubleshooting guide](https://github.com/sanketvgh/envbuckets/blob/main/docs/troubleshooting.md).

## Uninstall

First, run these commands in each project where you use envbuckets:

```sh
envbuckets uninstall -n       # preview
envbuckets uninstall          # restore active files and remove the checkout hook
```

The config, ignore rules, and other buckets are kept. Keep any bucket files you
still need. See [leaving envbuckets](https://github.com/sanketvgh/envbuckets/blob/main/docs/uninstall.md)
if a file could not be restored. Resolve any errors before removing the CLI.

Then remove the CLI. For a global npm install:

```sh
npm uninstall -g envbuckets
```

For a downloaded binary or `go install`, remove the installed `envbuckets`
executable (`envbuckets.exe` on Windows) from the directory where you installed
it. Removing the CLI does not restore project files or remove project hooks;
run `envbuckets uninstall` in those projects first.

## More detail

- [All user guides](https://github.com/sanketvgh/envbuckets/blob/main/docs/index.md)
- [Setup and adding files](https://github.com/sanketvgh/envbuckets/blob/main/docs/setup.md)
- [Monorepos](https://github.com/sanketvgh/envbuckets/blob/main/docs/monorepos.md)
- [Troubleshooting](https://github.com/sanketvgh/envbuckets/blob/main/docs/troubleshooting.md)
- [Switching buckets](https://github.com/sanketvgh/envbuckets/blob/main/docs/switch.md)
- [Status and branch mappings](https://github.com/sanketvgh/envbuckets/blob/main/docs/status.md)
- [Leaving envbuckets](https://github.com/sanketvgh/envbuckets/blob/main/docs/uninstall.md)
- [Contributor checks](https://github.com/sanketvgh/envbuckets/blob/main/docs-eb/DEVELOPMENT.md)
- [Acceptance coverage](https://github.com/sanketvgh/envbuckets/blob/main/docs-eb/ACCEPTANCE.md)
