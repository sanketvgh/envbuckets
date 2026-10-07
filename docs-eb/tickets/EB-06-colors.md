# EB-06: Colored output

**Goal:** make output easier to scan in a terminal without changing what it says.

**Scope:**

- Add color to EB-01's output helper: `fatal:` and `error:` red, `warning:` and `hint:` yellow. Everything else stays plain.
- In `status`, show the bucket in use in green (like the current branch in `git branch`) and problem paths in red (like unstaged files in `git status`).
- Turn color on only when the stream is a terminal and `NO_COLOR` is unset. Check stdout and stderr separately.
- On Windows, enable virtual terminal processing; if that fails, print without color.
- A few ANSI codes behind the helper are enough; no styling library.

## Acceptance criteria

- Piped or redirected output, and any output with `NO_COLOR` set, contains no escape codes.
- Message text is identical with and without color. Tests compare uncolored output.
- Windows Terminal and the classic Windows console show colors, not escape codes.
- Hook output is colored only when Git runs it in a terminal.
