# EB-02: Switch engine and hook

**Goal:** link the right bucket after a branch switch, and on demand.

**Status:** Passed. All five phases and exit criteria are complete using the verification alternative accepted by the user on 2026-10-09. CI passed on `f1c2133` ([Actions run `37951292646`](https://github.com/sanketvgh/envbuckets/actions/runs/37951292646)). Local `task check` passed formatting, lint, schema validation, and unit tests; its integration stage failed solely because Windows lacks symlink privilege. Passing Linux/macOS/Windows CI integration results cover that stage. The user deferred a full local rerun and environment setup. Phase 1 was confirmed complete by the user.

Implementation: `internal/switcher/`, `internal/cli/cmd_switch.go`, and `internal/block/hook.go`. User documentation: [Switching buckets](../SWITCH.md). Real-Git cases are in `testdata/script/{switch,checkout,hooks,hooks-paths,hooks-lfs,switch-safety,switch-unreadable}.txtar`; hook installation is exercised directly because `init` belongs to EB-03.

References checked during implementation: installed Go 1.27.1 `go doc os.Root`, `os.Root.Rename`, `os.Root.MkdirAll`, and `os.Rename`; the current official [`os` documentation](https://pkg.go.dev/os) (Go 1.27.2 when fetched); and Grep MCP's current [`git-lfs/git-lfs` hook installer](https://github.com/git-lfs/git-lfs/blob/main/lfs/hook.go).

**Scope:**

- Pick the target bucket from the branch using EB-01's rule matching.
- Plan every path change before applying any. The plan is a list of typed actions (link, remove, skip with reason, preflight error). Apply and `-n` both consume that list, so dry-run output and exit codes come from the same plan.
- Link target files with relative symlinks; remove envbuckets links whose path is not in the target.
- Skip and report real files and symlinks that do not point into `.env.d/`.
- `switch` (back to the branch's bucket, repairs missing links) and `switch <bucket>` (any bucket, any time). No state is stored: going back always means the branch's bucket from the rules.
- After a switch that leaves the links on a bucket other than the branch's, print `hint: This branch uses '<bucket>'. Run "envbuckets switch" to go back.`
- `switch -n`: print the plan and stop before applying it.
- Find the hooks directory with `git rev-parse --git-path hooks`, never a hardcoded `.git/hooks`. It honors `core.hooksPath` and resolves to the shared directory inside a linked worktree. If that directory is outside the repo (a shared `core.hooksPath`), stop with `fatal:` and a hint, because the hook would affect every repo that uses it.
- The hook script uses `#!/bin/sh` and mode 0755. It runs `command -v envbuckets >/dev/null 2>&1 || exit 0`, calls envbuckets with `|| true`, and ends with `exit 0`. No path may return a non-zero status; tools such as pre-commit (exit 1) and Git LFS (exit 2) fail when their binary is missing, and this hook must not.
- Install the marker-guarded hook block, appended after any existing hook content. Replace an existing envbuckets block, including the alpha's `# >>> envbuckets v1 >>>` block, instead of adding a second one.

## Implementation phases

Complete and review each phase checklist before starting the next. The final exit checklist still applies to the whole ticket.

### Phase 1: Inspect EB-01 interfaces and define the switch plan

- [x] Review the existing path, config, rule-matching, and CLI interfaces from EB-01.
- [x] Define the typed plan actions for link, remove, skip with reason, and preflight error.
- [x] Confirm the planner validates every path before any filesystem changes and can drive both real runs and `-n`.

**Checkpoint:** the plan's inputs, actions, safety checks, and exit-code behavior are clear before applying changes.

### Phase 2: Implement safe bucket switching

- [x] Resolve the branch bucket and explicit bucket requests, including detached HEAD behavior.
- [x] Plan target links, obsolete envbuckets links, and blocked paths; never alter real files or foreign symlinks.
- [x] Apply relative symlink changes atomically where supported and ensure reruns converge.
- [x] Implement missing-bucket fallback, missing-default behavior, dry-run output, success output, skip errors, and temporary-switch hint.

**Checkpoint:** switch behavior and dry-run behavior are driven by the same plan and match the acceptance criteria.

### Phase 3: Implement checkout hook management

- [x] Resolve the hook directory with `git rev-parse --git-path hooks` and reject a shared hooks directory outside the repository.
- [x] Install or replace the marker-guarded hook block, preserving existing content and replacing the alpha v1 block.
- [x] Ensure the hook ignores file checkouts and detached HEAD, handles missing binary/config/errors, and always exits 0.

**Checkpoint:** hook installation and execution satisfy the ticket's path, compatibility, and nonblocking requirements.

### Phase 4: Add integration coverage and documentation

- [x] Add integration scripts for switch, fallback, blocked paths, detached HEAD, broken config, and checkout behavior.
- [x] Cover hook compatibility: Git LFS install orders, legacy early exit/failure, alpha block, linked worktree, shared `core.hooksPath`, and overwritten hook.
- [x] Update user-facing docs for switch and hook behavior where needed.

**Checkpoint:** every scenario listed in the ticket has a corresponding integration case and expected result.

### Phase 5: Verify and close EB-02

- [x] Run the integration scripts and inspect their results.
- [x] Verify on Linux, macOS, and Windows as available, including Git for Windows invoking the hook through `sh`.
- [x] Complete `task check` coverage through passing local formatting/lint/schema/unit checks and cross-platform CI integration; the user accepted this alternative to a full local rerun on Windows without symlink privilege.

**Checkpoint:** all applicable exit checklist items are checked before starting EB-03, EB-04, or EB-05.

## Verification evidence

Reviewed with `gh run view` and `gh run watch`. Run `37951292646` passed the Linux lint/unit/build/package job and all three integration jobs on `f1c2133`.

| Exit criteria | Evidence |
| --- | --- |
| Typed plan, preflight, dry-run parity | `TestPlanIsReadOnlyAndPreflightBlocksWrites`, `TestSwitchBlockedPathsAndDryRunParity`, `switch.txtar` |
| Switch output, temporary override, fallback, detached HEAD | CLI unit tests and `switch.txtar` / `checkout.txtar` |
| Real files, tracked paths, foreign/noncanonical links, unsafe symlinks | `switch-safety.txtar`, planner/CLI tests, and code review of destination revalidation |
| Atomic Unix swaps and recovery | `fsx.LinkFile` stages beside the destination and uses `Root.Rename`; filesystem swap tests and `TestRerunConvergesAfterPartialApply` |
| Hook markers, exit behavior, Git LFS, legacy hooks | Hook unit tests, `hooks.txtar`, `hooks-lfs.txtar`, and `checkout.txtar` |
| Actual hooks directory, linked worktree, shared/symlinked path refusal | `hooks-paths.txtar` and `InstallRepoHook` review; installation is exercised directly until EB-03 adds `init` |
| Linux, macOS, Windows | All integration matrix jobs passed; the Unix unreadable-file case passed on Linux/macOS and intentionally skips Windows |
| `task check` coverage (user-accepted alternative) | Local formatting, lint, schema validation (8 valid examples and 9 invalid fixtures), and unit tests passed on 2026-10-09. The local integration stage failed with "A required privilege is not held by the client"; all integration jobs passed in CI on the same implementation. Full local rerun and environment setup deferred by the user |

The noncanonical-link case found during the exit review was fixed in `f1c2133` and verified by the same CI run. A legacy hook's earlier `exit 1` still fails Git before our appended block runs; that documented limitation is explicitly covered, while every envbuckets-controlled exit path succeeds.

## Acceptance criteria

- Target files link at their repo paths. Links absent from the target are removed, and their bucket files remain.
- Real files and foreign symlinks are never changed. Only canonical relative targets into `.env.d/<bucket>/` are managed; absolute or non-canonical targets are foreign. Other paths still switch; each skipped path gets an `error:` line, and the command exits 1.
- Success prints one line (`Switched to bucket 'prod'` or `Already on bucket 'dev'`); links removed because the target lacks the file are not listed.
- Each link swap is atomic. Rerunning `switch` after an interruption converges.
- When the branch's bucket is missing, the hook and plain `switch` link the default bucket instead, printing `warning: bucket '<b>' does not exist; using '<default>' (default)` and a hint to create it with `switch -c`. Coming from `prod`, nothing from `prod` stays linked. If the default is missing too, the links stay and the warning says so.
- `switch <bucket>` naming a missing bucket stops with `fatal: no bucket named '<b>'` and a hint, and changes nothing.
- The hook prints `envbuckets: Switched to bucket '<b>' (<reason>)` only when the bucket changes, and nothing when it stays the same.
- After `switch prod` on a branch that uses `dev`, both `switch` and the next branch checkout link `dev` again.
- On a detached HEAD, `switch <bucket>` works and plain `switch` stops with `fatal:` and a hint to name a bucket.
- With a broken config, the hook changes nothing and prints one `envbuckets: warning:` line naming the problem.
- `switch -n` prints every planned link and removal, changes nothing, and exits with the code a real run would return.
- The hook ignores file checkouts and detached HEAD, never prompts, exits 0 on every error, and does nothing when the binary is not on `PATH`.
- A shared, external, or symlinked hook path is refused without writing through the link.
- Integration scripts cover a normal checkout, a glob rule, same-bucket silence, missing bucket falling back to the default (including from a `prod` branch), missing default, blocked path, file only in the old bucket, file checkout, detached HEAD, broken config, missing binary, an alpha hook block, a repo with a Git LFS `post-checkout` hook (installed before and after `envbuckets init`), a legacy hook that exits 1 or exits early, a linked worktree, a shared `core.hooksPath`, a symlinked hook path, non-canonical foreign links, and a hook file that `git lfs install --force` has overwritten.

## Gaps, risks, and tradeoffs

**Gaps**

- No other tool appends a block to an existing hook, so there is no prior art for a hook with a different shebang.
- Git 2.54 added hooks defined in config (`hook.<name>.command`), which would avoid editing hook files. This ticket does not use them; it is a possible later change if the hook file causes problems.
- A full local `task check` rerun awaits a symlink-capable environment and is deferred by the user. Its checks are covered by the accepted local/CI alternative above. Git for Windows invoking the hook through `sh` is covered by the passing Windows integration job.

**Risks**

- `git lfs install --force` overwrites the hook and drops our block, and an earlier `exit` in a user's hook skips it. EB-04 `status` warns when the hook is missing; the block is placed so an earlier `exit` is the only way it is skipped, and it is tested.
- A shared `core.hooksPath` outside the repo would make the hook run for every repo, so the ticket stops with `fatal:`.
- A bug in the "never block Git" paths would block a checkout. Every exit path needs a test.
- Rename onto an existing link is not atomic on Windows.

**Tradeoffs**

- Appending a block keeps the user's hook content in place, unlike lefthook and pre-commit, which move it aside. The cost is more edge cases and a harder uninstall.
- Always exiting 0 hides failures from Git; the user must read the `envbuckets:` lines to notice problems.
- Refusing a shared `core.hooksPath` is safer than installing there, but those users cannot use the tool until they change their setup.

## Exit checklist

Tick every box before starting EB-03, EB-04, or EB-05.

- [x] The plan is a typed action list; real runs and `-n` both use it and return the same exit code.
- [x] `switch`, `switch <bucket>`, and `switch -n` behave as written, including the one-line output and the `hint:` after a temporary switch.
- [x] Missing bucket falls back to the default bucket with a warning; nothing from the old bucket stays linked. A missing default leaves links alone with a warning.
- [x] Real files and foreign symlinks are never touched; other paths still switch; exit 1 with an `error:` per skipped path.
- [x] Each link swap is atomic on Unix; rerunning after an interruption converges.
- [x] Detached HEAD: `switch <bucket>` works, plain `switch` stops with `fatal:`.
- [x] The hook block is marker-guarded, appended after existing content, never duplicated, and replaces the alpha `v1` block.
- [x] The hook directory comes from `git rev-parse --git-path hooks`; a shared `core.hooksPath` outside the repo stops with `fatal:`.
- [x] The hook exits 0 on every path: missing binary, broken config, file checkout, detached HEAD.
- [x] Integration scripts pass for every case listed in the ticket, including Git LFS in both install orders, a legacy hook that exits early, a linked worktree, and an overwritten hook file.
- [x] Verified on Linux, macOS, and Windows (Git for Windows runs the hook through `sh`).
- [x] `task check` coverage accepted through passed local formatting/lint/schema/unit checks plus passing cross-platform CI integration. A full local rerun is deferred by the user because this Windows session lacks symlink privilege.
