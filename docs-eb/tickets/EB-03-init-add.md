# EB-03: init, add, and switch -c

**Goal:** get files into buckets.

**Status:** passed; local checks and same-code cross-platform CI accepted under the Windows verification rule.

**Scope:**

- `init`: create the config if missing, with `$schema` as the first key and `default` set to `dev`, create the default bucket folder, write the `.gitignore` block, install the hook, and import untracked `.env` and `.env.*` files. Find candidates through Git so fully ignored folders like `node_modules/` are never walked. Replace the alpha's `.gitignore` block (`# >>> envbuckets v1 >>>`) instead of adding a second one. Print `Adding <path> to bucket '<b>'` per file, then `Initialized envbuckets in <repo>/.env.d/`; with no files found, add `hint: No local files found. Create them, then run "envbuckets add <file>".`
- `add <file>...`: check every path first and stop with `fatal:` if any cannot be added. Then move each into the current bucket with EB-01's move helper, leaving a link, and add the path to the ignore block. Silent on success.
- `switch -c <bucket>`: validate the name, refuse an existing bucket (`fatal: a bucket named '<b>' already exists`), create an empty file at each of the current bucket's paths, then switch and print `Switched to a new bucket '<b>'`.
- Current bucket: the bucket the links point to; with no links, the branch's bucket; with links to several buckets, stop and suggest `switch`.
- `-n` for `init`, `add`, and `switch -c`: print the plan and stop. A dry-run `init` still checks symlink support, using a temp link it removes right away. On Windows the probe tells `ERROR_PRIVILEGE_NOT_HELD` (1314) from other failures by unwrapping `*os.LinkError` with `errors.As`, and names Developer Mode in the message. It is a preflight error in the plan, so `init -n` reports it and returns the same exit code as a real run.

## Acceptance criteria

- Existing bucket files and real working files are never overwritten. `init` skips a collision with a `warning:`; `add` refuses it with `fatal:` before changing anything.
- A failure during `add` leaves the original file in place.
- `init` and `add` work on files the user cannot read (`chmod 000` on Linux and macOS), which proves they move rather than copy.
- `switch -c` creates zero-byte files and never touches the current bucket's files.
- Tracked files such as `.env.example` are never imported or added.
- `add` refuses directories and paths that are already links, including ones it already manages, with `fatal:` before changing anything.
- If an alpha `.envbuckets.toml` exists, `init` says it is unused and can be deleted.
- `init` on an alpha repo leaves exactly one envbuckets `.gitignore` block and one hook block.
- Rerunning `init` on a set-up repo changes nothing. On a fresh clone it installs the hook and creates only what is missing.
- `init` stops with a clear message when symlinks cannot be created, such as on Windows without Developer Mode. On the development machine used for this research (Windows, Developer Mode off) Go's `os.Symlink` fails with "A required privilege is not held by the client", which is the error to detect.
- The ignore block is marker-guarded, never duplicated, and skips paths Git already ignores.
- A symlinked `.gitignore` is refused without reading or modifying its target.
- `init` never moves a real `.env` into or through a symlinked bucket directory; the source stays in place and the unsafe bucket is reported.
- With `-n`, `init`, `add`, and `switch -c` leave the repo byte-identical and exit with the code a real run would return.

## Gaps, risks, and tradeoffs

**Gaps**

- The Developer Mode probe is the only reliable check, so a machine that allows links in one folder but not another may pass the probe and fail later.
- Git enumerates the untracked tree, skips submodule gitlinks, and prunes fully ignored folders. Very large nonignored trees still incur Git enumeration cost; envbuckets performs no additional directory walk for candidate discovery.

**Risks**

- Moving files rather than copying them means a crash between the hard link and the symlink rename relies on `switch` to repair the link.
- `add` on a file another process has open can fail on Windows.
- A bad `.gitignore` edit could hide or expose files; the marker block must be exact and idempotent.

**Tradeoffs**

- Move-and-link never reads contents (matches the safety rules), but there is no copy mode, so a user who wants a backup must make one first.
- `init` skipping an unimportable file with a `warning:` finishes more work than stopping, but a quiet skip can be missed in a long run.
- Probing for symlink support first adds one temporary file, which keeps `-n` honest.

## Phase checklist

- [x] Phase 1: review existing helpers; research Git discovery and Go filesystem APIs with MCP and `go doc`.
- [x] Phase 2: implement safe import planning, `init`, `add`, and `switch -c`, including dry runs.
- [x] Phase 3: add unit and real-Git integration coverage for the acceptance criteria.
- [x] Phase 4: run formatting/fixes, review the diff, run lint/checks, inspect CI, and record evidence and limitations.
- [x] Phase 5: commit/push authorized by the user; verify this code in cross-platform CI and close the exit checklist.

## Implementation and research notes

- Queried grep MCP for `os.OpenRoot` usage in [Go's own filesystem tests](https://github.com/golang/go/blob/master/src/os/root_test.go). Consulted local Go 1.27.1 documentation with `go doc os.Root`, `os.Root.Rename`, `os.Root.Link`, `os.Root.OpenFile`, `errors.AsType`, `syscall.ERROR_PRIVILEGE_NOT_HELD`, and `github.com/rogpeppe/go-internal/testscript`.
- Candidate discovery combines NUL-delimited `git ls-files --others --exclude-standard` with `--ignored --directory`. Synthetic Git tests verify that individually ignored files are found, wholly ignored folders are pruned, whitespace in filenames is preserved, and submodule gitlinks are not traversed. Git handles tree enumeration; envbuckets does not walk ignored directories to find imports.
- All import paths and metadata are checked before changes. Ignore blocks preserve user rules and legacy entries, collapse duplicate/alpha blocks, escape literal filename patterns, and skip paths already ignored by Git. Existing managed links can restore missing ignore entries when rerunning `init`.
- The move helper stages its symlink before attempting a hard link or rename. It removes the new hard link or restores the renamed source if installation fails. Injected failure tests cover both paths without reading real files.
- Current-bucket detection uses canonical managed links, rejects mixed links, and otherwise uses the branch mapping. `add` refuses a missing current bucket with a creation hint. A detached HEAD requires existing managed links to identify the current bucket. `switch -c` uses the existing switch planner against a virtual list of empty files, so preflight and dry runs need no bucket writes.

## Verification evidence (2026-10-09)

- `task fix`: passed, zero issues after resolving a root-scoped snapshot-test lint finding; automatic changes reviewed.
- `task lint:go`: passed, including formatting and modernize. The final `task check` also passed this stage.
- JSON formatting and schema validation: passed; eight PRODUCT.md examples accepted and nine invalid fixtures rejected.
- `go test ./...`: passed. Tests requiring symlinks skip on this Windows session, and Unix mode-000 tests skip on Windows. Portable Git discovery, ignore-block, preflight/usage, and privilege-error tests ran locally.
- `task check`: lint/schema/unit stages passed; integration failed only because Windows returned `ERROR_PRIVILEGE_NOT_HELD` (1314). All eight failing scripts were audited: `checkout`, `hooks`, `hooks-lfs`, `hooks-paths`, `switch`, `switch-safety`, `init-add`, and `init-add-safety`. Local log: `tmp/eb03-check.log`. Test temporary directories for this run were kept under repository `tmp/eb03-tests`.
- The `init`/`init -n` privilege test verifies that no metadata or source changes on failure; it also exercises valid `add`/`add -n` failure before imports. Wrapped LinkError classification specifically distinguishes 1314 from other errors and mentions Developer Mode only for that Windows privilege error.
- Inspected CI with `gh run list` and `gh run view`: [run 37956688010](https://github.com/sanketvgh/envbuckets/actions/runs/37956688010) passed all Linux/macOS/Windows integration and safeguard jobs on baseline `316ed53`. **That run does not verify these uncommitted changes.**
- At initial handoff, no commit or push had been performed. The user subsequently authorized commit and push to check these changes in CI. Symlink setup remains deferred as instructed. Successful import/link behavior, mode-000 behavior, and the full exit checklist require CI for the same code. Implementation and test coverage are present; symlink-dependent exit boxes remain open pending execution.
- First pushed implementation: `a8feb93`, [CI run 37960018643](https://github.com/sanketvgh/envbuckets/actions/runs/37960018643). Windows integration and the main lint/unit/build/snapshot job passed. Linux/macOS integration exposed a script error: this pinned testscript version rejects multiple paths in one `chmod` command, despite its package documentation describing `path...`. Split both permission changes into one call per file; application code did not change.
- Corrected commit `823e0d9`: [CI run 37960285426](https://github.com/sanketvgh/envbuckets/actions/runs/37960285426) **passed all seven jobs**: integration and safeguards on Linux, macOS, and Windows, plus the main Go lint/unit/build/snapshot/npm-artifact job. The unreadable-file integration script passed on Linux/macOS. The main unit job exercised the rerun/fresh-clone, mixed-link, zero-byte creation, dry-run snapshot, unsafe-metadata, literal-filename, and both injected move-failure tests.
- All exit criteria are now verified. Local `task check` integration remains blocked by Windows error 1314; the required local lint/schema/unit checks plus passing cross-platform CI for the same code satisfy AGENTS.md's Windows verification rule. Symlink setup remains deferred. Subsequent ticket/index updates change documentation only.

## Exit checklist

Tick every box before starting EB-07's end-to-end work.

- [x] `init` creates the config, default bucket, `.gitignore` block, and hook, and imports untracked `.env` and `.env.*` files; tracked files and ignored folders are skipped.
- [x] `init` is safe to rerun and on a fresh clone only adds what is missing.
- [x] `init` on an alpha repo leaves exactly one `.gitignore` block and one hook block, and reports a leftover `.envbuckets.toml` as unused.
- [x] `add` checks every path first and changes nothing if any fails; it refuses directories, links, and tracked files.
- [x] A failure during `add` leaves the original file in place.
- [x] `init` and `add` work on files with mode `000` on Linux and macOS (move, not copy).
- [x] `switch -c` creates zero-byte files, refuses an existing bucket, and never touches the current bucket.
- [x] "Current bucket" follows the rule: links, else the branch's bucket, else stop on mixed links.
- [x] `-n` leaves the repo byte-identical and returns the real run's exit code for `init`, `add`, and `switch -c`.
- [x] The symlink probe reports a clear message for Windows without Developer Mode (privilege error), also under `init -n`.
- [x] A symlinked `.gitignore` target is neither read nor modified.
- [x] A real `.env` remains at its source path when the bucket directory is symlinked.
- [x] Output matches the PRODUCT.md sample runs for these commands.
- [x] `task check` requirement satisfied under the Windows verification rule: local lint/schema/unit checks and same-code cross-platform CI passed; local integration is blocked only by symlink privilege.
