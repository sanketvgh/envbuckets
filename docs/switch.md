# Switching buckets

Choose a bucket now or return to the current branch's bucket:

```sh
envbuckets switch                 # use the current branch's bucket
envbuckets switch prod            # use prod until the next branch checkout
envbuckets switch -n prod         # preview changes without applying them
envbuckets switch -c prod         # create empty files at the current bucket's paths
```

The repository needs `.envbuckets.json` and bucket directories under `.env.d/`.
Run `envbuckets init` to create config, the default bucket, and the hook, then
`envbuckets add <file>` for files other than local environment files.
A bucket file has the same path inside the bucket as its working path in the
repository:

```text
.env.d/dev/.env
.env.d/dev/apps/api/key.json
.env.d/prod/.env
```

Switching to `dev` creates relative links at `.env` and `apps/api/key.json`.
Switching to `prod` replaces `.env` and removes the working `key.json` link;
its real file stays in the `dev` bucket. Managed file contents are never opened
or printed. If a switch stops halfway through, run it again to finish. It also
recreates missing links.

If a real file or a link envbuckets doesn't manage is in the way, `switch`
reports an `error:` and leaves it alone. Tracked files and unsafe paths are also
left alone. The other files still switch, but the command exits with code 1.
Use `switch -n` to preview the changes and any known problems. Files or
permissions can still change before the real command runs.

If the branch's bucket is missing, plain `switch` uses the default bucket with
a warning and a hint to create the missing bucket. If the default is also
missing, existing links stay in place. Explicitly requesting a missing bucket
stops with `fatal:` and changes nothing. On a detached HEAD, name a bucket;
plain `switch` has no branch mapping to return to.

## Checkout hook

The `init` command installs the hook. It asks Git for the hooks directory,
honors an in-repository `core.hooksPath`, and uses Git's
common metadata directory for linked worktrees. A custom hooks directory shared
outside the repository is refused. Symlinked hook paths and hooks directories
inside `.env.d/` are also refused.

The installer appends one marked block after existing hook content, replaces
legacy blocks, and makes the hook executable. The envbuckets block ignores file
checkouts and detached HEAD, is silent when the bucket stays the same, and exits
0 on missing binary, broken config, or switch errors.

An earlier `exit` in a user's hook prevents the appended block running. An earlier
nonzero exit remains a failure from that user's hook. Git LFS normally refuses
to overwrite a combined hook; `git lfs install --force` does overwrite it and
removes the envbuckets block. Reinstalling the envbuckets hook restores the block
alongside LFS. `envbuckets status` reports a missing hook block with a hint to
run `envbuckets init` to restore it.

## Windows

Symlink operations need Developer Mode or symlink privilege. `init` checks this
before moving any file.

Go's Windows rename uses one `MoveFileEx(MOVEFILE_REPLACE_EXISTING)` call;
envbuckets does not delete the existing link first. Microsoft does not document
that call as atomic. A multi-file switch can also stop between paths on any
platform. Fix the reported problems and rerun `switch` to finish switching
files. See [Go's implementation](https://go.dev/src/internal/syscall/windows/syscall_windows.go)
and [Microsoft's API documentation](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw).
