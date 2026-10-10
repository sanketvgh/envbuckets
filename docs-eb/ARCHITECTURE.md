# Architecture research: Go tooling and prior art

Written 2026-10-08 against `feat/docs-demo` at `1bf4fae` plus uncommitted EB-01 code in the working tree.

Research only. No code or docs outside this file were changed. `docs/adr/INDEX.md` does not exist, so there were no prior decisions to apply.

**Status:** historical research, with final-review corrections below. The
implementation now uses explicit command dispatch, command-specific plans, the
Ajv library through `tools/validate-config-schema.mjs`, and tested marker-block
hooks. Use [PRODUCT.md](PRODUCT.md), [DEVELOPMENT.md](DEVELOPMENT.md), and the
[tickets](tickets/README.md) for the current contract and verification results.
Recommendations and unresolved questions below describe the research stage.

Method: `go doc` (Go 1.27.1) and a local `git` experiment; Grep MCP, GitHub MCP (for SHAs and files) and DeepWiki; five parallel Haiku subagents, one per aspect (hooks, dry-run and symlinks, output and color, testing, config and schema). Evidence labels used below:

- **read**: file content was fetched and read.
- **summary**: DeepWiki or a subagent summary I did not check against the code.
- **local**: I ran it here.

Pinned commits: `direnv/direnv@e24ea74873aff78d5e371c85061dc7fafdeedd5a`, `evilmartians/lefthook@60c1ddd071810542eb3e639dca6529831dbb4b3e`, `asdf-vm/asdf@cb72590e9f9691b6e651aecfe4b0c9334adff41b`, `twpayne/chezmoi@a34c45c9f65ce873579c74fa5f61f4265af068a8`, `git-lfs/git-lfs@0043a645047926f4bd7f7091299095528253d575`, `rogpeppe/go-internal@38c169d642f5e6ee08f0c90fe56b64c5ae2db19e`, `cli/cli@8db3c3107a174c51de4758f53d5b27911650931b`, `charmbracelet/colorprofile@51fafca7b44cffe4004bd86edd6bd924f84ce0c9`, `charmbracelet/lipgloss@6a419c6543d3475a369ef08f6252a2a6f33be733`. Grep MCP does not return SHAs, so these are the default-branch heads at research time and will drift. Line ranges appear only where a file was read and counted; otherwise the symbol is cited.

## 1. Go tooling to reuse

| Need | Use | Ticket | Notes |
| --- | --- | --- | --- |
| Reject paths outside the repo, through symlinked folders, `..`, Windows device names | `os.OpenRoot` / `os.Root` (`Lstat`, `Readlink`, `Symlink`, `Link`, `Rename`, `Remove`, `MkdirAll`, `OpenRoot`, `FS`) | EB-01, 02, 03, 05 | Verified with `go doc`. Every method refuses to leave the root. Go's own `src/os/root_test.go` and `root_windows_test.go` cover case-insensitivity and `NUL`. |
| Cheap lexical pre-check | `path/filepath.IsLocal` | EB-01, 03 | Rejects absolute, empty, escaping paths and Windows reserved names. Lexical only; keep as a first filter before `os.Root`. |
| Scan buckets without following links | `os.Lstat`, `io/fs.WalkDir` over `Root.FS()` | EB-01, 04, 05 | Entry type comes from the directory read, no file contents touched. |
| Atomic link swap | temp symlink then `os.Rename` (or the `Root` methods) | EB-01, 02 | `go doc os.Rename`: "on non-Unix platforms Rename is not an atomic operation". See section 3. |
| Move into a bucket without the path vanishing | `os.Link`, then rename a symlink over the original | EB-01, 03 | Already the plan. |
| Strict config parsing | `encoding/json/v2` with `RejectUnknownMembers(true)` | EB-01 | Tested on Go 1.27.1 (section 9a). v1's `DisallowUnknownFields` accepts duplicate keys and wrong-case keys; v2 rejects both, plus trailing data and invalid UTF-8, and errors carry a JSON pointer. |
| Find the hooks directory | `git rev-parse --git-path hooks` | EB-02, 05 | **local**: honors `core.hooksPath` and, inside a linked worktree, resolves to the common dir's `hooks`. Do not hardcode `.git/hooks`. |
| Run git | `os/exec` (`LookPath`, `Cmd.Environ`) | EB-02 to 04 | Already in `internal/gitx`. |
| Version string | `runtime/debug.ReadBuildInfo` (`vcs.revision`) | EB-07 | asdf's `cmd/asdf/main.go` does exactly this (**read**). |
| Plain-text columns | `text/tabwriter` | EB-04 | Frozen, and it counts ANSI bytes, so it cannot align colored output. |
| Flag parsing | `flag.FlagSet` per subcommand | EB-03 to 05 | `flag` stops at the first non-flag argument, so `branches 'release/**' --bucket prod` fails and `branches --bucket prod 'release/**'` works. Decide before EB-04. |
| Fake FS in tests | `testing/fstest` | EB-01 | Read-only; symlink and rename tests need real `t.TempDir`. |
| Tests that call the real binary | `testscript.Main` | EB-02, 07 | See section 5. |

`go doc` and `go/doc` inspect Go source, not repos, so they do not help the product. Already adopted: `go-internal/testscript`, `git-lfs/wildmatch/v2`.

**Recommendation:** make `os.Root` the only way `internal/fsx` touches the repo, with `filepath.IsLocal` as a pre-check.

## 2. Hooks: install, uninstall, never block

Sources: the hook subagent (git-lfs, lefthook, husky, pre-commit; **summary** unless noted), DeepWiki on git-lfs, and my own check of lefthook's tree.

| Tool | Foreign hook | Own-hook marker | Missing binary | `core.hooksPath` | Uninstall |
| --- | --- | --- | --- | --- | --- |
| git-lfs (`lfs/hook.go`, summary) | Refuses unless the file is empty or a known older LFS version; `--manual` prints lines to paste | Compares full content with current and `upgradeables` | Hook script prints a message and exits 2 | Reads it via `config.HookDir()` | Removes the hook only if content matches a known LFS version |
| lefthook (`internal/command/install.go`, `lefthook.go` `cleanHook`; install.go is not at `internal/lefthook/`) | Renames to `<name>.old` | Marker line plus md5 checksum file under the git info dir | n/a | Refuses unless `--force` or `--reset-hooks-path` | Deletes its hook, renames `.old` back |
| pre-commit (`install_uninstall.py`, `hook-tmpl`) | Moves to `<hook>.legacy`, runs it in "migration mode" | Hash of known template versions | Prints an error and **exits 1**, which blocks Git | Refuses to install | Only if it is our script; restores `.legacy` |
| husky (`index.js`) | Not preserved | none | Install returns an error string | Sets `core.hooksPath` to `.husky/_` | n/a |

What this means for EB-02 and EB-05:

- Nobody appends a block to an existing hook. All four either refuse, move the file aside, or take over `core.hooksPath`. Our append-and-remove-block design is the least invasive, but it is also the one with no prior art to copy, so it needs the most tests.
- **git-lfs, read** (`git-lfs/git-lfs@0043a64 lfs/hook.go`): `Install(force)` (L84-L98) calls `Upgrade()` when the file exists. `matchesCurrent()` (L147-L172) reads at most 1024 bytes and accepts only the exact current hook, an empty file, or a known older LFS version; anything else returns "Hook already exists". `Uninstall()` (L126-L141) then leaves foreign content alone. `Install(true)` overwrites unconditionally. The LFS hook exits 2 when `git-lfs` is missing (L19 template). So `git lfs install` fails on a hook that has our block appended, and `git lfs install --force` silently deletes our block. `envbuckets init` is safe to rerun and restores it; `status` should warn when the block is missing (add to EB-04). Command layer (**read**, `commands/command_install.go`, `command_update.go`): `git lfs install` calls the same update path; on a hook conflict it prints the error plus "To resolve this, either: 1: run `git lfs update --manual` for instructions on how to merge hooks. 2: run `git lfs update --force` to overwrite your hook." and exits non-zero. `--manual` prints the lines to paste (`getHookInstallSteps`). So users with a conflict are told to merge by hand, which our appended block already does for them.
- Section 9a records Git 2.54 config hooks as an alternative that avoids the risks below; they are not adopted.
- The Git LFS conflict is real: `git lfs install` writes `post-checkout`, and LFS refuses to touch a hook it does not recognize. A repo using both needs `envbuckets init` after LFS, and LFS then finds a hook with our block in it. Each tool's check would fail in a different order. Test both orders in EB-02 and document that `git lfs update --manual` is the LFS-side path.
- Copy none of the missing-binary behavior of pre-commit (exit 1) or git-lfs (exit 2). Our hook should be `command -v envbuckets >/dev/null 2>&1 || exit 0`, then the call followed by `|| true`, then `exit 0` (matches PRODUCT.md rule 4).
- The subagent flagged lefthook's `createHooksIfNeeded`, which silently returns nil when writing the ghost hook fails. That is the opposite of our "print one warning" rule; do not copy it.
- Use `#!/bin/sh`, mode 0755, and resolve the directory with `git rev-parse --git-path hooks`. If it is outside the repo (a shared `core.hooksPath`), decide between refusing with `fatal:` (what lefthook and pre-commit do) and installing there. Recommended: refuse, because the hook would then affect every repo that uses that path.
- Uninstall: remove exactly the marker range, delete the file only if what remains is empty or a bare shebang, and write through a temp file plus rename. Add a test that every byte outside the block is unchanged.

## 3. Plan then apply, dry-run, and symlinks

Sources: dry-run subagent (read `chezmoi internal/chezmoi/dryrunsystem.go` in full, DeepWiki for stow), DeepWiki on chezmoi.

- **chezmoi:** a `System` interface wraps all OS access; `DryRunSystem` passes reads through and turns every write into `setModified()` returning nil (**read**). Pipeline is source state, target state, actual state, apply. Authors' tradeoff: dry-run says "would modify", not the exact change, and equivalence holds only if the wrapper is the sole path to writes.
- **GNU stow (read, `aspiers/stow@891c187b73d4654ec9800647af06f0cfd880e2fe lib/Stow.pm.in`):** `sub conflict` (L1390-L1400) only counts conflicts; the abort is documented in `bin/stow.in` and `doc/stow.texi`: a plain file at a target is "registered as a conflict" and stow will "refuse to proceed" (read via search snippet). Ownership is `find_stowed_path` (L1022-L1058): a link is owned only if its target is relative and resolves into the stow directory; absolute links are never owned. An unowned link is the conflict "existing target is not owned by stow" (L548-L553), and a real file where a link goes is a conflict unless `--adopt` (L646-L651). Unstow silently ignores unowned links (L925-L931). Rule to copy: plan first, never delete what you do not own. A link is owned only if it points into a package directory, and anything else at a target is a conflict that stow never deletes. `-n` runs the same planner.
- **Atomicity:** chezmoi uses `renameio.Symlink` (temp plus rename) on Unix and `RemoveAll` then `Symlink` on Windows, which has a gap (summary). So "atomic on Windows" cannot be claimed.
- **Windows symlinks:** `os.Symlink` first passes `SYMBOLIC_LINK_FLAG_ALLOW_UNPRIVILEGED_CREATE`, then retries without it. Failure shows as `*os.LinkError` wrapping `syscall.ERROR_PRIVILEGE_NOT_HELD` (1314). Go has no Developer Mode API; Go's `internal/testenv/testenv_windows.go` probes by creating a link and checking the error (summary, symbol only).

For envbuckets:

1. One planner returns a typed `[]Action` (move, link, remove, conflict, preflight error). Real runs and `-n` both consume it. "Would ..." lines and the exit code come from the plan, so they cannot diverge (acceptance criterion 13). A plain slice is simpler than chezmoi's `System` wrapper at this size.
2. Conflicts are found in the plan, not during apply. This matches the "ownership" rule: a symlink is ours only if it resolves into `.env.d/`; everything else in the way is skipped and reported.
3. The symlink probe in `init` (EB-03) should be a preflight action, so `init -n` reports it too.
4. EB-07 should state: Windows rename over an existing link may not be atomic; `switch` converges on rerun.

## 4. Command structure and config (direnv, asdf)

- **direnv (read):** `main.go` into `internal/cmd`, one `cmd_<name>.go` per command. A `Cmd{Name, Desc, Args, Aliases, Private, Action}` struct and a `CmdList` slice; `CommandsDispatch` loops over it (`commands.go` L33-L42 and the dispatch func). DeepWiki says the maintainers chose this over cobra for binary size (summary). `LoadConfig` merges env vars with an optional `direnv.toml` and logs a deprecation instead of silently ignoring an old key (`config.go` `LoadConfig`).
- **asdf (read):** `cmd/asdf/main.go` only builds the version and calls `cli.Execute`; logic sits in many small `internal/*` packages with fixture helpers (`repotest`, `installtest`). Config is env then `.asdfrc` then defaults, loaded lazily.
- **chezmoi (summary):** cobra, commands as methods on a central `Config` struct, with annotations controlling setup and teardown.
- **mise (summary):** config discovery walks up directories, with a trust model for config files that can run code. envbuckets has no executable config, so trust is out of scope, which is a good reason to keep the file data-only.
- **lefthook (summary):** several formats plus `extends`, `remotes`, and a local override file. This is the surface envbuckets deliberately avoids.

Recommendation: a `Cmd` table (name, one-line help, usage, flag setup, action) in `internal/cli`, one file per command as the tickets already split them. No cobra.

## 5. Testing

Sources: testing subagent (grep snippets, partly unread) and my earlier results.

- `testscript.Main(m, map[string]func(){...})` registers the CLI as a command so scripts call the real binary with no build step (`FiloSottile/age cmd/age-keygen/keygen_test.go`, **read** snippet). Hugo shares one params struct across script dirs.
- Conditions: chezmoi uses `[!exec:git] skip '...'` then `[windows] skip '...'`; Hugo's `Condition` errors on unknown names, which catches typos (snippets).
- Isolation: Hugo's `testSetupFunc` points HOME and cache dirs at the work dir. The agent found no sourced pattern for `GIT_CONFIG_GLOBAL` or fixed git dates; those are our recommendation: `HOME`, `XDG_CONFIG_HOME`, `GIT_CONFIG_GLOBAL`, `GIT_CONFIG_NOSYSTEM=1`, fixed author and committer name, email, date.
- Golden updates: `testscript.Params.UpdateScripts` rewrites `cmp` targets; gate it behind `TESTSCRIPT_UPDATE=1` in a `task` target and review the diff.
- git-lfs tests hooks with `t/t-install-custom-hooks-path.sh` (shell, not txtar). The txtar equivalent is `exec git commit` after the CLI installed the hook.
- No sourced 1000-branch benchmark was found. Plan: a Go `testing.B` in its own file, skipped under `-short`, creating branches with one `git update-ref --stdin` call, with a budget in the ticket instead of a timing assert.
- Risk: hook tests on Windows need Git for Windows to run the hook through sh. Test it explicitly instead of skipping, and name platform limits in EB-07 rather than blanket `[windows] skip`.
- asdf uses about 30 bats scripts; envbuckets already has the better tool.

## 6. Config, schema, and the glob matcher

- **lefthook (summary):** `gen/jsonschema.go` reflects Go types with `invopop/jsonschema` and stamps a "Last updated" comment, so a byte-compare drift check would fail daily.
- **invopop/jsonschema (README):** defaults to draft 2020-12 and adds `$id`. EB-01 needs draft-07, so generated output would need post-processing.
- **goreleaser (read, file level, `goreleaser/goreleaser@1942d44355492f44cd3774c57bd3c27fe544dfe9`):** `Taskfile.yml` has `schema:generate` (`go run . schema -o ./www/static/schema.json`) and `schema:validate` (`jv`). A daily `generate.yml` workflow regenerates the file and opens a bot PR if it changed, and PR CI (`build.yml`) has no up-to-date check, so drift is repaired after the fact, not blocked. `pkg/config/jsonschema_test.go` (`TestSlackJSONSchema`) reflects the config type with `invopop/jsonschema`, compiles it with `santhosh-tekuri/jsonschema/v5`, and validates real configs, but it tests the generated schema, not the committed file.
- **golangci-lint (read, `golangci/golangci-lint@d9df2fa53265776945734b5c5902489edc386095`):** `jsonschema/` holds versioned schema files embedded in `jsonschema.go` and loaded with `santhosh-tekuri/jsonschema/v6`. No drift check found in its workflows, and I did not find a generator.
- **SchemaStore (read, `SchemaStore/schemastore@c10b9c4086c8d160d82a76070271eda385ca47c6` `CONTRIBUTING.md`):** add the schema under `src/schemas/json`, positive tests under `src/test/<name>/`, optional negative tests under `src/negative_test/<name>/`, then a catalog entry in `src/api/json/catalog.json` with a `fileMatch`. CI runs Ajv checks and a coverage check (enum values, required fields). The guide recommends draft-07, matching EB-01. Not found: whether a `$schema` URL on `raw.githubusercontent.com` is accepted (the guide only uses such URLs as catalog `url`), and the exact catalog entry shape (the file was too large to fetch). Registering is optional in PRODUCT.md and stays out of the first release. Closed with Exa and Grep: a catalog entry is `{name, description, fileMatch, url, versions?}` (`schema-catalog.json` requires `name`, `url`, `description`), and `url` may point at an external host such as `raw.githubusercontent.com`; CONTRIBUTING shows the Ory Hydra entry. SchemaStore itself moved URLs around in 2025 (issues #4695, #4759), and a maintainer's concern was that raw GitHub URLs change when a default branch is renamed. For us: the `$schema` URL in PRODUCT.md points at `main` on raw.githubusercontent.com, so it breaks if `main` is renamed. Consider pinning it to a release tag in `init`'s output later.
- **Takeaway:** no project found blocks PRs on schema drift. Our structural Go test plus validating real configs in `go test` already goes further than any of them.
- **santhosh-tekuri/jsonschema (README):** v6 supports draft-07 and can be used as a library in Go tests, so the Go side can validate fixtures without ajv.
- **Globs:** `bmatcuk/doublestar` supports `{a,b}` braces that Git does not, and mid-pattern `**` acts like `*`. `go-git`'s gitignore matcher ports `wildmatch.c` (comment in `plumbing/format/gitignore/pattern.go`), with gitignore-specific semantics. Not applicable.
- **wildmatch vs Git (read by a subagent at `git-lfs/wildmatch@ad2401267b216ccdf6b0c61835b250552e5a7b4d`, plus a local probe of `internal/pattern`):** matching is always component-based, so `*` never crosses `/`; POSIX classes are supported; unknown classes and malformed patterns panic (our `compile` recovers, and `validateSyntax` pre-checks). The library treats a trailing `/` as "directories only, any depth", not `**`, which is why `compile` appends `**` itself. My probe: `\x` matches `x` and `wip\*` matches `wip*` as in Git. **Real divergence found locally:** `[]a]` passes `validateSyntax` but matches neither `]` nor `a`, while Git matches both. Other PRODUCT.md rows hold in the probe: `release/` matches `release/1.2/rc1` but not `release`; `release/**` does not match `release`; `**/hotfix` matches `hotfix`; `[[:digit:]]` matches `5`.
- **Git `onbranch` docs (read, `git/git@6de20f6092dcf9bdb1c8efe03db4b70c82b423dd Documentation/config.adoc`):** the pattern is matched against the checked-out branch name, and a trailing `/` adds `**`. The docs do not say whether a branch named exactly `foo` matches `foo/`, or whether `**/` is auto-prepended, so the Patterns table in PRODUCT.md must be checked against real Git (the differential tests in EB-01), not assumed.
- **Matcher decision:** `git-lfs/wildmatch/v2` is MIT-licensed (checked in the module cache), so keeping it behind `internal/pattern` is compatible with this project's MIT license. EB-01 was updated to say so and to add differential tests against real Git.

Recommendations for EB-01:

1. Keep the hand-written schema. Make the drift test structural: reflect over the config struct's json tags and compare key sets, `required`, `additionalProperties`, and the bucket-name pattern with the schema file.
2. One Go constant for the bucket-name regex, in a subset that means the same in RE2 and ECMA (decided: `^[A-Za-z0-9][A-Za-z0-9_-]*$`, so a bucket name cannot start with `-` or `_`). Avoid lookarounds and `\p{}`.
3. Validate every JSON example from `PRODUCT.md` and invalid fixtures against the schema in a Go test, and run the Ajv validator in `task lint`.
4. Wrap strict-parse errors as `.envbuckets.json: <problem>` built from `JSONPointer` and `ErrUnknownName` (section 9a, item 5), and test the error kind and key name, not Go's wording.
5. Differential tests: done once by hand for 40 cases against `includeIf onbranch:` (section 9a, item 2). Keep them as a committed fixture with Git's answers; the working tree already has `internal/pattern/testdata/git-patterns.json`.

## 7. Output and color (EB-06, shared helper in EB-01)

Source: output subagent (**summary**, symbols only).

- `lipgloss` pulls in `colorprofile`, which decides TTY and `NO_COLOR`. A writer must expose `Fd()` to count as a TTY; wrapping `os.Stderr` in a struct without `Fd()` silently drops color.
- `colorprofile` parses `NO_COLOR` with `strconv.ParseBool`, so `NO_COLOR=yes` is ignored, unlike the no-color.org rule (any non-empty value). PRODUCT.md says "when `NO_COLOR` is set". Check `os.Getenv("NO_COLOR") != ""` yourself before colorprofile.
- `lipgloss.Writer` is bound to stdout at init, so `Sprint*` for stderr uses stdout's decision. Build one stream type per output (`NewStream(w, env)`) and pass the environment explicitly.
- Lip Gloss provides `EnableLegacyWindowsANSI`, but it returns no result and does not restore the console mode (verified during EB-06). Use `golang.org/x/sys/windows` for stdout and stderr separately so failures produce plain text and each original mode is restored on exit. `cli/cli` adds `go-colorable` for old consoles.
- Git Bash and mintty ptys are not detected by `colorprofile`. `cli/cli` adds the fallback itself: `isTerminal(f)` is `ghTerm.IsTerminal(f) || isatty.IsCygwinTerminal(f.Fd())` (`cli/cli pkg/iostreams/iostreams.go`, **read** snippet near L603). Copy that if Git Bash users should get color. Decide if that is acceptable.
- Tests: any `bytes.Buffer` is non-TTY, so uncolored by default. Add one color test per profile with injected env. `lipgloss.Width` (`charmbracelet/lipgloss@6a419c6 size.go`, **read**) takes the max `ansi.StringWidth` per line, so it ignores ANSI sequences; `Style.Render` padding (`pad`) appends fixed runes and does not measure. `alignTextHorizontal` (`lipgloss@main align.go`, **read** snippet) also measures each line with `ansi.StringWidth`, and `charmbracelet/x ansi/width.go` documents that "ANSI escape codes are ignored and wide characters ... are accounted for", so Lip Gloss column padding is safe with color. EB-06 still tests that colored columns match uncolored ones. `golang.org/x/term@6226200 term_windows.go` `isTerminal` is `GetConsoleMode` (**read**); mintty is not mentioned there, so Git Bash would likely report non-terminal (inference, test it).
- Git prints advice to stderr and `advice.*` / `GIT_ADVICE=0` silence it. Optional: a similar switch for `hint:` lines. Not in PRODUCT.md, so only if wanted.
- Do not use termenv's background-color query for CLI output (up to 5 s timeout).

## 8. What the sources disagree on

- Hand-rolled vs library dispatch: direnv hand-rolls, chezmoi uses cobra. At eight commands the table is cheaper.
- Existing hooks: refuse (git-lfs, lefthook force flags), move aside (lefthook, pre-commit), or append (our plan). Appending had no prior implementation evidence during this research; EB-02 now tests coexistence, replacement, and removal, including Git LFS.
- Schema: generate (lefthook) vs hand-write. With three object types, hand-write plus a structural test.

## 9. Recommendation per part

1. **Tooling:** `os.Root` everywhere, `IsLocal` as pre-check, `git rev-parse --git-path hooks` for the hook location.
2. **Architecture:** `Cmd` table in `internal/cli`; one planner producing typed actions shared by real runs and `-n`; strict, data-only config; no config layers.
3. **Hooks:** keep the marker block, test both Git LFS orders, exit 0 on every path, and refuse a shared `core.hooksPath` with `fatal:`. Git 2.54 config hooks are the recorded alternative (section 9a).
4. **Tests:** testscript with a shared isolated Setup, real-binary registration, `UpdateScripts` behind an env var, a separate benchmark.
5. **Output:** one stream type per output, own `NO_COLOR` check, explicit Windows VT enabling.

## 9a. Direction changes to cut risk

Each item below was tested on this machine (Windows, Git 2.54.0) rather than argued.

1. **Git's config-based hooks (Git 2.54, April 2026): tested, not adopted.** Decision: EB-02 keeps the marker block only, to avoid two install paths. This stays on record as the option if hook-file problems show up. `hook.<name>.event` and `hook.<name>.command` in the local config run a command on a hook event, before any `.git/hooks` file ([Git 2.54 release notes](https://github.blog/open-source/git/highlights-from-git-2-54/), [git-hook docs](https://git-scm.com/docs/git-hook)). Local tests:
   - The command runs from the working tree root and receives the hook arguments (old HEAD, new HEAD, flag `1` for a branch checkout, `0` for a file checkout). A shell command line gets them as `"$@"` automatically, so the command must not append them itself.
   - A legacy `.git/hooks/post-checkout` that does `exit 1` does not stop the config hook, and the config hook ran in a linked worktree (local config is shared).
   - A missing binary in the command makes `git checkout` exit 1. The form `f() { command -v envbuckets >/dev/null 2>&1 || return 0; envbuckets hook post-checkout "$@" || true; }; f` printed nothing and left the checkout at exit 0, with all three arguments passed.
   - Removal is `git config --unset-all`, with nothing left behind.

   It would remove four risks of the marker-block plan: Git LFS overwriting the hook, an earlier `exit` in a user's hook, exact byte-range removal on uninstall, and a shared `core.hooksPath`. Cost of adopting it: two install paths (the block is still needed for Git older than 2.54, such as Ubuntu 24.04's 2.43) and a Git version check.
2. **Matcher: tested against real Git, one bug fixed.** 40 cases (the Patterns table, the rule examples, and extra classes and edge cases) went through `git config includeIf.onbranch:<pattern>` on a branch of that name, and through `internal/pattern`. 39 agree; `[]a]` does not. A leading `]` rewritten to `\]` fixes it. This closes the open question about `foo/`: real Git says `release/` does not match `release` but matches `release/1.2/rc1`. The 40 cases and Git's answers become a committed fixture.
3. **Windows rename: less uncertain, still not provable here.** Go's Windows `os.Rename` is one `MoveFileEx` call with `MOVEFILE_REPLACE_EXISTING` (read in Go's source under `GOROOT/src/os/file_windows.go` and `internal/syscall/windows`), and our code does no delete before it. Microsoft does not promise atomicity, so the ticket wording stays "converges on rerun". I could not run it: on this machine `os.Symlink` fails with "A required privilege is not held by the client", so Developer Mode is off. That confirms the error the `init` probe must detect, and it means symlink tests need CI or Developer Mode.
4. **`$schema` pinning (proposal, not applied).** The `$schema` URL in PRODUCT.md points at `main`. SchemaStore's own history shows raw GitHub URLs breaking when branches or hosting change. Pinning to a release tag removes that risk, but `init` would need the version at build time and dev builds would omit `$schema`. This changes PRODUCT.md, so it needs your decision.

5. **Config parsing: `encoding/json/v2` instead of v1.** Run on Go 1.27.1, with no `GOEXPERIMENT` set. The package is in Go's `api/go1.27.txt` and `JSONv2` is in the default experiment baseline (`internal/buildcfg/exp.go`). Same ten inputs through v1 (`DisallowUnknownFields` plus a trailing-data check) and v2 (`RejectUnknownMembers(true)`):

   | Input | v1 | v2 |
   | --- | --- | --- |
   | unknown key | rejected | rejected |
   | unknown key in a rule | rejected | rejected, with `/rules/0` |
   | duplicate key | **accepted** (last wins) | rejected |
   | wrong-case key (`Default`) | **accepted** | rejected |
   | invalid UTF-8 | **accepted** | rejected |
   | trailing data | rejected (with an extra check) | rejected |
   | trailing comma, wrong value type | rejected | rejected, with position or pointer |
   | missing `default` | accepted | accepted (no `required` option; check after decoding) |

   v1 would have let two typo classes through that the JSON Schema rejects, so the CLI and the schema would have disagreed (acceptance criterion 3 in PRODUCT.md). v2's error text can change between releases, so messages are built from `SemanticError.JSONPointer` and `ErrUnknownName`. EB-01 is updated.

PRODUCT.md keeps describing the `.git/hooks/post-checkout` block, which matches the decision above.

## 10. Third goal

The request left the third goal as a placeholder. By request it is out of scope for this report.

## 11. Gaps

- DeepWiki and subagent summaries were not checked against code unless marked read. Unread: stow's `bin/stow.in` abort code (its docs were read), and `ansi.StringWidth` beyond its doc comment.
- The mise point is a DeepWiki summary only. mise is Rust, so there is no code to reuse and it is dropped from the recommendations. Whether SchemaStore restricts a user file's `$schema` URL was not found; its docs only discuss catalog `url`.
- The subagent noted Grep MCP returned an error and an empty result on some queries.
- Nothing found measures a post-checkout hook with many branches, but Git's own perf tests show the setup: `t/perf/p6300-for-each-ref.sh` creates 10000 branches with one `git update-ref --stdin` call and times `git for-each-ref` over loose then packed refs, and a Git patch measured `update-ref --stdin` about 9.5 times faster than a `git branch` loop (983 ms vs 9.3 s). EB-07's benchmark should use that setup.
