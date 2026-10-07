# EB-04: status

**Goal:** answer "which files am I using, and is anything wrong?" in the shape of `git status`.

**Scope:**

- First line: `On branch <name>`, or `HEAD detached at <commit>`.
- Bucket line: `Using bucket 'dev' (default)` or `(rule '<pattern>')`. When the links differ from the branch's bucket, or that bucket is missing, say so and add a hint in parentheses on the next line.
- A section per kind of problem, each with a hint in parentheses and tab-indented `label:   path` lines: real file in the way, broken link, foreign symlink, links to several buckets, linked path not ignored by Git.
- Last line: `all N files linked`, or `X of N files linked`.
- Config errors and a missing config print `fatal:` lines; a missing config hints to run `envbuckets init`.

## Acceptance criteria

- Every problem above has a test with the exact uncolored output.
- `status` writes to stdout, never reads file contents, and changes nothing.
- It works on a detached HEAD and without a config.
