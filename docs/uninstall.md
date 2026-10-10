# Leaving envbuckets

Run `envbuckets uninstall -n` to preview, then `envbuckets uninstall` to restore
your active files as ordinary files at their working paths. Uninstall moves each
file out of its bucket and replaces its working link. It does not read or copy
the file's contents.

```console
$ envbuckets uninstall -n
Would move .env.d/dev/.env to .env
Would remove the envbuckets hook from .git/hooks/post-checkout
$ envbuckets uninstall
Moving .env.d/dev/.env to .env
Removing the envbuckets hook from .git/hooks/post-checkout
hint: Your other buckets are still in .env.d/. Delete it when you no longer need them.
```

Your config, ignore rules, and other buckets are kept. Files without a working
link also stay in their buckets. If some links use `dev` and others use `prod`,
each file is restored from the bucket its link points to. Uninstall still works
if `.envbuckets.json` is missing or invalid.

Uninstall removes envbuckets' marked block from the checkout hook, including
blocks left by older versions. It keeps the rest of the hook and its permissions.
If only a shebang or blank lines remain, it removes the hook file. If the hook
path is unsafe or the block is malformed, it reports the problem and leaves
the hook alone.

If a link points to a missing bucket file, uninstall reports the problem and
leaves the link alone. It restores the other files and removes its hook, but
exits with code 1. It won't overwrite ordinary files or links it doesn't manage.
Unsafe paths are left alone too. Use `uninstall -n` to see what can be restored
without moving anything. Files or permissions can still change before the real
command runs.

Uninstall looks for working links using the paths found in your buckets. If a
broken link's path is missing from every bucket, uninstall can't find it and
leaves it alone. It can find the link if another bucket still has that path.

If uninstall stops halfway through, run it again to restore the remaining
files. Rename is atomic on Unix; Go does not promise atomic rename on Windows.
Running uninstall again after it has finished does nothing and prints nothing.
Run `envbuckets init` to manage restored `.env` files again; use
`envbuckets add <file>` for other restored files. When manually using a bucket
other than the default, the default may already contain the same paths; resolve
those collisions before importing into it.

## Remove the CLI

Run `envbuckets uninstall` in every project where you use it before removing
the CLI. That restores each project's active files and removes its checkout
hook. It keeps your config, ignore rules, and other buckets.

Resolve any reported errors and rerun uninstall before removing the CLI.

For a global npm install:

```sh
npm uninstall -g envbuckets
```

If you downloaded a binary or used `go install`, remove the installed
`envbuckets` executable (`envbuckets.exe` on Windows) from its installation
directory. Removing the executable alone does not restore files or remove
hooks in your projects.
