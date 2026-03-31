//go:build linux

package platform

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/pratik-anurag/arpkit/internal/microarch"
	"github.com/pratik-anurag/arpkit/internal/numa"
	"github.com/pratik-anurag/arpkit/internal/parser"
	"github.com/pratik-anurag/arpkit/internal/pcie"
	"github.com/pratik-anurag/arpkit/internal/power"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

func collectPlatform(opts Options) (*topology.MachineProfile, error) {
	return collectLinux(opts)
}

func collectLinux(opts Options) (*topology.MachineProfile, error) {
	fs := parser.NewReader(opts.Root)
	if opts.MaxFileBytes > 0 {
		fs.MaxFileBytes = opts.MaxFileBytes
	}
	m := &topology.MachineProfile{}

	hostname, _ := os.Hostname()
	m.Metadata.Hostname = hostname
	m.Metadata.OS = readLinuxPrettyName(fs)
	m.Metadata.Kernel = readLinuxKernel()
	m.Metadata.Arch = runtime.GOARCH
	m.CPU.Architecture = runtime.GOARCH

	cpuInfoText, _ := fs.ReadFile("/proc/cpuinfo")
	cpuInfo := microarch.ParseProcCPUInfo(cpuInfoText)
	augmentMicroarchFromSysfs(fs, &cpuInfo)
	m.CPU.Vendor = cpuInfo.Vendor
	m.CPU.ModelName = cpuInfo.ModelName
	m.Microarch = microarch.FromLinux(runtime.GOARCH, cpuInfo)

	cpuIDs, err := discoverCPUIDs(fs)
	if err != nil {
		return nil, fmt.Errorf("discover cpus: %w", err)
	}
	if len(cpuIDs) == 0 {
		return nil, fmt.Errorf("no cpu entries found in sysfs")
	}

	onlineSet := readCPUSetFile(fs, "/sys/devices/system/cpu/online")
	offlineSet := readCPUSetFile(fs, "/sys/devices/system/cpu/offline")

	type coreKey struct {
		socket int
		local  int
	}
	coreByKey := make(map[coreKey]int)
	m.Threads = make([]topology.Thread, 0, len(cpuIDs))

	for _, cpuID := range cpuIDs {
		socketID := readIntOrDefault(fs, fmt.Sprintf("/sys/devices/system/cpu/cpu%d/topology/physical_package_id", cpuID), 0)
		coreLocalID := readIntOrDefault(fs, fmt.Sprintf("/sys/devices/system/cpu/cpu%d/topology/core_id", cpuID), cpuID)

		online := true
		if len(onlineSet) > 0 {
			_, online = onlineSet[cpuID]
		}
		if _, isOffline := offlineSet[cpuID]; isOffline {
			online = false
		}
		if fs.Exists(fmt.Sprintf("/sys/devices/system/cpu/cpu%d/online", cpuID)) {
			online = readIntOrDefault(fs, fmt.Sprintf("/sys/devices/system/cpu/cpu%d/online", cpuID), 1) == 1
		}

		key := coreKey{socket: socketID, local: coreLocalID}
		coreID, ok := coreByKey[key]
		if !ok {
			coreID = len(coreByKey)
			coreByKey[key] = coreID
			m.Cores = append(m.Cores, topology.Core{
				ID:       coreID,
				SocketID: socketID,
				NodeID:   -1,
				LocalID:  coreLocalID,
			})
		}

		m.Threads = append(m.Threads, topology.Thread{
			ID:          cpuID,
			SocketID:    socketID,
			CoreID:      coreID,
			CoreLocalID: coreLocalID,
			NodeID:      -1,
			Online:      online,
		})
	}

	freq := readLinuxFrequencies(fs, cpuIDs, cpuInfoText)
	m.CPU.Frequency = freq

	readLinuxNUMA(fs, m)
	readLinuxCaches(fs, m, cpuIDs)
	m.NumaDistance = numa.ReadDistanceMatrix(fs, nodeIDsFromNodes(m.Nodes))

	cmdline, _ := fs.ReadFile("/proc/cmdline")
	m.Isolation = numa.ParseIsolationCmdline(cmdline)
	m.CPU.IsolatedCPUs = numa.UnionIsolation(m.Isolation)

	m.Power = power.SnapshotLinux(fs)
	m.MemoryTopology = numa.DetectMemoryTopology(fs)
	m.PCIeAffinity = pcie.ScanLinux(fs)

	if len(m.Nodes) == 0 {
		node := topology.NUMANode{ID: 0}
		for _, cpu := range cpuIDs {
			node.CPUs = append(node.CPUs, cpu)
		}
		node.MemTotalBytes = readLinuxTotalMemBytes(fs)
		m.Nodes = []topology.NUMANode{node}
		if len(m.NumaDistance.NodeIDs) == 0 {
			m.NumaDistance = topology.NumaDistance{NodeIDs: []int{0}, Matrix: [][]int{{10}}}
		}
	}

	if m.Microarch.Vendor == "" {
		m.Microarch.Vendor = m.CPU.Vendor
	}
	if m.Microarch.MicroarchName == "" {
		m.Microarch.MicroarchName = "unknown"
	}
	applyReadIssues(m, fs)

	return m, nil
}

func applyReadIssues(m *topology.MachineProfile, fs *parser.Reader) {
	if m == nil || fs == nil {
		return
	}
	for _, issue := range fs.Issues() {
		m.Warnings = append(m.Warnings, fmt.Sprintf("skipped %s: file exceeds %d bytes", issue.Path, issue.Limit))
	}
	if len(m.Warnings) > 0 {
		m.Partial = true
	}
}

func augmentMicroarchFromSysfs(fs *parser.Reader, info *microarch.ProcCPUInfo) {
	if info == nil {
		return
	}
	if info.Family >= 0 && info.Model >= 0 && info.Stepping >= 0 {
		return
	}
	read := func(path string) int {
		v, err := fs.ReadFile(path)
		if err != nil {
			return -1
		}
		n, err := parseFlexibleInt(v)
		if err != nil {
			return -1
		}
		return n
	}
	if info.Family < 0 {
		info.Family = read("/sys/devices/system/cpu/cpu0/cpuid/family")
	}
	if info.Model < 0 {
		info.Model = read("/sys/devices/system/cpu/cpu0/cpuid/model")
	}
	if info.Stepping < 0 {
		info.Stepping = read("/sys/devices/system/cpu/cpu0/cpuid/stepping")
	}
}

func parseFlexibleInt(text string) (int, error) {
	v := strings.TrimSpace(text)
	if v == "" {
		return 0, fmt.Errorf("empty value")
	}
	if strings.HasPrefix(v, "0x") || strings.HasPrefix(v, "0X") {
		n, err := strconv.ParseInt(v, 0, 64)
		return int(n), err
	}
	n, err := strconv.Atoi(v)
	if err == nil {
		return n, nil
	}
	if hex, hexErr := strconv.ParseInt(v, 16, 64); hexErr == nil {
		return int(hex), nil
	}
	return 0, err
}

func discoverCPUIDs(fs *parser.Reader) ([]int, error) {
	entries, err := fs.ReadDir("/sys/devices/system/cpu")
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "cpu") {
			continue
		}
		idStr := strings.TrimPrefix(name, "cpu")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, nil
}

func readCPUSetFile(fs *parser.Reader, path string) map[int]struct{} {
	value, err := fs.ReadFile(path)
	if err != nil || value == "" {
		return nil
	}
	cpus, err := parser.ParseCPUList(value)
	if err != nil {
		return nil
	}
	set := make(map[int]struct{}, len(cpus))
	for _, cpu := range cpus {
		set[cpu] = struct{}{}
	}
	return set
}

func readIntOrDefault(fs *parser.Reader, path string, fallback int) int {
	v, err := fs.ReadInt(path)
	if err != nil {
		return fallback
	}
	return v
}

func readLinuxNUMA(fs *parser.Reader, m *topology.MachineProfile) {
	entries, err := fs.ReadDir("/sys/devices/system/node")
	if err != nil {
		return
	}

	cpuToNode := make(map[int]int)
	nodes := make([]topology.NUMANode, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "node") {
			continue
		}
		nodeID, err := strconv.Atoi(strings.TrimPrefix(name, "node"))
		if err != nil {
			continue
		}
		cpulistText, err := fs.ReadFile(fmt.Sprintf("/sys/devices/system/node/%s/cpulist", name))
		if err != nil {
			continue
		}
		cpus, err := parser.ParseCPUList(cpulistText)
		if err != nil {
			continue
		}
		for _, cpu := range cpus {
			cpuToNode[cpu] = nodeID
		}

		node := topology.NUMANode{ID: nodeID, CPUs: cpus}
		node.MemTotalBytes = readLinuxNodeMemBytes(fs, nodeID)
		nodes = append(nodes, node)
	}

	for i := range m.Threads {
		if nodeID, ok := cpuToNode[m.Threads[i].ID]; ok {
			m.Threads[i].NodeID = nodeID
		}
	}
	coreNode := make(map[int]int)
	for _, thread := range m.Threads {
		if thread.NodeID >= 0 {
			coreNode[thread.CoreID] = thread.NodeID
		}
	}
	for i := range m.Cores {
		if nodeID, ok := coreNode[m.Cores[i].ID]; ok {
			m.Cores[i].NodeID = nodeID
		}
	}

	m.Nodes = append(m.Nodes, nodes...)
}

func readLinuxNodeMemBytes(fs *parser.Reader, nodeID int) uint64 {
	path := fmt.Sprintf("/sys/devices/system/node/node%d/meminfo", nodeID)
	text, err := fs.ReadFile(path)
	if err != nil {
		return 0
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if !strings.Contains(line, "MemTotal") {
			continue
		}
		fields := strings.Fields(line)
		for i, f := range fields {
			if strings.HasPrefix(f, "MemTotal") && i+1 < len(fields) {
				kb, err := strconv.ParseUint(fields[i+1], 10, 64)
				if err == nil {
					return kb * 1024
				}
			}
		}
	}
	return 0
}

func readLinuxTotalMemBytes(fs *parser.Reader) uint64 {
	text, err := fs.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err == nil {
			return kb * 1024
		}
	}
	return 0
}

func readLinuxCaches(fs *parser.Reader, m *topology.MachineProfile, cpuIDs []int) {
	seen := map[string]struct{}{}
	for _, cpuID := range cpuIDs {
		entries, err := fs.ReadDir(fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cache", cpuID))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "index") {
				continue
			}
			base := fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cache/%s", cpuID, entry.Name())
			level := readIntOrDefault(fs, base+"/level", 0)
			if level == 0 {
				continue
			}
			typeName, err := fs.ReadFile(base + "/type")
			if err != nil {
				continue
			}
			typeName = strings.ToLower(strings.TrimSpace(typeName))

			sizeText, err := fs.ReadFile(base + "/size")
			if err != nil {
				continue
			}
			sizeBytes := parseSizeBytes(sizeText)
			if sizeBytes == 0 {
				continue
			}

			sharedText, err := fs.ReadFile(base + "/shared_cpu_list")
			if err != nil {
				continue
			}
			sharedCPUs, err := parser.ParseCPUList(sharedText)
			if err != nil {
				continue
			}
			key := fmt.Sprintf("%d|%s|%d|%s", level, typeName, sizeBytes, parser.FormatCPUList(sharedCPUs))
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			m.Caches = append(m.Caches, topology.Cache{
				Level:         level,
				Type:          typeName,
				SizeBytes:     sizeBytes,
				SharedCPUList: sharedCPUs,
			})
		}
	}
}

func readLinuxFrequencies(fs *parser.Reader, cpuIDs []int, cpuInfoText string) topology.FrequencyInfo {
	curSum := 0
	curCount := 0
	minSeen := 0
	maxSeen := 0

	for _, cpuID := range cpuIDs {
		base := fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq", cpuID)
		curKHz, curErr := fs.ReadInt(base + "/scaling_cur_freq")
		minKHz, minErr := fs.ReadInt(base + "/scaling_min_freq")
		maxKHz, maxErr := fs.ReadInt(base + "/scaling_max_freq")

		if curErr == nil && curKHz > 0 {
			curSum += curKHz / 1000
			curCount++
		}
		if minErr == nil && minKHz > 0 {
			mhz := minKHz / 1000
			if minSeen == 0 || mhz < minSeen {
				minSeen = mhz
			}
		}
		if maxErr == nil && maxKHz > 0 {
			mhz := maxKHz / 1000
			if mhz > maxSeen {
				maxSeen = mhz
			}
		}
	}

	current := 0
	if curCount > 0 {
		current = curSum / curCount
	} else if mhz := parseFallbackCPUMHz(cpuInfoText); mhz > 0 {
		current = mhz
	}

	return topology.FrequencyInfo{
		CurrentMHz: current,
		MinMHz:     minSeen,
		MaxMHz:     maxSeen,
	}
}

func parseFallbackCPUMHz(cpuInfoText string) int {
	for _, line := range strings.Split(cpuInfoText, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "cpu MHz") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err == nil {
			return int(f + 0.5)
		}
	}
	return 0
}

func parseSizeBytes(v string) uint64 {
	v = strings.TrimSpace(strings.ToUpper(v))
	if v == "" {
		return 0
	}
	mult := uint64(1)
	switch {
	case strings.HasSuffix(v, "K"):
		mult = 1024
		v = strings.TrimSuffix(v, "K")
	case strings.HasSuffix(v, "M"):
		mult = 1024 * 1024
		v = strings.TrimSuffix(v, "M")
	case strings.HasSuffix(v, "G"):
		mult = 1024 * 1024 * 1024
		v = strings.TrimSuffix(v, "G")
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0
	}
	return n * mult
}

func readLinuxPrettyName(fs *parser.Reader) string {
	text, err := fs.ReadFile("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, "PRETTY_NAME=") {
			continue
		}
		v := strings.TrimPrefix(line, "PRETTY_NAME=")
		v = strings.Trim(v, `"`)
		if v != "" {
			return v
		}
	}
	return "Linux"
}

func readLinuxKernel() string {
	var u syscall.Utsname
	if err := syscall.Uname(&u); err != nil {
		return ""
	}
	return utsToString(u.Release[:])
}

func utsToString(b []int8) string {
	buf := make([]byte, 0, len(b))
	for _, c := range b {
		if c == 0 {
			break
		}
		buf = append(buf, byte(c))
	}
	return string(buf)
}

func nodeIDsFromNodes(nodes []topology.NUMANode) []int {
	ids := make([]int, 0, len(nodes))
	for _, node := range nodes {
		ids = append(ids, node.ID)
	}
	sort.Ints(ids)
	return ids
}
