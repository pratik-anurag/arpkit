package util

import (
	"os"
)

func IsTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		return false
	}
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}
