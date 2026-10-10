// Package output centralizes Git-style command messages and stream routing.
package output

import (
	"fmt"
	"io"
)

// Writer routes user-facing output to the standard Git-style streams.
type Writer struct {
	Out  io.Writer
	Err  io.Writer
	Hook bool
}

func (w Writer) message(stream io.Writer, kind, format string, args ...any) {
	label := kind + ":"
	switch kind {
	case "fatal", "error":
		label = Red(stream, label)
	case "warning", "hint":
		label = yellow(stream, label)
	}
	prefix := label + " "
	if w.Hook {
		prefix = "envbuckets: " + prefix
	}
	fmt.Fprintf(stream, prefix+format+"\n", args...)
}

// Fatal writes a fatal diagnostic to stderr.
func (w Writer) Fatal(format string, args ...any) { w.message(w.Err, "fatal", format, args...) }

// Error writes an error diagnostic to stderr.
func (w Writer) Error(format string, args ...any) { w.message(w.Err, "error", format, args...) }

// Warning writes a warning diagnostic to stderr.
func (w Writer) Warning(format string, args ...any) { w.message(w.Err, "warning", format, args...) }

// Hint writes a hint diagnostic to stderr.
func (w Writer) Hint(format string, args ...any) { w.message(w.Err, "hint", format, args...) }

// Would writes a dry-run action to stdout.
func (w Writer) Would(format string, args ...any) { fmt.Fprintf(w.Out, "Would "+format+"\n", args...) }

// Info writes an informational message to stderr, like Git command progress.
func (w Writer) Info(format string, args ...any) {
	if w.Hook {
		format = "envbuckets: " + format
	}
	fmt.Fprintf(w.Err, format+"\n", args...)
}

// List writes a status or branch listing to stdout.
func (w Writer) List(format string, args ...any) { fmt.Fprintf(w.Out, format, args...) }

// Usage writes plain usage text to stderr.
func (w Writer) Usage(format string, args ...any) { fmt.Fprintf(w.Err, "usage: "+format+"\n", args...) }
