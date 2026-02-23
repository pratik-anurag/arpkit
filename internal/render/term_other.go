//go:build !linux && !darwin

package render

import "fmt"

func termWidthFromFD(fd uintptr) (int, error) {
	return 0, fmt.Errorf("terminal width detection unsupported")
}
