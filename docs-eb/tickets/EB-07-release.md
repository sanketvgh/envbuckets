# EB-07: Release and acceptance

**Goal:** ship the new product as one coherent CLI.

**Scope:** one shared test setup for all integration scripts (`HOME`, `XDG_CONFIG_HOME`, `GIT_CONFIG_GLOBAL`, `GIT_CONFIG_NOSYSTEM=1`, fixed git author, committer, and dates) and the CLI registered with `testscript.Main` so scripts and the hook call the real command; golden updates only behind `TESTSCRIPT_UPDATE=1` in a task target; a `testing.B` benchmark for 1000 branches in its own file, skipped under `-short`, creating branches with one `git update-ref --stdin` call (the setup in Git's own `t/perf/p6300-for-each-ref.sh`) and judged against a budget in the ticket; a documented Windows note: Go's rename on Windows is one `MoveFileEx(MOVEFILE_REPLACE_EXISTING)` call with no delete step in our code, but Microsoft does not document it as atomic, so `switch` converges on rerun after an interruption; and a note that symlink tests need Windows Developer Mode, which was off on the research machine, so run them in CI or enable it locally; README, user docs, and command help for the new product, replacing EB-00's placeholder README before any release (the npm package ships it); finish `AGENTS.md` and `CLAUDE.md` for the finished CLI; make sure `schema/envbuckets.schema.json` is on `main` so the `$schema` URL resolves, and show editor setup in the README; end-to-end scripts for every acceptance criterion in `PRODUCT.md`; npm package smoke test.

## Phase checklist

**PR #6 continuation:** integration of the newer `main` security/formatting
commits adds metadata-read hardening and restores the blocking security,
CodeQL, and dependency-review gates. [Fresh CI on `032178f`](https://github.com/sanketvgh/envbuckets/actions/runs/38038426065)
passed all ten jobs, including blocking security and all cross-platform
integration, safeguards, and npm smoke. CodeQL and dependency review also
passed; all three advisory scanners explicitly found no vulnerabilities.
EB-10 records the action-comment correction and full evidence. The follow-up
changes only tickets; the earlier `7d6dd44` record below remains historical.

**EB-10 handoff (2026-10-10):** the Windows Terminal visual check is deferred
until after release at the user's request and no longer blocks this ticket.
Public guides now live under `docs/` with lowercase slug filenames. Verify the
schema and public documentation URLs on remote `main` after merging the PR;
the schema gate still blocks publication. The release-runner benchmark review
remains owned by this ticket, using the budgets below.

**Final-code evidence:** [CI run 38030155164 on `7d6dd44`](https://github.com/sanketvgh/envbuckets/actions/runs/38030155164)
passed all nine jobs, including cross-platform integration/npm smoke and
Linux race checks. All three advisory vulnerability scanners explicitly reported
no vulnerabilities. EB-10 records the Windows runner warning and successful
unchanged-code retry. The subsequent handoff update changes only tickets and
restores the user's early-alpha README warning; runtime, packaging code, schema,
tests, and user guides match the tested SHA. The final README was reverified in
a local snapshot. This completes the pre-PR verification; the remote `main`
schema and release-runner benchmark gates remain open until after merge.

- [x] 1. Audit acceptance coverage; isolate the shared integration setup and gate golden updates.
- [x] 2. Check PRODUCT.md examples against the real CLI, fill acceptance gaps, and measure the branch benchmark.
- [ ] 3. Replace release documentation and help; document Windows limits and verify the schema URL.
- [x] 4. Add and run a packed npm smoke test in its own playground Git repository; preserve release gates.
- [x] 5. Run formatting, lint, checks, and security; inspect CI and record release blockers.

## Implementation and verification evidence

- Acceptance coverage is mapped in [ACCEPTANCE.md](../ACCEPTANCE.md). The new
  `acceptance-*.txtar` scripts use the real command and compare the documentation
  itself, with a coverage test that checks matching commands. Existing pattern,
  schema, filesystem, color, hook, and dry-run checks remain in place.
- All integration scripts share isolated HOME/USERPROFILE/XDG settings, a private
  global Git config, disabled system config, and fixed author/committer/dates.
  Golden rewriting is enabled only with `TESTSCRIPT_UPDATE=1`, supplied by the
  explicit `task test:integration:update` target. Documentation assertions are
  never automatically rewritten.
- The README, help, user docs, AGENTS.md, and CLAUDE.md describe the finished CLI.
  PRODUCT.md's dry-switch and uninstall examples now follow actual sorted path
  order. Legacy compatibility fixtures retain coverage without stale alpha copy;
  prerelease channels and Git's `[:alpha:]` character class are intentional.
- The published schema URL stays on `main` for continuity. GitHub's content API
  returned 404 for `schema/envbuckets.schema.json` on remote `main` at
  `e22256c61914e504ed220b033dacda1974031417`. Merging the schema there remains a
  release blocker. The release workflow now checks both availability and matching
  bytes before tagging. The user authorized committing and pushing this work to
  the current branch; merging to main, tagging, and publishing remain separate.
- Packed npm smoke is implemented in `tools/npm-smoke.mjs` and exposed through
  `task test:npm` / `task playground`. It creates its own Git repository, installs
  local tarballs offline, checks the shipped README, calls the npm launcher, tests
  the installed hook, hashes all four dry runs, and exercises restoration and
  reinitialization. Linux/macOS/Windows CI run it; release runs it before tagging.
- Release checks now use the same pinned linter as Task/CI and require full checks
  and security before tagging. Integration-tagged Go tests are included in local
  lint/fix, and the smoke script is included in the JS formatter check. Tool-pin
  guards now verify both CI and release workflows.

### Branch benchmark budget

The existing `internal/cli/bench_test.go` uses `testing.B`, skips under `-short`,
and creates extra refs with one `git update-ref --stdin` call, following
[Git's perf setup](https://github.com/git/git/blob/master/t/perf/p6300-for-each-ref.sh).
Setup is outside `b.Loop` timings; the hook fixture has no managed files and
measures branch-independent dispatch/planning, not a workload of file swaps.

First local measurement: Windows/amd64, Go 1.27.2, Intel Core i5-12500H;
`go test -run '^$' -bench BenchmarkReports -benchtime=3x -count=3 -benchmem ./internal/cli`.

| Case            | Three observed timings         | Review budget on this machine |
| --------------- | ------------------------------ | ----------------------------- |
| `branches/1`    | 264.46 / 220.37 / 229.22 ms/op | 500 ms/op                     |
| `branches/1000` | 515.48 / 522.49 / 437.15 ms/op | 1,000 ms/op                   |
| `hook/1`        | 167.80 / 161.61 / 134.43 ms/op | 350 ms/op                     |
| `hook/1000`     | 131.72 / 125.40 / 179.32 ms/op | 350 ms/op                     |

All cases fit the budgets. These short, warm-cache samples establish a review
baseline, not a cross-runner timing guarantee. Remeasure on the release runner
and investigate budget overruns; no flaky timing assertion is added.

### Platform and CI limits

The Windows session still lacks symlink privilege. The new walkthrough/clone/team
scripts reach the same required-privilege error as existing integration tests;
Developer Mode remains deferred per AGENTS.md. Rule examples, the 1000-ref
enumeration, and documentation sample coverage pass without symlinks. Permission
scripts skip Windows explicitly because `chmod 000` needs Unix mode bits.

The baseline [CI run 38025251583](https://github.com/sanketvgh/envbuckets/actions/runs/38025251583)
passed on `b8985234154aec597397b7276afe7313ebfbe33f`. The EB-07 implementation
is now verified by the same-code run recorded below. EB-06's manual Windows
Terminal review remains pending separately.

### Local verification (2026-10-10)

- `task fix` and the reviewed follow-up `task lint:go` pass with zero findings,
  including integration-tagged tests, using the existing pinned tools under
  ignored `tmp/eb09/bin` through process-local PATH only. Final `task lint` also
  passes after the CI/release pin guards and smoke script were finalized.
- `task check`: Go/JS formatting, Go lint, all 8 documented schema examples,
  all 9 invalid schema fixtures, and shuffled unit tests pass. Integration fails
  only at symlink creation in 14 scripts with Windows's required-privilege error.
  This is not a passing local full check. Passing same-code CI below supplies
  the accepted AGENTS.md Windows verification alternative.
- A focused shuffled run passes `TestAcceptanceSampleCoverage` and the smoke,
  isolation, acceptance-rules, and acceptance-scale scripts. Isolation verifies
  the initial branch and fixed author/committer/Unix dates even with a conflicting
  synthetic XDG config. The benchmark's `-short` skip is verified separately.
- `task security`: the existing pinned govulncheck v1.8.0 reports no vulnerabilities.
- `task snapshot`: GoReleaser config validation, all six target builds, archives,
  checksums, and npm layout pass for `0.2.0-next`. No tag or package is published.
- `task test:npm`: local tarballs pack and install offline; the shipped README,
  npm launcher, platform/version selection, independent Git root, and usage exit
  code pass. It then stops at `init -n` for the same Windows symlink privilege
  limit in the final `playground/eb07-tTSUqf` run after refreshing npm layout with
  the final README. CI below verifies the remaining smoke steps. The script
  isolates Git variables, npm config/cache/prefix, and all synthetic files inside
  its new playground; existing playground data is retained.
- Reviewed `gh run view 38025251583`: all seven baseline jobs passed (Linux CI
  plus integration/safeguards on Linux, macOS, and Windows). At the initial local
  verification stage, no commit, push, or release had been performed.

### Same-code CI verification

The user authorized commit and push. Implementation commit
`e211da42fd1a80cdfd7a77f8ac8bf2e908ab6675` was pushed to `feat/docs-demo`.
[CI run 38027455901](https://github.com/sanketvgh/envbuckets/actions/runs/38027455901)
passed all nine jobs and every step, including advisory scanner steps:

- Linux lint, shuffled unit tests, required race tests, six-target release
  snapshot, npm layout, and the complete packed npm smoke test.
- Full real-Git integration and documentation sample checks on Linux, macOS,
  and Windows, including the symlink-dependent walkthroughs and offline checks.
- Safeguards, schema validation, and vulnerability scans on all three platforms.
- Complete packed npm smoke tests on macOS and Windows, including all four
  dry runs, the installed checkout hook, uninstall, and reinitialization.

Local lint/schema/unit/security results and this same-code cross-platform CI
satisfy AGENTS.md's Windows verification alternative. Local Developer Mode
setup stays deferred. Phases 2, 4, and 5 are verified; phase 3 remains open only
until the schema reaches `main`. No merge, tag, or publication was performed.

## Acceptance criteria

- Every acceptance criterion in `PRODUCT.md` has an automated test or a documented platform limit.
- Every sample run in `PRODUCT.md` matches real output, checked by an integration script.
- `task check` and `task security` pass.
- The packed npm package works in the playground from its own Git repository.

## Gaps, risks, and tradeoffs

**Gaps**

- The local benchmark budget is measured above; release-runner measurements are still pending.
- The npm package smoke test on Windows depends on Developer Mode in CI, which may not be available.
- `$schema` URL stability: it points at `main` on raw GitHub, so renaming the branch breaks editor validation for existing configs.

**Risks**

- Blanket `[windows] skip` lines can hide real Windows bugs. Each skip must name a reason or a documented platform limit.
- Timing assertions are flaky across runners, so the benchmark uses a budget, not a hard assert.
- Golden updates can hide regressions; they stay behind `TESTSCRIPT_UPDATE=1` and are reviewed in the diff.
- A release without the schema on `main` leaves the `$schema` URL returning 404.

**Tradeoffs**

- Registering the real command with `testscript.Main` avoids a build step per test, but it is not the packed binary; the npm smoke test covers that.
- One script per acceptance criterion is easy to trace to PRODUCT.md, but it means many scripts and slower runs.
- Pinning `$schema` to a release tag would be safer than `main`, but needs a version bump in every generated config.

## Exit checklist

Tick every box before cutting a release.

- [x] Every PRODUCT.md acceptance criterion has an automated test or a documented platform limit.
- [x] Every PRODUCT.md sample run matches real output, checked by an integration script.
- [x] The shared test setup isolates `HOME`, git config, and git author and date.
- [x] Golden updates run only with `TESTSCRIPT_UPDATE=1`.
- [x] The 1000-branch benchmark has a set budget and runs outside `-short`.
- [x] Windows limits are documented: rename atomicity and symlink privilege.
- [ ] `schema/envbuckets.schema.json` is on `main` and the `$schema` URL decision is made and applied.
- [x] README, user docs, command help, `AGENTS.md`, and `CLAUDE.md` describe the finished CLI.
- [x] `task check` and `task security` pass via local lint/schema/unit/security plus same-code CI under the accepted Windows verification alternative.
- [x] The packed npm package works in the playground from its own Git repository.
- [x] No leftover alpha text remains in the repo outside `docs-eb/` (intentional prerelease channels and Git character classes remain).
