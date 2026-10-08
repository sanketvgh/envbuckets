# EB-06: Colored output with Lip Gloss

**Goal:** make output easier to scan in a terminal without changing what it says.

**Scope:**

- Style output with [Lip Gloss](https://github.com/charmbracelet/lipgloss), behind EB-01's output helper so commands never call it directly.
- Add color: `fatal:` and `error:` red, `warning:` and `hint:` yellow. Everything else stays plain.
- Pad the `branches` and `status` columns with Lip Gloss, which measures visible width; padding with `fmt` would count color codes and misalign the columns.
- In `status`, show the bucket in use in green (like the current branch in `git branch`) and problem paths in red (like unstaged files in `git status`). In `branches`, show the current branch in green.
- Turn color on only when the stream is a terminal and `NO_COLOR` is unset. Treat any non-empty `NO_COLOR` as set, checked by us (`os.Getenv("NO_COLOR") != ""`) before the library runs, because `colorprofile` parses it as a boolean and would ignore `NO_COLOR=yes`. Detect stdout and stderr separately, since one can be a terminal while the other is piped.
- Build one stream type per output, taking the writer and the environment explicitly. Do not use `lipgloss.Writer` or `Sprint*` for stderr: they use stdout's decision. Keep `Fd()` reachable on any wrapper, or detection silently turns color off.
- On Windows, enable `ENABLE_VIRTUAL_TERMINAL_PROCESSING` on stdout and stderr separately with `golang.org/x/sys/windows`, restore the previous mode on exit, and print without color if it cannot be enabled. Neither Lip Gloss nor `colorprofile` does this. Treat a Git Bash or mintty terminal as a terminal, as `cli/cli` does with an `IsCygwinTerminal` check next to the normal one, because `colorprofile` does not detect them.

## Acceptance criteria

- Piped or redirected output, and any output with `NO_COLOR` set, contains no escape codes.
- Message text is identical with and without color. Tests compare uncolored output.
- Windows Terminal and the classic Windows console show colors, not escape codes.
- Hook output is colored only when Git runs it in a terminal.
- Colored `branches` and `status` columns line up exactly like the uncolored ones.
- `task security` passes after adding the dependency.

## Gaps, risks, and tradeoffs

**Gaps**

- The Git Bash and mintty fallback (`IsCygwinTerminal`) is copied from `cli/cli` and untested here.
- Behavior on very old Windows consoles without virtual terminal mode is not verified; the plan is to print without color.

**Risks**

- `lipgloss.Writer` and `Sprint*` use stdout's color decision, so using them for stderr would color piped stderr. Commands must go through the output helper only.
- `colorprofile` ignores `NO_COLOR=yes`; our own non-empty check must run first.
- Wrapping `os.Stderr` in a type without `Fd()` silently turns color off.
- Not restoring the console mode after enabling virtual terminal mode can leave a user's terminal changed.

**Tradeoffs**

- Lip Gloss is a heavy dependency for four colors, but PRODUCT.md names it and it measures visible width, which `fmt` padding cannot.
- Treating any non-empty `NO_COLOR` as off follows the no-color.org rule; the library's boolean parsing would be simpler but wrong for `yes`.
- Extra Windows-specific code (`x/sys/windows`) is needed because neither library enables virtual terminal mode.

## Exit checklist

Tick every box before starting EB-07's release work.

- [ ] Piped output, redirected output, and any non-empty `NO_COLOR` produce no escape codes.
- [ ] stdout and stderr are detected separately, through one stream type per output; nothing calls Lip Gloss directly.
- [ ] Message text is identical with and without color; tests compare uncolored output.
- [ ] `branches` and `status` columns line up the same colored and uncolored.
- [ ] Windows virtual terminal mode is enabled per stream and restored on exit; with failure, output is plain.
- [ ] Windows Terminal and the classic console show colors, not escape codes (checked by hand or in CI).
- [ ] Git Bash behavior is decided and tested.
- [ ] Hook output is colored only when Git runs it in a terminal.
- [ ] `task security` passes after the new dependency.
- [ ] `task check` passes.
