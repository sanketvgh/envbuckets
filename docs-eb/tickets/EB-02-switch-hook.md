# EB-02: Switch engine and hook

**Goal:** link the right bucket after a branch switch, and on demand.

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

## Acceptance criteria

- Target files link at their repo paths. Links absent from the target are removed, and their bucket files remain.
- Real files and foreign symlinks are never changed. Other paths still switch; each skipped path gets an `error:` line, and the command exits 1.
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
- Integration scripts cover a normal checkout, a glob rule, same-bucket silence, missing bucket falling back to the default (including from a `prod` branch), missing default, blocked path, file only in the old bucket, file checkout, detached HEAD, broken config, missing binary, an alpha hook block, a repo with a Git LFS `post-checkout` hook (installed before and after `envbuckets init`), a legacy hook that exits 1 or exits early, a linked worktree, a shared `core.hooksPath`, and a hook file that `git lfs install --force` has overwritten.

## Gaps, risks, and tradeoffs

**Gaps**

- No other tool appends a block to an existing hook, so there is no prior art for a hook with a different shebang.
- Git 2.54 added hooks defined in config (`hook.<name>.command`), which would avoid editing hook files. This ticket does not use them; it is a possible later change if the hook file causes problems.
- Git for Windows running the hook through `sh` is assumed, not tested.

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

- [ ] The plan is a typed action list; real runs and `-n` both use it and return the same exit code.
- [ ] `switch`, `switch <bucket>`, and `switch -n` behave as written, including the one-line output and the `hint:` after a temporary switch.
- [ ] Missing bucket falls back to the default bucket with a warning; nothing from the old bucket stays linked. A missing default leaves links alone with a warning.
- [ ] Real files and foreign symlinks are never touched; other paths still switch; exit 1 with an `error:` per skipped path.
- [ ] Each link swap is atomic on Unix; rerunning after an interruption converges.
- [ ] Detached HEAD: `switch <bucket>` works, plain `switch` stops with `fatal:`.
- [ ] The hook block is marker-guarded, appended after existing content, never duplicated, and replaces the alpha `v1` block.
- [ ] The hook directory comes from `git rev-parse --git-path hooks`; a shared `core.hooksPath` outside the repo stops with `fatal:`.
- [ ] The hook exits 0 on every path: missing binary, broken config, file checkout, detached HEAD.
- [ ] Integration scripts pass for every case listed in the ticket, including Git LFS in both install orders, a legacy hook that exits early, a linked worktree, and an overwritten hook file.
- [ ] Verified on Linux, macOS, and Windows (Git for Windows runs the hook through `sh`).
- [ ] `task check` passes.
