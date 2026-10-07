# EB-00: Clear the alpha code (pre-work)

**Goal:** remove the alpha code, tests, and docs whose model conflicts with [`PRODUCT.md`](../PRODUCT.md), so EB-01 to EB-07 build on a small, clean base instead of working around it. Keep the generic plumbing.

**Why first:** the alpha is built around scopes, `.git/config` branch pins, TOML rule editing, a `*` catch-all, and `--json`. None of that survives, and it already gets in the way: EB-01's Git-style matcher breaks the alpha's `*` catch-all and `testdata/script/branch_link.txtar`.

**Base:** `origin/main` (`e22256c`). Local `main` and `feat/docs-demo` are behind it.

## Delete

| Path                                                                                                                                                                                                         | What it is                                    | Why it goes                                                                                   |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------- | --------------------------------------------------------------------------------------------- |
| `internal/cli/` command files: `cmd_apply.go`, `cmd_bucket.go`, `cmd_check.go`, `cmd_hook.go`, `cmd_init.go`, `cmd_link.go`, `cmd_map.go`, `cmd_scope.go`, `cmd_status.go`, `cmd_uninstall.go`, `cmd_use.go` | Alpha commands                                | Every one is scope-, pin-, or TOML-based; EB-02 to EB-05 write the new commands               |
| `internal/cli/`: `bucket_all.go`, `gitignore.go`, `help_groups.go`, `json.go`, `json_plan.go`, `json_readiness.go`, `map_explain.go`, `project.go`, `readiness.go`, `scaffold.go`, `scope_cleanup.go`        | Alpha support code                            | Scope resolution, per-scope ignore lines, `--json`, `check`, `init --scaffold`, `map explain` |
| `internal/cli/*_test.go` for the files above, and `setup()` in `harness_test.go`                                                                                                                             | Alpha tests                                   | They test removed behavior; `setup()` builds an alpha scope repo                              |
| `internal/config/`                                                                                                                                                                                           | TOML config, scopes, rule editing, `CatchAll` | EB-01 writes the JSON config                                                                  |
| `internal/gitx`: `BranchExists`, `BranchLink`, `LinkedBucket`, `SetLink`, `Unlink`, `Links`                                                                                                                  | `.git/config` branch pins                     | The product never uses `.git/config`                                                          |
| `testdata/script/*.txtar` (19 scripts)                                                                                                                                                                       | Alpha integration scripts                     | They assert alpha behavior                                                                    |
| `docs/` (10 pages)                                                                                                                                                                                           | Alpha user docs                               | Contradict the product; EB-07 writes new docs                                                 |
| `Taskfile.yml`: `bench:fixture`, `bench`, `bench:compare`                                                                                                                                                    | Hook benchmarks                               | The fixture calls `bucket add` and `map add`; EB-04 rebuilds them                             |
| `github.com/BurntSushi/toml`                                                                                                                                                                                 | TOML parser                                   | Unused once `internal/config` goes; run `go mod tidy`                                         |

## Keep

| Path                                                                                                                                                             | Why                                                                                                                                                 |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/block`                                                                                                                                                 | Marker-guarded hook and `.gitignore` blocks. It finds any `# >>> envbuckets ` block, which EB-02 and EB-03 need to replace the alpha's `v1` blocks. |
| `internal/fsx`                                                                                                                                                   | Symlink inspection, atomic symlink swaps through `os.Root`, atomic writes for the config and `.gitignore`.                                          |
| `internal/gitx`: `Root`, `Branch`, `HooksDir` (with its local-hook-path check), `run`                                                                            | Repo root, current branch, and safe hook location.                                                                                                  |
| `internal/pattern`                                                                                                                                               | EB-01 replaces it, and work on that is in progress. Do not touch it here.                                                                           |
| `main.go`, `integration_test.go` (git sandbox, `readlink` and `regular` commands), `harness_test.go` helpers (`newRepo`, `run`, `ok`, `hook`, `requireSymlinks`) | Generic test and entry-point plumbing.                                                                                                              |
| `npm/`, `tools/npmpkg`, `.goreleaser.yaml`, `.github/` workflows, `.golangci.yml`, oxfmt setup, the other Taskfile tasks                                         | Packaging, CI, and security gates do not change.                                                                                                    |

## Replace with a minimal skeleton

- **`internal/cli/cli.go`:** keep `Run`, `Env`, flag parsing, and error reporting. Commands: `version`, `help`, and `hook`.
  - `hook` stays as a silent no-op that exits 0. Repos with the alpha hook installed still call `envbuckets hook`, and checkouts must stay quiet until EB-02.
  - An unknown command prints `usage:` and exits 2.
  - Exit codes follow `PRODUCT.md`: `0`, `1`, `2`. This replaces the alpha's `0`-`4`; update `main.go`, which uses `ExitEnv`.
- **`testdata/script/`:** one smoke script, because testscript fails on an empty folder (`no txtar nor txt scripts found`). It runs `envbuckets version` and checks that `envbuckets hook` prints nothing and exits 0.
- **`README.md`:** a short notice that envbuckets is being rebuilt, a link to `docs-eb/PRODUCT.md`, and how to install the last alpha from npm. Keep the badges. `tools/npmpkg` copies this README into the npm package, so EB-07 must replace it before the next release.
- **`AGENTS.md` and `CLAUDE.md`:** point to `docs-eb/PRODUCT.md` and the tickets. Remove the alpha rules that would steer agents wrong: the `.env.d/<bucket>/.env` description, scope rules, keeping `--json` schema 1 fields, and links to `docs/`. Keep the check, security, hook, main-gate, and commit rules.

## Carry over before deleting

`internal/cli/security_paths_test.go` holds attack cases worth keeping. Rewrite these in EB-01's path tests (or the matching command ticket), then delete the file:

- A symlinked bucket directory cannot write or delete outside the repo.
- A symlinked bucket file is not treated as a bucket entry.
- A link into a bucket with a non-canonical target counts as foreign.
- An external or symlinked hook path is refused.
- A symlinked `.gitignore` cannot be used to read other files.
- `uninstall` does not move files from outside the buckets.
- `init` does not move a real `.env` through a symlinked bucket directory.

The scope and `bucket rm --purge` cases go with their features.

## Acceptance criteria

- `go build ./...`, `go vet ./...`, `task lint`, `go test ./...`, and `task test:integration` pass, and the integration run executes the smoke script on Linux, macOS, and Windows in CI.
- `go mod tidy -diff` prints nothing, and `go.mod` has no TOML dependency.
- Outside `docs-eb/`, no code, script, or doc mentions scopes, the `use`, `apply`, `check`, `link`, `unlink`, `map`, or `scope` commands, `.envbuckets.toml`, or `--json`.
- In a repo with the alpha hook block, a branch checkout with the new binary prints nothing and succeeds.
- The security workflow passes after the dependency change (`task security`).

## Notes

- Releases are manual (`workflow_dispatch`), so `main` can sit between EB-00 and EB-07 without shipping a half-built CLI. Do not cut a release until EB-07.
- The uncommitted EB-01 matcher work on `feat/docs-demo` changes `internal/pattern` (kept here) plus `internal/config/config.go` and `docs/configuration.md` (deleted here). Land this ticket first, then rebase that work and drop those two edits.
