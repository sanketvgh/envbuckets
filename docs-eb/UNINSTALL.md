# Leaving envbuckets

Run `envbuckets uninstall -n` to preview, then `envbuckets uninstall` to restore
your active files as ordinary files at their working paths. Each bucket file is
moved over its link with a single rename; its contents are never read or copied.

```console
$ envbuckets uninstall -n
Would move .env.d/dev/.env to .env
Would remove the envbuckets hook from .git/hooks/post-checkout
$ envbuckets uninstall
Moving .env.d/dev/.env to .env
Removing the envbuckets hook from .git/hooks/post-checkout
hint: Your other buckets are still in .env.d/. Delete it when you no longer need them.
```

The config, `.gitignore` block, other buckets, and unlinked bucket files stay in
place. Each link identifies its own active bucket, even when links point to
different buckets. Missing or invalid config does not prevent uninstall.

Only complete, exactly matched envbuckets hook blocks are removed, including
the alpha's `envbuckets v1` blocks. Custom hook bytes and permissions are kept.
A hook with only a shebang and whitespace left is deleted. Git's configured
hooks path is used; unsafe paths and malformed blocks produce an error and stay
untouched.

Broken managed links get an `error:` line and stay in place; healthy files and
hook removal still complete, with exit code 1. Foreign links, real working files,
and unsafe paths are never overwritten. Dry runs perform the same validation
and report the same known errors without changing anything. Filesystem changes
or write failures after validation can still cause a real run to fail.

Uninstall finds candidate paths from bucket entries without walking the project.
A broken link whose path has disappeared from every bucket cannot be discovered
this way and remains untouched. It can report a broken link when another bucket
still contains that path.

After an interrupted run, restored files and remaining links stay usable, and
rerunning finishes the remaining moves. Rename is atomic on Unix; Go does not
promise atomic rename on Windows. Repeating a completed uninstall is silent.
Run `envbuckets init` to manage restored `.env` files again; use
`envbuckets add <file>` for other restored files. When manually using a bucket
other than the default, the default may already contain the same paths; resolve
those collisions before importing into it.
