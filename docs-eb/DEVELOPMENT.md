# Developer checks

Use the Go version from `go.mod`, Task, Node.js, pnpm, and golangci-lint.
The required golangci-lint version is `GOLANGCI_LINT_VERSION` in `Taskfile.yml`;
keep the CI and release workflow pins in sync when upgrading it. `fmt`, `fix`, and `lint:go`
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
`lint:version` also checks the CI/release pins without requiring a local scanner. The
scan contacts the Go vulnerability database and checks reachable Go code; it
does not read managed environment files. It is advisory in the three-platform
safeguards matrix and separate from `task check`. Existing gosec, build, lint,
integration, and release gates remain blocking.

The main Linux CI job also runs `go test -race -shuffle=on -count=1 ./...` as a
required check. The initial trial found no races and added about ten seconds
relative to ordinary unit tests on that runner. Race detection requires cgo;
it stays out of local `task check` and the macOS/Windows jobs. Enable it on
additional platforms only with trial evidence.

## Release acceptance

All txtar scripts share one setup: private `HOME`, `USERPROFILE`,
`XDG_CONFIG_HOME`, and global Git config; system Git config is disabled; Git
author, committer, and dates are fixed. `testscript.Main` registers the real CLI
so installed hooks resolve the same command as the scripts.

`task test:integration` also checks that every PRODUCT.md CLI transcript has a
script assertion. The `acceptance-*.txtar` scripts compare output against the
documentation itself. Only fixture paths, CRLF, and displayed tabs are normalized;
Git's own checkout chatter is excluded when checking envbuckets hook messages.
Sample commands and output must stay in sync. `task test:integration:update`
sets `TESTSCRIPT_UPDATE=1` to allow txtar golden updates. Ordinary tests never
rewrite goldens, and documentation assertions always require manual review.

`task bench` runs the branch and hook benchmarks separately from tests. They
use one `git update-ref --stdin` setup call and are skipped with `-short`.
EB-07 records timings and review budgets; there are no timing assertions.

`task playground` builds a snapshot and runs `task test:npm`. The smoke test
packs the umbrella and current platform packages, installs them offline, and
exercises the npm launcher and checkout hook in a new Git repository under
ignored `playground/`. It checks all four dry runs, imports, reports, switching,
uninstall, and reinitialization using synthetic files. Existing playground
repositories are retained. Windows needs symlink privilege; that limitation
also applies to this smoke test.

The release workflow runs checks, security, npm smoke, and a comparison with
the published schema on `main` before creating a tag. The schema URL remains
on `main`; changing its branch requires an explicit compatibility decision.
Use the [acceptance map](ACCEPTANCE.md) and EB-07 exit checklist before release.
