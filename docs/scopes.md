# Scopes and monorepos

A scope is a directory with its own `.env` and `.env.d/` buckets. The
repository has one shared rule file and hook. In a simple repository, the
root is the implicit scope. Once you register explicit scopes, add `.` as
`root` too if the root still needs its own `.env`.

## Register apps

```sh
envbuckets scope add apps/api --name api --into dev
envbuckets scope add apps/web --name web
envbuckets scope add . --name root
envbuckets scope list
```

The directory must already exist inside the repository. The name defaults
to its directory name. Nested scopes are not supported. `--into` moves an
existing real `.env` into that bucket; without it, setup can ask which
bucket to use. Rules and pins choose one bucket _name_ for the branch,
while each scope supplies its own file for that name.

## Work across scopes

```sh
envbuckets bucket add qa --all
envbuckets bucket list --all
envbuckets use qa --all
envbuckets check
envbuckets apply --scope api
```

`--scope <name>` selects one registered scope. For bucket and manual-use
commands without `--scope`, the current directory chooses its containing
scope. `check` and `apply` cover all scopes by default. `--all` is
available for `bucket add`, `bucket list`, and `use`, and cannot be paired
with `--scope`. Check the matrix before making a rule for a bucket that
does not exist in every app.

## Unregister or erase an app

```sh
envbuckets scope rm web
envbuckets scope purge apps/web
```

`scope rm` unregisters the name but keeps `.env.d/` and its `.gitignore`
lines so local values remain hidden from Git. Use `scope rm web --purge`
to erase data during removal, or `scope purge apps/web` later. Purge takes
a repo-relative path, only works after unregistering, and asks you to
type `DELETE`. It refuses paths outside the repository or through symlinks.
A real `.env` or foreign symlink is left in place and remains ignored.

See [Buckets](buckets.md) for per-scope values and
[Uninstall and safety](uninstall-and-safety.md) before deleting data.
