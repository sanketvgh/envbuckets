# envbuckets

**Stop copy-pasting `.env` files every time you switch branches.**

[![ci](https://github.com/sanketvgh/envbuckets/actions/workflows/ci.yml/badge.svg)](https://github.com/sanketvgh/envbuckets/actions/workflows/ci.yml)
[![npm](https://img.shields.io/npm/v/envbuckets?label=npm)](https://www.npmjs.com/package/envbuckets)
[![license](https://img.shields.io/github/license/sanketvgh/envbuckets)](LICENSE)
[![website](https://img.shields.io/badge/heysanket.com-333?logo=googlechrome&logoColor=white)](https://heysanket.com)
[![x](https://img.shields.io/badge/x-@sanketvgh-000?logo=x&logoColor=white)](https://x.com/sanketvgh)
[![linkedin](https://img.shields.io/badge/linkedin-sanketvgh-0A66C2?logo=linkedin&logoColor=white)](https://www.linkedin.com/in/sanketvgh)

> [!WARNING]
> **Early alpha, under active development.**
> Commands and config may change. Star or watch the repo to stay tuned.

envbuckets keeps one `.env` file per bucket (`dev`, `staging`, `prod`) and
points `.env` at the right one for the branch you are on. A git hook does
the switch every time you check out a branch. It never reads what is inside
your env files.

```text
.env -> .env.d/staging/.env      # on main
.env -> .env.d/prod/.env         # on release/*
```

## Install

```sh
npm install -g envbuckets
```

Windows, macOS, Linux. Node 18+. No Go required.

On Windows, you need Developer Mode (Settings > System > For developers)
or an admin shell so the tool can create symlinks. `init` checks this before
installing the hook or writing project configuration.

## Quick start

```sh
envbuckets init --into dev          # moves your .env into .env.d/dev/ and installs the hook
envbuckets bucket add prod          # creates an empty .env.d/prod/.env for you to fill in
envbuckets map add "*" dev          # catch-all, always kept last
envbuckets map add "release/*" prod # goes in before the catch-all

git checkout -b release/1.2
# envbuckets: dev -> prod (release/*) - 1 scope switched, next: restart your dev servers

envbuckets check                    # exit 0 only if every .env points where it should
```

Have a monorepo? `envbuckets scope add apps/api --name api` gives each app
its own `.env.d/`. The branch rules are shared by all of them.

For setup, every command and flag, monorepos, and recovery, see the
[documentation guide](docs/README.md).
For automation and coding agents, see [CLI JSON output](docs/agent-integration.md).

## Everyday workflows

**Check before you start services.** `envbuckets check` looks at the
current branch, works out the bucket it should use, and verifies every
scope: the folder exists, `.env` is a working link, the bucket file exists,
and the active bucket is the expected one. It exits 0 only when all of that
holds, so it fits in a `predev` script:

```sh
envbuckets check && npm run dev
```

It checks structure, not values. A bucket file that exists can still hold
wrong or empty values.

**Apply after changing things.** Changed a rule, filled in a bucket that
was missing, or deleted a broken `.env`? `envbuckets apply` points every
scope at the current branch's bucket without another checkout. Preview first
with `--dry-run`, which writes nothing and returns the same exit code. Limit
either one with `--scope <name>`.

**Clone a repo that already uses envbuckets.** The rules are committed, the
values are not:

```sh
envbuckets init --scaffold          # hook + an empty file for every bucket the rules name
# fill .env.d/<bucket>/.env in your editor
envbuckets apply
```

`--scaffold` never overwrites a file, never creates missing app folders,
and never switches to an empty file for you. Have a real `.env` already?
`envbuckets init --scaffold --into dev` moves it into `dev` first and
creates the rest empty.

**Edit rules.** Rules are checked top to bottom and the first match wins.

```sh
envbuckets map update "release/*" staging           # new bucket, same position
envbuckets map move "release/hotfix-*" --before "release/*"
envbuckets map explain release/2.0                  # which rule wins, and is the file there?
```

`map explain` works for branches you have not created yet and changes
nothing.

**Pin one branch.** `envbuckets link prod` pins the current branch (or
`--branch <name>`) to a bucket. A pin beats every rule, in `check`,
`status`, `apply`, `map explain`, and the hook. It is stored in
`.git/config`, so it stays on your machine. `envbuckets unlink` removes it.

**Work across a monorepo.**

```sh
envbuckets bucket add qa --all      # empty .env.d/qa/.env in every existing app scope
envbuckets bucket list --all        # which bucket exists in which app
envbuckets use qa --all             # switch every app by hand until the next checkout
```

```text
bucket   root     api      web      used by
dev      active   active   active   rule *
prod     present  present  missing  rule release/*
qa       present  present  present  pin feature/qa
```

`present` means the file exists. Its contents are never checked.

**Remove an app.** `envbuckets scope rm web` unregisters the app but keeps
its `.env.d/` and the `.gitignore` lines that hide it. When you want the
values gone, `envbuckets scope purge apps/web` deletes `apps/web/.env.d/`
after you type `DELETE`. It only works on a folder that is no longer
registered, refuses paths outside the repo or through symlinks, and leaves
a real `.env` in place.

## How it works

```text
your-repo/
|-- .env -> .env.d/dev/.env      symlink, ignored by git
|-- .env.d/                      ignored by git
|   |-- dev/
|   |   `-- .env                 a real file with your local values
|   `-- prod/
|       `-- .env
|-- .envbuckets.toml             committed, maps branch patterns to buckets
`-- .git/
    `-- hooks/
        `-- post-checkout        runs `envbuckets hook`
```

In a monorepo each scope has its own `.env.d/`, and there is one shared rule
file:

```text
your-repo/
|-- .envbuckets.toml             rules and scopes
|-- apps/
|   |-- api/
|   |   |-- .env -> .env.d/dev/.env
|   |   `-- .env.d/
|   |       |-- dev/.env
|   |       `-- prod/.env
|   `-- web/
|       |-- .env -> .env.d/dev/.env
|       `-- .env.d/
|           |-- dev/.env
|           `-- prod/.env
`-- .git/hooks/post-checkout
```

- **Bucket**: a named set of values. It is just a folder `.env.d/<name>/`
  with a real `.env` inside.
- **Rule**: a branch pattern that points to a bucket. `*` matches anything,
  including `/`. The first rule that matches wins. Rules are committed, so
  the whole team shares them. The values are never committed.
- **Link**: a pin from one branch to a bucket, set with `envbuckets link`.
  It wins over the rules and lives in `.git/config`, so it is never
  committed. Renaming or deleting the branch carries or drops it.
- **Scope**: a folder that has its own `.env`. A normal repo has one scope,
  the root. A monorepo adds one scope per app.

## Commands

| Command                                    | What it does                                                                                              |
| ------------------------------------------ | --------------------------------------------------------------------------------------------------------- |
| `init [--into <bucket>] [--scaffold]`      | Move your `.env` into a bucket, install the hook, update `.gitignore`. `--scaffold` creates rule buckets  |
| `status`                                   | Show the branch, the pin or rule that matched, and each scope's active and expected bucket                |
| `check [--scope <name>]`                   | Exit 0 only if every scope is on the expected bucket and its file exists                                  |
| `apply [--scope <name>] [--dry-run]`       | Point scopes at the current branch's bucket now, repairing missing links                                  |
| `use <bucket> [--scope <name> \| --all]`   | Switch by hand. The next checkout that matches a rule switches it back                                    |
| `link <bucket>`                            | Pin the current branch (or `--branch <name>`) to a bucket. Overrides rules, stays local to your clone     |
| `unlink`                                   | Remove the pin so the rules apply again                                                                   |
| `bucket add\|rm\|list`                     | Add, remove, or list buckets in the current scope. `add` and `list` take `--all`                          |
| `map add\|update\|move\|rm\|list\|explain` | Edit, reorder, list, or explain branch rules                                                              |
| `scope add\|rm\|purge\|list`               | Add, remove, or list scopes (for monorepos). `purge` deletes data a removed scope left behind             |
| `uninstall [--purge]`                      | Turn `.env` back into a real file and remove the hook. `--purge` also deletes `.env.d/` after you confirm |

The documented commands explain themselves with `--help`, for example
`envbuckets map move --help`.

**Exit codes.** `0` ok, `1` blocked or incomplete, `2` config missing or
unreadable, `3` usage error, `4` environment problem (not a git repo, no
symlink support). `apply`, `use --all`, and `bucket add --all` keep going
past a scope that fails and return `1` when any scope fails. `check` returns
`1` when any selected scope is unready. `init --scaffold` skips missing scope
directories and returns `1` if a bucket file could not be created. The scopes
that succeeded keep their changes: each link swap is atomic, the set of
them is not.

## Guarantees

- **It never reads your values.** No command reads, prints, or parses the
  contents of an env file. Files are only moved, linked, or checked to see
  if they exist.
- **You never lose an env.** If the bucket file for the new branch is
  missing, the scope keeps its current link and prints a warning. Each swap
  is atomic, so `.env` never disappears, even if the process is killed
  halfway.
- **It never replaces your own files.** A real `.env`, or a `.env` symlink
  pointing outside `.env.d/`, is reported and left alone by the hook,
  `apply`, and `use`. No command writes over an existing bucket file.
- **Bucket paths stay in the repo.** Symlinked scope and bucket directories are
  refused, and a symlinked bucket `.env` is not counted as a bucket file.
  `bucket rm` needs `--purge` and typed confirmation if the directory holds
  anything beyond an empty `.env`.
- **It never blocks a checkout.** The hook always exits 0 and never creates,
  moves, or deletes files.
- **It stays inside the repo and offline.** No `$HOME`, no temp folders, no
  network.
- **You can always remove it.** `uninstall` never reads the config, so a
  broken config cannot lock you in.

## Development

```sh
task check              # lint and unit tests
task test:integration   # real git runs the real binary (needs symlinks; use WSL on Windows)
task bench              # time the hook with hyperfine (Linux or WSL)
task bench:compare BASE=main
```

## License

[MIT](LICENSE)
