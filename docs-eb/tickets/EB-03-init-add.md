# EB-03: init, add, and switch -c

**Goal:** get files into buckets.

**Scope:**

- `init`: create the config if missing (default `dev`), create the default bucket folder, write the `.gitignore` block, install the hook, and import untracked `.env` and `.env.*` files. Find candidates through Git so fully ignored folders like `node_modules/` are never walked.
- `add <file>...`: check every path first and stop with `fatal:` if any cannot be added. Then copy each into the current bucket, verify the copy, atomically replace the file with a link, and add the path to the ignore block. Silent on success.
- `switch -c <bucket>`: validate the name, refuse an existing bucket, copy the current bucket's files, then switch.
- Current bucket: the bucket the links point to; with no links, the branch's bucket; with links to several buckets, stop and suggest `switch`.
- `-n` for `init`, `add`, and `switch -c`: print the plan and stop. A dry-run `init` still checks symlink support, using a temp link it removes right away.

## Acceptance criteria

- Existing bucket files and real working files are never overwritten. `init` skips a collision with a `warning:`; `add` refuses it with `fatal:` before changing anything.
- A failure during `add` leaves the original file in place.
- Tracked files such as `.env.example` are never imported or added.
- Rerunning `init` on a set-up repo changes nothing. On a fresh clone it installs the hook and creates only what is missing.
- `init` stops with a clear message when symlinks cannot be created, such as on Windows without Developer Mode.
- The ignore block is marker-guarded, never duplicated, and skips paths Git already ignores.
- With `-n`, `init`, `add`, and `switch -c` leave the repo byte-identical and exit with the code a real run would return.
