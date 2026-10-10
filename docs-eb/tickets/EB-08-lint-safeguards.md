# EB-08: Earlier lint and format safeguards

**Goal:** catch simple Go mistakes (formatting, outdated idioms, avoidable `fmt.Errorf`) before CI, with the same results locally and in CI. This ticket stays last; do it after EB-07 ships.

**Status:** Done. Implemented ahead of EB-07 at the user's request. [CI run 37954669747](https://github.com/sanketvgh/envbuckets/actions/runs/37954669747) passed for implementation commit `fcdfa8c`.

## Implementation phases

- [x] Review existing tooling and choose the optional hook and CI scope.
- [x] Add the pinned version guard, Go tasks, default modernize analyzers, and editor safeguard; fix all existing findings.
- [x] Verify synthetic cases locally and add cross-platform CI safeguards without changing the existing gates.
- [x] Verify local tooling and record the results and developer workflow.
- [x] Verify Linux/macOS/Windows CI and the integration suite for this exact code before closing the ticket.

**Scope:**

- Enable `modernize` in `.golangci.yml` with its default analyzers. Disable an analyzer by name only after it proves noisy. Fix every existing finding in the same change, including the committed EB-01 code in `internal/config/schema_test.go` that was never checked.
- Pin golangci-lint in one place. Add `GOLANGCI_LINT_VERSION` to `Taskfile.yml` and an internal `lint:version` task that fails when the installed version differs from it or when `.github/workflows/ci.yml` pins a different one.
- Add `task fix` (`golangci-lint fmt ./...` then `golangci-lint run --fix ./...`) and `task lint:go` (version check, `fmt --diff`, `run`). `task lint` calls `lint:go`, then the existing oxfmt and schema steps. `task check` and CI keep the same coverage.
- Add one `AGENTS.md` line: after editing Go files, run `task fix`, then `task lint:go`.
- Add `"source.fixAll": "explicit"` to the Go `editor.codeActionsOnSave` in `.vscode/settings.json`.
- Optional, decided at the start of the ticket: an opt-in `.githooks/pre-commit` (mode 0755) and `task hooks:install`. The hook runs `golangci-lint fmt --diff ./...` and `golangci-lint run --new-from-rev=HEAD ./...` only when staged `.go` files exist. It exits 0 when `golangci-lint` is missing. It sets `core.hooksPath`, which replaces today's empty `.husky/_` value.

The implemented workflow is documented in [Developer checks](../DEVELOPMENT.md). Test every Taskfile change on Windows, Linux, and macOS.

## Acceptance criteria

- `task lint:go` fails on an unformatted file, an `if/else` chain that should be a `switch`, a `fmt.Errorf` without verbs, `errors.As` with a pointer target, `reflect.TypeOf` of a constant type, and `strings.Split` in a `range`.
- `task fix` repairs those cases, leaves code that needs judgment alone, and never touches files outside `./...`.
- `lint:version` fails with a clear message when the local golangci-lint, the Taskfile pin, or the `ci.yml` pin disagree.
- `task lint` and CI cover everything they did before. The oxfmt and schema steps still run in `task lint`.
- No new rule changes how files are read, parsed, logged, or printed, and none relaxes the filesystem or symlink checks.
- With the optional hook installed, a commit with no staged Go files is not slowed, and a missing `golangci-lint` never blocks a commit.

## Gaps, risks, and tradeoffs

**Gaps**

- The new safeguards matrix runs oxfmt and `validate:schema` through `task lint` on Linux, macOS, and Windows; the former local/CI coverage gap is closed.
- No separate Windows-target lint step is needed: there are no Windows-only source files or platform build tags, and the safeguards matrix lints on Windows directly.
- A `forbidigo` rule limiting raw `os.*` filesystem calls and printing outside `internal/fsx` and the CLI layer would back the safety constraints. The config loader legitimately reads files, so the exceptions need design. Deferred.
- Linter runtime before and after enabling `modernize` is recorded below.

**Risks**

- A golangci-lint upgrade can add `modernize` analyzers and new findings. The pin and the drift check make upgrades deliberate.
- `run --fix` rewrites code. It stays manual and is reviewed in the diff.
- The pre-commit hook can annoy contributors or conflict with their own `core.hooksPath`. It stays opt-in and never blocks when tools are missing, like the product's own checkout hook.
- Version preconditions use Task's shell builtins instead of external `grep`, avoiding a Git-for-Windows PATH dependency. The CI pin check accepts LF and CRLF and expects the existing unquoted, two-space-indented env pin.
- The pinned linter can report a stale `gocritic` if/else finding while `modernize` fixes it. `task fix` retains that nonzero status; run `task lint:go` afterwards and resolve any remaining findings.

**Tradeoffs**

- The version lives in the Taskfile and in `ci.yml`, with a drift check instead of one source. A `go tool` entry in `go.mod` would give one source, but the golangci-lint docs discourage installing it that way and it pulls its dependencies into this module.
- `modernize` with default analyzers is the lowest-noise option today. Adding style linters (`gocyclo`, `funlen`, `lll`, `godot`) would raise the finding count without catching these mistakes, so they are out of scope.
- The editor setting helps only developers who use VS Code. CI stays the gate.

## Exit checklist

Tick every box to close the project's ticket list.

- [x] `modernize` is enabled and the repo is clean under it, including `internal/config/schema_test.go`.
- [x] `GOLANGCI_LINT_VERSION` is pinned in `Taskfile.yml` and `lint:version` fails on any mismatch with the installed binary or `ci.yml`.
- [x] `task fix` and `task lint:go` exist; `task lint` and `task check` still run every earlier step.
- [x] The `AGENTS.md` line and the `.vscode` setting are added.
- [x] The optional pre-commit hook is added and verified, or the decision to skip it is recorded here.
- [x] The CI decisions are recorded here: oxfmt and schema steps, and the Windows-target lint step.
- [x] Runtime of `task lint:go` before and after is recorded here.
- [x] Checked on Linux, macOS, and Windows.
- [x] Local lint/schema/unit checks and cross-platform CI integration pass via the accepted verification alternative; existing CI and security gates are unchanged.

## Decisions and verification

- Optional pre-commit hook: skipped at the start of this work to preserve contributor hook configuration; `core.hooksPath` is untouched.
- CI: add a safeguards matrix on Linux, macOS, and Windows with Node/pnpm, pinned Task, and `task lint`. This closes the oxfmt/schema gap and exercises the version guard. CI does not run `task fix`. Existing CI jobs, permissions, and release gates are preserved.
- Windows-target lint: no separate `GOOS=windows` step; there are no Windows-only source files or platform build tags. The safeguards matrix runs the real Windows linter.
- Runtime: the original equivalent Go lint commands (`fmt --diff` plus `run`) took 13.29 seconds locally. After enabling modernize, `task lint:go` including the version guard took 16.53 seconds. These are single wall-clock measurements with differing cache state, not a benchmark.
- Local Windows checks: `task fix`, `task lint:go`, oxfmt, schema validation, Go unit tests, and the standalone modernize run pass. All seven existing modernize findings were fixed without suppressions.
- One-time synthetic checks rejected all six requested mistake classes, verified their repairs, preserved a dynamic formatting case and a file outside `./...`, and rejected Taskfile/binary and CI pin mismatches. CRLF pin checks also passed. The dedicated test script and task were removed at the user's request; they are not retained or run in CI.
- `task check` reaches the real-Git integration scripts but fails solely because this Windows session lacks symlink privilege. The accepted local/CI alternative is satisfied by the local checks above plus [CI run 37954669747](https://github.com/sanketvgh/envbuckets/actions/runs/37954669747) on `fcdfa8c`: safeguards and integration passed on Linux, macOS, and Windows, and the original CI job passed lint, unit tests, the release snapshot, and npm package layout. All exit criteria are complete.
