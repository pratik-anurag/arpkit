//go:build linux || darwin

package render

import (
	"fmt"
	"syscall"
	"unsafe"
)

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func termWidthFromFD(fd uintptr) (int, error) {
	ws := &winsize{}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(ws)))
	if errno != 0 {
		return 0, fmt.Errorf("ioctl TIOCGWINSZ: %w", errno)
	}
	if ws.Col == 0 {
		return 0, fmt.Errorf("terminal width unavailable")
	}
	return int(ws.Col), nil
}
