package render

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type ansiStyle struct {
	enabled bool
}

type nodePalette struct {
	Border int
	Title  int
	Socket int
	CPU    int
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func newANSI(enabled bool) ansiStyle {
	return ansiStyle{enabled: enabled}
}

func (a ansiStyle) wrap(code string, s string) string {
	if !a.enabled || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (a ansiStyle) key(s string) string {
	return a.wrap("1;37", s)
}

func (a ansiStyle) dim(s string) string {
	return a.wrap("2;37", s)
}

func (a ansiStyle) good(s string) string {
	return a.wrap("1;32", s)
}

func (a ansiStyle) warn(s string) string {
	return a.wrap("1;33", s)
}

func (a ansiStyle) danger(s string) string {
	return a.wrap("1;31", s)
}

func (a ansiStyle) border(color int, s string) string {
	return a.wrap(fmt.Sprintf("2;38;5;%d", color), s)
}

func (a ansiStyle) title(color int, s string) string {
	return a.wrap(fmt.Sprintf("1;38;5;%d", color), s)
}

func (a ansiStyle) socket(color int, s string) string {
	return a.wrap(fmt.Sprintf("38;5;%d", color), s)
}

func (a ansiStyle) cpu(color int, s string) string {
	return a.wrap(fmt.Sprintf("1;38;5;%d", color), s)
}

func paletteForNode(nodeID int) nodePalette {
	palettes := []nodePalette{
		{Border: 33, Title: 39, Socket: 32, CPU: 81},
		{Border: 35, Title: 42, Socket: 36, CPU: 118},
		{Border: 166, Title: 208, Socket: 214, CPU: 220},
		{Border: 127, Title: 171, Socket: 177, CPU: 219},
		{Border: 31, Title: 45, Socket: 37, CPU: 51},
		{Border: 88, Title: 203, Socket: 167, CPU: 210},
	}
	if nodeID < 0 {
		return nodePalette{Border: 244, Title: 250, Socket: 245, CPU: 252}
	}
	return palettes[nodeID%len(palettes)]
}

func stripANSI(s string) string {
	if s == "" {
		return ""
	}
	return ansiPattern.ReplaceAllString(s, "")
}

func visibleLen(s string) int {
	return utf8.RuneCountInString(stripANSI(s))
}

func MeasureVisibleWidth(s string) int {
	return visibleLen(s)
}

func padRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if visibleLen(s) >= width {
		return truncatePlain(stripANSI(s), width)
	}
	return s + strings.Repeat(" ", width-visibleLen(s))
}

func truncatePlain(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

func clampLine(s string, width int) string {
	plain := stripANSI(s)
	if utf8.RuneCountInString(plain) <= width {
		return s
	}
	return truncatePlain(plain, width)
}
