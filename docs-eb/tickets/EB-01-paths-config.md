# EB-01: Paths, buckets, and config

**Goal:** the shared building blocks every command uses.

**Scope:**

- Find the repo root.
- Validate managed paths: relative, inside the repo, not under `.git/` (any case, including Windows short names like `GIT~1`) or `.env.d/`, no symlinked parent folders, and not tracked by Git.
- Scan a bucket without following symlinks. Return regular files only, skip clutter (`.DS_Store`, `Thumbs.db`, `desktop.ini`, `*~`, `*.swp`), and report other unsafe entries.
- Parse `.envbuckets.json` strictly with the standard library: `default` plus an ordered `rules` array of `{ "branch", "bucket" }` objects. Reject unknown keys (`DisallowUnknownFields`), trailing data, and invalid bucket names. An optional `$schema` string is allowed and ignored; nothing ever fetches it.
- Write `schema/envbuckets.schema.json` in JSON Schema draft-07, which editors support widely: `$schema`, a required `default`, `rules` items with `branch` and `bucket`, `additionalProperties: false` at every level, the bucket-name pattern, and a `description` on each key so editors show help on hover. Glob syntax stays a CLI check; JSON Schema cannot express Git's glob rules.
- Match rules in order, first match wins, using Git's glob rules from the Patterns table in `PRODUCT.md`: `*` and `?` stop at `/`; `**` crosses `/` only at the start, at the end, or between slashes, and acts like `*` elsewhere; a trailing `/` adds `**`; `[...]` sets with ranges, `!`/`^` negation, and POSIX classes; `\` escapes. Write a small matcher from Git's documentation (gitignore's PATTERN FORMAT and `includeIf "onbranch:"`). Do not copy Git's `wildmatch.c`: Git is GPL and this project is MIT. Go's `path.Match` lacks `**` and POSIX classes, and glob libraries add `{a,b}` braces that Git does not support.
- Reject broken patterns, such as an unclosed `[`, when loading the config.
- Rename-based helpers that never read or copy file contents:
  - Link: create a symlink next to the target and rename it into place.
  - Move into a bucket without the original path ever disappearing: hard-link the file into the bucket, then rename a new symlink over the original. Where hard links fail, rename the file into the bucket and create the link right after; a crash in between leaves the file safe in the bucket, and `switch` repairs the link.
- One output helper for Git-style messages (`fatal:`, `error:`, `warning:`, `hint:`, `Would ...`), the `envbuckets: ` hook prefix, and the stdout/stderr split from the Output section of `PRODUCT.md`, so dry runs and colors each live in one place.

## Acceptance criteria

- Every accepted path resolves inside the repo without following a symlinked folder.
- Bucket scans never read file contents.
- Config errors name the file and the problem, such as an unknown key or a missing `default`.
- A Go test checks that the schema and the config type agree on keys, required fields, and the bucket-name pattern, so they cannot drift apart.
- Every JSON config in `PRODUCT.md` passes the CLI parser and the schema. `task lint` validates them against the schema with a JSON Schema validator run through pnpm (such as `ajv-cli`), next to `oxfmt`.
- A leftover alpha `.envbuckets.toml` is never read.
- Unit tests cover traversal, `.git` variants, symlinked parents, tracked files, clutter names, and rule order.
- The alpha's path-attack cases listed in EB-00 are rewritten here and pass.
- Table tests cover every row of the Patterns table, `**` next to and away from `/`, `**/` matching zero folders, character classes, escapes, and broken patterns.

**Out of scope:** commands.
