# EB-05: uninstall

**Goal:** leave the project working without envbuckets, with no data lost.

**Status:** implemented; local lint/schema/unit checks passed. Same-code
cross-platform CI is still required before this ticket can be closed.

**Scope:** find envbuckets links without reading the config: take the candidate paths from the files in `.env.d/*/` and keep the ones that are links into `.env.d/`, so the repo is never walked. Rename each link's target file from the active bucket over the link (one atomic rename, so the path never disappears and nothing is copied), then remove the hook block, including the alpha's `envbuckets v1` block. Other buckets, the config, and the ignore block stay. Print `Moving <bucket path> to <path>` per file, `Removing the envbuckets hook from .git/hooks/post-checkout`, and a hint that the other buckets are still in `.env.d/`. `uninstall -n` prints the same lines as `Would ...` and stops.

## Phase checklist

- [x] Phase 1: review filesystem/hook APIs and define the safe uninstall plan.
- [x] Phase 2: implement file restoration, contained hook removal, and CLI output.
- [x] Phase 3: add unit/integration coverage and user documentation.
- [x] Phase 4: run formatting, lint, and authorized local checks; review changes (Windows integration limitation recorded below).
- [ ] Phase 5: inspect CI and record exit-criterion evidence or deferred verification (baseline inspected; same-code CI pending).

## Acceptance criteria

- Each formerly linked path is now the active bucket's file, moved rather than copied. Other buckets are unchanged.
- Running `init` afterwards imports the same files again without collisions.
- Broken links get an `error:` line and are left alone; the rest completes, and the exit code is 1.
- Only the envbuckets hook block (the exact marker range) is removed. Other hook content stays byte-identical and executable, and a hook file whose remaining content is empty or only a shebang and whitespace is deleted. The file is rewritten through a temp file and a rename, so the hook is never half-written. A test compares every byte outside the block before and after.
- Works with a missing or broken config.
- Running it twice, or on a repo that was never set up, does nothing and exits 0.
- Files outside `.env.d/` are never moved during uninstall; foreign links and their targets remain unchanged.
- If interrupted, the project still runs and rerunning finishes the job.
- `uninstall -n` changes nothing and exits with the code a real run would return.

## Gaps, risks, and tradeoffs

**Gaps**

- If the user edited the hook inside our marker block, exact-range removal drops those edits. The ticket does not say whether to warn.
- Uninstall never walks the worktree. A deleted active target is reported only if its path still exists in another bucket. A link whose path disappeared from every bucket cannot be discovered by this scan and remains untouched.
- Reinitialization imports restored `.env` files into the config's default bucket. Other restored file types still require `add`. If a non-default bucket was active and the retained default bucket already contains those paths, `init` correctly reports collisions rather than overwriting the retained files.

**Risks**

- A bug in the marker-range removal could delete user hook content. The byte-identical test outside the block is the guard.
- Moving the active bucket's file over the link is atomic on Unix, but on Windows it may not be, so an interruption can leave a gap.
- An interrupted uninstall leaves some files moved and some links in place; rerunning must finish the job.

**Tradeoffs**

- Keeping the config, the `.gitignore` block, and the other buckets avoids data loss, but the repo is not fully back to its pre-install state.
- Deleting the hook file when only a shebang remains leaves a clean tree, but could remove a file the user created intentionally.
- Moving instead of copying keeps the "never read contents" rule, but there is no undo besides `init`.

## Exit checklist

Tick every box before starting EB-07's end-to-end work.

- [ ] Each formerly linked path ends up as the active bucket's file, moved not copied; other buckets are unchanged.
- [ ] Running `init` afterwards imports the same files again without collisions.
- [ ] Broken links get an `error:` line and are left alone; the rest completes; exit 1.
- [ ] Only the envbuckets hook block is removed; every other byte of the hook is unchanged and still executable; a hook with only a shebang or nothing left is deleted.
- [x] The hook file is rewritten through a temp file and a rename.
- [ ] Works with a missing or broken config.
- [ ] Running twice, or on a repo never set up, does nothing and exits 0.
- [ ] A foreign link whose target is outside `.env.d/` and its target remain unchanged.
- [ ] An interrupted run leaves the project working and rerunning finishes it.
- [ ] `uninstall -n` changes nothing and returns the real exit code.
- [ ] `task check` passes.

## Implementation notes

- `uninstall` accepts `-n` / `--dry-run`, uses the repository root even from a subdirectory, and never loads config or enumerates branches. Candidate paths are deduplicated and sorted from `.env.d/<valid bucket>/` entries, including unsafe entries so a linked unsafe target can be reported. Bucket symlinks are not followed.
- A working link must match the existing canonical relative link recognition used by the switcher. Each link selects its own source bucket, so temporary selections and mixed buckets can be restored independently. Real working paths and foreign links are left alone. Tracked/reserved paths, unsafe parents, nonregular targets, and missing targets are rejected before mutation.
- `fsx.RestoreBucketFile` revalidates the source, working link, and expected target, then uses one contained `os.Root.Rename`. It does not remove the working link first, open either file, or copy bytes. Direct file identity assertions verify that a successful move preserves the original bucket file; tests capture identity before mutation using `File.Stat`, which avoids Windows's deferred path-based `SameFile` lookup.
- Hook installation, inspection, and removal share Git's existing contained hooks-path checks, including linked-worktree metadata. Removal accepts only exact paired current and alpha `v1` markers, removes all complete supported blocks, and refuses nested, mismatched, or truncated supported markers without modifying the hook. Unknown-version blocks and marker-like text remain unchanged. Edits inside our complete marker ranges are removed as specified; no additional warning is emitted.
- Hook rewrites use the existing synced temporary-file-and-rename writer. Outside bytes (LF/CRLF, whitespace, and missing final newline) and original permission bits are preserved. Hook file identity tests verify replacement instead of an in-place rewrite. Only an empty or shebang/whitespace remainder is deleted, and hooks with no supported block remain untouched.
- Unit tests cover restoration/reinitialization, mixed buckets without config, broken/foreign/real paths, partial completion and resume, no-op reruns, hook-only uninstall, usage, malformed markers, and unsafe source/working/hook paths. New real-Git scripts are `uninstall`, `uninstall-empty`, and `uninstall-unreadable`; the last checks mode-000 bucket/config files and an unrelated unreadable directory that must never be walked.
- User-facing instructions are in [Leaving envbuckets](../UNINSTALL.md), linked from the README and reflected in command help. The only existing test-harness change removes its unused stdin argument and adjusts callers, as required by the final `unparam` check.
- API research used installed Go 1.27.2 documentation/source plus [Go's rename documentation](https://pkg.go.dev/os#Rename). The documented Windows atomicity limitation remains; symlink privilege setup is deferred under AGENTS.md.

## Verification evidence (2026-10-10)

- `task fix` passed with zero issues on the final Go source/tests; changes were reviewed. `task lint:go` passed with zero issues, including formatting and modernize. Commands used the existing repository-local `tmp/eb09/bin/golangci-lint.exe` v2.14.0 through a temporary PATH prefix; tool pins and global installations were unchanged. Logs: `tmp/eb05-fix-final.log` and `tmp/eb05-lint-go-final.log`.
- Final `task check` passed its Go lint, JSON formatting, config schema, and shuffled unit stages. Schema validation accepted eight PRODUCT.md examples and rejected nine invalid fixtures. All unit packages passed; symlink-dependent cases skip because this Windows session cannot create symlinks. Log: `tmp/eb05-check-final.log`.
- Integration failed only at symlink creation with Windows `ERROR_PRIVILEGE_NOT_HELD` (1314). Reviewed all ten failing scripts: `switch-safety`, `hooks-paths`, `reports`, `hooks`, `checkout`, `switch`, `uninstall`, `init-add-safety`, `init-add`, and `hooks-lfs`. The new `uninstall` fails during its initial `init`, before uninstall runs. Unix permission scripts, including `uninstall-unreadable`, skip on Windows. Synthetic test temporary directories stayed under `tmp/eb05-tests`; symlink setup remains deferred.
- The final portable `TestScript/uninstall-empty` passed independently, verifying no-op behavior before setup, alpha-hook removal at `core.hooksPath`, exact preview output and no preview writes, broken-config preservation, repeated uninstall, and rejection/preservation of a truncated hook. Log: `tmp/eb05-uninstall-empty.log`.
- Focused shuffled EB-05 unit tests passed. Portable cases verify malformed-hook preservation, non-directory bucket-root errors, usage/no-op/hook-only behavior, all supported marker ranges, byte-identical outside content with LF/CRLF and no final newline, preserved Windows permissions, hook replacement identity, and empty/shebang-only deletion. The focused log explicitly records skipped restoration, reinitialization, foreign-link, unsafe-symlink, and resume cases: `tmp/eb05-focused.log`.
- Inspected CI with `gh run list` and `gh run view`. Baseline [run 38020738680](https://github.com/sanketvgh/envbuckets/actions/runs/38020738680) on `50359f6` passed all seven jobs. It predates EB-05 and does **not** verify these changes. Existing CI already runs the new filesystem unit tests and all integration scripts on Linux, macOS, and Windows; no workflow/security gates were changed.
- `git diff --check` passed. Reviewed the task file list and final production/test changes; no files were staged, committed, or pushed. Only EB-05 implementation, tests, documentation, and the necessary test-helper lint fix are included.
- Phase 5 and the remaining exit boxes stay open until passing same-code cross-platform CI verifies the locally skipped cases. Reinitialization coverage demonstrates no collisions for restored default-bucket `.env` paths and for an explicit `add` of an arbitrary file; the retained-default collision caveat above is intentional data-loss prevention and needs to be accepted when closing the ticket.
