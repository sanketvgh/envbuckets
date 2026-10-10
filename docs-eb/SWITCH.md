# Switching buckets

EB-02 adds these commands to the rebuild:

```sh
envbuckets switch                 # use the current branch's bucket
envbuckets switch prod            # use prod until the next branch checkout
envbuckets switch -n prod         # preview changes without applying them
```

The repository needs `.envbuckets.json` and bucket directories under `.env.d/`.
The `init`, `add`, and `switch -c` setup commands arrive in EB-03. Until then,
prepare those directories and config manually. A bucket file has the same path
inside the bucket as its working path in the repository:

```text
.env.d/dev/.env
.env.d/dev/apps/api/key.json
.env.d/prod/.env
```

Switching to `dev` creates relative links at `.env` and `apps/api/key.json`.
Switching to `prod` replaces `.env` and removes the working `key.json` link;
its real file stays in the `dev` bucket. Managed file contents are never opened
or printed. Running `switch` again repairs missing links and finishes a switch
interrupted between individual link replacements.

Real files, foreign symlinks, tracked paths, and unsafe bucket entries are
preserved. Each blocked path gets an `error:` line; other safe paths still
switch, and the command exits 1. A dry run reports the same planned errors and
exit status without changing links or creating directories. Filesystem failures
that occur while applying changes, such as permissions changing after planning,
are reported by the real run.

If the branch's bucket is missing, plain `switch` uses the default bucket with
a warning and a hint to create the missing bucket. If the default is also
missing, existing links stay in place. Explicitly requesting a missing bucket
stops with `fatal:` and changes nothing. On a detached HEAD, name a bucket;
plain `switch` has no branch mapping to return to.

## Checkout hook

The hook installer is ready for EB-03's `init` command. It resolves the hooks
directory through Git, honors an in-repository `core.hooksPath`, and uses Git's
common metadata directory for linked worktrees. A custom hooks directory shared
outside the repository is refused. Symlinked hook paths and hooks directories
inside `.env.d/` are also refused.

The installer appends one marked block after existing hook content, replaces
alpha blocks, and makes the hook executable. The envbuckets block ignores file
checkouts and detached HEAD, is silent when the bucket stays the same, and exits
0 on missing binary, broken config, or switch errors.

An earlier `exit` in a user's hook prevents the appended block running. An earlier
nonzero exit remains a failure from that user's hook. Git LFS normally refuses
to overwrite a combined hook; `git lfs install --force` does overwrite it and
removes the envbuckets block. Reinstalling the envbuckets hook restores the block
alongside LFS. `envbuckets status` reports a missing hook block with a hint to
run `envbuckets init` to restore it.

## CI verification

Unit tests cover the planner, CLI behavior, value blindness, and hook installation.
The txtar scripts cover real Git checkouts, fallback, blocked paths, hook conflicts,
Git LFS install orders, symlink safety, and linked worktrees. The LFS script skips
when `git-lfs` is unavailable; the unreadable-file script uses Unix mode bits.
The existing CI workflow runs unit tests on Linux and scripts on Linux, macOS,
and Windows. [Run `37951292646`](https://github.com/sanketvgh/envbuckets/actions/runs/37951292646)
passed those jobs on `f1c2133`. Local formatting, lint, schema validation, and unit
tests also passed. The local integration stage requires symlink privilege that
this Windows session lacks; the user accepted the passing CI results for that
stage and deferred environment setup. EB-02 is passed.
