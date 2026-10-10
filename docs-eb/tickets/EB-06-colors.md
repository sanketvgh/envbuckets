# EB-06: Colored output with Lip Gloss

**Goal:** make output easier to scan in a terminal without changing what it says.

**Status:** implemented and verified by [passing cross-platform CI on `a50e2cf`](https://github.com/sanketvgh/envbuckets/actions/runs/38024972163); the user deferred Windows Terminal visual confirmation until after release on 2026-10-10. It is no longer a pre-PR or release blocker. Local lint/schema/unit/security checks pass. The local integration symlink limitation is accepted under AGENTS.md with same-code Linux/macOS/Windows integration evidence.

## Phase checklist

- [x] Phase 1: review output call sites, current Lip Gloss APIs, and per-stream terminal/Windows behavior.
- [x] Phase 2: implement stream styling, diagnostic routing, report padding, and Windows mode restoration.
- [x] Phase 3: add unit/integration coverage and user documentation.
- [x] Phase 4: run formatting, lint, security, and authorized local checks; review the diff (Windows integration limitation recorded below).
- [x] Phase 5: inspect same-code CI and record exit-criterion evidence and remaining limitations.

**Scope:**

- Style output with [Lip Gloss](https://github.com/charmbracelet/lipgloss), behind EB-01's output helper so commands never call it directly.
- Add color: `fatal:` and `error:` red, `warning:` and `hint:` yellow. Everything else stays plain.
- Pad the `branches` and `status` columns with Lip Gloss, which measures visible width; padding with `fmt` would count color codes and misalign the columns.
- In `status`, show the bucket in use in green (like the current branch in `git branch`) and problem paths in red (like unstaged files in `git status`). In `branches`, show the current branch in green.
- Turn color on only when the stream is a terminal and `NO_COLOR` is unset. Treat any non-empty `NO_COLOR` as set, checked by us (`os.Getenv("NO_COLOR") != ""`) before the library runs, because `colorprofile` parses it as a boolean and would ignore `NO_COLOR=yes`. Detect stdout and stderr separately, since one can be a terminal while the other is piped.
- Build one stream type per output, taking the writer and the environment explicitly. Do not use `lipgloss.Writer` or `Sprint*` for stderr: they use stdout's decision. Keep `Fd()` reachable on any wrapper, or detection silently turns color off.
- On Windows, enable `ENABLE_VIRTUAL_TERMINAL_PROCESSING` on stdout and stderr separately with `golang.org/x/sys/windows`, restore the previous mode on exit, and print without color if it cannot be enabled. Lip Gloss provides `EnableLegacyWindowsANSI`, but it does not report failures or restore the prior mode; this ticket needs both. Treat a Git Bash or mintty terminal as a terminal, as `cli/cli` does with an `IsCygwinTerminal` check next to the normal one, because `colorprofile` does not detect them.

## Acceptance criteria

- Piped or redirected output, and any output with `NO_COLOR` set, contains no escape codes.
- Message text is identical with and without color. Tests compare uncolored output.
- Windows Terminal and the classic Windows console show colors, not escape codes.
- Hook output is colored only when Git runs it in a terminal.
- Colored `branches` and `status` columns line up exactly like the uncolored ones.
- `task security` passes after adding the dependency.

## Gaps, risks, and tradeoffs

**Gaps**

- The Git Bash and mintty fallback uses `go-isatty`'s `IsCygwinTerminal`, as `cli/cli` does. Real terminal-shaped Windows named pipes test detection and colored CLI output; a live Git Bash/mintty session is not available here.
- Behavior on very old Windows consoles without virtual terminal mode is not verified; the plan is to print without color.

**Risks**

- `lipgloss.Writer` and `Sprint*` use stdout's color decision, so using them for stderr would color piped stderr. Commands must go through the output helper only.
- `colorprofile` ignores `NO_COLOR=yes`; our own non-empty check must run first.
- Wrapping `os.Stderr` in a type without `Fd()` silently turns color off.
- Not restoring the console mode after enabling virtual terminal mode can leave a user's terminal changed.

**Tradeoffs**

- Lip Gloss is a heavy dependency for four colors, but PRODUCT.md names it and it measures visible width, which `fmt` padding cannot.
- Treating any non-empty `NO_COLOR` as off follows the no-color.org rule; the library's boolean parsing would be simpler but wrong for `yes`.
- Extra Windows-specific code (`x/sys/windows`) is needed to detect virtual terminal mode failures and restore the prior mode. Lip Gloss's convenience helper enables the mode without providing either capability.

## Exit checklist

Complete the automated checks before release. The user deferred the manual
Windows Terminal review until after release; keep that item open until observed.

- [x] Piped output, redirected output, and any non-empty `NO_COLOR` produce no escape codes.
- [x] stdout and stderr are detected separately, through one stream type per output; commands never call Lip Gloss directly.
- [x] Message text is identical with and without color; tests compare uncolored output.
- [x] `branches` and `status` columns line up the same colored and uncolored.
- [x] Windows virtual terminal mode is enabled per stream and restored on exit; with failure, output is plain.
- [ ] Windows Terminal shows colors, not escape codes (user-owned review deferred until after release on 2026-10-10); classic-console interpretation is already verified locally and in CI.
- [x] Git Bash behavior is decided and tested (real terminal-shaped MSYS named pipe handles; live-shell limitation noted above).
- [x] Hook output is colored only when Git runs it in a terminal.
- [x] `task security` passes after the new dependency.
- [x] `task check` verification accepted: local lint/schema/unit checks plus passing same-code cross-platform CI, under AGENTS.md's Windows symlink exception.

## Implementation notes

- Lip Gloss v2.0.6 (`charm.land/lipgloss/v2`) stays entirely inside `internal/output`. Its `Style.Render` API renders basic ANSI colors without global stdout profile detection. We do not call `lipgloss.Writer`, `Sprint*`, or its print helpers, and do not depend on `colorprofile` parsing `NO_COLOR`. API research used the [current Lip Gloss documentation](https://pkg.go.dev/charm.land/lipgloss/v2), installed dependency source, and [GitHub CLI's stream detection](https://github.com/cli/cli/blob/trunk/pkg/iostreams/iostreams.go).
- `cli.Run` constructs one `output.Stream` for each writer with an explicit `Getenv` function; `main` passes `os.Getenv`. Any non-empty `NO_COLOR` short-circuits detection and console operations, including `yes`, `false`, `0`, and whitespace. `TERM=dumb` also disables color. Force-color environment variables cannot override a pipe or `NO_COLOR`. Wrappers forward `Fd()`; writer-only buffers remain plain.
- Normal terminal detection and `IsCygwinTerminal` come from `go-isatty` v0.0.24. Console handles enable `ENABLE_VIRTUAL_TERMINAL_PROCESSING` with `x/sys/windows` and capture the original mode. Failed get/set operations fall back to plain text; already-enabled modes are left alone. Deferred stream closes run before `main` calls `os.Exit`, restore in reverse initialization order, and never close stdout/stderr.
- All command and switcher output now goes through the existing `output.Writer`. Only diagnostic labels are red/yellow. The current branch name, active bucket names, and problem paths use per-stream styles. Missing/requested bucket names, informational messages, usage, and dry-run text stay plain. A positive-length hook argument guard makes the existing safe slice bounds explicit to the pinned gosec analyzer without suppressing its findings or changing hook behavior.
- Report padding and width measurement use Lip Gloss with tab expansion disabled. Existing status label spacing and plain report fixtures remain byte-identical. Styling renders spans around CR/LF separately so Lip Gloss cannot normalize unusual path names. Tests cover ASCII, CJK, combining characters, emoji, tabs, whitespace, and line-break preservation.
- Portable unit tests cover color policy, both stdout/stderr terminal combinations, diagnostic colors, hook prefixes, exact stripped text, visible column widths, real pipes/redirected files, descriptor forwarding, and idempotent restoration. Windows tests exercise failed/already-enabled console modes, a real hidden classic console, and actual MSYS terminal-shaped pipe detection through CLI reports and hook diagnostics. Tests use only synthetic fixture values.
- The `colors` txtar script checks exact plain reports, usage, and hook warnings with captured streams, force-color variables, and non-empty `NO_COLOR` values. It needs no symlinks. [Status and branch mappings](../../docs/status.md) documents the behavior. CI's existing Linux/macOS/Windows integration matrix now also runs `internal/output` and `internal/cli` unit tests so Windows-specific tests execute in CI; all previous gates and tool pins remain unchanged.

## Verification evidence (2026-10-10)

- Final `task fix` passed with zero issues; reviewed the complete implementation diff and new source/tests. `task lint:go` passed with zero issues, including formatting and modernize. Logs: `tmp/eb06-fix-final.log` and `tmp/eb06-lint-go-final.log`. Used the existing repository-local `tmp/eb09/bin` tools via a temporary PATH prefix; no global installations or tool pins changed.
- `task security` passed after dependency addition: **No vulnerabilities found.** Log: `tmp/eb06-security.log`.
- Final `task check` passed Go lint, JSON formatting, config schema validation, and all shuffled unit packages. Schema validation accepted eight PRODUCT.md examples and rejected nine invalid fixtures. This includes the real hidden-console test, MSYS pipe/colored CLI checks, and whitespace-preservation tests. Log: `tmp/eb06-check-final.log`; focused output/CLI evidence is also in `tmp/eb06-focused.log`.
- Full integration failed only at Windows symlink creation (`ERROR_PRIVILEGE_NOT_HELD`, 1314). Audited all ten failing scripts: `switch-safety`, `hooks-paths`, `hooks-lfs`, `hooks`, `reports`, `checkout`, `switch`, `uninstall`, `init-add-safety`, and `init-add`. Each failure is the documented privilege limitation; symlink-dependent unit cases skip. Test temporary directories were under `tmp/eb06-tests`. Symlink setup remains deferred under AGENTS.md.
- The new `go test -tags integration -shuffle=on -run '^TestScript/colors$' -count=1 -v .` passed independently. It verified the built executable against real Git and exact output fixtures. Log: `tmp/eb06-colors-integration.log`.
- Initial CI inspection found [run 38022743989](https://github.com/sanketvgh/envbuckets/actions/runs/38022743989) on `393864e` passed all seven jobs, but predates these changes and does **not** verify EB-06. The CI continuation below records the subsequent same-code verification.
- The hidden classic-console test verifies real mode enable/restore, one visible cell for a styled character, and attribute reset, without opening a visible console or modifying the user's terminal. Windows Terminal visual confirmation remains open; live Git Bash/mintty and very old consoles were not available. Named-pipe and injected failure tests cover their intended detection/fallback paths.
- `git diff --check` passed. At the initial implementation review, no files were staged, committed, or pushed; the user subsequently authorized commit/push as recorded below. Only EB-06 implementation, dependencies, tests, documentation, required output routing, and CI test coverage are changed.

## Upstream README and examples review (2026-10-10)

- Read the [upstream README](https://github.com/charmbracelet/lipgloss), examples directory, and the standalone color, layout, languages/ANSI table, and simple list examples. They demonstrate `Style.Render`, value-based style reuse, visible-width layout, and print-time color downsampling. Our basic ANSI palette and explicit per-stream policy preserve the ticket's stricter environment/terminal rules; adaptive background queries and decorative layouts are unnecessary for its Git-style messages.
- Verified [writer.go](https://github.com/charmbracelet/lipgloss/blob/main/writer.go): `Print*` and `Sprint*` use the global stdout profile, while `Fprint*` constructs a writer for its supplied stream using `os.Environ()`. The latter supports separate streams, but does not provide our explicit environment injection or Windows mode lifetime. The implementation continues to use ordinary writer-directed printing through `output.Stream`.
- Verified [ansi_windows.go](https://github.com/charmbracelet/lipgloss/blob/main/ansi_windows.go) against the installed v2.0.6 source. Corrected the scope/tradeoff claim that Lip Gloss cannot enable virtual terminal mode: its `EnableLegacyWindowsANSI` helper can enable it, but returns no result and has no restoration. Our wrapper is still needed for the ticket's fallback/restoration requirements. No Go code changed in this review; previous test evidence remains applicable.

## CI continuation (2026-10-10)

**Later user instruction:** during EB-10, the user said "i'll check after release"
for the Windows Terminal visual review. This explicitly supersedes the blocking
statements in the earlier verification record below. Keep the manual checkbox
open until observed; automated console/color gates remain required.

- Committed EB-06 as `dfe3732` (`✨ feat(output): add per-stream terminal colors`) and pushed it to `feat/docs-demo` at the user's request. Local and remote commit IDs matched.
- [Run 38024526850](https://github.com/sanketvgh/envbuckets/actions/runs/38024526850) passed six of seven jobs: Linux lint/unit/race/build/snapshot checks, Linux/Windows integration, and safeguards on all three platforms. The new macOS output/CLI unit step exposed an existing Unicode fixture assumption: Git precomposed the decomposed `café` ref into `café`, causing two exact-text expectations in `TestBranchesVariants` to fail. Output unit tests passed; the macOS integration scripts were skipped after that unit-step failure. Log: `tmp/eb06-ci-macos-failed.log`.
- The Unicode test now sets `core.precomposeUnicode=false` in its private repository before creating branches. This retains the combining-character fixture on macOS and preserves its exact-text/width assertions on all platforms. [Git documents this macOS setting](https://git-scm.com/docs/git-config#Documentation/git-config.txt-coreprecomposeUnicode). No production code or CI gate changed. Committed and pushed as `a50e2cf` (`✅ test(reports): preserve Unicode fixture on macOS`).
- Correction validation: `task fix`, reviewed diff, `task lint:go`, and focused `TestBranchesVariants` passed. `task check` again passed lint, JSON/schema checks, and all unit packages; only the same ten integration scripts failed for unavailable Windows symlink privilege. Logs: `tmp/eb06-ci-fix.log`, `tmp/eb06-ci-lint-go.log`, `tmp/eb06-ci-unicode.log`, and `tmp/eb06-ci-check.log`.
- [Run 38024972163](https://github.com/sanketvgh/envbuckets/actions/runs/38024972163) on `a50e2cf` passed all seven jobs: Linux lint/unit/race/build/snapshot, integration on Linux/macOS/Windows, and safeguards on Linux/macOS/Windows. Inspected successful macOS and Windows job logs: output/CLI unit packages and `TestScript/colors` passed, as did the complete integration suite. This verifies the Unicode correction and the locally privilege-blocked scripts. Windows output tests include the real hidden classic console and MSYS pipe tests without skip paths. Logs: `tmp/eb06-ci-macos-passed.log`, `tmp/eb06-ci-windows-passed.log`, and `tmp/eb06-ci-correction-watch.log`.
- CI safeguards also passed lint and schema validation; the Linux vulnerability scan reported **No vulnerabilities found** (`tmp/eb06-ci-linux-safeguards.log`). Phase 5 and check verification are complete. The Windows Terminal visual criterion remains unticked; EB-07's release work remains blocked until that check is recorded. Classic-console ANSI interpretation/restoration is verified locally and in CI; live Git Bash/mintty and very old consoles retain the limitations recorded above.
