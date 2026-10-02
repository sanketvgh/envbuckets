# CLI output and AI agents

envbuckets has a CLI integration for scripts and coding agents. It needs no
MCP server, network connection, or access to env file contents. Pass
`--json` before or after the command name:

```sh
envbuckets --json status
envbuckets check --json
envbuckets apply --dry-run --json
```

For people, failures use `envbuckets: <category>: <reason>` on stderr.
Nonfatal warnings use `envbuckets: warning:`. Successful commands describe
what changed. Display wording is free to improve; scripts and agents
should use the JSON contract below.

JSON mode prints one object to stdout and returns the command's process
exit code. It never waits for interactive input. The envelope has schema
version 1:

```json
{
  "schema_version": 1,
  "command": "check",
  "ok": false,
  "exit_code": 1,
  "data": {
    "branch": "feature/login",
    "resolved": true,
    "ready": false,
    "target": {
      "bucket": "dev",
      "source": "rule",
      "pattern": "*",
      "priority": 1
    },
    "scopes": [
      {
        "name": "root",
        "path": ".",
        "directory_exists": true,
        "link_state": "missing",
        "expected_file_exists": true,
        "ready": false,
        "issues": ["link_missing"]
      }
    ]
  },
  "error": {
    "category": "blocked",
    "message": "selected scopes are not structurally ready"
  },
  "output": "branch: feature/login\n..."
}
```

`schema_version`, `command`, `ok`, and `exit_code` are always present.
`error` appears on failure and has a `blocked`, `config`, `usage`, or
`environment` category. `warnings` holds nonfatal stderr text when present.
`output` is display text and may
change; do not parse it. `status`, `check`, `apply`, and `use --all`
return structured `data`. Other commands have the common envelope and error.

For `status` and `check`, `data.target.source` is `rule` or `pin`. The
target is absent when the branch is unresolved. Each scope reports a
`link_state` of `managed_symlink`, `missing`, `real_file`,
`foreign_symlink`, or `unknown`. `issues` uses these codes:

| Code                                                      | Meaning                                             |
| --------------------------------------------------------- | --------------------------------------------------- |
| `scope_missing`, `unsafe_scope_path`                      | Scope directory absent or unsafe.                   |
| `link_inspection_failed`                                  | `.env` could not be inspected.                      |
| `link_missing`, `real_env`, `foreign_link`, `broken_link` | `.env` link absent, unmanaged, or broken.           |
| `expected_file_missing`, `bucket_mismatch`                | Expected file or selection is wrong.                |
| `unresolved`                                              | No branch mapping applies, including detached HEAD. |

Use `data.ready` or the exit code to gate work. `check` returns exit 1
for an unready scope; `status` is read-only and can return 0 even when
`data.ready` is false. Issue codes and JSON field names are stable within
schema version 1. Human display text is not an API.

`apply` and `use --all` return a plan/result with `bucket`, `dry_run`,
`changed`, `unchanged`, `failed`, and `scopes`. Each scope has an `action`:
`would_change`, `changed`, `unchanged`, `blocked`, or `failed`. A blocked
or failed row has a human-facing `reason`. In a dry run, `changed` counts
planned changes; no link is written.

## Suggested agent workflow

1. Run `envbuckets check --json`. Inspect `data.ready`, `data.target`,
   and each scope's `issues`.
2. If a bucket file is missing, create it with `bucket add` only when an
   empty file is appropriate. Ask the user to fill secrets in their editor;
   do not request, read, or print env values.
3. Preview a repair with `envbuckets apply --dry-run --json`, then run
   `envbuckets apply --json` if it only changes managed links. Use
   `status --json` to inspect the result.
4. Treat a real `.env`, foreign symlink, unsafe path, or destructive
   `--purge` as a user decision. JSON mode does not accept typed
   confirmations, so purge operations requiring one are blocked.

`--json` also works with `init`, `bucket`, `map`, `scope`, `use`, `link`,
`unlink`, and `uninstall`. For a real `.env`, pass `init --into <bucket>`
so no prompt is needed. See [Status and recovery](status-and-recovery.md)
and the [command reference](command-reference.md) for command behavior.
