# EB-03: init, add, and switch -c

**Goal:** get files into buckets.

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
- With `-n`, `init`, `add`, and `switch -c` leave the repo byte-identical and exit with the code a real run would return.

## Gaps, risks, and tradeoffs

**Gaps**

- The Developer Mode probe is the only reliable check, so a machine that allows links in one folder but not another may pass the probe and fail later.
- `init` finding candidates through Git needs a decision for repos with submodules or a very large untracked tree.

**Risks**

- Moving files rather than copying them means a crash between the hard link and the symlink rename relies on `switch` to repair the link.
- `add` on a file another process has open can fail on Windows.
- A bad `.gitignore` edit could hide or expose files; the marker block must be exact and idempotent.

**Tradeoffs**

- Move-and-link never reads contents (matches the safety rules), but there is no copy mode, so a user who wants a backup must make one first.
- `init` skipping an unimportable file with a `warning:` finishes more work than stopping, but a quiet skip can be missed in a long run.
- Probing for symlink support first adds one temporary file, which keeps `-n` honest.

## Exit checklist

Tick every box before starting EB-07's end-to-end work.

- [ ] `init` creates the config, default bucket, `.gitignore` block, and hook, and imports untracked `.env` and `.env.*` files; tracked files and ignored folders are skipped.
- [ ] `init` is safe to rerun and on a fresh clone only adds what is missing.
- [ ] `init` on an alpha repo leaves exactly one `.gitignore` block and one hook block, and reports a leftover `.envbuckets.toml` as unused.
- [ ] `add` checks every path first and changes nothing if any fails; it refuses directories, links, and tracked files.
- [ ] A failure during `add` leaves the original file in place.
- [ ] `init` and `add` work on files with mode `000` on Linux and macOS (move, not copy).
- [ ] `switch -c` creates zero-byte files, refuses an existing bucket, and never touches the current bucket.
- [ ] "Current bucket" follows the rule: links, else the branch's bucket, else stop on mixed links.
- [ ] `-n` leaves the repo byte-identical and returns the real run's exit code for `init`, `add`, and `switch -c`.
- [ ] The symlink probe reports a clear message for Windows without Developer Mode (privilege error), also under `init -n`.
- [ ] Output matches the PRODUCT.md sample runs for these commands.
- [ ] `task check` passes.
