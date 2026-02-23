package render

import (
	"regexp"
	"strings"

	"github.com/pratik-anurag/arpkit/internal/topology"
)

type SystemInfo struct {
	OSName        string
	OSFamily      string
	Kernel        string
	Arch          string
	CPUVendor     string
	MicroarchName string
}

type normalizedSystemInfo struct {
	OSName        string
	OSShort       string
	OSFamily      string
	Distro        string
	Kernel        string
	Arch          string
	Vendor        string
	MicroarchName string
}

type pillTokens struct {
	os        string
	kernel    string
	arch      string
	vendor    string
	microarch string
}

var macOSVersionRE = regexp.MustCompile(`(?i)macos\s+([0-9]+(?:\.[0-9]+)*)`)

func BuildPill(info SystemInfo, termWidth int, color bool) string {
	return BuildPillWithTheme(info, termWidth, color, "auto")
}

func BuildPillWithTheme(info SystemInfo, termWidth int, color bool, colorTheme string) string {
	n := normalizeSystemInfo(info)
	if termWidth <= 0 {
		termWidth = defaultTermWidth
	}

	t := choosePillTokens(n, termWidth)
	if color {
		return renderPillColored(t, n, colorTheme)
	}
	return renderPillPlain(t)
}

func systemInfoFromProfile(m *topology.MachineProfile) SystemInfo {
	if m == nil {
		return SystemInfo{}
	}
	vendor := strings.TrimSpace(m.CPU.Vendor)
	if vendor == "" {
		vendor = strings.TrimSpace(m.Microarch.Vendor)
	}
	return SystemInfo{
		OSName:        m.Metadata.OS,
		OSFamily:      inferOSFamily(m.Metadata.OS),
		Kernel:        m.Metadata.Kernel,
		Arch:          firstNonEmpty(m.CPU.Architecture, m.Metadata.Arch),
		CPUVendor:     vendor,
		MicroarchName: valueOrUnknown(strings.TrimSpace(m.Microarch.MicroarchName)),
	}
}

func normalizeSystemInfo(info SystemInfo) normalizedSystemInfo {
	osName := strings.TrimSpace(info.OSName)
	osFamily := strings.ToLower(strings.TrimSpace(info.OSFamily))
	if osFamily == "" {
		osFamily = inferOSFamily(osName)
	}

	kernel := strings.TrimSpace(info.Kernel)
	if osFamily == "darwin" {
		if v := extractMacOSVersion(osName); v != "" {
			kernel = v
		}
	}

	longOS := osName
	if longOS == "" {
		if osFamily == "darwin" {
			longOS = "macOS"
		} else {
			longOS = "Linux"
		}
	}

	return normalizedSystemInfo{
		OSName:        longOS,
		OSShort:       shortenOSToken(longOS),
		OSFamily:      osFamily,
		Distro:        detectDistro(longOS),
		Kernel:        valueOrUnknown(kernel),
		Arch:          normalizeArch(info.Arch),
		Vendor:        canonicalVendor(info.CPUVendor),
		MicroarchName: valueOrUnknown(strings.TrimSpace(info.MicroarchName)),
	}
}

func choosePillTokens(info normalizedSystemInfo, width int) pillTokens {
	base := pillTokens{
		os:        info.OSName,
		kernel:    info.Kernel,
		arch:      info.Arch,
		vendor:    prettyVendor(info.Vendor),
		microarch: info.MicroarchName,
	}
	if base.vendor == "" {
		base.vendor = "Unknown"
	}

	if visibleLen(renderPillPlain(base)) <= width {
		return base
	}

	noKernel := base
	noKernel.kernel = ""
	if visibleLen(renderPillPlain(noKernel)) <= width {
		return noKernel
	}

	shortOS := noKernel
	shortOS.os = info.OSShort
	if visibleLen(renderPillPlain(shortOS)) <= width {
		return shortOS
	}

	noMicro := shortOS
	noMicro.microarch = ""
	if visibleLen(renderPillPlain(noMicro)) <= width {
		return noMicro
	}

	fallback := pillTokens{
		os:     info.OSShort,
		arch:   info.Arch,
		vendor: prettyVendor(info.Vendor),
	}
	if fallback.vendor == "" {
		fallback.vendor = "Unknown"
	}
	if visibleLen(renderPillPlain(fallback)) <= width {
		return fallback
	}

	plain := renderPillPlain(fallback)
	if visibleLen(plain) <= width {
		return fallback
	}
	available := width - visibleLen("[  |  |  ]") - visibleLen(fallback.arch) - visibleLen(fallback.vendor)
	if available < 3 {
		fallback.os = truncatePlain(fallback.os, 3)
		return fallback
	}
	fallback.os = truncatePlain(fallback.os, available)
	return fallback
}

func renderPillPlain(tokens pillTokens) string {
	values := make([]string, 0, 5)
	if tokens.os != "" {
		values = append(values, tokens.os)
	}
	if tokens.kernel != "" {
		values = append(values, tokens.kernel)
	}
	if tokens.arch != "" {
		values = append(values, tokens.arch)
	}
	if tokens.vendor != "" {
		values = append(values, tokens.vendor)
	}
	if tokens.microarch != "" {
		values = append(values, tokens.microarch)
	}
	return "[ " + strings.Join(values, " | ") + " ]"
}

func renderPillColored(tokens pillTokens, info normalizedSystemInfo, colorTheme string) string {
	ansi := newANSI(true)
	accent := distroAccent(info, colorTheme)
	vendorColor := vendorAccent(info.Vendor)
	microColor := 111

	values := make([]string, 0, 5)
	if tokens.os != "" {
		values = append(values, ansi.title(accent, tokens.os))
	}
	if tokens.kernel != "" {
		values = append(values, tokens.kernel)
	}
	if tokens.arch != "" {
		values = append(values, tokens.arch)
	}
	if tokens.vendor != "" {
		values = append(values, ansi.title(vendorColor, tokens.vendor))
	}
	if tokens.microarch != "" {
		values = append(values, ansi.title(microColor, tokens.microarch))
	}

	sep := ansi.dim(" | ")
	return ansi.dim("[") + " " + strings.Join(values, sep) + " " + ansi.dim("]")
}

func distroAccent(info normalizedSystemInfo, colorTheme string) int {
	if strings.EqualFold(colorTheme, "distro") && info.Distro == "" && info.OSFamily == "linux" {
		return 39
	}
	switch info.Distro {
	case "ubuntu":
		return 208
	case "arch":
		return 39
	case "debian":
		return 161
	case "fedora":
		return 33
	case "rhel":
		return 160
	case "alpine":
		return 45
	case "macos":
		return 117
	default:
		if info.OSFamily == "darwin" {
			return 117
		}
		return 76
	}
}

func vendorAccent(vendor string) int {
	switch vendor {
	case "intel":
		return 39
	case "amd":
		return 197
	case "apple":
		return 111
	case "arm":
		return 45
	default:
		return 252
	}
}

func detectDistro(osName string) string {
	l := strings.ToLower(strings.TrimSpace(osName))
	switch {
	case strings.Contains(l, "ubuntu"):
		return "ubuntu"
	case strings.Contains(l, "arch"):
		return "arch"
	case strings.Contains(l, "debian"):
		return "debian"
	case strings.Contains(l, "fedora"):
		return "fedora"
	case strings.Contains(l, "rhel"),
		strings.Contains(l, "red hat"),
		strings.Contains(l, "centos"),
		strings.Contains(l, "rocky"),
		strings.Contains(l, "alma"):
		return "rhel"
	case strings.Contains(l, "alpine"):
		return "alpine"
	case strings.Contains(l, "macos"), strings.Contains(l, "darwin"):
		return "macos"
	default:
		return ""
	}
}

func shortenOSToken(osName string) string {
	trimmed := strings.TrimSpace(osName)
	if trimmed == "" {
		return "Linux"
	}
	l := strings.ToLower(trimmed)
	switch {
	case strings.Contains(l, "ubuntu"):
		return "Ubuntu"
	case strings.Contains(l, "arch"):
		return "Arch Linux"
	case strings.Contains(l, "debian"):
		return "Debian"
	case strings.Contains(l, "fedora"):
		return "Fedora"
	case strings.Contains(l, "rhel"), strings.Contains(l, "red hat"):
		return "RHEL"
	case strings.Contains(l, "centos"):
		return "CentOS"
	case strings.Contains(l, "alpine"):
		return "Alpine"
	case strings.Contains(l, "macos"):
		return "macOS"
	default:
		fields := strings.Fields(trimmed)
		if len(fields) > 0 {
			return fields[0]
		}
		return trimmed
	}
}

func extractMacOSVersion(osName string) string {
	match := macOSVersionRE.FindStringSubmatch(strings.TrimSpace(osName))
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func inferOSFamily(osName string) string {
	l := strings.ToLower(strings.TrimSpace(osName))
	switch {
	case strings.Contains(l, "mac"), strings.Contains(l, "darwin"):
		return "darwin"
	case l == "":
		return "linux"
	default:
		return "linux"
	}
}

func normalizeArch(arch string) string {
	a := strings.ToLower(strings.TrimSpace(arch))
	switch a {
	case "x86_64", "x64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		if a == "" {
			return "unknown"
		}
		return a
	}
}

func canonicalVendor(vendor string) string {
	v := strings.ToLower(strings.TrimSpace(vendor))
	switch {
	case strings.Contains(v, "intel"):
		return "intel"
	case strings.Contains(v, "amd"):
		return "amd"
	case strings.Contains(v, "apple"):
		return "apple"
	case strings.Contains(v, "arm"), strings.Contains(v, "0x41"):
		return "arm"
	default:
		return "unknown"
	}
}

func prettyVendor(vendor string) string {
	switch vendor {
	case "intel":
		return "Intel"
	case "amd":
		return "AMD"
	case "apple":
		return "Apple"
	case "arm":
		return "ARM"
	default:
		return ""
	}
}

func valueOrUnknown(v string) string {
	if strings.TrimSpace(v) == "" {
		return "unknown"
	}
	return strings.TrimSpace(v)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
