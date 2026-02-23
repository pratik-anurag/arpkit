package render

import "strings"

func drawBox(width int, title string, body []string) []string {
	if width < 4 {
		width = 4
	}
	contentWidth := width - 2
	lines := make([]string, 0, len(body)+2)
	lines = append(lines, boxTop(contentWidth, title))
	if len(body) == 0 {
		lines = append(lines, "│"+strings.Repeat(" ", contentWidth)+"│")
	} else {
		for _, line := range body {
			fitted := line
			if visibleLen(fitted) > contentWidth {
				fitted = truncatePlain(stripANSI(fitted), contentWidth)
			}
			lines = append(lines, "│"+fitted+strings.Repeat(" ", contentWidth-visibleLen(fitted))+"│")
		}
	}
	lines = append(lines, "└"+strings.Repeat("─", contentWidth)+"┘")
	return lines
}

func boxTop(contentWidth int, title string) string {
	if title == "" {
		return "┌" + strings.Repeat("─", contentWidth) + "┐"
	}
	label := " " + truncatePlain(title, max(contentWidth-2, 1)) + " "
	if len([]rune(label)) > contentWidth {
		label = truncatePlain(label, contentWidth)
	}
	left := 1
	if contentWidth-len([]rune(label))-left < 0 {
		left = 0
	}
	right := contentWidth - len([]rune(label)) - left
	if right < 0 {
		right = 0
	}
	return "┌" + strings.Repeat("─", left) + label + strings.Repeat("─", right) + "┐"
}

func composeHorizontal(blocks [][]string, gap int) []string {
	if len(blocks) == 0 {
		return nil
	}
	if gap < 0 {
		gap = 0
	}
	maxHeight := 0
	widths := make([]int, len(blocks))
	for i, block := range blocks {
		if len(block) > maxHeight {
			maxHeight = len(block)
		}
		if len(block) > 0 {
			widths[i] = visibleLen(block[0])
		}
	}

	out := make([]string, 0, maxHeight)
	gapText := strings.Repeat(" ", gap)
	for row := 0; row < maxHeight; row++ {
		parts := make([]string, 0, len(blocks))
		for i, block := range blocks {
			if row < len(block) {
				parts = append(parts, padRight(block[row], widths[i]))
			} else {
				parts = append(parts, strings.Repeat(" ", widths[i]))
			}
		}
		out = append(out, strings.Join(parts, gapText))
	}
	return out
}

func colorizeBorders(lines []string, apply func(string) string) []string {
	if apply == nil {
		return lines
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		var b strings.Builder
		for _, r := range line {
			if isBorderRune(r) {
				b.WriteString(apply(string(r)))
			} else {
				b.WriteRune(r)
			}
		}
		out[i] = b.String()
	}
	return out
}

func isBorderRune(r rune) bool {
	switch r {
	case '┌', '┐', '└', '┘', '│', '─', '├', '┤', '┬', '┴', '┼':
		return true
	default:
		return false
	}
}
