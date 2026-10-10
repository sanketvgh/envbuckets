# Setup and adding files

Run envbuckets from a Git repository. Git and symlink support are required;
Windows needs Developer Mode or symlink privilege. See the
[README](../README.md) for installation.

## First setup

Start with local files at their normal project paths:

```sh
envbuckets init -n
envbuckets init
envbuckets status
```

`init` creates missing config, the default bucket, the ignore block, and the
checkout hook. New config uses `dev` as the default. Existing config is kept,
and imports go into its configured default bucket even when a branch rule
selects another bucket. Run `envbuckets switch` afterwards to follow that rule.

`init` looks for untracked `.env` and `.env.*` files throughout the project.
Tracked examples stay untouched. `.git/`, `.env.d/`, and ignored directories
such as `node_modules/` are skipped. A file already ignored by Git can still be
imported when its parent directories are not ignored.

Each imported file moves once into `.env.d/<default>/<project-path>`; its
working path becomes a relative symlink. Editing that link edits the bucket
file. Rerunning `init` preserves existing setup and imports newly discovered
files. Individual import problems produce warnings while other files proceed.

Commit `.envbuckets.json` and `.gitignore`. Keep `.env.d/` and managed working
paths ignored. Each teammate supplies their own private files.

## Fresh clone

Shared config and ignores arrive through Git; private buckets do not:

```sh
envbuckets init
cp .env.example .env
# Fill in your own local values using your editor.
envbuckets add .env
```

Use your editor or file manager to copy the example on Windows if `cp` is
unavailable. `init` creates an empty default bucket when there are no local
files. It does not download another developer's values.

If this branch maps to a bucket that doesn't exist locally, `add` will ask for
it. Before adding your first file, create that bucket's empty directory. For
example, if `main` maps to `staging`:

```sh
mkdir .env.d/staging
envbuckets add .env
```

With no working links yet, `add` uses the branch's bucket, even if `switch`
would fall back to the default. Once you have linked files, `switch -c <name>`
can create another bucket with empty files at the same paths.

## Add other files

Create regular files at their working paths before adding them:

```sh
envbuckets add -n apps/api/service-account.json settings/local.json
envbuckets add apps/api/service-account.json settings/local.json
```

`add` uses the bucket the working links point to, including a manual override.
With no links, it uses the branch's bucket. If links point to multiple buckets,
run `envbuckets switch` to select one before adding files. Every argument is
validated before any file moves; one bad path rejects the whole request.
Successful `add` is silent. Directories, symlinks, tracked files, paths outside
the repo, and paths through symlinked directories are refused.

Files keep their project paths inside each bucket. `add` updates the ignore
block, unless Git already ignores the path. It never overwrites a path already
present in the destination bucket. See [troubleshooting](troubleshooting.md)
for collisions or replaced working links.

## Another environment

```sh
envbuckets switch -c prod
# Fill in the new empty files through their working paths.
envbuckets switch
```

`switch -c` creates empty files at the current bucket's paths. It does not copy
values. An existing bucket name is refused. Plain `switch` returns to the
branch's mapping. Edit `.envbuckets.json` to assign branches; the first matching
Git glob wins, otherwise `default` is used. See [switching](switch.md) and
[branch reports](status.md).
