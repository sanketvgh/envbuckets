# EB-01: Paths, buckets, and config

**Goal:** the shared building blocks every command uses.

**Status:** passed. `task check` passes locally, and GitHub Actions run [37946531174](https://github.com/sanketvgh/envbuckets/actions/runs/37946531174) passed on commit `d74be56`, including Linux, macOS, and Windows integration jobs. The shared filesystem attack cases pass here; command-specific cases from EB-00 are handed off to EB-02, EB-03, and EB-05 below.

**Scope:**

- Find the repo root.
- Validate managed paths with `filepath.IsLocal` as a first check, then do every filesystem operation inside the repo through one `os.Root` opened at the repo root (`Lstat`, `Readlink`, `Symlink`, `Link`, `Rename`, `Remove`, `MkdirAll`), so no path can leave the repo. `os.Root` still follows symlinks that stay inside the root (Go docs: "Methods on Root will follow symbolic links, but symbolic links may not reference a location outside the root"), so the rule against symlinked parent folders needs its own `Lstat` check on every parent component. Paths must be relative, inside the repo, not under `.git/` (any case, including Windows short names like `GIT~1`) or `.env.d/`, no symlinked parent folders, and not tracked by Git.
- Scan a bucket without following symlinks. Return regular files only, skip clutter (`.DS_Store`, `Thumbs.db`, `desktop.ini`, `*~`, `*.swp`), and report other unsafe entries.
- Parse `.envbuckets.json` strictly with the standard library's `encoding/json/v2` (part of the Go 1.27 API; `go.mod` already requires Go 1.27.1): `jsonv2.Unmarshal(data, &cfg, jsonv2.RejectUnknownMembers(true))`. The config is `default` plus an ordered `rules` array of `{ "branch", "bucket" }` objects. Unlike `encoding/json` v1, v2 by default rejects duplicate keys, invalid UTF-8, trailing data after the top-level value, and keys with the wrong letter case; v1 accepted `{"Default":"dev"}` and `{"default":"dev","default":"prod"}` silently, which the schema would flag but the CLI would not. v2 has no `required` option, so a missing or empty `default` and invalid bucket names are checked after decoding. Turn v2 errors into `.envbuckets.json: <problem>` (commands add their own prefix, such as `fatal: invalid ` in `PRODUCT.md`'s sample runs), with names in single quotes as the Output rules require (`unknown key 'bukcet'`): unwrap `*json.SemanticError` for `JSONPointer` (for example `/rules/0`) and `errors.Is(err, json.ErrUnknownName)` for an unknown key; v2's own text ("cannot unmarshal JSON string into Go main.Config: unknown object member name") is not fit to show users. An optional `$schema` string is allowed and ignored; nothing ever fetches it.
- Write `schema/envbuckets.schema.json` in JSON Schema draft-07, which editors support widely: `$schema`, a required `default`, `rules` items with `branch` and `bucket`, `additionalProperties: false` at every level, the bucket-name pattern, and a `description` on each key so editors show help on hover. Glob syntax stays a CLI check; JSON Schema cannot express Git's glob rules. Keep the schema hand-written: with three object types, generating it (as lefthook does with `invopop/jsonschema`) adds draft 2020-12 output and non-deterministic comments that need post-processing.
- Define the bucket-name regex once as a Go constant, in a form that means the same in Go's RE2 and in ECMA regex (`^[A-Za-z0-9][A-Za-z0-9_-]*$`: it must start with a letter or digit, so a bucket name can never look like a flag such as `-x`): no lookarounds, no `\p{}`. The parser, the schema, and the drift test all use it.
- Match rules in order, first match wins, using Git's glob rules from the Patterns table in `PRODUCT.md`: `*` and `?` stop at `/`; `**` crosses `/` only at the start, at the end, or between slashes, and acts like `*` elsewhere; a trailing `/` adds `**`; `[...]` sets with ranges, `!`/`^` negation, and POSIX classes; `\` escapes. Use `github.com/git-lfs/wildmatch/v2` (MIT, already adopted in `internal/pattern`) behind the existing `pattern` package, so nothing else imports it. Do not copy Git's `wildmatch.c`: Git is GPL and this project is MIT. Go's `path.Match` lacks `**` and POSIX classes, and `doublestar` adds `{a,b}` braces that Git does not support. Back the Patterns table with differential tests that run the same (pattern, branch) cases through real Git (`includeIf "onbranch:"`) where it applies, so the library's behavior is checked, not assumed. A 40-case run against real Git 2.54 (`includeIf "onbranch:"`) found one divergence: `[]a]` (and `[!]a]`) matches neither `]` nor `a` in the library, where Git matches both. Fix it before the library sees the pattern by rewriting a `]` that directly follows `[`, `[!`, or `[^` to `\]`; the rewritten `[\]a]` was checked to match `]` and `a` and not `b`. Commit the cases and Git's answers as a fixture file, with a task that regenerates the answers from real Git, so the matcher is tested against recorded Git output on machines without the right Git version.
- Reject broken patterns, such as an unclosed `[`, when loading the config.
- Rename-based helpers that never read or copy file contents:
  - Link: create a symlink next to the target and rename it into place.
  - Move into a bucket without the original path ever disappearing: hard-link the file into the bucket, then rename a new symlink over the original. Where hard links fail, rename the file into the bucket and create the link right after; a crash in between leaves the file safe in the bucket, and `switch` repairs the link.
- One output helper for Git-style messages (`fatal:`, `error:`, `warning:`, `hint:`, `Would ...`), the `envbuckets: ` hook prefix, and the stdout/stderr split from the Output section of `PRODUCT.md`, so dry runs and colors each live in one place.

## Acceptance criteria

- Every accepted path resolves inside the repo without following a symlinked folder.
- Bucket scans never read file contents.
- Config errors name the file and the problem, such as an unknown key or a missing `default`. Tests check the error kind and the key name, not the standard library's wording.
- Tests cover what v1 would have accepted and v2 rejects: a duplicate key, a wrong-case key (`Default`), invalid UTF-8, trailing data, a trailing comma, and a wrong value type.
- A Go test reflects over the config type's `json` tags and checks that the schema agrees on keys, required fields, `additionalProperties: false` at every level, and the bucket-name pattern, so they cannot drift apart. It compares structure, not generated bytes.
- A Go test validates every JSON config in `PRODUCT.md` and a set of invalid fixtures against the schema with a draft-07 validator library (such as `santhosh-tekuri/jsonschema`), so the Go side does not depend on pnpm.
- Every JSON config in `PRODUCT.md` passes the CLI parser and the schema. `task lint` also validates them with Ajv through `tools/validate-config-schema.mjs`, next to `oxfmt`; the two schema validators cross-check each other.
- A leftover alpha `.envbuckets.toml` is never read.
- Unit tests cover traversal, `.git` variants, symlinked parents, tracked files, clutter names, and rule order.
- Every EB-00 path-attack case is assigned to this ticket's shared-path tests or the ticket implementing the relevant command; the EB-01 filesystem cases pass.
- Table tests cover every row of the Patterns table, `**` next to and away from `/`, `**/` matching zero folders, character classes, escapes, and broken patterns.

**Out of scope:** commands.

### Carried-over security cases

- EB-01 tests the symlinked bucket directory and symlinked bucket-file scan cases through `fsx`.
- EB-02 tests that non-canonical bucket links are foreign and that external or symlinked hook paths are refused.
- EB-03 tests that a symlinked `.gitignore` is never read or modified and that `init` does not move a real `.env` through a symlinked bucket directory.
- EB-05 tests that `uninstall` leaves files outside the buckets untouched.

## EB-02 to EB-06 handoff API

- `gitx.Root(cwd)` discovers the worktree root; open it once with `fsx.OpenRepo(root)` and close the returned `*fsx.Repo` when the command finishes.
- `(*fsx.Repo).ValidatePath(path)` rejects non-local, reserved, symlink-parent, and Git-tracked managed paths. `(*fsx.Repo).ScanBucket(bucket)` returns regular-file paths and unsafe entry paths separately without reading file contents.
- `config.Load(repo.Root)` reads `.envbuckets.json` through the root. `config.Parse(data)` is available to tests and non-filesystem callers. `Config.BucketFor(branch)` returns the first matching bucket and its rule pattern, or the default bucket and an empty pattern.
- `fsx.LinkFile(root, link, target)` installs a relative symlink without replacing a real file. `fsx.MoveFileToBucket(root, source, destination, linkTarget)` returns whether the hard-link route succeeded; on fallback errors the original remains safely in the bucket for link repair.
- `output.Writer` centralizes Git-style prefixes, the hook prefix, and stream routing. `Fatal`, `Error`, `Warning`, and `Hint` write diagnostics to stderr; `Info` writes command progress to stderr; `Would` and `List` write dry-run actions and listings to stdout.

## Gaps, risks, and tradeoffs

**Gaps**

- 40 patterns from the Patterns table and the rule examples were run against real Git 2.54: all agree with `internal/pattern` except `[]a]`, which is fixed as above. Real Git confirms `release/` does not match `release`, `release/**` does not match `release`, `**/hotfix` matches `hotfix`, and mid-pattern `**` (`a**b`) acts like `*`. Cases not in the fixture may still differ, so new rule syntax needs a fixture row.
- Whether an editor accepts the `$schema` raw GitHub URL after `main` is renamed is untested.

**Risks**

- The library panics on malformed patterns. `compile` recovers and `validateSyntax` pre-checks, so a new syntax case that slips past `validateSyntax` must still not crash the hook.
- `os.Root` limits are not fully known on Windows (reserved names, case-insensitive paths). Go's own tests cover some of it; ours must cover the rest.
- Windows rename onto an existing link may not be atomic, so the move helpers can leave a short gap there.
- The text of v2 errors "may change over time" (Go docs for `SemanticError`), so our messages are built from `JSONPointer` and `ErrUnknownName`, never from `err.Error()`.

**Tradeoffs**

- A hand-written schema plus a structural test is cheaper than generating it and keeps draft-07 exact, but a key added to the Go type without a schema change is caught only by the test, not by the compiler.
- Using the wildmatch library saves writing a matcher but ties behavior to a dependency; writing our own would match Git exactly only if we also test it against Git.
- Two validators (a Go library and Ajv) catch more mistakes but add a dependency and a pnpm step.
- `encoding/json/v2` is stricter by default than v1, which suits a hand-edited config. The cost is newer error types and error text that we map ourselves; the package is in the Go 1.27 API file and enabled by default in 1.27.1 without `GOEXPERIMENT`.

## Exit checklist

Tick every box before starting EB-02. Later tickets build on these without reopening this one.

**Paths**

- [x] All repo file operations go through one `os.Root` opened at the repo root.
- [x] Symlinked parent folders are refused by an explicit `Lstat` check on each parent component.
- [x] `.git/` (any case, `GIT~1`), `.env.d/`, absolute, `..`, and tracked paths are refused.
- [x] Path tests for EB-01's shared filesystem cases pass; command-level EB-00 attacks are assigned to the matching command tickets.

**Bucket scan**

- [x] Regular files only, clutter skipped, unsafe entries reported, no symlink followed, no content read.

**Config**

- [x] `encoding/json/v2` with `RejectUnknownMembers(true)`; missing `default` and bad bucket names checked after decoding.
- [x] Tests cover unknown key, nested unknown key, duplicate key, wrong-case key, invalid UTF-8, trailing data, trailing comma, wrong type.
- [x] Error messages are built from `JSONPointer` and `ErrUnknownName` as `.envbuckets.json: <problem>`.
- [x] A leftover `.envbuckets.toml` is never read.

**Schema**

- [x] Draft-07 schema with `additionalProperties: false` everywhere and a description on every key.
- [x] One bucket-name regex constant, valid in RE2 and ECMA, requiring a first character that is a letter or digit (tests: `-x`, `_x`, `x-`, `x_y` and an empty name).
- [x] The structural drift test passes.
- [x] PRODUCT.md configs pass the parser, the Go validator, and Ajv; invalid fixtures fail all three.

**Matcher**

- [x] First match wins; broken patterns are config errors.
- [x] Leading-`]` fix in; `[]a]` matches `]` and `a`.
- [x] The Git-recorded fixture (40+ cases) is committed with a regeneration task.
- [x] Every row of the Patterns table has a test.

**Helpers**

- [x] Link and move helpers work, never read or copy contents, and pass the `chmod 000` test on Linux and macOS.
- [x] A simulated crash between steps leaves the file safe and rerunning repairs the link.
- [x] The output helper covers every prefix and the stdout/stderr split.

**Hand-off**

- [x] Names used by EB-02 to EB-06 are fixed and documented in the ticket.
- [x] `task check` passes.
