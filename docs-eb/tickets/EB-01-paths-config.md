# EB-01: Paths, buckets, and config

**Goal:** the shared building blocks every command uses.

**Scope:**

- Find the repo root.
- Validate managed paths: relative, inside the repo, not under `.git/` (any case, including Windows short names like `GIT~1`) or `.env.d/`, no symlinked parent folders, and not tracked by Git.
- Scan a bucket without following symlinks. Return regular files only, skip clutter (`.DS_Store`, `Thumbs.db`, `desktop.ini`, `*~`, `*.swp`), and report other unsafe entries.
- Parse `.envbuckets.json` strictly with the standard library: `default` plus an ordered `rules` array of `{ "branch", "bucket" }` objects. Reject unknown keys (`DisallowUnknownFields`), trailing data, and invalid bucket names.
- Match rules: first match wins, `*` matches any text including `/`.
- Atomic replace helper that writes a temp file next to the target and renames it into place.
- One output helper for Git-style messages (`fatal:`, `error:`, `warning:`, `hint:`, `Would ...`), the `envbuckets: ` hook prefix, and the stdout/stderr split from the Output section of `PRODUCT.md`, so dry runs and colors each live in one place.

## Acceptance criteria

- Every accepted path resolves inside the repo without following a symlinked folder.
- Bucket scans never read file contents.
- Config errors name the file and the problem, such as an unknown key or a missing `default`.
- A leftover alpha `.envbuckets.toml` is never read; `init` and `status` say it is unused and can be deleted.
- Unit tests cover traversal, `.git` variants, symlinked parents, tracked files, clutter names, and rule order.

**Out of scope:** commands.
