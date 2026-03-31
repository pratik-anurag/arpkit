//go:build linux

package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectLinuxMarksPartialOnOversizedOptionalFile(t *testing.T) {
	root := t.TempDir()

	writeFile := func(rel string, value string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error: %v", rel, err)
		}
	}

	writeFile("etc/os-release", `PRETTY_NAME="Fixture Linux"`+"\n")
	writeFile("proc/cpuinfo", strings.Repeat("processor\t: 0\n", 16))
	writeFile("proc/meminfo", "MemTotal:       1024 kB\n")
	writeFile("sys/devices/system/cpu/online", "0\n")
	writeFile("sys/devices/system/cpu/cpu0/topology/physical_package_id", "0\n")
	writeFile("sys/devices/system/cpu/cpu0/topology/core_id", "0\n")

	profile, err := Collect(Options{
		Root:         root,
		MaxFileBytes: 32,
	})
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}
	if !profile.Partial {
		t.Fatal("Collect() partial = false, want true")
	}
	if len(profile.Warnings) != 1 {
		t.Fatalf("Collect() warnings = %v, want 1 warning", profile.Warnings)
	}
	if !strings.Contains(profile.Warnings[0], "/proc/cpuinfo") {
		t.Fatalf("warning = %q, want /proc/cpuinfo", profile.Warnings[0])
	}
	if len(profile.Threads) != 1 {
		t.Fatalf("threads = %d, want 1", len(profile.Threads))
	}
}
