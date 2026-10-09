# EB-09: Stricter bug-catching checks

**Goal:** catch real bugs earlier by adding only the checks that prove useful on this codebase. This ticket stays last. Each check is added only if a trial run shows findings that are mostly real.

**Status:** Local implementation and Linux race trial verified; awaiting CI for the final blocking-race configuration. Work stays in the main worktree. The user authorized commits and pushes to complete verification and record Passed status.

## Implementation phases

- [x] Inspect existing checks and record baseline runtime.
- [x] Trial each candidate without automatic fixes; classify findings and choose checks.
- [x] Configure retained checks and fix actionable findings; document narrow exclusions.
- [x] Verify synthetic violations, repeated shuffle seeds, local checks, and tool-pin drift.
- [ ] Verify the same code in Linux/macOS/Windows CI and record final evidence.

**Scope:**

- Trial first, on a throwaway branch with no `--fix`. Run `golangci-lint run` with each candidate below and record the finding count and how many are real. Drop any check that is mostly noise.
- `govet` with `enable-all`, disabling `fieldalignment` and `shadow`.
- `errcheck` with `check-type-assertions: true` and `check-blank: true`. Each `exclude-functions` entry carries a reason.
- `forbidigo` with `analyze-types: true`. Forbid raw `os.*` filesystem calls (such as `os.ReadFile`, `os.Symlink`, `os.Lstat`, `os.Remove`, `os.Rename`) and `fmt.Print*` outside `internal/fsx` and the CLI layer. Exclusions cover the config loader and tests. Every use of `//nolint:forbidigo` needs a reason, which `nolintlint` already enforces.
- `go test -shuffle=on` in `task test` and CI. Fix any order dependence it exposes.
- `go test -race` on one Linux CI job only, unless the trial shows value elsewhere. Record the runtime cost.
- `govulncheck ./...` as a CI step, non-blocking at first. Record any false positives. Check what `task security` already covers before adding it.
- Pin any new tool version in the Taskfile and in `ci.yml`, using the drift check from EB-08.

Exact config comes from the research summary for this ticket and the trial results.

## Acceptance criteria

- Each added check has a recorded trial result: findings, how many were real, and the decision.
- A check that is mostly noise is not added. The ticket records why.
- `forbidigo` flags a direct `os.*` filesystem call or a print in a package outside the allowed ones, and passes on `internal/fsx` and the config loader.
- `-shuffle` passes repeatedly with different seeds.
- No new rule reads, parses, logs, or prints real environment-file contents, or weakens the filesystem and symlink checks.
- Existing CI and security gates are unchanged.

## Gaps, risks, and tradeoffs

**Gaps**

- No evidence was found for coverage thresholds, `GOOS=windows go vet`, or build-tag handling. They are out of scope unless a bug shows the need.
- Research line ranges and several source SHAs were unverified. Treat the cited projects as pointers.
- All candidates now have trial evidence below, including the Linux race trial. Findings remain limited to the code and tests exercised.

**Risks**

- `govet` `enable-all` can add analyzers on a golangci-lint upgrade. The version pin makes upgrades deliberate.
- `errcheck` strict mode needs many exclusions for cleanup and close calls. Over-excluding hollows it out.
- `forbidigo` needs hand-written exclusions. Too few cause false positives. Too many weaken the guard.
- `-race` is slower and needs cgo. `govulncheck` can false-positive on interface calls and misses reflect calls.

**Tradeoffs**

- `nilaway` is excluded: it is not in golangci-lint and reports false positives, so it would be a separate advisory tool.
- `goleak`, `fieldalignment`, `wrapcheck`, `varnamelen`, `contextcheck`, and `paralleltest` are excluded for low payoff or high noise. `exhaustive`, `testifylint`, and `durationcheck` are skipped until a matching mistake appears.
- Fuzz smoke is excluded for now. Revisit if a parser bug appears in the config loader or hook block parser.
- A narrow `forbidigo` rule set gives the strongest protection for the safety rules but costs the most to maintain.

## Exit checklist

Tick every box to close the project's ticket list.

- [x] A trial result is recorded for every candidate check, with a keep or drop decision.
- [x] Kept linters are configured in `.golangci.yml`, and the repo is clean under them.
- [x] `forbidigo` rules and exclusions are documented, and each exclusion has a reason.
- [x] `-shuffle` is on in `task test` and CI. `-race` and `govulncheck` decisions are recorded.
- [x] New tool versions are pinned with the EB-08 drift check.
- [x] Runtime of `task check` before and after is recorded.
- [ ] Checked on Linux, macOS, and Windows.
- [ ] `task check` passes, and existing CI and security gates are unchanged.

## Trial setup and baseline

- User override: work in the main worktree on `feat/docs-demo`; the temporary `chore/eb09-trial` worktree and branch were removed before trials. Candidate configs and logs are under ignored `tmp/eb09/`; all trial linter runs omit `--fix`.
- Baseline `task check`: 128.83 seconds, exit 201. Lint, schema validation, and all unit packages passed. Integration failed solely on Windows symlink privilege; setup remains deferred per AGENTS.md. This is a failing full-check timing, not a successful benchmark.
- There is no `task security` or security workflow in the current checkout. Existing security coverage is the blocking `gosec` linter, the CI build/release snapshot; these remain unchanged.
- Existing [CI run 37960775401](https://github.com/sanketvgh/envbuckets/actions/runs/37960775401) passed on `30f5ba8`, including Linux/macOS/Windows safeguards and integration. It verifies the baseline only, not these uncommitted changes.

## Candidate trials and decisions

The first three linter trials used golangci-lint 2.13.2, Go 1.27.1, unlimited finding counts, and separate configs without fixes or the std-error-handling preset. Times below are single wall-clock samples with mixed cache state.

| Candidate | Findings and real issues | Decision | Trial time |
| --- | --- | --- | --- |
| Expanded govet | 0 findings, 0 current bugs; final pinned linter catches all 3 planted unused-field-write, nil-dereference, and non-slice-sort bugs | Keep: no false positives, and the extra analyzers catch concrete mistakes | 4.25 s |
| Strict errcheck | 109 diagnostics: 12 actionable unchecked test reads/stat/directory listings; 31 closes, 42 output writes, 14 cleanup removals, 2 cleanup chmods, 7 deliberate assertions, 1 deliberately ignored symlink-probe outcome | Drop both stricter settings: 97/109 are low-value here. Fix the 12 test errors, retain default errcheck, and add no exclude-functions entries | 3.43 s |
| Initial forbidigo scope | 17 diagnostics, 0 unsafe operations demonstrated: 6 hook-metadata operations and 11 release-packaging operations | Drop this noisy scope. Refine with packaging exclusions and six line-specific metadata explanations, then verify actual prohibited synthetic operations | 4.78 s |
| Shuffle | All unit packages pass with seeds 1, 42, and 20261009; 0 order-dependent failures | Keep; enable in unit and integration commands locally and in CI | 43.95 / 21.79 / 21.69 s |
| Race | Linux CI trial passes with 0 race reports, 0 false positives; local Windows has no cgo/gcc and its Linux Docker daemon is unavailable | Keep as a required step in the existing Linux job only; no race flag in local task check or macOS/Windows jobs | About 12 s vs 3 s plain (step timestamps); exact shell timings below |
| govulncheck v1.8.0 | 1 reachable standard-library vulnerability, 1 real: GO-2026-6604 in Go 1.27.1. The 12 additional module-only vulnerabilities are unreachable and do not fail the scan; no false positive was identified | Keep advisory scans on all three CI platforms, because the real finding is Windows-specific. Add separate task security and pin checks | 6.54 s (scan only) |

### Vulnerability and tool compatibility

- [GO-2026-6604](https://pkg.go.dev/vuln/GO-2026-6604) describes Windows junction escapes in Root.Mkdir/MkdirAll and is fixed in Go 1.27.2. Raise the go.mod requirement from 1.27.1 to 1.27.2. The resulting task security scan passed with no vulnerabilities in 4.25 seconds.
- The installed golangci-lint 2.13.2 uses x/tools 0.49.0 and cannot decode Go 1.27.2 export-data version 5; task fix failed with typecheck errors, which were not suppressed. Upgrade both linter pins to 2.14.0 (x/tools 0.50.0 supports version 5). Final local task fix and task lint:go pass with no findings, using a 2.14.0 binary built with Go 1.27.2 under tmp/eb09/bin.
- The scanner is installed only under ignored tmp/eb09/bin for this work; local PATH changes are process-scoped. No user tool installations or settings are changed.

### Guard scope and exclusions

- The os filesystem function list covers reads, writes, opens, directory creation/listing, stat/link operations, removals/renames, permissions/ownership, timestamps, and truncation. Type analysis catches renamed imports. It deliberately does not forbid os.Root methods, metadata constants, or writer-directed fmt.Fprint* calls. It is an architectural guard, not a proof that every operation is safe.
- internal/fsx implements the guarded operations; internal/cli owns command output and filesystem preflight. Tests use synthetic fixtures; internal/config/config.go loads only the shared config. tools/npmpkg is a release-packaging executable that reads build metadata and copies artifacts, rather than managed files. Each exclusion is limited to forbidigo and has a nearby reason.
- Hook metadata has six individually explained suppressions; there is no whole-package block exclusion. The legacy path-based hook API assumes its caller supplies a hook script. Existing safety checks and behavior are unchanged.
- nolintlint explanation and specific-linter requirements are now explicit (the defaults did not actually enforce the ticket's assumed explanation requirement). No errcheck function exclusion is added.

### Primary references verified during implementation

- [Linter settings](https://golangci-lint.run/docs/linters/configuration/) for govet, errcheck, forbidigo, and nolintlint.
- [Go vulnerability tutorial](https://go.dev/doc/tutorial/govulncheck) and the advisory above for reachable vulnerability scanning.
- [Race detector](https://go.dev/doc/articles/race_detector) for cgo requirements and runtime considerations.
- [Linter Go support](https://golangci-lint.run/docs/welcome/faq/) for compiler compatibility.

## Local verification

- Final `task check`: 59.93 seconds, exit 201, with lint/format/schema and unit tests passing; integration is still blocked solely by Windows symlink privilege. Before/after timings have different Go/linter versions and cache state, so the faster final sample does not establish a performance improvement.
- Repeated unit tests on final Go 1.27.2 code pass with `-shuffle=1`, `42`, and `20261009`, each with `-count=1`. Tests still use synthetic file contents exclusively.
- A separate, ignored synthetic module copies the final linter config. `forbidigo` rejects 85/85 prohibited references: each of 25 os filesystem functions and 3 fmt.Print variants in three disallowed paths, plus a dot import. Renamed imports are recognized. It emits no findings in fsx, CLI, the exact config loader, tests, or packaging. A sibling package named notfsx and a non-loader config file are correctly rejected, proving exclusions are narrow. Keep this refined scope: 85 real synthetic violations, zero false positives; the noisy initial scope is dropped.
- Expanded govet rejects 3/3 planted bugs through unusedwrite, nilness, and sortslice. `nolintlint` rejects both a forbidigo suppression without an explanation and a suppression without a named linter.
- Tool-pin checks reject scanner CI drift through both lint and security, scanner binary mismatch, linter CI drift, and linter binary mismatch with the intended messages. Matching CRLF Taskfile/CI pins pass lint and security. Probe edits are restored byte-for-byte.
- Synthetic source is compiled for analysis only, never executed. Trial fixtures, logs, and binaries stay under ignored tmp/eb09; no helper is added to normal CI or project commands.
- [Trial run 37963815199](https://github.com/sanketvgh/envbuckets/actions/runs/37963815199) on `d8db05e` passed the Linux CI job, including the advisory race trial, unit tests, release snapshot, and npm layout. All three integration jobs also passed. The race trial reported no failures and took about 12 seconds versus 3 seconds plain by step timestamps. Keep it as a blocking Linux-only check and re-run the final configuration before marking Passed. Cross-platform safeguards are still being reviewed.
