# EB-02: Switch engine and hook

**Goal:** link the right bucket after a branch switch, and on demand.

**Scope:**

- Pick the target bucket from the branch using EB-01's rule matching.
- Plan every path change before applying any.
- Link target files with relative symlinks; remove envbuckets links whose path is not in the target.
- Skip and report real files and symlinks that do not point into `.env.d/`.
- `switch` (branch's bucket, repairs missing links) and `switch <bucket>`.
- `switch -n`: print the plan and stop before applying it.
- Install the marker-guarded hook block, appended after any existing hook content.

## Acceptance criteria

- Target files link at their repo paths. Links absent from the target are removed, and their bucket files remain.
- Real files and foreign symlinks are never changed. Other paths still switch; each skipped path gets an `error:` line, and the command exits 1.
- Success prints one line (`Switched to bucket 'prod'` or `Already on bucket 'dev'`); links removed because the target lacks the file are not listed.
- Each link swap is atomic. Rerunning `switch` after an interruption converges.
- A missing bucket changes nothing and prints a warning.
- `switch -n` prints every planned link and removal, changes nothing, and exits with the code a real run would return.
- The hook ignores file checkouts and detached HEAD, never prompts, exits 0 on every error, and does nothing when the binary is not on `PATH`.
- Integration scripts cover a normal checkout, missing bucket, blocked path, file only in the old bucket, file checkout, broken config, and missing binary.
