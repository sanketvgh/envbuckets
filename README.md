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
or an admin shell so the tool can create symlinks.

## Quick start

```sh
envbuckets init --into dev          # moves your .env into .env.d/dev/ and installs the hook
envbuckets bucket add prod          # creates an empty .env.d/prod/.env for you to fill in
envbuckets map add "release/*" prod
envbuckets map add "*" dev          # catch-all, must be last

git checkout -b release/1.2
# envbuckets: dev -> prod (release/*) - 1 scope switched, next: restart your dev servers

envbuckets status
```

Have a monorepo? `envbuckets scope add apps/api --name api` gives each app
its own `.env.d/`. The branch rules are shared by all of them.

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
- **Scope**: a folder that has its own `.env`. A normal repo has one scope,
  the root. A monorepo adds one scope per app.

## Commands

| Command                  | What it does                                                                                               |
| ------------------------ | ---------------------------------------------------------------------------------------------------------- |
| `init [--into <bucket>]` | Move your `.env` into a bucket, install the hook, update `.gitignore`                                      |
| `status`                 | Show the branch, the rule that matched, and which bucket each scope is on                                  |
| `use <bucket>`           | Switch by hand. The next checkout that matches a rule switches it back                                     |
| `bucket add\|rm\|list`   | Add, remove, or list buckets in the current scope                                                          |
| `map add\|rm\|list`      | Add, remove, or list branch rules                                                                          |
| `scope add\|rm\|list`    | Add, remove, or list scopes (for monorepos)                                                                |
| `uninstall [--purge]`    | Turn `.env` back into a real file and remove the hook. `--purge` also deletes `.env.d/` after you confirm  |

## Guarantees

- **It never reads your values.** No command reads, prints, or parses the
  contents of an env file. Files are only moved, linked, or checked to see
  if they exist.
- **You never lose an env.** If the bucket file for the new branch is
  missing, the scope keeps its current link and prints a warning. Each swap
  is atomic, so `.env` never disappears, even if the process is killed
  halfway.
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
