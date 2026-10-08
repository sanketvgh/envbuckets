# EB-05: uninstall

**Goal:** leave the project working without envbuckets, with no data lost.

**Scope:** find envbuckets links without reading the config: take the candidate paths from the files in `.env.d/*/` and keep the ones that are links into `.env.d/`, so the repo is never walked. Rename each link's target file from the active bucket over the link (one atomic rename, so the path never disappears and nothing is copied), then remove the hook block, including the alpha's `envbuckets v1` block. Other buckets, the config, and the ignore block stay. Print `Moving <bucket path> to <path>` per file, `Removing the envbuckets hook from .git/hooks/post-checkout`, and a hint that the other buckets are still in `.env.d/`. `uninstall -n` prints the same lines as `Would ...` and stops.

## Acceptance criteria

- Each formerly linked path is now the active bucket's file, moved rather than copied. Other buckets are unchanged.
- Running `init` afterwards imports the same files again without collisions.
- Broken links get an `error:` line and are left alone; the rest completes, and the exit code is 1.
- Only the envbuckets hook block (the exact marker range) is removed. Other hook content stays byte-identical and executable, and a hook file whose remaining content is empty or only a shebang and whitespace is deleted. The file is rewritten through a temp file and a rename, so the hook is never half-written. A test compares every byte outside the block before and after.
- Works with a missing or broken config.
- Running it twice, or on a repo that was never set up, does nothing and exits 0.
- If interrupted, the project still runs and rerunning finishes the job.
- `uninstall -n` changes nothing and exits with the code a real run would return.

## Gaps, risks, and tradeoffs

**Gaps**

- If the user edited the hook inside our marker block, exact-range removal drops those edits. The ticket does not say whether to warn.
- Uninstall without a readable config relies on scanning `.env.d/*/` for links; a bucket file deleted by hand leaves an orphan link that is reported, not repaired.

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
- [ ] The hook file is rewritten through a temp file and a rename.
- [ ] Works with a missing or broken config.
- [ ] Running twice, or on a repo never set up, does nothing and exits 0.
- [ ] An interrupted run leaves the project working and rerunning finishes it.
- [ ] `uninstall -n` changes nothing and returns the real exit code.
- [ ] `task check` passes.
