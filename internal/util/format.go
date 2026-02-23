package util

import (
	"fmt"
)

func HumanBytes(v uint64) string {
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%d B", v)
	}
	div, exp := uint64(unit), 0
	for n := v / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffixes := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	if exp >= len(suffixes) {
		exp = len(suffixes) - 1
	}
	value := float64(v) / float64(div)
	if value >= 10 {
		return fmt.Sprintf("%.0f %s", value, suffixes[exp])
	}
	return fmt.Sprintf("%.1f %s", value, suffixes[exp])
}

func HumanMHz(v int) string {
	if v <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%d MHz", v)
}
