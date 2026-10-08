# EB-04: status and branches

**Goal:** answer "which files am I using, is anything wrong, and which branch uses which bucket?" in the shape of `git status` and `git branch`.

## status

- First line: `On branch <name>`, or `HEAD detached at <commit>`.
- Bucket line: `Using bucket 'dev' (default)` or `(rule '<pattern>')`. On a detached HEAD there is no reason, just `Using bucket 'dev'` and a hint that links stay as they are. When the links differ from the branch's bucket, say so and add a hint in parentheses on the next line. When the branch's bucket is missing, show the fallback: `Using bucket 'dev' (default), because 'prod' (rule '<pattern>') does not exist` with the hint `(use "envbuckets switch -c prod" to create it)`.
- Links pointing at several buckets replace the bucket line with `Using a mix of buckets: 'dev', 'prod'` and the hint `(use "envbuckets switch" to link one bucket)`.
- Problem sections, each with a hint in parentheses and tab-indented lines, shown only when not empty:
  - `Files not linked:` with `real file:`, `broken link:`, and `foreign link:` labels.
  - `Not ignored by Git:` with the paths, and the hint `(add them to .gitignore)`.
  - `Hook not installed:` when the envbuckets block is missing from the post-checkout hook, with the hint `(use "envbuckets init" to install it)`. Tools such as `git lfs install --force` overwrite the hook file and drop the block.
- Last line: `all N files linked` (`1 file linked` for one), or `X of N files linked`.
- Config errors and a missing config print `fatal:` lines; a missing config hints to run `envbuckets init`.
- A leftover alpha `.envbuckets.toml` gets a `warning:` that it is unused and can be deleted.

## branches

- `branches` lists local branches sorted by name, like `git branch`: a `*` marks the current branch, then aligned columns for branch, bucket, and reason (`(default)` or `(rule '<pattern>')`).
- Plain `branches` shows the current branch and every branch a rule matches, nothing else: other branches use `default` by definition, so there is no count or summary line for them.
- Names, patterns, and `--bucket` show every match, including branches that use the default. `'**'` lists every branch, so there is no `--all` flag.
- The current branch's line adds `, using '<bucket>' for now` when the links point at another bucket.
- A bucket missing from `.env.d/` shows `(rule '<pattern>', missing; falls back to '<default>')`, matching what a checkout would link.
- `branches <name>...` checks only the given names, in the given order, whether or not they exist as branches.
- An argument containing `*`, `?`, or `[` is a pattern: it lists the local branches it matches, using the EB-01 matcher. Branch names cannot contain those characters, so names and patterns never clash.
- `--bucket <name>` keeps only the branches that use that bucket. It combines with names and patterns.
- Read all local branches with one `git for-each-ref refs/heads` call and match in memory; never run Git once per branch.

## Acceptance criteria

- Every status problem and every branches variant above has a test with the exact uncolored output.
- Both commands write to stdout, never read file contents, and change nothing.
- `status` works on a detached HEAD. Without a config it prints `fatal:` with a hint to run `envbuckets init`, and exits 1.
- `branches` uses the same matcher as the hook, so its answer always matches what a checkout would do.
- Rebuild `task bench` (removed in EB-00) on the new CLI, with a 1000-branch case. The hook's time does not change with the number of branches.

## Gaps, risks, and tradeoffs

**Gaps**

- Reading all branches with one `git for-each-ref` call is the plan, but nothing is measured yet for 1000 or more branches. EB-07 adds it.
- Git's docs do not state how `onbranch` treats a name equal to a pattern with a trailing `/`, so `branches` output for such names relies on the differential tests.

**Risks**

- `branches` and the hook must use the same matcher and the same missing-bucket fallback, or `branches` will say one thing and a checkout do another.
- Column alignment with color depends on Lip Gloss width measuring; if it misreads wide characters, columns drift.
- A branch name with non-ASCII characters must not break the column layout.

**Tradeoffs**

- Showing only the current branch and rule-matched branches keeps output short, but a user must pass `'**'` to see everything.
- Treating any argument containing `*`, `?`, or `[` as a pattern is unambiguous because branch names cannot contain them, but a user who types a bracket by mistake gets a pattern error instead of "no such branch".
- The `status` warning about a missing hook block helps after tools like `git lfs install --force`, at the cost of one more check on every `status`.

## Exit checklist

Tick every box before starting EB-06 or EB-07.

- [ ] Every `status` problem section and line has a test with exact uncolored output.
- [ ] Every `branches` variant has a test: plain, names, patterns, `--bucket`, current-branch marker, temporary bucket, missing bucket fallback.
- [ ] `status` works on a detached HEAD; without a config it prints `fatal:` with the `init` hint and exits 1.
- [ ] `status` reports a missing hook block with a hint.
- [ ] Both commands write to stdout, never read file contents, and change nothing.
- [ ] `branches` uses the same matcher and fallback as the hook, with a test that compares them.
- [ ] Branches are read with one `git for-each-ref` call.
- [ ] The 1000-branch benchmark exists and the hook time does not grow with the branch count.
- [ ] `task check` passes.
