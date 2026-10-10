package output

import (
	"errors"
	"os"
	"os/exec"
	"reflect"
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
)

// A CREATE_NO_WINDOW child has its own hidden classic console, so this checks
// the real Windows APIs without altering or opening the user's terminal.
func TestHiddenWindowsConsole(t *testing.T) {
	if os.Getenv("EB06_CONSOLE_CHILD") != "1" {
		exe, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(exe, "-test.run=^TestHiddenWindowsConsole$", "-test.v") //nolint:gosec // Only re-executes this test binary with fixed test arguments.
		cmd.Env = append(os.Environ(), "EB06_CONSOLE_CHILD=1")
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("hidden console: %v\n%s", err, out)
		}
		return
	}
	name, err := windows.UTF16PtrFromString("CONOUT$")
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		handle, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		f := os.NewFile(uintptr(handle), "hidden console")
		defer f.Close()
		var initial uint32
		if err := windows.GetConsoleMode(handle, &initial); err != nil {
			t.Fatal(err)
		}
		initial &^= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
		if err := windows.SetConsoleMode(handle, initial); err != nil {
			t.Fatal(err)
		}
		s := NewStream(f, nil)
		if !s.color {
			t.Fatal("hidden classic console was not colored")
		}
		var enabled uint32
		if err := windows.GetConsoleMode(handle, &enabled); err != nil || enabled&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING == 0 {
			t.Fatalf("mode=%d err=%v", enabled, err)
		}
		var before, after windows.ConsoleScreenBufferInfo
		if err := windows.GetConsoleScreenBufferInfo(handle, &before); err != nil {
			t.Fatal(err)
		}
		Writer{Out: s}.List("%s", Red(s, "X"))
		if err := windows.GetConsoleScreenBufferInfo(handle, &after); err != nil {
			t.Fatal(err)
		}
		if after.CursorPosition.X != before.CursorPosition.X+1 || after.CursorPosition.Y != before.CursorPosition.Y || after.Attributes != before.Attributes {
			t.Fatalf("console printed escapes or failed to reset color: before=%+v after=%+v", before, after)
		}
		Writer{Err: s}.Error("synthetic")
		if err := windows.GetConsoleScreenBufferInfo(handle, &after); err != nil {
			t.Fatal(err)
		}
		// Literal escapes would occupy cells. A parsed diagnostic newline moves
		// to column zero on the next row and leaves the default attributes intact.
		if after.CursorPosition.X != 0 || after.CursorPosition.Y != before.CursorPosition.Y+1 || after.Attributes != before.Attributes {
			t.Fatalf("console did not interpret/reset ANSI: before=%+v after=%+v", before, after)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		var restored uint32
		if err := windows.GetConsoleMode(handle, &restored); err != nil || restored != initial {
			t.Fatalf("restored=%d initial=%d err=%v", restored, initial, err)
		}
	}
}

func TestConsoleModeLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		mode                     uint32
		getFails, setFails, want bool
	}{
		{name: "enable and restore", mode: windows.ENABLE_PROCESSED_OUTPUT, want: true},
		{name: "already enabled", mode: windows.ENABLE_PROCESSED_OUTPUT | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING, want: true},
		{name: "old console", setFails: true},
		{name: "not a console", getFails: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var modes []uint32
			get := func(h windows.Handle, mode *uint32) error {
				if h != 42 {
					t.Fatal("wrong stream handle")
				}
				if tc.getFails {
					return errors.New("cannot read mode")
				}
				*mode = tc.mode
				return nil
			}
			set := func(h windows.Handle, mode uint32) error {
				if h != 42 {
					t.Fatal("wrong stream handle")
				}
				modes = append(modes, mode)
				if tc.setFails {
					return errors.New("cannot set mode")
				}
				return nil
			}
			color, restore := enableConsoleMode(42, get, set)
			if color != tc.want {
				t.Fatalf("color=%v", color)
			}
			if restore != nil {
				if err := restore(); err != nil {
					t.Fatal(err)
				}
			}
			var want []uint32
			if !tc.getFails && tc.mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING == 0 {
				want = append(want, tc.mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
				if !tc.setFails {
					want = append(want, tc.mode)
				}
			}
			if !reflect.DeepEqual(modes, want) {
				t.Fatalf("modes=%v want=%v", modes, want)
			}
		})
	}
}
