# Status, checks, and recovery

`status` explains the current branch's pin or winning rule and reports
each scope's active and expected bucket, missing files, broken links, and
unmanaged `.env` files. Detached HEAD and an unmatched branch are reported
explicitly; checkout leaves `.env` unchanged in either case. It is useful
for inspection; use `check` as a gate:

```sh
envbuckets status
envbuckets check
envbuckets check --scope api
envbuckets check && npm run dev
```

`check` exits 0 only when the branch resolves to a bucket and every
selected scope has its directory, a working managed `.env` link, the
expected real bucket file, and the expected bucket active. It checks
structure, not whether values are correct or nonempty. `status` displays
state without requiring it to be ready.

| Symptom                            | Next step                                                           |
| ---------------------------------- | ------------------------------------------------------------------- |
| No rule for the branch             | Add a rule with `map add`, or pin it with `link`                    |
| Detached HEAD                      | Check out a branch, or manually select a bucket with `use`          |
| Expected bucket file missing       | Create it with `bucket add` if absent, fill it in, then run `apply` |
| Managed `.env` link missing        | Run `apply` after its target bucket exists                          |
| Active and expected buckets differ | Preview `apply --dry-run`, then run `apply`                         |
| Broken managed link                | Restore its bucket file, then run `apply`                           |
| Real `.env` file                   | Use `init --into <bucket>` to move it into a bucket                 |
| Foreign `.env` symlink             | Inspect and remove or replace it yourself; the CLI leaves it alone  |
| Scope directory missing            | Restore the directory or unregister the scope                       |
| Config unreadable                  | Restore `.envbuckets.toml` from Git                                 |

`apply` reports blockers but keeps successfully changed scopes. It never
creates a bucket file, and a real `.env` or foreign link is left untouched.
For a new clone, follow [Getting started](getting-started.md). For a rule
whose order or target is wrong, see [Configuration and rules](configuration.md).
