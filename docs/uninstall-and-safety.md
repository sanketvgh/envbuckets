# Uninstall and data safety

envbuckets keeps env values local and ignored by Git. It does not read or
print them. The committed `.envbuckets.toml` contains only rules and scope
paths; local branch pins live in `.git/config`. If you need to stop using
envbuckets, decide whether to keep or delete the bucket data.

## Deactivate and keep data

```sh
envbuckets uninstall
```

This restores each safely managed `.env` link to a real file and removes
envbuckets' block from the `post-checkout` hook. It keeps `.env.d/`, the
config, local pins, and the `.gitignore` protection. You can reactivate
later with `envbuckets init`. Custom code outside the envbuckets hook
block is preserved.

## Deactivate and erase data

```sh
envbuckets uninstall --purge
```

This additionally removes bucket directories, the config, local pins, and
the envbuckets `.gitignore` block after showing what will be deleted and
asking you to type `DELETE`. Back up wanted values before confirming. If
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
- Commands that erase nonempty bucket data require explicit `--purge` and
  typed confirmation. Scope purge also requires typed confirmation.
- `check` verifies paths and links, not the contents or correctness of
  your secrets. Review values in your editor before using them.

See [Status and recovery](status-and-recovery.md) for blocked operations.
