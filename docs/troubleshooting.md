# Troubleshooting

Start with `envbuckets status`. Preview a repair with `envbuckets switch -n`,
resolve the reported obstruction, then run `envbuckets switch`.

| Symptom | What to do |
| --- | --- |
| Windows cannot create symlinks | Enable Developer Mode or use an account with symlink privilege, then rerun. `init` checks support before moving files. |
| `init` does not import a file | Check whether Git tracks it or ignores a parent directory. Use `add` for other regular local files; tracked files must stay outside envbuckets. |
| A bucket already contains the path | Preserve both files and choose which one to keep using your editor or file manager before moving either. Imports never overwrite a bucket file. |
| Links point to more than one bucket | Resolve blocked paths, then run `switch` to select the branch's bucket, or `switch <bucket>` for an explicit choice. Retry `add` or `switch -c` afterwards. |
| A mapped bucket is missing | The hook and plain `switch` fall back to the default with a warning. If you have linked files, use `switch -c <name>` to create it with empty files at the same paths. With no links yet, create `.env.d/<name>/` before adding your first file. |
| The default bucket is missing too | Existing links stay as they are. Create the required bucket directory and local files, then rerun `switch`. |
| `switch` reports a real file in the way | Preserve the working file and existing bucket file. After choosing the values to keep, move the working file to the same path inside the intended bucket, then rerun `switch`. |
| A foreign symlink or symlinked directory blocks a path | Preserve it and move it aside with your own tools if you want envbuckets to manage that path. envbuckets refuses to follow it. |
| Config is invalid | Fix the reported key, bucket name, or pattern in `.envbuckets.json`. The hook warns and leaves links alone; other commands fail until config is fixed. |
| HEAD is detached | Links stay as they are. Use `switch <bucket>` to choose explicitly; plain `switch` requires a branch. |
| Branch checkout does not switch files | Use `status` to check the hook. Rerun `init` if another tool replaced it. An earlier `exit` in your existing hook can prevent the appended block from running. |
| Hooks are outside the repository | Configure a hooks directory inside this repository before running `init`. Shared external or symlinked hook paths are refused. |
| Output contains no color | Pipes, redirected output, `TERM=dumb`, and any non-empty `NO_COLOR` disable color. Consoles unable to enable virtual terminal processing also use plain text. |

Some editors save by replacing a symlink with a regular file. In that case,
`status` reports the real file and `switch` preserves it; use the recovery above
to keep the edit in the intended bucket. A multi-file switch can stop between
paths. Resolve the error and rerun to finish; there is no rollback of the whole
switch. [Switching](switch.md) explains Windows rename limits.

`init` imports into the configured default, while `add` uses the active bucket.
After uninstalling from another bucket, the default may still contain the same
paths. Preserve both versions and resolve those collisions before reinitializing.
See [setup](setup.md) and [uninstall](uninstall.md).

Older `.envbuckets.toml` configs are unused and are never migrated. Remove the
old config, run `init`, and resolve any reported collisions before importing.
Share command output and path descriptions when asking for help; keep private
environment values out of bug reports.
