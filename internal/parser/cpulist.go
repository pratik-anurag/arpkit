package parser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pratik-anurag/arpkit/internal/topology"
)

// ParseCPUList parses Linux CPU list syntax, including optional stride ranges (0-10:2).
func ParseCPUList(input string) ([]int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	parts := strings.Split(input, ",")
	values := make([]int, 0, len(parts))
	for _, p := range parts {
		part := strings.TrimSpace(p)
		if part == "" {
			continue
		}
		expanded, err := expandCPUToken(part)
		if err != nil {
			return nil, err
		}
		values = append(values, expanded...)
	}
	return topology.SortedUniqueInts(values), nil
}

func expandCPUToken(token string) ([]int, error) {
	if strings.Contains(token, "-") {
		var stride int
		stride = 1
		rangePart := token
		if strings.Contains(token, ":") {
			chunks := strings.Split(token, ":")
			if len(chunks) != 2 {
				return nil, fmt.Errorf("invalid cpu token %q", token)
			}
			rangePart = chunks[0]
			s, err := strconv.Atoi(chunks[1])
			if err != nil || s <= 0 {
				return nil, fmt.Errorf("invalid cpu stride in %q", token)
			}
			stride = s
		}

		r := strings.Split(rangePart, "-")
		if len(r) != 2 {
			return nil, fmt.Errorf("invalid cpu range %q", token)
		}
		start, err := strconv.Atoi(strings.TrimSpace(r[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid cpu range %q", token)
		}
		end, err := strconv.Atoi(strings.TrimSpace(r[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid cpu range %q", token)
		}
		if end < start {
			return nil, fmt.Errorf("invalid descending range %q", token)
		}
		out := make([]int, 0, (end-start)/stride+1)
		for i := start; i <= end; i += stride {
			out = append(out, i)
		}
		return out, nil
	}

	v, err := strconv.Atoi(token)
	if err != nil {
		return nil, fmt.Errorf("invalid cpu token %q", token)
	}
	return []int{v}, nil
}

func FormatCPUList(values []int) string {
	values = topology.SortedUniqueInts(values)
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return strconv.Itoa(values[0])
	}

	var b strings.Builder
	start := values[0]
	prev := values[0]
	for i := 1; i < len(values); i++ {
		if values[i] == prev+1 {
			prev = values[i]
			continue
		}
		writeRange(&b, start, prev)
		b.WriteByte(',')
		start = values[i]
		prev = values[i]
	}
	writeRange(&b, start, prev)
	return b.String()
}

func writeRange(b *strings.Builder, start, end int) {
	if start == end {
		b.WriteString(strconv.Itoa(start))
		return
	}
	b.WriteString(strconv.Itoa(start))
	b.WriteByte('-')
	b.WriteString(strconv.Itoa(end))
}

func SortCPUList(values []int) []int {
	out := append([]int(nil), values...)
	sort.Ints(out)
	return out
}
