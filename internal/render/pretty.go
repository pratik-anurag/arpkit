package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pratik-anurag/arpkit/internal/microarch"
	"github.com/pratik-anurag/arpkit/internal/topology"
	"github.com/pratik-anurag/arpkit/internal/util"
)

type prettySection struct {
	Title string
	Lines []string
}

func RenderPretty(m *topology.MachineProfile, opts Options) (string, error) {
	if m == nil {
		return "", fmt.Errorf("nil machine profile")
	}

	width := opts.Width
	if width <= 0 {
		width = defaultTermWidth
	}
	if width < 52 {
		width = 52
	}

	ansi := newANSI(opts.ColorEnabled())
	out := make([]string, 0, 256)

	if !opts.NoPill {
		pillColor := ansi.enabled && !strings.EqualFold(opts.ColorTheme, "mono")
		pill := BuildPillWithTheme(systemInfoFromProfile(m), width, pillColor, opts.ColorTheme)
		out = append(out, clampLine(pill, width))
		if !opts.Compact {
			out = append(out, "")
		}
	}

	sections := make([]prettySection, 0, 10)
	if showSummarySection(opts) {
		sections = append(sections, prettySection{
			Title: "Summary",
			Lines: renderSummarySection(m, opts, ansi),
		})
	}
	if opts.sectionEnabled("topology") {
		sections = append(sections, prettySection{
			Title: "Topology",
			Lines: renderTopologySection(m, opts, width, ansi),
		})
	}
	if showCacheSection(opts) {
		sections = append(sections, prettySection{
			Title: "Cache",
			Lines: renderCacheSection(m, opts, ansi),
		})
	}
	if showMemorySection(opts) {
		sections = append(sections, prettySection{
			Title: "Memory",
			Lines: renderMemorySection(m, opts, ansi),
		})
	}
	if opts.sectionEnabled("power") {
		sections = append(sections, prettySection{
			Title: "Power",
			Lines: renderPowerSection(m, opts, ansi),
		})
	}
	if shouldShowIsolationSection(m, opts) {
		sections = append(sections, prettySection{
			Title: "Isolation",
			Lines: renderIsolationSection(m, opts, ansi),
		})
	}
	if shouldShowPCIeSection(opts) {
		sections = append(sections, prettySection{
			Title: "PCIe NUMA Affinity",
			Lines: renderPCIeSection(m, ansi),
		})
	}
	if opts.sectionEnabled("posture") {
		sections = append(sections, prettySection{
			Title: "Architecture Posture",
			Lines: renderPostureSection(m.Posture, opts, ansi),
		})
	}
	if opts.Debug && len(m.Warnings) > 0 {
		sections = append(sections, prettySection{
			Title: "Debug",
			Lines: renderDebugSection(m.Warnings),
		})
	}

	for i, section := range sections {
		block := renderSectionBlock(section, width, opts.Unicode, ansi)
		for _, line := range block {
			out = append(out, clampLine(line, width))
		}
		if !opts.Compact && i != len(sections)-1 {
			out = append(out, "")
		}
	}

	if len(out) == 0 {
		return "", nil
	}
	return strings.Join(out, "\n") + "\n", nil
}

func showSummarySection(opts Options) bool {
	return opts.sectionEnabled("summary") || opts.sectionEnabled("freq") || opts.sectionEnabled("microarch")
}

func showCacheSection(opts Options) bool {
	return opts.sectionEnabled("cache") || opts.sectionEnabled("llc")
}

func showMemorySection(opts Options) bool {
	return opts.sectionEnabled("mem") || opts.sectionEnabled("memtop")
}

func shouldShowIsolationSection(m *topology.MachineProfile, opts Options) bool {
	if opts.HasOnly() {
		return opts.sectionEnabled("isolation") || opts.sectionEnabled("notes")
	}
	if opts.sectionEnabled("isolation") || opts.Profile == "verbose" {
		return true
	}
	return len(m.Isolation.Isolated) > 0 || len(m.Isolation.NoHZFull) > 0 || len(m.Isolation.RCUNOCBS) > 0
}

func shouldShowPCIeSection(opts Options) bool {
	if opts.HasOnly() {
		return opts.sectionEnabled("pcie")
	}
	return opts.sectionEnabled("pcie") || opts.Profile == "verbose"
}

func renderSectionBlock(section prettySection, width int, unicode bool, ansi ansiStyle) []string {
	if len(section.Lines) == 0 {
		return nil
	}
	header := section.Title

	underlineChar := "-"
	if unicode {
		underlineChar = "─"
	}
	underlineLen := minInt(width, maxInt(12, MeasureVisibleWidth(header)+4))
	underline := strings.Repeat(underlineChar, underlineLen)

	if ansi.enabled {
		header = ansi.key(header)
		underline = ansi.dim(underline)
	}

	out := make([]string, 0, len(section.Lines)+2)
	out = append(out, header, underline)
	out = append(out, section.Lines...)
	return out
}

func renderSummarySection(m *topology.MachineProfile, opts Options, ansi ansiStyle) []string {
	lines := make([]string, 0, 8)
	lines = append(lines, kv("CPU", valueOrNA(m.CPU.ModelName), opts.Compact, ansi))

	numaCount := "N/A"
	if m.CPU.NUMANodes > 0 {
		numaCount = fmt.Sprintf("%d", m.CPU.NUMANodes)
	}
	topologyText := fmt.Sprintf("Sockets: %d  NUMA: %s  Cores: %d  Threads: %d  SMT: %s", m.CPU.Sockets, numaCount, m.CPU.Cores, m.CPU.Threads, onOff(m.CPU.SMT))
	lines = append(lines, kv("Topology", topologyText, opts.Compact, ansi))

	if opts.sectionEnabled("freq") {
		freqText := fmt.Sprintf("cur=%s  min=%s  max=%s", util.HumanMHz(m.CPU.Frequency.CurrentMHz), util.HumanMHz(m.CPU.Frequency.MinMHz), util.HumanMHz(m.CPU.Frequency.MaxMHz))
		lines = append(lines, kv("Freq", freqText, opts.Compact, ansi))
	}

	if opts.sectionEnabled("microarch") {
		micro := valueOrNA(m.Microarch.MicroarchName)
		lines = append(lines, kv("uArch", micro, opts.Compact, ansi))
		features := microarch.FeatureList(m.CPU.Architecture, m.Microarch.ISAFeatures)
		featureText := "none"
		if len(features) > 0 {
			featureText = strings.Join(features, ",")
		}
		lines = append(lines, kv("ISA", featureText, opts.Compact, ansi))
	}
	return lines
}

func renderTopologySection(m *topology.MachineProfile, opts Options, width int, ansi ansiStyle) []string {
	lines := make([]string, 0, 32)
	if opts.NoDiagram {
		lines = append(lines, "diagram disabled (--no-diagram)")
	} else {
		diagram := RenderDiagram(m, DiagramConfig{Width: width, Wide: opts.Wide, ColorEnabled: ansi.enabled})
		lines = append(lines, strings.Split(diagram, "\n")...)
	}

	showDistance := opts.sectionEnabled("distance") || (!opts.HasOnly() && opts.Profile == "verbose")
	if !showDistance {
		return lines
	}

	if !opts.Compact {
		lines = append(lines, "")
	}
	lines = append(lines, kv("NUMA Distance", "", opts.Compact, ansi))
	lines = append(lines, renderDistanceLines(m)...)
	return lines
}

func renderDistanceLines(m *topology.MachineProfile) []string {
	if len(m.NumaDistance.NodeIDs) == 0 || len(m.NumaDistance.Matrix) == 0 {
		return []string{"  N/A"}
	}
	body := make([]string, 0, len(m.NumaDistance.NodeIDs)+1)
	header := "      "
	for _, id := range m.NumaDistance.NodeIDs {
		header += fmt.Sprintf("N%-4d", id)
	}
	body = append(body, "  "+header)
	for i, id := range m.NumaDistance.NodeIDs {
		row := fmt.Sprintf("N%-4d", id)
		if i < len(m.NumaDistance.Matrix) {
			for _, v := range m.NumaDistance.Matrix[i] {
				row += fmt.Sprintf("%-5d", v)
			}
		}
		body = append(body, "  "+row)
	}
	return body
}

func renderCacheSection(m *topology.MachineProfile, opts Options, ansi ansiStyle) []string {
	lines := make([]string, 0, 10)
	cache := summarizeCaches(m.Caches, m.CPU.ThreadsPerCore, m.CPU.Threads)
	lines = append(lines, kv("L1i", cache[0].value, opts.Compact, ansi))
	lines = append(lines, kv("L1d", cache[1].value, opts.Compact, ansi))
	lines = append(lines, kv("L2", cache[2].value, opts.Compact, ansi))
	lines = append(lines, kv("L3", cache[3].value, opts.Compact, ansi))

	showLLC := opts.sectionEnabled("llc") || (!opts.HasOnly() && opts.Profile == "verbose") || len(m.LLCGroups) > 0
	if !showLLC {
		return lines
	}

	if len(m.LLCGroups) == 0 {
		val := "none"
		if strings.EqualFold(m.Metadata.OS, "macOS") {
			val = "N/A"
		}
		lines = append(lines, kv("LLC Groups", val, opts.Compact, ansi))
		return lines
	}

	for i, group := range m.LLCGroups {
		cpuset := group.CPUSet
		if cpuset == "" {
			cpuset = topology.FormatCPUSet(group.CPUs)
		}
		value := fmt.Sprintf("Group %d: CPUs %s", group.ID, cpuset)
		if i == 0 {
			lines = append(lines, kv("LLC Groups", value, opts.Compact, ansi))
		} else {
			lines = append(lines, continuation(value, opts.Compact))
		}
	}
	return lines
}

func renderMemorySection(m *topology.MachineProfile, opts Options, ansi ansiStyle) []string {
	lines := make([]string, 0, len(m.MemoryDistribution.Nodes)+6)
	total := "n/a"
	if m.MemoryDistribution.TotalBytes > 0 {
		total = util.HumanBytes(m.MemoryDistribution.TotalBytes)
	}
	lines = append(lines, kv("Total", total, opts.Compact, ansi))

	if len(m.MemoryDistribution.Nodes) == 0 {
		lines = append(lines, kv("NUMA", "n/a", opts.Compact, ansi))
	} else {
		for _, node := range m.MemoryDistribution.Nodes {
			lines = append(lines, kv(fmt.Sprintf("NUMA%d", node.NodeID), fmt.Sprintf("%s (%.1f%%)", util.HumanBytes(node.TotalBytes), node.Percent), opts.Compact, ansi))
		}
	}

	showMemTopo := opts.sectionEnabled("memtop") || (!opts.HasOnly() && opts.Profile == "verbose")
	if !showMemTopo {
		return lines
	}
	if !m.MemoryTopology.Known {
		lines = append(lines, kv("DIMMs", "unknown", opts.Compact, ansi))
		lines = append(lines, kv("Channels", "unknown", opts.Compact, ansi))
		return lines
	}

	pop := "unknown"
	if m.MemoryTopology.DIMMsPopulated >= 0 && m.MemoryTopology.DIMMsTotal >= 0 {
		pop = fmt.Sprintf("%d/%d", m.MemoryTopology.DIMMsPopulated, m.MemoryTopology.DIMMsTotal)
	}
	channels := "unknown"
	if m.MemoryTopology.Channels >= 0 {
		channels = fmt.Sprintf("%d", m.MemoryTopology.Channels)
	}
	lines = append(lines, kv("DIMMs", pop, opts.Compact, ansi))
	lines = append(lines, kv("Channels", channels, opts.Compact, ansi))
	return lines
}

func renderPowerSection(m *topology.MachineProfile, opts Options, ansi ansiStyle) []string {
	return []string{
		kv("Governor", valueOrNA(m.Power.Governor), opts.Compact, ansi),
		kv("Driver", valueOrNA(m.Power.Driver), opts.Compact, ansi),
		kv("Turbo/Boost", valueOrNA(m.Power.TurboBoost), opts.Compact, ansi),
	}
}

func renderIsolationSection(m *topology.MachineProfile, opts Options, ansi ansiStyle) []string {
	lines := []string{
		kv("isolated", listOrNone(m.Isolation.Isolated), opts.Compact, ansi),
		kv("nohz_full", listOrNone(m.Isolation.NoHZFull), opts.Compact, ansi),
		kv("rcu_nocbs", listOrNone(m.Isolation.RCUNOCBS), opts.Compact, ansi),
	}
	if opts.sectionEnabled("notes") || (!opts.HasOnly() && opts.Profile == "verbose") {
		lines = append(lines, kv("offline", listOrNone(m.CPU.OfflineCPUs), opts.Compact, ansi))
	}
	return lines
}

func renderPCIeSection(m *topology.MachineProfile, ansi ansiStyle) []string {
	if len(m.PCIeAffinity) == 0 {
		return []string{"N/A"}
	}
	lines := make([]string, 0, len(m.PCIeAffinity))
	for _, entry := range m.PCIeAffinity {
		numa := "N/A"
		if entry.NUMANode >= 0 {
			numa = fmt.Sprintf("NUMA %d", entry.NUMANode)
		}
		lines = append(lines, fmt.Sprintf("%s (%s) -> %s", entry.Name, entry.DeviceType, numa))
	}
	return lines
}

func renderPostureSection(posture topology.Posture, opts Options, ansi ansiStyle) []string {
	lines := make([]string, 0, len(posture.Checks)+1)
	for _, check := range posture.Checks {
		line := fmt.Sprintf("%-7s %s: %s", check.Status, check.Name, check.Reason)
		switch check.Status {
		case "pass":
			line = ansi.good(line)
		case "warn":
			line = ansi.warn(line)
		default:
			line = ansi.dim(line)
		}
		lines = append(lines, continuation(line, opts.Compact))
	}
	lines = append(lines, kv("SMT", posture.SMT, opts.Compact, ansi))
	return lines
}

func renderDebugSection(warnings []string) []string {
	lines := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		lines = append(lines, "- "+warning)
	}
	return lines
}

func kv(label string, value string, compact bool, ansi ansiStyle) string {
	if value == "" {
		value = "n/a"
	}
	if compact {
		key := label + ":"
		if ansi.enabled {
			return ansi.dim(key) + " " + value
		}
		return key + " " + value
	}
	key := fmt.Sprintf("%-10s", label+":")
	if ansi.enabled {
		return ansi.dim(key) + " " + value
	}
	return key + " " + value
}

func continuation(value string, compact bool) string {
	if compact {
		return "  " + value
	}
	return strings.Repeat(" ", 11) + value
}

type cacheLine struct {
	name  string
	value string
}

func summarizeCaches(caches []topology.Cache, threadsPerCore int, totalThreads int) []cacheLine {
	out := make([]cacheLine, 0, 4)
	out = append(out, cacheLine{name: "L1i", value: cacheLineValue(caches, 1, "instruction", threadsPerCore, totalThreads)})
	out = append(out, cacheLine{name: "L1d", value: cacheLineValue(caches, 1, "data", threadsPerCore, totalThreads)})
	out = append(out, cacheLine{name: "L2", value: cacheLineValue(caches, 2, "", threadsPerCore, totalThreads)})
	out = append(out, cacheLine{name: "L3", value: cacheLineValue(caches, 3, "", threadsPerCore, totalThreads)})
	return out
}

func cacheLineValue(caches []topology.Cache, level int, kind string, threadsPerCore int, totalThreads int) string {
	filtered := make([]topology.Cache, 0)
	for _, cache := range caches {
		if cache.Level != level {
			continue
		}
		if kind != "" {
			if normalizeCacheType(cache.Type) != kind {
				continue
			}
		} else if level >= 2 {
			if normalizeCacheType(cache.Type) != "unified" && level != 2 {
				continue
			}
		}
		filtered = append(filtered, cache)
	}
	if len(filtered) == 0 {
		return "n/a"
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].SizeBytes != filtered[j].SizeBytes {
			return filtered[i].SizeBytes > filtered[j].SizeBytes
		}
		return len(filtered[i].SharedCPUList) < len(filtered[j].SharedCPUList)
	})

	representative := filtered[0]
	shared := len(representative.SharedCPUList)
	size := util.HumanBytes(representative.SizeBytes)

	scope := "shared"
	if shared <= maxInt(threadsPerCore, 1) {
		scope = "per-core"
	} else if totalThreads > 0 && shared >= totalThreads {
		scope = "shared(all)"
	} else {
		scope = fmt.Sprintf("shared by %d threads", shared)
	}

	if scope == "per-core" {
		return fmt.Sprintf("%s per-core", size)
	}
	if scope == "shared(all)" {
		return fmt.Sprintf("%s shared", size)
	}
	if len(filtered) > 1 {
		return fmt.Sprintf("%s %s (%d groups)", size, scope, len(filtered))
	}
	return fmt.Sprintf("%s %s", size, scope)
}

func normalizeCacheType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	switch t {
	case "instruction", "inst", "i":
		return "instruction"
	case "data", "d":
		return "data"
	default:
		return "unified"
	}
}

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func valueOrNA(v string) string {
	if strings.TrimSpace(v) == "" {
		return "n/a"
	}
	return v
}

func listOrNone(values []int) string {
	if len(values) == 0 {
		return "none"
	}
	return topology.FormatCPUSet(values)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
