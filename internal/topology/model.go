package topology

import (
	"sort"
	"strconv"
	"strings"
)

// MachineProfile is the canonical structured output for arpkit.
type MachineProfile struct {
	Metadata           Metadata           `json:"metadata"`
	CPU                CPUInfo            `json:"cpu"`
	Microarch          MicroarchInfo      `json:"microarch"`
	Nodes              []NUMANode         `json:"nodes"`
	NumaDistance       NumaDistance       `json:"numa_distance"`
	Sockets            []Socket           `json:"sockets"`
	Cores              []Core             `json:"cores"`
	Threads            []Thread           `json:"threads"`
	Caches             []Cache            `json:"caches"`
	LLCGroups          []CPUGroup         `json:"llc_groups"`
	MemoryDistribution MemoryDistribution `json:"memory_distribution"`
	MemoryTopology     MemoryTopology     `json:"memory_topology"`
	Isolation          IsolationInfo      `json:"isolation"`
	Power              PowerInfo          `json:"power"`
	PCIeAffinity       []PCIEAffinity     `json:"pcie_affinity"`
	Posture            Posture            `json:"posture"`
	Flags              Flags              `json:"flags"`
	Warnings           []string           `json:"warnings,omitempty"`
	Partial            bool               `json:"partial"`
}

type Metadata struct {
	ToolVersion string `json:"tool_version"`
	Hostname    string `json:"hostname"`
	OS          string `json:"os"`
	Kernel      string `json:"kernel"`
	Arch        string `json:"arch"`
}

type CPUInfo struct {
	Architecture   string        `json:"architecture"`
	Vendor         string        `json:"vendor"`
	ModelName      string        `json:"model_name"`
	Sockets        int           `json:"sockets"`
	NUMANodes      int           `json:"numa_nodes"`
	Cores          int           `json:"cores"`
	Threads        int           `json:"threads"`
	ThreadsPerCore int           `json:"threads_per_core"`
	SMT            bool          `json:"smt"`
	OnlineCPUs     []int         `json:"online_cpus"`
	OfflineCPUs    []int         `json:"offline_cpus"`
	IsolatedCPUs   []int         `json:"isolated_cpus"`
	Frequency      FrequencyInfo `json:"frequency"`
}

type FrequencyInfo struct {
	CurrentMHz int `json:"current_mhz"`
	MinMHz     int `json:"min_mhz"`
	MaxMHz     int `json:"max_mhz"`
}

type MicroarchInfo struct {
	MicroarchName        string         `json:"microarch_name"`
	Vendor               string         `json:"vendor"`
	Family               int            `json:"family"`
	Model                int            `json:"model"`
	Stepping             int            `json:"stepping"`
	ISAFeatures          FeatureSummary `json:"isa_features"`
	AVX512LikelyDisabled bool           `json:"avx512_likely_disabled"`
	RawFlags             []string       `json:"raw_flags"`
}

type FeatureSummary struct {
	AVX     bool `json:"avx"`
	AVX2    bool `json:"avx2"`
	AVX512F bool `json:"avx512f"`
	AES     bool `json:"aes"`
	BMI1    bool `json:"bmi1"`
	BMI2    bool `json:"bmi2"`
	FMA     bool `json:"fma"`
	SHA     bool `json:"sha"`

	SVE   bool `json:"sve"`
	SVE2  bool `json:"sve2"`
	SHA1  bool `json:"sha1"`
	SHA2  bool `json:"sha2"`
	CRC32 bool `json:"crc32"`
}

type NUMANode struct {
	ID            int    `json:"id"`
	CPUs          []int  `json:"cpus"`
	MemTotalBytes uint64 `json:"mem_total_bytes"`
}

type NumaDistance struct {
	NodeIDs []int   `json:"node_ids"`
	Matrix  [][]int `json:"matrix"`
}

type Socket struct {
	ID      int   `json:"id"`
	CPUs    []int `json:"cpus"`
	CoreIDs []int `json:"core_ids"`
	NodeIDs []int `json:"node_ids"`
}

type Core struct {
	ID        int   `json:"id"`
	SocketID  int   `json:"socket_id"`
	NodeID    int   `json:"node_id"`
	LocalID   int   `json:"local_id"`
	ThreadIDs []int `json:"thread_ids"`
}

type Thread struct {
	ID          int  `json:"id"`
	SocketID    int  `json:"socket_id"`
	CoreID      int  `json:"core_id"`
	CoreLocalID int  `json:"core_local_id"`
	NodeID      int  `json:"node_id"`
	Online      bool `json:"online"`
}

type Cache struct {
	Level         int    `json:"level"`
	Type          string `json:"type"`
	SizeBytes     uint64 `json:"size_bytes"`
	SharedCPUList []int  `json:"shared_cpu_list"`
}

type CPUGroup struct {
	ID     int    `json:"id"`
	CPUs   []int  `json:"cpus"`
	CPUSet string `json:"cpuset"`
}

type MemoryDistribution struct {
	TotalBytes uint64                   `json:"total_bytes"`
	Nodes      []NodeMemoryDistribution `json:"nodes"`
}

type NodeMemoryDistribution struct {
	NodeID     int     `json:"node_id"`
	TotalBytes uint64  `json:"total_bytes"`
	Percent    float64 `json:"percent"`
}

type MemoryTopology struct {
	Known          bool `json:"known"`
	DIMMsPopulated int  `json:"dimms_populated"`
	DIMMsTotal     int  `json:"dimms_total"`
	Channels       int  `json:"channels"`
}

type IsolationInfo struct {
	Isolated []int `json:"isolated"`
	NoHZFull []int `json:"nohz_full"`
	RCUNOCBS []int `json:"rcu_nocbs"`
}

type PowerInfo struct {
	Governor   string `json:"governor"`
	Driver     string `json:"driver"`
	TurboBoost string `json:"turbo_boost"`
}

type PCIEAffinity struct {
	BDF        string `json:"bdf"`
	DeviceType string `json:"device_type"`
	Name       string `json:"name"`
	Class      string `json:"class"`
	NUMANode   int    `json:"numa_node"`
}

type Posture struct {
	Score  float64        `json:"score"`
	Checks []PostureCheck `json:"checks"`
	SMT    string         `json:"smt_note"`
}

type PostureCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type Flags struct {
	SMT          bool  `json:"smt"`
	IsolatedCPUs []int `json:"isolated_cpus"`
	OfflineCPUs  []int `json:"offline_cpus"`
}

func (m *MachineProfile) Sort() {
	for i := range m.Nodes {
		m.Nodes[i].CPUs = SortedUniqueInts(m.Nodes[i].CPUs)
	}
	for i := range m.Sockets {
		m.Sockets[i].CPUs = SortedUniqueInts(m.Sockets[i].CPUs)
		m.Sockets[i].CoreIDs = SortedUniqueInts(m.Sockets[i].CoreIDs)
		m.Sockets[i].NodeIDs = SortedUniqueInts(m.Sockets[i].NodeIDs)
	}
	for i := range m.Cores {
		m.Cores[i].ThreadIDs = SortedUniqueInts(m.Cores[i].ThreadIDs)
	}
	for i := range m.Caches {
		m.Caches[i].SharedCPUList = SortedUniqueInts(m.Caches[i].SharedCPUList)
	}
	for i := range m.LLCGroups {
		m.LLCGroups[i].CPUs = SortedUniqueInts(m.LLCGroups[i].CPUs)
		if m.LLCGroups[i].CPUSet == "" {
			m.LLCGroups[i].CPUSet = FormatCPUSet(m.LLCGroups[i].CPUs)
		}
	}

	m.CPU.OnlineCPUs = SortedUniqueInts(m.CPU.OnlineCPUs)
	m.CPU.OfflineCPUs = SortedUniqueInts(m.CPU.OfflineCPUs)
	m.CPU.IsolatedCPUs = SortedUniqueInts(m.CPU.IsolatedCPUs)
	m.Flags.IsolatedCPUs = SortedUniqueInts(m.Flags.IsolatedCPUs)
	m.Flags.OfflineCPUs = SortedUniqueInts(m.Flags.OfflineCPUs)
	m.Isolation.Isolated = SortedUniqueInts(m.Isolation.Isolated)
	m.Isolation.NoHZFull = SortedUniqueInts(m.Isolation.NoHZFull)
	m.Isolation.RCUNOCBS = SortedUniqueInts(m.Isolation.RCUNOCBS)
	m.Microarch.RawFlags = sortStrings(m.Microarch.RawFlags)

	sort.Slice(m.Nodes, func(i, j int) bool { return m.Nodes[i].ID < m.Nodes[j].ID })
	sort.Slice(m.Sockets, func(i, j int) bool { return m.Sockets[i].ID < m.Sockets[j].ID })
	sort.Slice(m.Cores, func(i, j int) bool {
		if m.Cores[i].SocketID != m.Cores[j].SocketID {
			return m.Cores[i].SocketID < m.Cores[j].SocketID
		}
		if m.Cores[i].LocalID != m.Cores[j].LocalID {
			return m.Cores[i].LocalID < m.Cores[j].LocalID
		}
		return m.Cores[i].ID < m.Cores[j].ID
	})
	sort.Slice(m.Threads, func(i, j int) bool { return m.Threads[i].ID < m.Threads[j].ID })
	sort.Slice(m.Caches, func(i, j int) bool {
		if m.Caches[i].Level != m.Caches[j].Level {
			return m.Caches[i].Level < m.Caches[j].Level
		}
		if m.Caches[i].Type != m.Caches[j].Type {
			return m.Caches[i].Type < m.Caches[j].Type
		}
		if m.Caches[i].SizeBytes != m.Caches[j].SizeBytes {
			return m.Caches[i].SizeBytes < m.Caches[j].SizeBytes
		}
		return FormatIntSlice(m.Caches[i].SharedCPUList) < FormatIntSlice(m.Caches[j].SharedCPUList)
	})
	sort.Slice(m.LLCGroups, func(i, j int) bool {
		if m.LLCGroups[i].ID != m.LLCGroups[j].ID {
			return m.LLCGroups[i].ID < m.LLCGroups[j].ID
		}
		return m.LLCGroups[i].CPUSet < m.LLCGroups[j].CPUSet
	})
	sort.Slice(m.MemoryDistribution.Nodes, func(i, j int) bool {
		return m.MemoryDistribution.Nodes[i].NodeID < m.MemoryDistribution.Nodes[j].NodeID
	})
	sort.Slice(m.PCIeAffinity, func(i, j int) bool {
		if m.PCIeAffinity[i].DeviceType != m.PCIeAffinity[j].DeviceType {
			return m.PCIeAffinity[i].DeviceType < m.PCIeAffinity[j].DeviceType
		}
		if m.PCIeAffinity[i].Name != m.PCIeAffinity[j].Name {
			return m.PCIeAffinity[i].Name < m.PCIeAffinity[j].Name
		}
		return m.PCIeAffinity[i].BDF < m.PCIeAffinity[j].BDF
	})

	m.sortNUMADistance()
}

func (m *MachineProfile) sortNUMADistance() {
	if len(m.NumaDistance.NodeIDs) == 0 {
		return
	}
	nodeIDs := append([]int(nil), m.NumaDistance.NodeIDs...)
	sort.Ints(nodeIDs)
	if len(m.NumaDistance.Matrix) != len(m.NumaDistance.NodeIDs) {
		m.NumaDistance.NodeIDs = nodeIDs
		return
	}

	indexByNode := make(map[int]int, len(m.NumaDistance.NodeIDs))
	for i, id := range m.NumaDistance.NodeIDs {
		indexByNode[id] = i
	}

	sortedMatrix := make([][]int, len(nodeIDs))
	for i, rowNode := range nodeIDs {
		oldRowIdx, ok := indexByNode[rowNode]
		if !ok || oldRowIdx >= len(m.NumaDistance.Matrix) {
			sortedMatrix[i] = make([]int, len(nodeIDs))
			continue
		}
		oldRow := m.NumaDistance.Matrix[oldRowIdx]
		newRow := make([]int, len(nodeIDs))
		for j, colNode := range nodeIDs {
			oldColIdx, ok := indexByNode[colNode]
			if !ok || oldColIdx >= len(oldRow) {
				continue
			}
			newRow[j] = oldRow[oldColIdx]
		}
		sortedMatrix[i] = newRow
	}
	m.NumaDistance.NodeIDs = nodeIDs
	m.NumaDistance.Matrix = sortedMatrix
}

func SortedUniqueInts(values []int) []int {
	if len(values) == 0 {
		return []int{}
	}
	out := make([]int, len(values))
	copy(out, values)
	sort.Ints(out)
	w := 1
	for i := 1; i < len(out); i++ {
		if out[i] != out[i-1] {
			out[w] = out[i]
			w++
		}
	}
	return out[:w]
}

func FormatCPUSet(values []int) string {
	values = SortedUniqueInts(values)
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return strconv.Itoa(values[0])
	}

	var b strings.Builder
	start := values[0]
	prev := values[0]
	for i := 1; i < len(values); i++ {
		if values[i] == prev+1 {
			prev = values[i]
			continue
		}
		writeCPUSetRange(&b, start, prev)
		b.WriteByte(',')
		start = values[i]
		prev = values[i]
	}
	writeCPUSetRange(&b, start, prev)
	return b.String()
}

func writeCPUSetRange(b *strings.Builder, start, end int) {
	if start == end {
		b.WriteString(strconv.Itoa(start))
		return
	}
	b.WriteString(strconv.Itoa(start))
	b.WriteByte('-')
	b.WriteString(strconv.Itoa(end))
}

func FormatIntSlice(values []int) string {
	if len(values) == 0 {
		return ""
	}
	b := make([]byte, 0, len(values)*3)
	for i, v := range values {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, itoa(v)...)
	}
	return string(b)
}

func itoa(v int) []byte {
	if v == 0 {
		return []byte{'0'}
	}
	n := v
	if n < 0 {
		n = -n
	}
	buf := make([]byte, 0, 12)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	if v < 0 {
		buf = append(buf, '-')
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return buf
}

func sortStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	w := 1
	for i := 1; i < len(out); i++ {
		if out[i] != out[i-1] {
			out[w] = out[i]
			w++
		}
	}
	return out[:w]
}
