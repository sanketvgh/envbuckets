// Package block upserts and removes the marker-guarded envbuckets section
// in files shared with the user: the post-checkout hook and .gitignore.
package block

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/fsx"
)

const (
	// Begin is the opening marker written to shared files.
	Begin = "# >>> envbuckets >>>"
	// End is the closing marker written to shared files.
	End = "# <<< envbuckets <<<"

	beginPrefix = "# >>> envbuckets "
	endPrefix   = "# <<< envbuckets "
)

// HookBody is the shell snippet inside the hook block.
const HookBody = "command -v envbuckets >/dev/null 2>&1 || exit 0\nenvbuckets hook \"$@\" || true\nexit 0"

func render(body string) []byte {
	return []byte(Begin + "\n" + body + "\n" + End + "\n")
}

func find(content []byte) (start, end int, ok bool) {
	pos, begin := 0, -1
	for _, line := range bytes.SplitAfter(content, []byte("\n")) {
		text := strings.TrimRight(string(line), "\r\n")
		if begin < 0 && strings.HasPrefix(text, beginPrefix) && strings.HasSuffix(text, " >>>") {
			begin = pos
		} else if begin >= 0 && strings.HasPrefix(text, endPrefix) && strings.HasSuffix(text, " <<<") {
			return begin, pos + len(line), true
		}
		pos += len(line)
	}
	return 0, 0, false
}

// Body returns the lines between the markers, or nil when absent.
func Body(content []byte) []string {
	start, end, ok := find(content)
	if !ok {
		return nil
	}
	inner := content[start:end]
	lines := strings.Split(strings.TrimRight(string(inner), "\n"), "\n")
	if len(lines) < 2 {
		return nil
	}
	return lines[1 : len(lines)-1]
}

// Upsert replaces the existing block or appends one, reporting whether
// the content changed.
func Upsert(content []byte, body string) ([]byte, bool) {
	block := render(body)
	start, end, ok := find(content)
	if !ok {
		out := append([]byte(nil), content...)
		if len(out) > 0 && out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		return append(out, block...), true
	}
	if bytes.Equal(content[start:end], block) {
		return content, false
	}
	out := append([]byte(nil), content[:start]...)
	out = append(out, block...)
	return append(out, content[end:]...), true
}

// Remove strips the block, reporting whether one was found.
func Remove(content []byte) ([]byte, bool) {
	start, end, ok := find(content)
	if !ok {
		return content, false
	}
	out := append([]byte(nil), content[:start]...)
	return append(out, content[end:]...), true
}

// HookResult reports what InstallHook or RemoveHook did.
type HookResult int

// Outcomes of hook file operations.
const (
	HookUnchanged HookResult = iota
	HookWritten
	HookDeleted
)

// InstallHook appends the envbuckets block to the hook at path, creating
// the file with a shebang when absent and keeping it executable.
func InstallHook(path string) (HookResult, error) {
	if bucketPath(path) {
		return HookUnchanged, errors.New("hooks directory is inside a bucket tree")
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return HookUnchanged, err
	}
	defer root.Close()
	return installHookRoot(root, filepath.Base(path))
}

// RemoveHook strips the block, deleting the file when only a shebang and
// blank lines remain and otherwise writing the rest back unchanged.
func RemoveHook(path string) (HookResult, error) {
	existing, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return HookUnchanged, nil
	}
	if err != nil {
		return HookUnchanged, err
	}
	out, found := Remove(existing)
	if !found {
		return HookUnchanged, nil
	}
	if onlyShebang(out) {
		if err := os.Remove(path); err != nil {
			return HookUnchanged, err
		}
		return HookDeleted, nil
	}
	if err := fsx.WriteFileAtomic(path, out, 0o755); err != nil {
		return HookUnchanged, err
	}
	return HookWritten, os.Chmod(path, 0o755) //nolint:gosec // git hooks must be executable
}

func onlyShebang(content []byte) bool {
	for i, line := range strings.Split(string(content), "\n") {
		l := strings.TrimSpace(line)
		if l == "" || (i == 0 && strings.HasPrefix(l, "#!")) {
			continue
		}
		return false
	}
	return true
}

// MergeLines appends any required lines missing from existing, never
// dropping any, so files on disk stay ignored until an explicit purge.
func MergeLines(existing, required []string) []string {
	seen := make(map[string]bool, len(existing))
	out := make([]string, 0, len(existing)+len(required))
	for _, l := range existing {
		if l == "" || seen[l] {
			continue
		}
		seen[l] = true
		out = append(out, l)
	}
	for _, l := range required {
		if seen[l] {
			continue
		}
		seen[l] = true
		out = append(out, l)
	}
	return out
}

// DropLines removes the given lines from the block lines.
func DropLines(existing, drop []string) []string {
	gone := make(map[string]bool, len(drop))
	for _, l := range drop {
		gone[l] = true
	}
	out := make([]string, 0, len(existing))
	for _, l := range existing {
		if !gone[l] {
			out = append(out, l)
		}
	}
	return out
}
