# Command reference

Run `envbuckets help` for the top-level summary, or append `--help` to a
documented command or subcommand for built-in usage. Commands run from
inside a Git repository unless noted otherwise. A bucket or scope name
uses letters, digits, `_`, and `-`, starting with a letter or digit.

| Command                         | Flags                                       | Action                                                                                               |
| ------------------------------- | ------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `init`                          | `--into <bucket>`, `--scaffold`             | Install config, hook, and ignore block; bootstrap real `.env` files; optionally create rule buckets. |
| `status`                        | —                                           | Show current branch resolution and per-scope actual/expected state.                                  |
| `check`                         | `--scope <name>`                            | Exit 0 only when selected scopes are structurally ready. Defaults to all.                            |
| `apply`                         | `--scope <name>`, `--dry-run`               | Apply the current branch's pin or rule now. Defaults to all.                                         |
| `use <bucket>`                  | `--scope <name>` or `--all`                 | Manually point one or every scope at a bucket.                                                       |
| `link <bucket>`                 | `--branch <name>`                           | Pin a local branch to a bucket; defaults to current branch.                                          |
| `unlink`                        | `--branch <name>`                           | Remove a local pin; defaults to current branch.                                                      |
| `bucket add <name>`             | `--scope <name>` or `--all`                 | Create empty bucket file(s).                                                                         |
| `bucket rm <name>`              | `--scope <name>`, `--purge`                 | Remove an unused bucket; data needs `--purge` and confirmation.                                      |
| `bucket list`                   | `--scope <name>` or `--all`                 | List one scope or a cross-scope availability matrix.                                                 |
| `map add <pattern> <bucket>`    | —                                           | Add a shared rule before the catch-all.                                                              |
| `map update <pattern> <bucket>` | —                                           | Change a rule's bucket without moving it.                                                            |
| `map move <pattern>`            | `--before <pattern>` or `--after <pattern>` | Reorder a rule.                                                                                      |
| `map rm <pattern>`              | —                                           | Remove a rule.                                                                                       |
| `map list`                      | —                                           | List rules in priority order and local pins.                                                         |
| `map explain <branch>`          | —                                           | Explain a branch's pin/rule and bucket availability without checkout.                                |
| `scope add <path>`              | `--name <name>`, `--into <bucket>`          | Register an existing directory and bootstrap its `.env`.                                             |
| `scope rm <name>`               | `--purge`                                   | Unregister; keep data by default.                                                                    |
| `scope purge <path>`            | —                                           | Delete data from an unregistered scope after confirmation.                                           |
| `scope list`                    | —                                           | List registered scopes and their `.env` state.                                                       |
| `uninstall`                     | `--purge`                                   | Deactivate and restore real `.env` files; optionally erase data.                                     |
| `version`                       | —                                           | Print the CLI version.                                                                               |
| `help`                          | —                                           | Print the top-level help.                                                                            |

`hook` is the command installed in the local `post-checkout` hook. Git
passes its checkout arguments; normally you do not run it yourself.

For `bucket` and `use` without a scope flag, the scope containing the
current directory is selected. `check` and `apply` default to all scopes.
`--all` and `--scope` cannot be combined. Rule edits are shared through
`.envbuckets.toml`; local pins are not. See the guides from the
[documentation index](README.md) for examples and safety details.

## Exit codes

| Code | Meaning                                                                          |
| ---- | -------------------------------------------------------------------------------- |
| `0`  | Successful operation or a healthy `check`. The hook always exits 0.              |
| `1`  | Blocked or incomplete; for example a missing file, unsafe link, or failed scope. |
| `2`  | Missing or unreadable configuration.                                             |
| `3`  | Invalid command usage.                                                           |
| `4`  | Environment problem, such as no Git repository or unavailable symlink support.   |

Bulk operations can finish some scopes and still exit 1. They do not roll
back successful scope changes. `status` is an inspection command; use
`check` when a script must stop on an unready checkout.
