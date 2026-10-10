package output

import "golang.org/x/sys/windows"

func enableVirtualTerminal(fd uintptr) (bool, func() error) {
	return enableConsoleMode(windows.Handle(fd), windows.GetConsoleMode, windows.SetConsoleMode)
}

func enableConsoleMode(handle windows.Handle, get func(windows.Handle, *uint32) error, set func(windows.Handle, uint32) error) (bool, func() error) {
	var previous uint32
	if err := get(handle, &previous); err != nil {
		return false, nil
	}
	if previous&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true, nil
	}
	if err := set(handle, previous|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return false, nil
	}
	return true, func() error { return set(handle, previous) }
}
