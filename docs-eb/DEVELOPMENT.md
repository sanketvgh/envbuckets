# Developer checks

Use the Go version from `go.mod`, Task, Node.js, pnpm, and golangci-lint.
The required golangci-lint version is `GOLANGCI_LINT_VERSION` in `Taskfile.yml`;
keep the `ci.yml` pin in sync when upgrading it. `fmt`, `fix`, and `lint:go`
stop with an installation hint if the binary is missing or its version differs.

After editing Go files, run:

```sh
task fix
task lint:go
```

Review the diff after `fix`. It formats Go files and applies the configured
linters' automatic fixes, including modernization. Some findings still require
judgment. The pinned linter can also return an old `ifElseChain` diagnostic after
modernize has rewritten the chain; `lint:go` checks the resulting code. Do not
ignore findings that remain on that second check.

`task lint` runs Go checks, oxfmt, and config schema validation.
`task check` runs lint, followed by Go unit tests and real-Git integration tests.
No pre-commit hook is installed by these commands.

CI runs `lint` on Linux, macOS, and Windows without rewriting the checkout.
The existing build, release snapshot, and integration jobs remain.

Integration tests require symlink support. When the local Windows session lacks
that privilege, retain the local lint, schema, and unit-test results and verify
the same code through the cross-platform integration job before closing a ticket.

Tool behavior is described in the [golangci-lint CLI documentation](https://golangci-lint.run/docs/configuration/cli/)
and [Task precondition documentation](https://taskfile.dev/docs/guide/conditional-execution).
