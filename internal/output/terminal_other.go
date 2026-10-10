//go:build !windows

package output

func enableVirtualTerminal(_ uintptr) (bool, func() error) { return true, nil }
