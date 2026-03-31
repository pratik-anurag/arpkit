package render

import (
	"fmt"
	"strings"
	"unicode"
)

func sanitizeText(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			switch {
			case r < 0x20 || r == 0x7f:
				fmt.Fprintf(&b, "\\x%02x", r)
			case unicode.IsControl(r):
				fmt.Fprintf(&b, "\\u%04x", r)
			default:
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
