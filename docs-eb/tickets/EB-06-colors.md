# EB-06: Colored output with Lip Gloss

**Goal:** make output easier to scan in a terminal without changing what it says.

**Scope:**

- Style output with [Lip Gloss](https://github.com/charmbracelet/lipgloss), behind EB-01's output helper so commands never call it directly.
- Add color: `fatal:` and `error:` red, `warning:` and `hint:` yellow. Everything else stays plain.
- Pad the `branches` and `status` columns with Lip Gloss, which measures visible width; padding with `fmt` would count color codes and misalign the columns.
- In `status`, show the bucket in use in green (like the current branch in `git branch`) and problem paths in red (like unstaged files in `git status`). In `branches`, show the current branch in green.
- Turn color on only when the stream is a terminal and `NO_COLOR` is unset. Detect stdout and stderr separately, since one can be a terminal while the other is piped.
- On Windows, make sure the console's virtual terminal mode is on; if it cannot be, print without color.

## Acceptance criteria

- Piped or redirected output, and any output with `NO_COLOR` set, contains no escape codes.
- Message text is identical with and without color. Tests compare uncolored output.
- Windows Terminal and the classic Windows console show colors, not escape codes.
- Hook output is colored only when Git runs it in a terminal.
- Colored `branches` and `status` columns line up exactly like the uncolored ones.
- `task security` passes after adding the dependency.
