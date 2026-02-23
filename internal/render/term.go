package render

import (
	"os"
	"strconv"
)

const defaultTermWidth = 80

func DetectWidth(file *os.File) int {
	if w := widthFromEnv(); w > 0 {
		return w
	}
	if file != nil {
		if w, err := termWidthFromFD(file.Fd()); err == nil && w > 0 {
			return w
		}
	}
	return defaultTermWidth
}

func widthFromEnv() int {
	value := os.Getenv("COLUMNS")
	if value == "" {
		return 0
	}
	w, err := strconv.Atoi(value)
	if err != nil || w <= 0 {
		return 0
	}
	return w
}
