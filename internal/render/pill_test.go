package render_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pratik-anurag/arpkit/internal/render"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

func TestPillGoldenBySystemAndWidth(t *testing.T) {
	type scenario struct {
		name     string
		osName   string
		osFamily string
		kernel   string
		arch     string
		vendor   string
		uarch    string
	}
	scenarios := []scenario{
		{
			name:     "ubuntu_amd64_intel_skylake",
			osName:   "Ubuntu 24.04",
			osFamily: "linux",
			kernel:   "6.8.0-31-generic",
			arch:     "amd64",
			vendor:   "GenuineIntel",
			uarch:    "Skylake",
		},
		{
			name:     "arch_amd64_amd_zen3",
			osName:   "Arch Linux",
			osFamily: "linux",
			kernel:   "6.7.9-arch1-1",
			arch:     "amd64",
			vendor:   "AuthenticAMD",
			uarch:    "Zen 3",
		},
		{
			name:     "linux_arm64_arm_unknown",
			osName:   "Linux",
			osFamily: "linux",
			kernel:   "6.6.15",
			arch:     "arm64",
			vendor:   "ARM",
			uarch:    "unknown",
		},
		{
			name:     "darwin_arm64_apple_m2",
			osName:   "macOS 14.3",
			osFamily: "darwin",
			kernel:   "23.3.0",
			arch:     "arm64",
			vendor:   "Apple",
			uarch:    "M2",
		},
	}
	widths := []int{60, 80, 140}

	for _, sc := range scenarios {
		for _, width := range widths {
			t.Run(sc.name+"_w"+itoa(width), func(t *testing.T) {
				profile := pillFixture(sc)
				out, err := render.RenderPretty(profile, render.Options{
					ColorMode: "never",
					Profile:   "min",
					Width:     width,
					Version:   "test",
				})
				if err != nil {
					t.Fatalf("RenderPretty() error: %v", err)
				}
				golden := "pill_" + sc.name + "_w" + itoa(width) + ".golden"
				assertGolden(t, golden, out)
			})
		}
	}
}

func TestPillTruncationOrder(t *testing.T) {
	info := render.SystemInfo{
		OSName:        "Ubuntu 24.04",
		OSFamily:      "linux",
		Kernel:        "6.8.0-31-generic",
		Arch:          "amd64",
		CPUVendor:     "GenuineIntel",
		MicroarchName: "Skylake",
	}

	full := render.BuildPill(info, 200, false)
	if !strings.Contains(full, "6.8.0-31-generic") || !strings.Contains(full, "Skylake") {
		t.Fatalf("expected full pill tokens, got: %s", full)
	}

	noKernel := render.BuildPill(info, 56, false)
	if strings.Contains(noKernel, "6.8.0-31-generic") {
		t.Fatalf("kernel should be dropped first: %s", noKernel)
	}

	shortOS := render.BuildPill(info, 40, false)
	if strings.Contains(shortOS, "Ubuntu 24.04") {
		t.Fatalf("long OS token should be shortened: %s", shortOS)
	}

	noMicro := render.BuildPill(info, 32, false)
	if strings.Contains(noMicro, "Skylake") {
		t.Fatalf("microarch should be dropped at narrow width: %s", noMicro)
	}

	fallback := render.BuildPill(info, 30, false)
	if !strings.Contains(fallback, "Ubuntu") || !strings.Contains(fallback, "amd64") || !strings.Contains(fallback, "Intel") {
		t.Fatalf("expected fallback os/arch/vendor form, got: %s", fallback)
	}
}

func TestMeasureVisibleWidthANSI(t *testing.T) {
	input := "\x1b[1;31m[ Ubuntu | amd64 | Intel ]\x1b[0m"
	got := render.MeasureVisibleWidth(input)
	want := len("[ Ubuntu | amd64 | Intel ]")
	if got != want {
		t.Fatalf("MeasureVisibleWidth()=%d want=%d", got, want)
	}
}

func pillFixture(sc struct {
	name     string
	osName   string
	osFamily string
	kernel   string
	arch     string
	vendor   string
	uarch    string
}) *topology.MachineProfile {
	return &topology.MachineProfile{
		Metadata: topology.Metadata{
			Hostname: "pill-host",
			OS:       sc.osName,
			Kernel:   sc.kernel,
			Arch:     sc.arch,
		},
		CPU: topology.CPUInfo{
			Architecture:   sc.arch,
			Vendor:         sc.vendor,
			ModelName:      "Pill CPU",
			Sockets:        1,
			NUMANodes:      1,
			Cores:          4,
			Threads:        8,
			ThreadsPerCore: 2,
			SMT:            true,
		},
		Microarch: topology.MicroarchInfo{
			MicroarchName: sc.uarch,
			Vendor:        sc.vendor,
		},
	}
}

func assertGolden(t *testing.T, goldenName string, got string) {
	t.Helper()
	goldenPath := filepath.Join("testdata", goldenName)
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.Getenv("UPDATE_GOLDEN") == "1" {
			if writeErr := os.WriteFile(goldenPath, []byte(got), 0o644); writeErr != nil {
				t.Fatalf("write golden: %v", writeErr)
			}
			return
		}
		t.Fatalf("read golden: %v", err)
	}
	if got != string(want) {
		if os.Getenv("UPDATE_GOLDEN") == "1" {
			if writeErr := os.WriteFile(goldenPath, []byte(got), 0o644); writeErr != nil {
				t.Fatalf("update golden: %v", writeErr)
			}
			return
		}
		t.Fatalf("output mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", goldenName, got, string(want))
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 8)
	for v > 0 {
		buf = append(buf, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
