package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"golang.org/x/sys/windows"
)

type terminalBuffer struct {
	bytes.Buffer
	handle windows.Handle
}

func (b *terminalBuffer) Fd() uintptr { return uintptr(b.handle) }

// Use a real MSYS terminal-shaped pipe handle. Writes are captured in memory,
// so tests do not need an interactive shell or change the user's console mode.
func testTerminal(t *testing.T) *terminalBuffer {
	t.Helper()
	name, err := windows.UTF16PtrFromString(`\\.\pipe\msys-envbuckets-pty99-to-master`)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateNamedPipe(name, windows.PIPE_ACCESS_DUPLEX, windows.PIPE_TYPE_BYTE, windows.PIPE_UNLIMITED_INSTANCES, 4096, 4096, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := windows.CloseHandle(h); err != nil {
			t.Error(err)
		}
	})
	return &terminalBuffer{handle: h}
}

func TestColoredCommandsAndReports(t *testing.T) {
	r := reportRepo(t)
	r.git("branch", "漢字")
	r.git("branch", "café")
	r.write(".env.d/dev/.env", "SYNTHETIC_BUCKET")
	r.write(".env", "SYNTHETIC_REAL")
	for _, args := range [][]string{{"branches", "**"}, {"status"}, {"switch", "-x"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out, errout := testTerminal(t), testTerminal(t)
			code := Run(args, Env{Cwd: r.root, Stdout: out, Stderr: errout})
			plain := r.run(args...)
			if code != plain.code || ansi.Strip(out.String()) != plain.stdout || ansi.Strip(errout.String()) != plain.stderr {
				t.Fatalf("text changed: out=%q err=%q plain=%+v", out.String(), errout.String(), plain)
			}
			switch args[0] {
			case "branches":
				if !strings.Contains(out.String(), "\x1b[32mmain") {
					t.Fatalf("branch not green: %q", out.String())
				}
			case "status":
				if !strings.Contains(out.String(), "\x1b[32mdev") || !strings.Contains(out.String(), "\x1b[31m.env") {
					t.Fatalf("status colors=%q", out.String())
				}
			case "switch":
				if !strings.Contains(errout.String(), "\x1b[31merror:") {
					t.Fatalf("diagnostic not red: %q", errout.String())
				}
			}
		})
	}
	for _, value := range []string{"yes", "false", "0", " "} {
		out, errout := testTerminal(t), testTerminal(t)
		Run([]string{"status"}, Env{Cwd: r.root, Stdout: out, Stderr: errout, Getenv: func(key string) string {
			if key == "NO_COLOR" {
				return value
			}
			return ""
		}})
		if strings.Contains(out.String()+errout.String(), "\x1b") {
			t.Fatalf("NO_COLOR=%q emitted escapes", value)
		}
	}
	// Hook config failure always exits zero; terminal stderr and piped stdout
	// must make separate decisions, exactly as when Git launches the hook.
	if err := os.Remove(r.path(".envbuckets.json")); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	stderr := testTerminal(t)
	code := Run([]string{"hook", "old", "new", "1"}, Env{Cwd: r.root, Stdout: &stdout, Stderr: stderr})
	plain := r.run("hook", "old", "new", "1")
	if code != ExitOK || stdout.Len() != 0 || ansi.Strip(stderr.String()) != plain.stderr || !strings.Contains(stderr.String(), "envbuckets: \x1b[33mwarning:") {
		t.Fatalf("hook code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}
}
