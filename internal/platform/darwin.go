//go:build darwin

package platform

import (
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"syscall"

	"github.com/pratik-anurag/arpkit/internal/microarch"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

func collectPlatform(opts Options) (*topology.MachineProfile, error) {
	m := &topology.MachineProfile{}

	hostname, _ := os.Hostname()
	m.Metadata.Hostname = hostname
	m.Metadata.OS = "macOS"
	m.Metadata.Kernel = readSysctlString("kern.osrelease")
	m.Metadata.Arch = runtime.GOARCH
	m.CPU.Architecture = runtime.GOARCH
	m.CPU.NUMANodes = -1
	m.CPU.ModelName = firstNonEmpty(readSysctlString("machdep.cpu.brand_string"), readSysctlString("hw.model"))
	m.CPU.Vendor = readSysctlString("machdep.cpu.vendor")

	family := int(readSysctlUint32("machdep.cpu.family"))
	model := int(readSysctlUint32("machdep.cpu.model"))
	stepping := int(readSysctlUint32("machdep.cpu.stepping"))
	if family == 0 {
		family = -1
	}
	if model == 0 {
		model = -1
	}
	if stepping == 0 {
		stepping = -1
	}

	m.Microarch = microarch.FromDarwin(runtime.GOARCH, m.CPU.Vendor, m.CPU.ModelName, family, model, stepping, hasSysctlFeature)
	if m.Microarch.MicroarchName == "" {
		m.Microarch.MicroarchName = "unknown"
	}

	physical := int(readSysctlUint32("hw.physicalcpu"))
	if physical == 0 {
		physical = int(readSysctlUint32("hw.ncpu"))
	}
	logical := int(readSysctlUint32("hw.logicalcpu"))
	if logical == 0 {
		logical = physical
	}
	if physical == 0 || logical == 0 {
		return nil, fmt.Errorf("unable to determine cpu topology via sysctl")
	}

	packages := int(readSysctlUint32("hw.packages"))
	if packages < 1 {
		packages = 1
	}

	threadsPerCore := 1
	if logical >= physical {
		threadsPerCore = logical / physical
	}
	if threadsPerCore < 1 {
		threadsPerCore = 1
	}

	m.Cores = make([]topology.Core, 0, physical)
	for core := 0; core < physical; core++ {
		socketID := (core * packages) / physical
		m.Cores = append(m.Cores, topology.Core{
			ID:       core,
			SocketID: socketID,
			NodeID:   -1,
			LocalID:  core,
		})
	}

	m.Threads = make([]topology.Thread, 0, logical)
	for threadID := 0; threadID < logical; threadID++ {
		coreID := threadID % physical
		m.Threads = append(m.Threads, topology.Thread{
			ID:          threadID,
			SocketID:    m.Cores[coreID].SocketID,
			CoreID:      coreID,
			CoreLocalID: m.Cores[coreID].LocalID,
			NodeID:      -1,
			Online:      true,
		})
	}

	totalMem := readSysctlUint64("hw.memsize")
	m.MemoryDistribution.TotalBytes = totalMem

	m.CPU.Frequency = topology.FrequencyInfo{
		CurrentMHz: int(readSysctlUint64("hw.cpufrequency") / 1_000_000),
		MinMHz:     int(readSysctlUint64("hw.cpufrequency_min") / 1_000_000),
		MaxMHz:     int(readSysctlUint64("hw.cpufrequency_max") / 1_000_000),
	}

	sharedCore := make([]int, 0, threadsPerCore)
	for i := 0; i < threadsPerCore; i++ {
		id := i * physical
		if id < logical {
			sharedCore = append(sharedCore, id)
		}
	}
	sharedAll := make([]int, logical)
	for i := range sharedAll {
		sharedAll[i] = i
	}

	appendCache := func(level int, kind string, key string, shared []int) {
		size := readSysctlUint64(key)
		if size == 0 {
			return
		}
		copyShared := append([]int(nil), shared...)
		m.Caches = append(m.Caches, topology.Cache{
			Level:         level,
			Type:          kind,
			SizeBytes:     size,
			SharedCPUList: copyShared,
		})
	}
	appendCache(1, "instruction", "hw.l1icachesize", sharedCore)
	appendCache(1, "data", "hw.l1dcachesize", sharedCore)
	appendCache(2, "unified", "hw.l2cachesize", sharedCore)
	appendCache(3, "unified", "hw.l3cachesize", sharedAll)

	m.Power = topology.PowerInfo{Governor: "n/a", Driver: "n/a", TurboBoost: "n/a"}
	m.MemoryTopology = topology.MemoryTopology{Known: false, DIMMsPopulated: -1, DIMMsTotal: -1, Channels: -1}
	m.NumaDistance = topology.NumaDistance{NodeIDs: []int{}, Matrix: [][]int{}}
	m.Isolation = topology.IsolationInfo{Isolated: []int{}, NoHZFull: []int{}, RCUNOCBS: []int{}}
	m.PCIeAffinity = []topology.PCIEAffinity{}

	return m, nil
}

func hasSysctlFeature(key string) bool {
	if v, err := syscall.Sysctl(key); err == nil {
		if v == "1" || v == "true" {
			return true
		}
		raw := []byte(v)
		if len(raw) > 0 && raw[0] == 1 {
			return true
		}
		if n, parseErr := strconv.ParseUint(stringsTrimNull(v), 10, 64); parseErr == nil {
			return n > 0
		}
	}
	if v, err := syscall.SysctlUint32(key); err == nil {
		return v > 0
	}
	return false
}

func stringsTrimNull(v string) string {
	b := []byte(v)
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return v
}

func readSysctlString(key string) string {
	v, err := syscall.Sysctl(key)
	if err != nil {
		return ""
	}
	return stringsTrimNull(v)
}

func readSysctlUint32(key string) uint32 {
	return uint32(readSysctlUint(key))
}

func readSysctlUint64(key string) uint64 {
	return readSysctlUint(key)
}

func readSysctlUint(key string) uint64 {
	v, err := syscall.Sysctl(key)
	if err != nil || v == "" {
		return 0
	}
	raw := []byte(v)
	if len(raw) > 8 {
		n, parseErr := strconv.ParseUint(stringsTrimNull(v), 10, 64)
		if parseErr == nil {
			return n
		}
		raw = raw[:8]
	}
	buf := make([]byte, 8)
	copy(buf, raw)
	return binary.LittleEndian.Uint64(buf)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
