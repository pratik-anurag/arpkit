package microarch

import (
	"strconv"
	"strings"

	"github.com/pratik-anurag/arpkit/internal/topology"
)

type ProcCPUInfo struct {
	Vendor    string
	ModelName string
	Family    int
	Model     int
	Stepping  int
	Flags     map[string]bool
}

func ParseProcCPUInfo(text string) ProcCPUInfo {
	out := ProcCPUInfo{
		Family:   -1,
		Model:    -1,
		Stepping: -1,
		Flags:    map[string]bool{},
	}
	if strings.TrimSpace(text) == "" {
		return out
	}
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "vendor_id", "CPU implementer":
			if out.Vendor == "" {
				out.Vendor = value
			}
		case "model name", "Hardware", "Processor":
			if out.ModelName == "" {
				out.ModelName = value
			}
		case "cpu family":
			if out.Family < 0 {
				if v, err := strconv.Atoi(value); err == nil {
					out.Family = v
				}
			}
		case "model":
			if out.Model < 0 {
				if v, err := strconv.Atoi(value); err == nil {
					out.Model = v
				}
			}
		case "stepping":
			if out.Stepping < 0 {
				if v, err := strconv.Atoi(value); err == nil {
					out.Stepping = v
				}
			}
		case "flags", "Features":
			for _, f := range strings.Fields(strings.ToLower(value)) {
				out.Flags[f] = true
			}
		}
	}
	return out
}

func FromLinux(arch string, parsed ProcCPUInfo) topology.MicroarchInfo {
	features := summarizeFeatures(arch, parsed.Flags)
	name := DetectMicroarchName(arch, parsed.Vendor, parsed.Family, parsed.Model, parsed.ModelName)
	avx512LikelyDisabled := likelyHasAVX512(name) && !features.AVX512F

	flags := make([]string, 0, len(parsed.Flags))
	for f := range parsed.Flags {
		flags = append(flags, f)
	}

	return topology.MicroarchInfo{
		MicroarchName:        name,
		Vendor:               parsed.Vendor,
		Family:               parsed.Family,
		Model:                parsed.Model,
		Stepping:             parsed.Stepping,
		ISAFeatures:          features,
		AVX512LikelyDisabled: avx512LikelyDisabled,
		RawFlags:             flags,
	}
}

func FromDarwin(arch string, vendor string, modelName string, family int, model int, stepping int, hasFeature func(key string) bool) topology.MicroarchInfo {
	features := topology.FeatureSummary{}
	if hasFeature != nil {
		features = summarizeDarwinFeatures(arch, hasFeature)
	}
	name := DetectMicroarchName(arch, vendor, family, model, modelName)

	return topology.MicroarchInfo{
		MicroarchName:        name,
		Vendor:               vendor,
		Family:               family,
		Model:                model,
		Stepping:             stepping,
		ISAFeatures:          features,
		AVX512LikelyDisabled: likelyHasAVX512(name) && !features.AVX512F,
		RawFlags:             []string{},
	}
}

func summarizeFeatures(arch string, flags map[string]bool) topology.FeatureSummary {
	f := topology.FeatureSummary{}
	for key := range flags {
		k := strings.ToLower(strings.TrimSpace(key))
		switch k {
		case "avx":
			f.AVX = true
		case "avx2":
			f.AVX2 = true
		case "avx512f":
			f.AVX512F = true
		case "aes", "aesni", "pmull":
			f.AES = true
		case "bmi1":
			f.BMI1 = true
		case "bmi2":
			f.BMI2 = true
		case "fma", "fma4":
			f.FMA = true
		case "sha_ni", "sha":
			f.SHA = true
		case "sve":
			f.SVE = true
		case "sve2":
			f.SVE2 = true
		case "sha1":
			f.SHA1 = true
		case "sha2", "sha256", "sha512":
			f.SHA2 = true
		case "crc32":
			f.CRC32 = true
		}
	}

	arch = strings.ToLower(arch)
	if arch == "arm64" || arch == "aarch64" {
		if f.SHA1 || f.SHA2 {
			f.SHA = true
		}
	}
	return f
}

func summarizeDarwinFeatures(arch string, hasFeature func(key string) bool) topology.FeatureSummary {
	f := topology.FeatureSummary{}
	if arch == "amd64" {
		f.AVX = hasFeature("hw.optional.avx1_0")
		f.AVX2 = hasFeature("hw.optional.avx2_0")
		f.AVX512F = hasFeature("hw.optional.avx512f")
		f.AES = hasFeature("hw.optional.aes")
		f.BMI1 = hasFeature("hw.optional.bmi1")
		f.BMI2 = hasFeature("hw.optional.bmi2")
		f.FMA = hasFeature("hw.optional.fma")
		f.SHA = hasFeature("hw.optional.sha1") || hasFeature("hw.optional.sha2")
		return f
	}

	f.AES = anyFeature(hasFeature, "hw.optional.arm.FEAT_AES", "hw.optional.aes")
	f.SVE = anyFeature(hasFeature, "hw.optional.sve", "hw.optional.arm.FEAT_SVE")
	f.SVE2 = anyFeature(hasFeature, "hw.optional.sve2", "hw.optional.arm.FEAT_SVE2")
	f.SHA1 = anyFeature(hasFeature, "hw.optional.arm.FEAT_SHA1", "hw.optional.sha1")
	f.SHA2 = anyFeature(hasFeature, "hw.optional.arm.FEAT_SHA256", "hw.optional.sha2")
	f.CRC32 = anyFeature(hasFeature, "hw.optional.armv8_crc32", "hw.optional.arm.FEAT_CRC32")
	f.SHA = f.SHA1 || f.SHA2
	return f
}

func anyFeature(hasFeature func(key string) bool, keys ...string) bool {
	for _, key := range keys {
		if hasFeature(key) {
			return true
		}
	}
	return false
}

func DetectMicroarchName(arch string, vendor string, family int, model int, modelName string) string {
	lArch := strings.ToLower(arch)
	lVendor := strings.ToLower(vendor)
	lModel := strings.ToLower(modelName)

	if strings.Contains(lModel, "graviton3") {
		return "Graviton3"
	}
	if strings.Contains(lModel, "graviton2") {
		return "Graviton2"
	}
	if strings.Contains(lModel, "neoverse") && strings.Contains(lModel, "n1") {
		return "Neoverse N1"
	}
	if strings.Contains(lModel, "apple m1") {
		return "Apple M1"
	}
	if strings.Contains(lModel, "apple m2") {
		return "Apple M2"
	}
	if strings.Contains(lModel, "apple m3") {
		return "Apple M3"
	}
	if strings.Contains(lModel, "apple m4") {
		return "Apple M4"
	}

	if lArch == "amd64" || lArch == "x86_64" {
		if strings.Contains(lVendor, "intel") {
			if family == 6 {
				switch model {
				case 0x4e, 0x5e, 0x8e, 0x9e:
					return "Skylake"
				case 0x55:
					return "Skylake-SP/Cascade Lake"
				case 0x6a, 0x6c:
					return "Ice Lake"
				case 0x8f:
					return "Sapphire Rapids"
				case 0x97, 0x9a:
					return "Alder Lake"
				case 0xa7, 0xbf:
					return "Raptor Lake"
				}
			}
			if strings.Contains(lModel, "ice lake") {
				return "Ice Lake"
			}
			if strings.Contains(lModel, "sapphire") {
				return "Sapphire Rapids"
			}
		}
		if strings.Contains(lVendor, "amd") {
			switch family {
			case 23:
				if model >= 0x30 {
					return "Zen 2"
				}
				return "Zen/Zen+"
			case 25:
				if model >= 0x60 {
					return "Zen 4"
				}
				return "Zen 3"
			case 26:
				return "Zen 5"
			}
			if strings.Contains(lModel, "epyc") && strings.Contains(lModel, "milan") {
				return "Zen 3"
			}
		}
	}

	if lArch == "arm64" || lArch == "aarch64" {
		if strings.Contains(lModel, "neoverse n1") {
			return "Neoverse N1"
		}
		if strings.Contains(lModel, "neoverse v1") {
			return "Neoverse V1"
		}
		if strings.Contains(lModel, "graviton") {
			return "Graviton"
		}
	}

	return "unknown"
}

func likelyHasAVX512(name string) bool {
	l := strings.ToLower(name)
	if strings.Contains(l, "sapphire") || strings.Contains(l, "ice lake") {
		return true
	}
	if strings.Contains(l, "skylake-sp") || strings.Contains(l, "cascade") {
		return true
	}
	if strings.Contains(l, "zen 4") || strings.Contains(l, "zen 5") {
		return true
	}
	return false
}

func FeatureList(arch string, f topology.FeatureSummary) []string {
	arch = strings.ToLower(arch)
	out := make([]string, 0, 8)
	if arch == "amd64" || arch == "x86_64" {
		if f.AVX {
			out = append(out, "avx")
		}
		if f.AVX2 {
			out = append(out, "avx2")
		}
		if f.AVX512F {
			out = append(out, "avx512f")
		}
		if f.AES {
			out = append(out, "aes")
		}
		if f.BMI1 {
			out = append(out, "bmi1")
		}
		if f.BMI2 {
			out = append(out, "bmi2")
		}
		if f.FMA {
			out = append(out, "fma")
		}
		if f.SHA {
			out = append(out, "sha")
		}
		return out
	}
	if f.SVE {
		out = append(out, "sve")
	}
	if f.SVE2 {
		out = append(out, "sve2")
	}
	if f.AES {
		out = append(out, "aes")
	}
	if f.SHA1 {
		out = append(out, "sha1")
	}
	if f.SHA2 {
		out = append(out, "sha2")
	}
	if f.CRC32 {
		out = append(out, "crc32")
	}
	return out
}
