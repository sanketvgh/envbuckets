# EB-03: init, add, and switch -c

**Goal:** get files into buckets.

**Scope:**

- `init`: create the config if missing, with `$schema` as the first key and `default` set to `dev`, create the default bucket folder, write the `.gitignore` block, install the hook, and import untracked `.env` and `.env.*` files. Find candidates through Git so fully ignored folders like `node_modules/` are never walked. Replace the alpha's `.gitignore` block (`# >>> envbuckets v1 >>>`) instead of adding a second one. Print `Adding <path> to bucket '<b>'` per file, then `Initialized envbuckets in <repo>/.env.d/`; with no files found, add `hint: No local files found. Create them, then run "envbuckets add <file>".`
- `add <file>...`: check every path first and stop with `fatal:` if any cannot be added. Then move each into the current bucket with EB-01's move helper, leaving a link, and add the path to the ignore block. Silent on success.
- `switch -c <bucket>`: validate the name, refuse an existing bucket (`fatal: a bucket named '<b>' already exists`), create an empty file at each of the current bucket's paths, then switch and print `Switched to a new bucket '<b>'`.
- Current bucket: the bucket the links point to; with no links, the branch's bucket; with links to several buckets, stop and suggest `switch`.
- `-n` for `init`, `add`, and `switch -c`: print the plan and stop. A dry-run `init` still checks symlink support, using a temp link it removes right away.

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
- `init` stops with a clear message when symlinks cannot be created, such as on Windows without Developer Mode.
- The ignore block is marker-guarded, never duplicated, and skips paths Git already ignores.
- With `-n`, `init`, `add`, and `switch -c` leave the repo byte-identical and exit with the code a real run would return.
