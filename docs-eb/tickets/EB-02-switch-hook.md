# EB-02: Switch engine and hook

**Goal:** link the right bucket after a branch switch, and on demand.

**Scope:**

- Pick the target bucket from the branch using EB-01's rule matching.
- Plan every path change before applying any.
- Link target files with relative symlinks; remove envbuckets links whose path is not in the target.
- Skip and report real files and symlinks that do not point into `.env.d/`.
- `switch` (back to the branch's bucket, repairs missing links) and `switch <bucket>` (any bucket, any time). No state is stored: going back always means the branch's bucket from the rules.
- After a switch that leaves the links on a bucket other than the branch's, print `hint: This branch uses '<bucket>'. Run "envbuckets switch" to go back.`
- `switch -n`: print the plan and stop before applying it.
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
- Integration scripts cover a normal checkout, a glob rule, same-bucket silence, missing bucket falling back to the default (including from a `prod` branch), missing default, blocked path, file only in the old bucket, file checkout, detached HEAD, broken config, missing binary, and an alpha hook block.
