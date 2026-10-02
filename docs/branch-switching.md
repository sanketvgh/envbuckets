# Branch switching and local pins

`init` installs a `post-checkout` hook. After a branch checkout, the hook
finds a local pin or the first matching shared rule and points each scope's
`.env` at that bucket's file. It prints a reminder to restart services
after a switch. The hook always exits 0, so it never blocks Git checkout.
It does not create bucket files, move real `.env` files, or replace foreign
symlinks. Detached HEAD and unmatched branches leave `.env` as-is.

## Switch now without a checkout

```sh
envbuckets apply --dry-run
envbuckets apply
envbuckets apply --scope api
```

`apply` uses the current branch's pin or first matching rule. It repairs a
missing managed `.env` link when the target bucket file exists. A dry run
shows the same plan and blockers without writing, with the same exit code
as a real run. It does not create a missing bucket file or overwrite a real
`.env` or foreign link. Across scopes, successful changes remain even if
another scope fails.

## Switch manually

```sh
envbuckets use staging
envbuckets use staging --scope api
envbuckets use staging --all
```

`use` points at a named bucket without changing a rule or pin. With no
flag it uses the scope containing your current directory. `--all` targets
every configured scope and reports partial failures with exit 1. A later
checkout can switch it back according to the branch's pin or rule. Use
`link` if the choice should persist across checkouts.

## Pin a branch locally

```sh
envbuckets link staging
envbuckets link prod --branch release/2.0
envbuckets unlink
envbuckets unlink --branch release/2.0
```

A pin overrides rules for that branch in the hook, `status`, `check`,
`apply`, and `map explain`. It is kept in local `.git/config`, not in the
committed `.envbuckets.toml`. `link` changes the mapping and attempts to
switch immediately when the named branch is current. `unlink` removes the
pin and attempts to apply the matching rule immediately for the current
branch. If the managed `.env` link is missing, run `envbuckets apply` to
repair it. A pin for another branch takes effect when you check out that
branch. Branch rename and deletion carry or drop local pins respectively.

Use [Status and recovery](status-and-recovery.md) to see why a switch was
skipped, and [Configuration and rules](configuration.md) to change the
shared mapping.
