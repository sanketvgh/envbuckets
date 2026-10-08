# EB-07: Release and acceptance

**Goal:** ship the new product as one coherent CLI.

**Scope:** one shared test setup for all integration scripts (`HOME`, `XDG_CONFIG_HOME`, `GIT_CONFIG_GLOBAL`, `GIT_CONFIG_NOSYSTEM=1`, fixed git author, committer, and dates) and the CLI registered with `testscript.Main` so scripts and the hook call the real command; golden updates only behind `TESTSCRIPT_UPDATE=1` in a task target; a `testing.B` benchmark for 1000 branches in its own file, skipped under `-short`, creating branches with one `git update-ref --stdin` call (the setup in Git's own `t/perf/p6300-for-each-ref.sh`) and judged against a budget in the ticket; a documented Windows note: Go's rename on Windows is one `MoveFileEx(MOVEFILE_REPLACE_EXISTING)` call with no delete step in our code, but Microsoft does not document it as atomic, so `switch` converges on rerun after an interruption; and a note that symlink tests need Windows Developer Mode, which was off on the research machine, so run them in CI or enable it locally; README, user docs, and command help for the new product, replacing EB-00's placeholder README before any release (the npm package ships it); finish `AGENTS.md` and `CLAUDE.md` for the finished CLI; make sure `schema/envbuckets.schema.json` is on `main` so the `$schema` URL resolves, and show editor setup in the README; end-to-end scripts for every acceptance criterion in `PRODUCT.md`; npm package smoke test.

## Acceptance criteria

- Every acceptance criterion in `PRODUCT.md` has an automated test or a documented platform limit.
- Every sample run in `PRODUCT.md` matches real output, checked by an integration script.
- `task check` and `task security` pass.
- The packed npm package works in the playground from its own Git repository.

## Gaps, risks, and tradeoffs

**Gaps**

- No measured budget exists yet for the 1000-branch benchmark; the first run sets it. Git's own perf tests show the setup but not our numbers.
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

- [ ] Every PRODUCT.md acceptance criterion has an automated test or a documented platform limit.
- [ ] Every PRODUCT.md sample run matches real output, checked by an integration script.
- [ ] The shared test setup isolates `HOME`, git config, and git author and date.
- [ ] Golden updates run only with `TESTSCRIPT_UPDATE=1`.
- [ ] The 1000-branch benchmark has a set budget and runs outside `-short`.
- [ ] Windows limits are documented: rename atomicity and symlink privilege.
- [ ] `schema/envbuckets.schema.json` is on `main` and the `$schema` URL decision is made and applied.
- [ ] README, user docs, command help, `AGENTS.md`, and `CLAUDE.md` describe the finished CLI.
- [ ] `task check` and `task security` pass.
- [ ] The packed npm package works in the playground from its own Git repository.
- [ ] No leftover alpha text remains in the repo outside `docs-eb/`.
