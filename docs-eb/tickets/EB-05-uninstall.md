# EB-05: uninstall

**Goal:** leave the project working without envbuckets, with no data lost.

**Scope:** find envbuckets links without reading the config: take the candidate paths from the files in `.env.d/*/` and keep the ones that are links into `.env.d/`, so the repo is never walked. Rename each link's target file from the active bucket over the link (one atomic rename, so the path never disappears and nothing is copied), then remove the hook block, including the alpha's `envbuckets v1` block. Other buckets, the config, and the ignore block stay. Print `Moving <bucket path> to <path>` per file, `Removing the envbuckets hook from .git/hooks/post-checkout`, and a hint that the other buckets are still in `.env.d/`. `uninstall -n` prints the same lines as `Would ...` and stops.

## Acceptance criteria

- Each formerly linked path is now the active bucket's file, moved rather than copied. Other buckets are unchanged.
- Running `init` afterwards imports the same files again without collisions.
- Broken links get an `error:` line and are left alone; the rest completes, and the exit code is 1.
- Only the envbuckets hook block is removed. Other hook content stays byte-identical and executable, and a hook file that held only our block is deleted.
- Works with a missing or broken config.
- Running it twice, or on a repo that was never set up, does nothing and exits 0.
- If interrupted, the project still runs and rerunning finishes the job.
- `uninstall -n` changes nothing and exits with the code a real run would return.
