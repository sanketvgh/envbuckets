# Uninstall and data safety

envbuckets keeps env values local and ignored by Git. It never reads, parses,
or prints env file contents. During uninstall, it renames the active bucket
file into a real `.env` without reading its bytes. The committed
`.envbuckets.toml` contains only rules and scope paths; local branch pins live
in `.git/config`.

## Deactivate and keep data

```sh
envbuckets uninstall
```

This moves each active bucket file into its scope's `.env`, replacing the
managed symlink atomically, and removes envbuckets' block from the
`post-checkout` hook. Inactive bucket files remain in `.env.d/`. The config,
local pins, and `.gitignore` protection are kept. You can reactivate later
with `envbuckets init --into <bucket>`. Custom hook code is preserved.

## Deactivate and erase data

```sh
envbuckets uninstall --purge
```

This additionally removes bucket directories, the config, local pins, and
the envbuckets `.gitignore` block after showing what will be deleted and
asking you to type `DELETE` before changing any files. JSON mode cannot
confirm, so `uninstall --purge --json` leaves the project unchanged. Back up
wanted values before confirming. If
a managed link cannot be safely restored to a real file, purge stops so
its ignore protection remains. A foreign or broken `.env` symlink is left
alone and must be resolved manually before purging.

`uninstall` discovers managed data even if the config is missing or
unreadable. For removing one app or bucket, use [Scopes](scopes.md) or
[Buckets](buckets.md) instead.

## Safety boundaries

- The checkout hook always exits 0, never creates bucket files, and leaves
  real `.env` files or links outside `.env.d/` untouched.
- Link replacement is atomic. If a target bucket file is missing, the
  previous managed link stays in place.
- Bucket and scope directory symlinks are refused. A symlink at a bucket's
  `.env` is not accepted as a real bucket file.
- A symlinked `.envbuckets.toml` or `.gitignore` is refused before its target
  can be read.
- The hook path must be inside the repo, with no symlinked directory or hook
  file. If Git's `core.hooksPath` points elsewhere, change it before running
  `init` or `uninstall`.
- Commands that erase nonempty bucket data require explicit `--purge` and
  typed confirmation. Scope purge also requires typed confirmation.
- `check` verifies paths and links, not the contents or correctness of
  your secrets. Review values in your editor before using them.

See [Status and recovery](status-and-recovery.md) for blocked operations.
