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

Unit and integration tests use `-shuffle=on` locally and in CI. Reproduce a
reported order with `go test -shuffle=<seed> -count=1 ./...`.

Expanded vet checks exclude field alignment and shadowing. The filesystem and
output guard rejects raw filesystem functions in `os` and `fmt.Print*` outside
the filesystem helpers and CLI. Tests, the config loader, and release packaging
have documented exclusions; six hook metadata call sites have explained
suppressions. Root-contained `os.Root` methods and writer-directed `fmt.Fprint*`
remain available.

`task security` runs a separate vulnerability scan. Install the pinned scanner:

```sh
go install golang.org/x/vuln/cmd/govulncheck@v1.8.0
task security
```

Its binary version and CI pin must match `GOVULNCHECK_VERSION` in the Taskfile.
`lint:version` also checks that CI pin without requiring a local scanner. The
scan contacts the Go vulnerability database and checks reachable Go code; it
does not read managed environment files. It is advisory in the three-platform
safeguards matrix and separate from `task check`. Existing gosec, build, lint,
integration, and release gates remain blocking.

The main Linux CI job also runs `go test -race -shuffle=on -count=1 ./...` as a
required check. The initial trial found no races and added about nine seconds
relative to ordinary unit tests on that runner. Race detection requires cgo;
it stays out of local `task check` and the macOS/Windows jobs. Enable it on
additional platforms only with trial evidence.
