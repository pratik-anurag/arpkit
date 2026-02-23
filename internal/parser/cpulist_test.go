package parser

import (
	"reflect"
	"testing"
)

func TestParseCPUList(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
	}{
		{name: "ranges", input: "0-3,8,10-11", want: []int{0, 1, 2, 3, 8, 10, 11}},
		{name: "stride", input: "0-10:2", want: []int{0, 2, 4, 6, 8, 10}},
		{name: "dedupe", input: "1,3,2,3,1", want: []int{1, 2, 3}},
		{name: "invalid descending", input: "3-1", wantErr: true},
		{name: "invalid token", input: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCPUList(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCPUList(%q) error: %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseCPUList(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatCPUList(t *testing.T) {
	input := []int{9, 3, 2, 1, 8, 5, 4}
	got := FormatCPUList(input)
	want := "1-5,8-9"
	if got != want {
		t.Fatalf("FormatCPUList() = %q, want %q", got, want)
	}
}
