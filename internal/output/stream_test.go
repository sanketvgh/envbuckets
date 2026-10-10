package output

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

type descriptorWriter struct {
	bytes.Buffer
	fd uintptr
}

func (w *descriptorWriter) Fd() uintptr { return w.fd }

func TestStreamColorPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, noColor, term             string
		terminal, cygwin, enabled, want bool
	}{
		{name: "console", terminal: true, enabled: true, want: true},
		{name: "mode failure", terminal: true},
		{name: "pipe"},
		{name: "mintty", cygwin: true, want: true},
		{name: "dumb terminal", term: "dumb", terminal: true, enabled: true},
		{name: "no color yes", noColor: "yes", terminal: true, enabled: true},
		{name: "no color true", noColor: "true", terminal: true, enabled: true},
		{name: "no color false", noColor: "false", terminal: true, enabled: true},
		{name: "no color zero", noColor: "0", cygwin: true},
		{name: "no color whitespace", noColor: " ", terminal: true, enabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var detected, enabled int
			w := &descriptorWriter{fd: 42}
			s := newStream(w, func(key string) string {
				switch key {
				case "NO_COLOR":
					return tc.noColor
				case "TERM":
					return tc.term
				default:
					return "1" // FORCE_COLOR must never color a pipe.
				}
			}, terminalOps{
				isTerminal:       func(fd uintptr) bool { detected++; return fd == 42 && tc.terminal },
				isCygwinTerminal: func(fd uintptr) bool { detected++; return fd == 42 && tc.cygwin },
				enable:           func(_ uintptr) (bool, func() error) { enabled++; return tc.enabled, nil },
			})
			defer s.Close()
			Writer{Err: s}.Fatal("problem")
			got := w.String()
			if strings.Contains(got, "\x1b[") != tc.want || ansi.Strip(got) != "fatal: problem\n" {
				t.Fatalf("output=%q, want color=%v", got, tc.want)
			}
			if (tc.noColor != "" || tc.term == "dumb") && (detected != 0 || enabled != 0) {
				t.Fatal("disabled color performed terminal operations")
			}
			if tc.cygwin && enabled != 0 {
				t.Fatal("mintty pipe attempted console mode changes")
			}
		})
	}
}

func TestIndependentStreamsAndHookText(t *testing.T) {
	for _, stdoutTTY := range []bool{false, true} {
		for _, stderrTTY := range []bool{false, true} {
			for _, hook := range []bool{false, true} {
				out, errout := &descriptorWriter{fd: 1}, &descriptorWriter{fd: 2}
				ops := terminalOps{
					isTerminal:       func(fd uintptr) bool { return fd == 1 && stdoutTTY || fd == 2 && stderrTTY },
					isCygwinTerminal: func(uintptr) bool { return false },
					enable:           func(uintptr) (bool, func() error) { return true, nil },
				}
				stdout, stderr := newStream(out, nil, ops), newStream(errout, nil, ops)
				w := Writer{Out: stdout, Err: stderr, Hook: hook}
				w.Fatal("fatal text")
				w.Error("error text")
				w.Warning("warning text")
				w.Hint("hint text")
				w.Info("Switched to bucket 'dev'")
				w.Usage("envbuckets switch")
				w.Would("link .env")
				w.List("%s\n", Green(stdout, "main"))
				var plainOut, plainErr bytes.Buffer
				plain := Writer{Out: &plainOut, Err: &plainErr, Hook: hook}
				plain.Fatal("fatal text")
				plain.Error("error text")
				plain.Warning("warning text")
				plain.Hint("hint text")
				plain.Info("Switched to bucket 'dev'")
				plain.Usage("envbuckets switch")
				plain.Would("link .env")
				plain.List("main\n")
				if ansi.Strip(out.String()) != plainOut.String() || ansi.Strip(errout.String()) != plainErr.String() {
					t.Fatal("styling changed message text")
				}
				if strings.Contains(out.String(), "\x1b[") != stdoutTTY || strings.Contains(errout.String(), "\x1b[") != stderrTTY {
					t.Fatalf("stdoutTTY=%v stderrTTY=%v hook=%v: out=%q err=%q", stdoutTTY, stderrTTY, hook, out.String(), errout.String())
				}
				if stderrTTY && (!strings.Contains(errout.String(), "\x1b[31mfatal:") || !strings.Contains(errout.String(), "\x1b[33mwarning:")) {
					t.Fatalf("incorrect diagnostic colors: %q", errout.String())
				}
			}
		}
	}
}

func TestVisibleColumnPadding(t *testing.T) {
	s := &Stream{color: true}
	for _, text := range []string{"main", "漢字", "café", "👩‍💻", "warning: file"} {
		colored := Green(s, text)
		if Width(colored) != Width(text) || ansi.Strip(Pad(colored, 16)) != Pad(text, 16) || Width(Pad(colored, 16)) != 16 {
			t.Fatalf("padding changed visible width: %q", text)
		}
		if ansi.Strip(Red(s, text)) != text || Pad(text, 0) != text {
			t.Fatalf("styling changed text: %q", text)
		}
	}
}

func TestStyledTextPreservesWhitespace(t *testing.T) {
	s := &Stream{color: true}
	for _, text := range []string{"", " leading and trailing ", "path\tname", "path\nname", "path\r\nname\n", "path\rname"} {
		if got := ansi.Strip(Red(s, text)); got != text {
			t.Fatalf("styled=%q want=%q", got, text)
		}
	}
}

func TestStreamRestorationAndDescriptor(t *testing.T) {
	var restored int
	w := &descriptorWriter{fd: 42}
	wantErr := errors.New("restore failed")
	wrapped := NewStream(w, func(string) string { return "yes" })
	s := newStream(wrapped, nil, terminalOps{
		isTerminal:       func(fd uintptr) bool { return fd == 42 },
		isCygwinTerminal: func(uintptr) bool { return false },
		enable: func(uintptr) (bool, func() error) {
			return true, func() error { restored++; return wantErr }
		},
	})
	if s.Fd() != 42 || !s.color {
		t.Fatal("wrapper lost its descriptor")
	}
	if !errors.Is(s.Close(), wantErr) || s.Close() != nil || restored != 1 {
		t.Fatal("restoration must run exactly once and return its error")
	}
	if n, err := s.Write([]byte("still open")); err != nil || n != len("still open") || w.String() != "still open" {
		t.Fatal("Close closed the underlying writer")
	}
	if NewStream(&bytes.Buffer{}, nil).Fd() != ^uintptr(0) {
		t.Fatal("non-file writer has a valid descriptor")
	}
}

func TestActualPipeAndRedirectArePlain(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, noColor := range []string{"", "yes", "false", "0"} {
		s := NewStream(w, func(key string) string {
			if key == "NO_COLOR" {
				return noColor
			}
			return "1"
		})
		if s.color {
			t.Fatal("real pipe was detected as a terminal")
		}
		Writer{Err: s}.Error("synthetic")
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil || strings.Contains(string(got), "\x1b") {
		t.Fatalf("pipe=%q err=%v", got, err)
	}
	f, err := os.CreateTemp(t.TempDir(), "redirect")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s := NewStream(f, nil)
	defer s.Close()
	Writer{Err: s}.Warning("synthetic")
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	got, err = io.ReadAll(f)
	if err != nil || string(got) != "warning: synthetic\n" {
		t.Fatalf("redirect=%q err=%v", got, err)
	}
}
