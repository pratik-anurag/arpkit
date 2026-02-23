package microarch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProcCPUInfoX86(t *testing.T) {
	cpuinfo := readFixture(t, "cpuinfo_x86.txt")
	got := ParseProcCPUInfo(cpuinfo)
	if got.Vendor != "GenuineIntel" {
		t.Fatalf("vendor=%q want GenuineIntel", got.Vendor)
	}
	if got.ModelName == "" {
		t.Fatal("model name should be parsed")
	}
	if got.Family != 6 || got.Model != 106 || got.Stepping != 6 {
		t.Fatalf("family/model/step=%d/%d/%d want 6/106/6", got.Family, got.Model, got.Stepping)
	}
	for _, feature := range []string{"avx", "avx2", "avx512f", "aes", "bmi1", "bmi2", "fma", "sha_ni"} {
		if !got.Flags[feature] {
			t.Fatalf("expected flag %q to be present", feature)
		}
	}
}

func TestFromLinuxX86FeatureSummaryAndMicroarch(t *testing.T) {
	parsed := ProcCPUInfo{
		Vendor:    "GenuineIntel",
		ModelName: "Intel(R) Xeon(R) Gold 6338 CPU @ 2.00GHz",
		Family:    6,
		Model:     106,
		Stepping:  6,
		Flags: map[string]bool{
			"avx":    true,
			"avx2":   true,
			"aes":    true,
			"bmi1":   true,
			"bmi2":   true,
			"fma":    true,
			"sha_ni": true,
		},
	}

	got := FromLinux("amd64", parsed)
	if got.MicroarchName != "Ice Lake" {
		t.Fatalf("microarch=%q want Ice Lake", got.MicroarchName)
	}
	if !got.ISAFeatures.AVX || !got.ISAFeatures.AVX2 || got.ISAFeatures.AVX512F == true {
		t.Fatalf("unexpected x86 ISA summary: %+v", got.ISAFeatures)
	}
	if !got.ISAFeatures.AES || !got.ISAFeatures.BMI1 || !got.ISAFeatures.BMI2 || !got.ISAFeatures.FMA || !got.ISAFeatures.SHA {
		t.Fatalf("missing expected x86 features: %+v", got.ISAFeatures)
	}
	if !got.AVX512LikelyDisabled {
		t.Fatal("expected AVX-512 likely disabled hint for Ice Lake without avx512f flag")
	}
}

func TestFromLinuxARMFeatureSummary(t *testing.T) {
	parsedFixture := ParseProcCPUInfo(readFixture(t, "cpuinfo_arm64.txt"))
	parsed := ProcCPUInfo{
		Vendor:    parsedFixture.Vendor,
		ModelName: parsedFixture.ModelName,
		Family:    -1,
		Model:     -1,
		Stepping:  -1,
		Flags:     parsedFixture.Flags,
	}

	got := FromLinux("arm64", parsed)
	if got.MicroarchName != "Neoverse N1" {
		t.Fatalf("microarch=%q want Neoverse N1", got.MicroarchName)
	}
	if !got.ISAFeatures.SVE || !got.ISAFeatures.SVE2 || !got.ISAFeatures.AES || !got.ISAFeatures.SHA1 || !got.ISAFeatures.SHA2 || !got.ISAFeatures.CRC32 {
		t.Fatalf("unexpected arm ISA summary: %+v", got.ISAFeatures)
	}
	if !got.ISAFeatures.SHA {
		t.Fatalf("expected SHA aggregate flag for arm features: %+v", got.ISAFeatures)
	}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}
