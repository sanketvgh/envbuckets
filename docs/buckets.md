# Buckets and local values

A bucket is a directory `<scope>/.env.d/<name>/` containing a real `.env`
file. Each scope has its own buckets; a shared rule can choose the same
name across scopes without sharing their values. envbuckets checks whether
the file exists, but does not read, parse, or validate its contents.

## Create and inspect

```sh
envbuckets bucket add staging
# Fill .env.d/staging/.env in your editor.
envbuckets bucket list
envbuckets bucket list --all
```

`bucket add` creates an empty file and never overwrites an existing one.
Without a flag, bucket commands use the scope containing the current
directory. Use `--scope api` for a named scope. `bucket add staging --all`
creates the file in every configured, existing scope; it continues after
a scope fails and exits 1 if any failed. It does not create missing scope
directories. `--all` and `--scope` cannot be combined.

`bucket list` marks the active bucket and shows rules that reference each
one. `bucket list --all` displays a bucket-by-scope matrix, including
names referenced by rules and local pins. `present` means a file exists,
`missing` means it does not, `active` means `.env` points there, and
`BROKEN` means an active link's file is absent.

## Remove

```sh
envbuckets bucket rm old --scope api
envbuckets bucket rm old --scope api --purge
```

Removal refuses a bucket selected by a rule or local pin, or currently
active in that scope. Remove those references or switch away first.
An otherwise unused bucket containing only an empty `.env` can be removed
without `--purge`. If it contains values or any other file, `--purge`
requires you to type `DELETE` before removal. Inspect and back up values
you need before confirming.

For switching, see [Branch switching](branch-switching.md). For several
apps, see [Scopes and monorepos](scopes.md).
