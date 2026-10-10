package output

import (
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-isatty"
)

// Stream owns one writer's color decision and terminal-mode lifetime.
// Use ordinary writer-directed printing; Lip Gloss's global print helpers
// decide color from stdout and cannot safely render stderr.
type Stream struct {
	writer  io.Writer
	color   bool
	restore func() error
}

type terminalOps struct {
	isTerminal       func(uintptr) bool
	isCygwinTerminal func(uintptr) bool
	enable           func(uintptr) (bool, func() error)
}

// NewStream detects only this writer. getenv supplies the command's environment;
// nil means an empty environment. NO_COLOR is deliberately not parsed as a bool.
func NewStream(writer io.Writer, getenv func(string) string) *Stream {
	return newStream(writer, getenv, terminalOps{isatty.IsTerminal, isatty.IsCygwinTerminal, enableVirtualTerminal})
}

func newStream(writer io.Writer, getenv func(string) string, ops terminalOps) *Stream {
	s := &Stream{writer: writer}
	if getenv != nil && (getenv("NO_COLOR") != "" || getenv("TERM") == "dumb") {
		return s
	}
	fd, ok := writer.(interface{ Fd() uintptr })
	if !ok {
		return s
	}
	switch {
	case ops.isTerminal(fd.Fd()):
		s.color, s.restore = ops.enable(fd.Fd())
	case ops.isCygwinTerminal(fd.Fd()):
		// MSYS/Cygwin terminal pipes already interpret ANSI. They are not
		// Windows console handles, so GetConsoleMode must not gate them.
		s.color = true
	}
	return s
}

// Write preserves io.Writer semantics and never makes a global color decision.
func (s *Stream) Write(p []byte) (int, error) { return s.writer.Write(p) }

// Fd preserves terminal detection through this wrapper. Non-file writers return
// an invalid descriptor, which terminal detectors reject.
func (s *Stream) Fd() uintptr {
	if fd, ok := s.writer.(interface{ Fd() uintptr }); ok {
		return fd.Fd()
	}
	return ^uintptr(0)
}

// Close restores the previous terminal mode once; it does not close the writer.
func (s *Stream) Close() error {
	if s.restore == nil {
		return nil
	}
	restore := s.restore
	s.restore = nil
	return restore()
}

func styled(writer io.Writer, text string, style lipgloss.Style) string {
	if s, ok := writer.(*Stream); ok && s.color {
		// Inline rendering removes newlines and block rendering normalizes CRLF.
		// Render individual spans so even unusual problem paths keep their bytes.
		style = style.Inline(true).TabWidth(-1)
		var b strings.Builder
		for {
			i := strings.IndexAny(text, "\r\n")
			if i < 0 {
				b.WriteString(style.Render(text))
				return b.String()
			}
			b.WriteString(style.Render(text[:i]))
			b.WriteByte(text[i])
			text = text[i+1:]
		}
	}
	return text
}

// Green styles the active branch or bucket on this stream.
func Green(writer io.Writer, text string) string {
	return styled(writer, text, lipgloss.NewStyle().Foreground(lipgloss.Green))
}

// Red styles a problem path on this stream.
func Red(writer io.Writer, text string) string {
	return styled(writer, text, lipgloss.NewStyle().Foreground(lipgloss.Red))
}

func yellow(writer io.Writer, text string) string {
	return styled(writer, text, lipgloss.NewStyle().Foreground(lipgloss.Yellow))
}

// Width measures terminal cells, excluding ANSI escapes.
func Width(text string) int { return lipgloss.Width(text) }

// Pad adds spaces to a column without counting color escapes or expanding tabs.
func Pad(text string, width int) string {
	return lipgloss.NewStyle().TabWidth(-1).PaddingRight(max(0, width-Width(text))).Render(text)
}
