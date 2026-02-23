package render

import "testing"

func TestLayoutForWidth(t *testing.T) {
	if got := layoutForWidth(140); got != layoutSideBySide {
		t.Fatalf("layoutForWidth(140)=%v want=%v", got, layoutSideBySide)
	}
	if got := layoutForWidth(100); got != layoutStacked {
		t.Fatalf("layoutForWidth(100)=%v want=%v", got, layoutStacked)
	}
	if got := layoutForWidth(60); got != layoutCompact {
		t.Fatalf("layoutForWidth(60)=%v want=%v", got, layoutCompact)
	}
}

func TestDrawBoxWidth(t *testing.T) {
	box := drawBox(24, "Title", []string{"line one", "line two"})
	if len(box) != 4 {
		t.Fatalf("box lines=%d want=4", len(box))
	}
	for _, line := range box {
		if got := visibleLen(line); got != 24 {
			t.Fatalf("line width=%d want=24 line=%q", got, line)
		}
	}
}

func TestComposeHorizontalWidth(t *testing.T) {
	left := drawBox(18, "L", []string{"a", "b"})
	right := drawBox(20, "R", []string{"x", "y"})
	rows := composeHorizontal([][]string{left, right}, 2)
	for _, row := range rows {
		if got := visibleLen(row); got != 40 {
			t.Fatalf("row width=%d want=40 row=%q", got, row)
		}
	}
}

func TestPackTokenLinesTruncates(t *testing.T) {
	tokens := []tokenLine{
		{plain: "C0 [0 1]", text: "C0 [0 1]", cores: 1},
		{plain: "C1 [2 3]", text: "C1 [2 3]", cores: 1},
		{plain: "C2 [4 5]", text: "C2 [4 5]", cores: 1},
		{plain: "C3 [6 7]", text: "C3 [6 7]", cores: 1},
	}
	lines, shown := packTokenLines(tokens, 16, 2)
	if len(lines) != 2 {
		t.Fatalf("lines=%d want=2", len(lines))
	}
	if shown >= len(tokens) {
		t.Fatalf("shown=%d expected truncation from %d tokens", shown, len(tokens))
	}
}

func TestCompactTokensCollapseRanges(t *testing.T) {
	cores := []diagramCore{
		{LocalID: 0, Threads: []int{0, 1}},
		{LocalID: 1, Threads: []int{2, 3}},
		{LocalID: 2, Threads: []int{4, 5}},
		{LocalID: 3, Threads: []int{6, 7}},
		{LocalID: 4, Threads: []int{8, 9}},
	}
	tokens := compactTokens(cores, 40, paletteForNode(0), newANSI(false))
	if len(tokens) == 0 {
		t.Fatal("expected collapsed token output")
	}
	if tokens[0].plain != "C0-4 [0-9]" {
		t.Fatalf("token[0]=%q want %q", tokens[0].plain, "C0-4 [0-9]")
	}
}
