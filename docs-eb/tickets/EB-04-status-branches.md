# EB-04: status and branches

**Goal:** answer "which files am I using, is anything wrong, and which branch uses which bucket?" in the shape of `git status` and `git branch`.

**Status:** passed; local checks and same-code cross-platform CI accepted under the Windows verification rule.

## Phase checklist

- [x] Phase 1: review the product, shared filesystem/config/switch helpers, and test conventions.
- [x] Phase 2: implement read-only status and branches with shared branch resolution and safe hook inspection.
- [x] Phase 3: add exact-output unit/integration tests and restore benchmarks with 1 and 1000 branches.
- [x] Phase 4: run fix, review the diff, run lint/checks and benchmarks, inspect CI, and record limitations.
- [x] Phase 5: user authorized commit/push; EB-04 changes pushed, same-code cross-platform CI verified, and exit checklist closed.

## status

- First line: `On branch <name>`, or `HEAD detached at <commit>`.
- Bucket line: `Using bucket 'dev' (default)` or `(rule '<pattern>')`. On a detached HEAD there is no reason, just `Using bucket 'dev'` and a hint that links stay as they are. When the links differ from the branch's bucket, say so and add a hint in parentheses on the next line. When the branch's bucket is missing, show the fallback: `Using bucket 'dev' (default), because 'prod' (rule '<pattern>') does not exist` with the hint `(use "envbuckets switch -c prod" to create it)`.
- Links pointing at several buckets replace the bucket line with `Using a mix of buckets: 'dev', 'prod'` and the hint `(use "envbuckets switch" to link one bucket)`.
- Problem sections, each with a hint in parentheses and tab-indented lines, shown only when not empty:
  - `Files not linked:` with `real file:`, `broken link:`, and `foreign link:` labels.
  - `Not ignored by Git:` with the paths, and the hint `(add them to .gitignore)`.
  - `Hook not installed:` when the envbuckets block is missing from the post-checkout hook, with the hint `(use "envbuckets init" to install it)`. Tools such as `git lfs install --force` overwrite the hook file and drop the block.
- Last line: `all N files linked` (`1 file linked` for one), or `X of N files linked`.
- Config errors and a missing config print `fatal:` lines; a missing config hints to run `envbuckets init`.
- A leftover alpha `.envbuckets.toml` gets a `warning:` that it is unused and can be deleted.

## branches

- `branches` lists local branches sorted by name, like `git branch`: a `*` marks the current branch, then aligned columns for branch, bucket, and reason (`(default)` or `(rule '<pattern>')`).
- Plain `branches` shows the current branch and every branch a rule matches, nothing else: other branches use `default` by definition, so there is no count or summary line for them.
- Names, patterns, and `--bucket` show every match, including branches that use the default. `'**'` lists every branch, so there is no `--all` flag.
- The current branch's line adds `, using '<bucket>' for now` when the links point at another bucket.
- A bucket missing from `.env.d/` shows `(rule '<pattern>', missing; falls back to '<default>')`, matching what a checkout would link.
- `branches <name>...` checks only the given names, in the given order, whether or not they exist as branches.
- An argument containing `*`, `?`, or `[` is a pattern: it lists the local branches it matches, using the EB-01 matcher. Branch names cannot contain those characters, so names and patterns never clash.
- `--bucket <name>` keeps only the branches that use that bucket. It combines with names and patterns.
- Read all local branches with one `git for-each-ref refs/heads` call and match in memory; never run Git once per branch.

## Acceptance criteria

- Every status problem and every branches variant above has a test with the exact uncolored output.
- Both commands write to stdout, never read file contents, and change nothing.
- `status` works on a detached HEAD. Without a config it prints `fatal:` with a hint to run `envbuckets init`, and exits 1.
- `branches` uses the same matcher as the hook, so its answer always matches what a checkout would do.
- Rebuild `task bench` (removed in EB-00) on the new CLI, with a 1000-branch case. The hook's time does not change with the number of branches.

## Gaps, risks, and tradeoffs

**Gaps**

- The 1000-branch benchmark is implemented and measured on Windows. EB-07 still needs cross-platform measurements and release budgets; timings here include Git process startup and fluctuate with machine load.
- Git's docs do not state how `onbranch` treats a name equal to a pattern with a trailing `/`, so `branches` output for such names relies on the differential tests.

**Risks**

- `branches` and the hook must use the same matcher and the same missing-bucket fallback, or `branches` will say one thing and a checkout do another.
- Column alignment with color depends on Lip Gloss width measuring; if it misreads wide characters, columns drift.
- A branch name with non-ASCII characters must not break the column layout.

**Tradeoffs**

- Showing only the current branch and rule-matched branches keeps output short, but a user must pass `'**'` to see everything.
- Treating any argument containing `*`, `?`, or `[` as a pattern is unambiguous because branch names cannot contain them, but a user who types a bracket by mistake gets a pattern error instead of "no such branch".
- The `status` warning about a missing hook block helps after tools like `git lfs install --force`, at the cost of one more check on every `status`.

## Exit checklist

Tick every box before starting EB-06 or EB-07.

- [x] Every `status` problem section and line has a test with exact uncolored output.
- [x] Every `branches` variant has a test: plain, names, patterns, `--bucket`, current-branch marker, temporary bucket, missing bucket fallback.
- [x] `status` works on a detached HEAD; without a config it prints `fatal:` with the `init` hint and exits 1.
- [x] `status` reports a missing hook block with a hint.
- [x] Both commands write to stdout, never read file contents, and change nothing.
- [x] `branches` uses the same matcher and fallback as the hook, with a test that compares them.
- [x] Branches are read with one `git for-each-ref` call.
- [x] The 1000-branch benchmark exists and the hook time does not grow with the branch count.
- [x] `task check` requirement satisfied under the Windows verification rule: local lint/schema/unit checks and same-code cross-platform CI passed; local integration is blocked only by symlink privilege.

Every exit criterion is now verified. The Linux unit/race job and cross-platform
integration jobs cover the cases skipped locally; local Windows symlink privilege
remains unavailable and its setup remains deferred.

## Implementation notes

- `status` reports the active links, a temporary selection, missing-bucket fallback, mixed buckets, replaced/foreign/dangling links, ignore omissions, and missing or truncated hook blocks. A wholly absent working path uses `missing link:`; a tracked/reserved path or unsafe working parent uses `unsafe path:`. No-links detached HEAD and deleted default buckets also have explicit output and tests.
- Reports read only config and validated hook metadata. Managed file inspection uses directory entries, `Lstat`, and `Readlink`; foreign targets and symlink parents are never followed. Unsafe config, bucket, and hook symlinks are rejected. Snapshot assertions read synthetic fixtures only and verify commands leave them unchanged; the unreadable-file script covers mode-000 bucket and blocking working files on Unix.
- `switcher.Resolve` shares first-match selection and directory-level missing-bucket fallback with the switch planner and reports. A portable test compares branch rows with switch plans for exact names, trailing-slash rules, first-match exceptions, character classes, and defaults; a symlink-dependent test and real-checkout script compare fallback with the hook.
- Branches use one sorted `git for-each-ref refs/heads/` call and match/filter in memory. Literal names retain input order, patterns expand sorted local names, and overlapping selectors are deduplicated. `--bucket` filters the configured mapping even when it falls back. Git trace asserts a single enumeration call. Full symbolic refs avoid ambiguity with a same-named tag; stripping only line endings preserves Unicode whitespace in names.
- Plain column widths use the already-pinned `golang.org/x/text/width` module plus zero-width combining marks; its dependency is now direct, with no version change. Exact tests cover CJK and decomposed accents. EB-06 remains responsible for color and styled width measurement.
- `task bench` uses `testing.B.Loop`, skips short runs, creates 999 additional refs in one `git update-ref --stdin` operation, and excludes setup from timings. Both cases use an empty bucket so measurements isolate reporting/hook inspection and Git startup, without needing symlink privilege. The hook never enumerates branch refs.
- User documentation is in [Status and branch mappings](../STATUS.md); command help and the README link it. Local API references consulted: Go 1.27.2 `testing.B.Loop` and the pinned `width` package and `Properties.Kind` documentation.

## Verification evidence (2026-10-09)

- `task fix`: passed with zero issues; automatic changes reviewed. `task lint:go`: passed on the final source and tests, including formatting and modernize. The global linter is v2.13.2, so these commands used the existing repository-local `tmp/eb09/bin/golangci-lint.exe` v2.14.0; pins and global tools were not changed.
- `task check`: lint, JSON formatting, schema validation, and shuffled unit stages passed. Schema checks accepted eight PRODUCT.md examples and rejected nine invalid fixtures. All packages' unit checks passed; symlink-dependent cases skip in this Windows session. The final additional `TestStatusUnsafeWorkingParent` exact-output test also passed separately after formatting/lint.
- Integration failed only at symlink creation with Windows `ERROR_PRIVILEGE_NOT_HELD` (1314). Audited all nine failing scripts: `checkout`, `hooks`, `hooks-lfs`, `hooks-paths`, `switch`, `switch-safety`, `init-add`, `init-add-safety`, and the new `reports`. Unix mode-000 scripts skip on Windows. Final log: `tmp/eb04-check-final.log`; test temporary repositories were contained in `tmp/eb04-tests`. Symlink setup remains deferred as instructed.
- The new portable `TestScript/reports-empty` passed independently on the final implementation, verifying exact output, missing/invalid-config exit 1, literal future names, current marker, missing hook detection, and unchanged config/hook state. Log: `tmp/eb04-reports-empty.log`.
- `task bench` passed. Windows amd64/i5-12500H: listing 1000 branches measured 435–732 ms across runs. Repeated two-second hook samples had medians of 152.7 ms (1 branch) and 130.4 ms (1000 branches), with about 3850 allocations in both cases: no branch-count increase observed. Timings fluctuate; no timing assertion or release budget is inferred. Logs: `tmp/eb04-bench.log`, `tmp/eb04-bench-final.log`, and `tmp/eb04-hook-bench.log`.
- Inspected CI using `gh run list`, `gh run view`, and failed logs. Baseline [run 37965030776](https://github.com/sanketvgh/envbuckets/actions/runs/37965030776) passed on `db8e991`; the earlier failure was an unrelated EB-09 snapshot test. A later baseline [run 37966039031](https://github.com/sanketvgh/envbuckets/actions/runs/37966039031) on `5c6154e` was in progress at inspection. Neither verifies these EB-04 changes.
- At the initial handoff, no EB-04 changes were staged, committed, or pushed. Concurrent EB-09 edits/commits were preserved. Same-code CI was still needed to audit the skipped symlink/mode-000 cases and close the final exit box under AGENTS.md's Windows verification rule. The completed verification is recorded below.

## Continuation review (2026-10-10)

- Re-reviewed the implementation, exact-output tests, integration fixtures, and task file list after EB-09 completed. `git diff --check` passed; the working tree contains only the reviewed EB-04 changes. Previous local verification remains applicable; no implementation changes were made during this review.
- Current baseline `200189a` passed [CI run 37967269484](https://github.com/sanketvgh/envbuckets/actions/runs/37967269484). EB-04 is still uncommitted, so that run does not satisfy its same-code CI requirement.
- The user explicitly authorized committing/pushing EB-04 and verifying CI on 2026-10-10. Symlink setup remains deferred; Phase 5 is completed by the verification below.

## Closing verification (2026-10-10)

- Committed and pushed implementation `bbef602` (`✨ feat(cli): add status and branch reports`) to `feat/docs-demo`. Reviewed all 22 staged files against baseline `200189a`: only EB-04 changes were included. In particular, the benchmark's two `testing.TB` substitutions are the only harness changes; EB-09's maintenance fix was already committed in the baseline.
- [CI run 38020427583](https://github.com/sanketvgh/envbuckets/actions/runs/38020427583) on `bbef602` **passed all seven jobs**: Linux/macOS/Windows integration and safeguards, plus the main Linux Go lint/unit/race/build/snapshot/npm-artifact job. Advisory vulnerability scans also passed on all three platforms.
- Audited completed integration logs: `reports` and `reports-empty` passed on Linux, macOS, and Windows. `reports-unreadable` passed on Linux and macOS and skipped on Windows with the explicit Unix-mode-bits reason. This verifies actual checkout fallback, exact stdout, hook overwrite detection, and operation with mode-000 managed files.
- The local lint/schema/unit evidence above plus passing cross-platform CI for this implementation satisfy AGENTS.md's Windows verification rule. All exit boxes are closed; EB-04 is passed. Local integration still encounters Windows error 1314, and symlink setup remains deferred.
- The subsequent ticket/index commit changes documentation only; the verified production code and tests are unchanged.
