# Getting started

Install the CLI with Node 18 or newer:

```sh
npm install -g envbuckets
```

It runs on Windows, macOS, and Linux. On Windows, enable Developer Mode or
use an administrator shell so it can create symlinks. `init` tests symlink
support before changing the project.

## Set up a repository with an existing `.env`

From inside a Git repository:

```sh
envbuckets init --into dev
envbuckets bucket add prod
# Fill .env.d/prod/.env in your editor.
envbuckets map add 'release/*' prod
envbuckets map add '*' dev
envbuckets status
```

`init` moves the existing root `.env` to `.env.d/dev/.env` and replaces it
with a symlink. It also creates `.envbuckets.toml`, installs the
`post-checkout` hook, and adds an envbuckets block to `.gitignore`. Without
`--into`, it asks which bucket should receive a real `.env`; an empty answer
leaves that file alone. Re-running `init` is safe and skips completed steps.

Commit `.envbuckets.toml` and the `.gitignore` change so teammates receive
the rules. Keep `.env.d/` and `.env` out of Git: they contain local values.
The hook lives in your local Git hooks directory and each clone needs its
own installation.

## Set up a fresh clone

The committed config describes buckets but does not include their values:

```sh
envbuckets init --scaffold
# Fill each .env.d/<bucket>/.env that this machine needs.
envbuckets apply
envbuckets check
```

`--scaffold` creates empty bucket files for buckets named by rules in
existing scope directories. It never overwrites a file, creates a missing
app directory, or activates an empty file. If you already have a real
`.env`, use `envbuckets init --scaffold --into dev` to move it into `dev`
before creating other buckets.

## What is stored where?

```text
repo/
  .env                 -> .env.d/dev/.env (local symlink)
  .env.d/dev/.env       local values, ignored by Git
  .env.d/prod/.env      local values, ignored by Git
  .envbuckets.toml      shared rules and scope names, commit this
  .gitignore            ignore rules, commit this
  .git/hooks/post-checkout  local hook installed by init
```

The next guides explain [rules](configuration.md), [buckets](buckets.md),
and [what a checkout does](branch-switching.md).
