package parser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReaderReadFileRejectsOversizedFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "proc", "cpuinfo")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(path, []byte("0123456789abcdef"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	reader := NewReader(root)
	reader.MaxFileBytes = 8

	for i := 0; i < 2; i++ {
		_, err := reader.ReadFile("/proc/cpuinfo")
		if err == nil {
			t.Fatal("ReadFile() error = nil, want oversized error")
		}
		var tooLarge *ErrFileTooLarge
		if !errors.As(err, &tooLarge) {
			t.Fatalf("ReadFile() error = %T, want *ErrFileTooLarge", err)
		}
		if tooLarge.Path != "/proc/cpuinfo" {
			t.Fatalf("tooLarge.Path = %q, want /proc/cpuinfo", tooLarge.Path)
		}
		if tooLarge.Limit != 8 {
			t.Fatalf("tooLarge.Limit = %d, want 8", tooLarge.Limit)
		}
	}

	issues := reader.Issues()
	if len(issues) != 1 {
		t.Fatalf("Issues() len = %d, want 1", len(issues))
	}
	if issues[0].Path != "/proc/cpuinfo" {
		t.Fatalf("Issues()[0].Path = %q, want /proc/cpuinfo", issues[0].Path)
	}
	if issues[0].Limit != 8 {
		t.Fatalf("Issues()[0].Limit = %d, want 8", issues[0].Limit)
	}
}
