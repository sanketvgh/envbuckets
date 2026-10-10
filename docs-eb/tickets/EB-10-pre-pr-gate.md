# EB-10: Final documentation and pre-PR gate

**Goal:** make the finished CLI, shipped user documentation, and verification
evidence ready for a PR to `main`.

**Status:** local implementation and documentation review complete; fresh
same-code CI is pending. Local integration/npm smoke are blocked only by the
documented Windows symlink privilege limit. The passing baseline CI predates
these packaging changes and cannot close this ticket. The user authorized final
documentation review, commit, and push on 2026-10-10, and explicitly asked not
to create a PR. Merge, tag, and publication require separate instructions.

**Depends on:** review EB-00 through EB-09, especially EB-06's remaining terminal
check and EB-07's release documentation and schema gate. Existing implementation
and test evidence should be reused where it covers the final code.

**Scope:** audit and finish README, user guides, command help, packaged docs,
acceptance coverage, ticket statuses, and the final PR checklist. Fix gaps found
in that review without adding new product features. Keep post-merge publishing
requirements in EB-07.

## Phase checklist

- [x] Phase 1: audit every ticket's exit checklist and record remaining blockers.
- [x] Phase 2: finish and review README, user guides, help, and contributor docs.
- [x] Phase 3: verify packaged documentation, examples, and links.
- [ ] Phase 4: review final-code checks and CI evidence; resolve actual failures.
- [ ] Phase 5: review the PR diff and record a ready/blocked decision with evidence.

Update this checklist and the ticket index after each phase. Continue from the
first unfinished phase when implementing this ticket.

## Phase 1: Remaining work audit

- Review EB-00 through EB-09 against their exit criteria, not just index labels.
  Record each unresolved item, its owning ticket, and how it will be verified.
- Track EB-06's Windows Terminal visual check using synthetic fixtures. Keep
  classic-console evidence and live Git Bash/mintty limitations accurate; do not
  treat automated console tests as an unperformed Windows Terminal review. The
  user deferred the manual review until after release on 2026-10-10; it is no
  longer a pre-PR or release blocker.
- Review EB-07's pending release-runner benchmark measurement and other recorded
  limitations. Complete applicable pre-PR work and list post-merge follow-up.
- Preserve prior evidence and update statuses truthfully. Do not mark EB-06 or
  EB-07 fully passed while their required exit items remain open.

## Phase 2: User and contributor documentation

- Review [README](../../README.md) against [PRODUCT.md](../PRODUCT.md) and actual
  CLI help. Cover installation and prerequisites, first setup, fresh clones,
  tracked examples, bucket creation, branch rules, manual switching, `add`,
  reports, dry runs, recovery, and uninstall.
- Make shared versus private files clear: commit `.envbuckets.json` and ignore
  rules; keep `.env.d/` and managed working paths ignored. Explain editing links,
  empty files from `switch -c`, and the absence of automatic TOML migration.
- Check first-match Git glob rules, quoted patterns, missing-bucket fallback,
  detached HEAD, unsafe path refusals, hook coexistence, and checkout behavior.
- Check Windows symlink requirements and interrupted-switch recovery, offline
  operation, stdout/stderr, exit codes, and terminal/`NO_COLOR` behavior.
- Review [switch.md](../../docs/switch.md), [status.md](../../docs/status.md), and
  [uninstall.md](../../docs/uninstall.md). Fill gaps in `init`/`add` and troubleshooting
  guidance in the README or a linked guide where needed.
- Keep command help, [DEVELOPMENT.md](../DEVELOPMENT.md), `AGENTS.md`, and
  `CLAUDE.md` consistent with the final CLI and check commands. Remove stale
  alpha instructions while preserving the intentional migration warning.

## Phase 3: Shipped docs and examples

- Check relative links and anchors in the repository. Verify public README guide
  links target files included in the PR; they must resolve on `main` after merge.
- Verify the npm tarball ships the final README and license. Its documentation
  links must work for readers on npm, not depend on unpublished repository files.
- Review install/version/platform claims against package metadata and release
  artifacts. Snapshot smoke verifies candidate packages; do not describe an
  unpublished candidate as an already available registry release.
- Keep PRODUCT.md sample assertions and [ACCEPTANCE.md](../ACCEPTANCE.md) current.
  Run the existing coverage/integration checks when authorized. Update goldens
  only through `task test:integration:update`, then review the diff.
- Include `schema/envbuckets.schema.json` in the PR and verify local schema/CLI
  agreement. Preserve the published `$schema` URL and editor setup instructions.
  Remote `main` availability and matching bytes are a post-merge release gate,
  not a prerequisite that prevents opening the PR that ships the schema.

## Phase 4: Verification evidence

- Follow [DEVELOPMENT.md](../DEVELOPMENT.md) and the user's local-test preference.
  For Go changes run `task fix`, review the diff, then `task lint:go`. Run
  `task check` when local testing is authorized; record what actually passed.
- Require passing checks and `task security`, plus a snapshot and packed npm
  smoke (`task playground`, or `task snapshot` followed by `task test:npm`).
  Verify final README changes are included in the packed artifacts.
- Inspect CI with `gh`. Record commit SHA, run URL, job results, and the code
  covered. Require Linux lint/unit/race/build/snapshot checks and full integration,
  safeguards, and npm smoke on the configured platforms. Inspect advisory scanner
  results explicitly; a green workflow alone does not prove security passed.
- If Windows symlink privilege alone blocks local integration/npm smoke, record
  local lint/schema/unit/security results and passing same-code cross-platform
  CI as allowed by `AGENTS.md`. Actual failures still block. Symlink setup stays
  deferred unless requested or the environment changes.
- Keep linter/scanner pins aligned across Task, CI, and release. Preserve all
  existing security and release gates; automatic fixes stay local.

## Phase 5: Pre-PR exit checklist

Tick every box before recording this ticket as ready for the PR to `main`.

- [x] Every earlier ticket is reviewed; remaining work has an owner. Final same-code CI is tracked separately below.
- [x] EB-06's Windows Terminal visual review has an explicit user-approved post-release deferral; automated console/color checks remain required.
- [x] README, user guides, help, and contributor docs describe the final behavior.
- [x] Repository links/anchors work; public guide and schema targets are present for inclusion in the PR.
- [x] The packed npm package contains the final README/license; public documentation URLs target files included in the planned PR and must resolve after merge.
- [x] PRODUCT.md examples and acceptance coverage agree with the final CLI; sample coverage passes locally and symlink-dependent transcripts await same-code CI.
- [ ] Final-code checks, security, snapshot, and cross-platform npm smoke have passing evidence, with accepted platform limits documented.
- [ ] CI evidence identifies the tested SHA and any subsequent changes; no unverified code change is covered by an older run.
- [x] The task diff and overall PR scope are reviewed for whitespace, stale docs, accidental generated artifacts, and private paths, without reading real environment-file contents.
- [x] The ticket index is accurate; EB-07 retains its post-merge schema/publishing gate.
- [x] A PR title and description are prepared with scope, behavior, validation, and remaining post-merge work; the title follows `<gitmoji> <type>(<scope>): <summary>`.
- [x] The current decision and supporting evidence are recorded below; a ready decision still requires fresh CI.

## Evidence and handoff

### Phase 1 audit and phase 2 documentation review (2026-10-10)

| Ticket | Final audit result / owner |
| --- | --- |
| EB-00 | Passing historical pre-work status, but original checklist was never ticked. Added a reconciliation note; temporary skeleton behavior is superseded. Final-code checks belong to EB-10. |
| EB-01 to EB-05 | All exit boxes checked; shared safety cases and command handoffs have recorded evidence. |
| EB-06 | Automated checks passed. User explicitly deferred Windows Terminal visual confirmation until after release; user owns that follow-up. |
| EB-07 | Schema availability/matching bytes on remote `main` and release-runner benchmark review remain post-merge release work. Publishing gate is preserved. |
| EB-08, EB-09 | Exit checklists complete with recorded CI evidence; tool pins and required safeguards remain unchanged. |

- User guides are under `docs/`, using lowercase slugs as requested:
  `index.md`, `setup.md`, `monorepos.md`, `switch.md`, `status.md`,
  `uninstall.md`, and `troubleshooting.md`. Updated README and ticket links
  after relocation.
- Reviewed README/help against the product contract and command source. Added
  default-versus-active import guidance, hook coexistence, setup/fresh-clone
  steps, collision recovery, and troubleshooting. Existing switching/status/
  uninstall guides retain their safety and Windows notes. Contributor workflow
  stays in `docs-eb/`; public guides no longer include contributor benchmark
  instructions. `AGENTS.md` and `CLAUDE.md` already match the final commands.
- Found a packaging gap: npm packages declared MIT but omitted the license;
  binary archives omitted the README and guides. Added license copies for all
  npm packages, packed license assertions, a synthetic packaging unit test,
  and README/guide inclusion in binary archives. Local packaging verification
  passed as recorded below; fresh CI remains pending.
- Fresh `gh` inspection of [CI run 38027668071](https://github.com/sanketvgh/envbuckets/actions/runs/38027668071)
  confirmed all nine jobs passed on baseline HEAD
  `f0fdf1dcee108379568c9e62cf99781e42bcfa4f`, including race, snapshots,
  cross-platform integration/npm smoke, and advisory scanner steps. That run
  predates EB-10's packaging changes and does not verify them.

### Local verification and final review (2026-10-10)

- Added the requested README Guarantees section and ASCII file-layout/checkout
  diagrams. Checked diagram characters and box alignment. Added monorepo setup
  guidance with mirrored app paths, one repo-wide bucket selection, branch rules,
  and differing file lists. The README and uninstall guide now explain project
  restoration followed by `npm uninstall -g envbuckets` or manual binary removal.
  Reworded the switching and uninstall guides using direct descriptions.
- Final review clarified the first `add` on a fresh clone when a branch maps to
  a missing local bucket: create that bucket directory before adding a file.
  Cross-checked this against `CurrentBucket` and import/create-bucket handling.
  Corrected PRODUCT.md's Windows atomicity claim without changing transcripts.
  Archived the obsolete root alpha handoff as `docs-eb/alpha-interfaces.md` with
  an explicit superseded notice, corrected historical Ajv/Lip Gloss references,
  and removed contributor-only implementation details from public guides.
- `task fix` passed with zero issues; reviewed the packaging diff and new test,
  then `task lint:go` passed with zero issues, including modernize. Logs:
  `tmp/eb10/fix-final.log`, `tmp/eb10/lint-go.log`.
- `task check` passed Go/JS formatting, lint, all eight PRODUCT.md schema examples,
  all nine invalid schema fixtures, and every shuffled unit package, including
  `tools/npmpkg`. Integration failed in 14 scripts, each at the required Windows
  symlink-privilege error; audited every failure. This is **not** a passing full
  local check. Logs: `tmp/eb10/check.log`, `tmp/eb10/integration-failures.json`.
- The focused `TestAcceptanceSampleCoverage` check passed. PRODUCT.md transcripts
  and goldens were unchanged. README and monorepo JSON examples separately passed
  the installed Ajv validator against the shipped schema. Logs:
  `tmp/eb10/acceptance-coverage.log`, `tmp/eb10/user-configs.log`.
- `task security` passed: **No vulnerabilities found.** Log:
  `tmp/eb10/security.log`. Tool pins and security/release gates are unchanged.
- Final `task snapshot` passed GoReleaser validation, six target builds, archives,
  checksums, and npm layout for `0.2.0-next`. Inspected all six tar/zip archives:
  root README/LICENSE and all seven `docs/*.md` files match source bytes. Every
  npm package has the source LICENSE, and the umbrella README matches source.
  Logs: `tmp/eb10/snapshot-final.log`, `tmp/eb10/artifacts-final.log`.
- Final packed `task test:npm` passed tarball content, installed README and both
  licenses, launcher/version, and usage checks, then stopped at `init -n` solely
  for Windows symlink privilege. Full smoke remains a same-code CI requirement.
  Log: `tmp/eb10/npm-smoke-final.log`. Synthetic fixtures stay in new isolated
  playground repositories; existing playground data was retained.
- Validated 82 documentation links/anchors across 27 Markdown files, local
  targets of public `main` links, and all seven lowercase guide filenames.
  Markdown whitespace and `git diff --check` passed. Remote guide URLs are an
  after-merge check. Evidence: `tmp/eb10/links-final.json`.
  `gh` rechecked the schema URL on remote `main`: still HTTP 404. EB-07's matching
  schema gate remains required before publishing.
- Reviewed the task diff and the branch-to-`origin/main` file list. Only intended
  docs, packaging, test, and ticket changes are in this task; snapshot outputs,
  temporary logs/scripts, and playground fixtures remain ignored. Go modules and
  the CLI runtime are unchanged. The user authorized commit and push after the
  final review, with an explicit instruction not to create a PR. Tag and
  publication are outside that request.

### PR draft for the completed branch

**Title:** `✨ feat(cli): ship branch-based environment buckets`

**Description:**

Replace the alpha scope/TOML model with committed `.envbuckets.json` branch rules
and private `.env.d/` buckets. The CLI imports and links local files with `init`
and `add`, follows branch mappings through the checkout hook, supports temporary
and newly created buckets, reports status/mappings, and restores active files
with `uninstall`. Legacy TOML configs are not migrated; the README explains setup
and recovery.

Ship the README, license, and lowercase user guides under `docs/`, including
monorepo guidance and CLI removal. Add editor schema validation, acceptance
coverage, terminal-only color, packed npm smoke, and the existing required
lint/race/security safeguards. Binary archives contain the docs; all npm packages
contain the license.

**Validation:** local fix/lint, schema/unit checks, acceptance sample coverage,
security, and six-platform snapshot/content verification passed. Local integration
and full npm smoke require Windows symlink privilege. The prior all-platform CI
on `f0fdf1d` passed, but final same-code CI must be recorded before this draft is
ready to open. Windows Terminal visual review is deferred until after release at
the user's request.

**After merge:** verify guide URLs and matching schema on remote `main`, then
complete EB-07's release-runner benchmark review and publishing checklist.

### Decision and next step

**Pending CI; not yet marked ready for the PR.** Local docs and packaging work
are complete. The user authorized the reviewed changes to be committed and
pushed. Inspect the new CI run
for the final SHA, including advisory security outcomes and full cross-platform
integration/npm smoke. Record the run here and in EB-07, then complete phases 4
and 5. Do not reuse the baseline run for the new packaging code.

### Historical planning baseline (2026-10-10)

- EB-06 records [CI run 38024972163 on `a50e2cf`](https://github.com/sanketvgh/envbuckets/actions/runs/38024972163)
  as passed, with Windows Terminal visual verification still open.
- EB-07 records [CI run 38027455901 on `e211da4`](https://github.com/sanketvgh/envbuckets/actions/runs/38027455901)
  as passed. Its README/help/user-doc exit item is checked; phase 3 remains open
  for the schema on `main`. These are historical records, not a fresh CI audit.
- README and the switching/status/uninstall guides already exist. This ticket
  schedules their final review and packaging verification rather than assuming
  documentation is missing or already fully shipped.
- At ticket creation, no implementation checks or fresh CI inspection had been
  performed. The local verification above supersedes that initial state; final
  committed-SHA CI verification remains pending.

After merge, verify the public guide links and matching schema on remote `main`,
record the evidence in EB-07, and complete its release exit checklist before any
publication. Commit and push are authorized. Do not create a PR; merging,
tagging, and publishing require separate user instructions.
