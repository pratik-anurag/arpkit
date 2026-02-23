package topology

import (
	"errors"
	"fmt"
	"strings"
)

// Normalize finalizes derived fields and deterministic ordering.
func Normalize(m *MachineProfile) error {
	if m == nil {
		return errors.New("nil machine profile")
	}

	if err := ensureCoreThreadLinks(m); err != nil {
		return err
	}
	ensureSocketLinks(m)
	ensureNodeLinks(m)
	computeMemoryDistribution(m)
	computeCPUCounts(m)
	dedupeCaches(m)
	computeLLCGroups(m)
	m.Sort()
	ensureNonNilSlices(m)
	return nil
}

func ensureCoreThreadLinks(m *MachineProfile) error {
	if len(m.Threads) == 0 {
		return nil
	}

	coreByID := make(map[int]int, len(m.Cores))
	for i := range m.Cores {
		coreByID[m.Cores[i].ID] = i
	}

	if len(m.Cores) == 0 {
		type coreKey struct {
			socketID int
			localID  int
			coreID   int
		}
		coreIndex := map[coreKey]int{}
		nextID := 0
		for i := range m.Threads {
			t := &m.Threads[i]
			key := coreKey{socketID: t.SocketID, localID: t.CoreLocalID, coreID: t.CoreID}
			if idx, ok := coreIndex[key]; ok {
				m.Cores[idx].ThreadIDs = append(m.Cores[idx].ThreadIDs, t.ID)
				t.CoreID = m.Cores[idx].ID
				continue
			}
			core := Core{
				ID:       nextID,
				SocketID: t.SocketID,
				NodeID:   t.NodeID,
				LocalID:  t.CoreLocalID,
			}
			if core.LocalID == 0 && key.localID == 0 && key.coreID != 0 {
				core.LocalID = key.coreID
			}
			core.ThreadIDs = []int{t.ID}
			m.Cores = append(m.Cores, core)
			coreIndex[key] = len(m.Cores) - 1
			t.CoreID = nextID
			nextID++
		}
		return nil
	}

	for i := range m.Threads {
		t := &m.Threads[i]
		idx, ok := coreByID[t.CoreID]
		if !ok {
			return fmt.Errorf("thread %d references unknown core %d", t.ID, t.CoreID)
		}
		m.Cores[idx].ThreadIDs = append(m.Cores[idx].ThreadIDs, t.ID)
		if m.Cores[idx].SocketID == 0 && t.SocketID != 0 {
			m.Cores[idx].SocketID = t.SocketID
		}
		if m.Cores[idx].NodeID == 0 && t.NodeID != 0 {
			m.Cores[idx].NodeID = t.NodeID
		}
		if m.Cores[idx].LocalID == 0 && t.CoreLocalID != 0 {
			m.Cores[idx].LocalID = t.CoreLocalID
		}
	}
	return nil
}

func ensureSocketLinks(m *MachineProfile) {
	socketByID := make(map[int]int, len(m.Sockets))
	for i := range m.Sockets {
		socketByID[m.Sockets[i].ID] = i
	}

	if len(m.Sockets) == 0 {
		for _, core := range m.Cores {
			if _, ok := socketByID[core.SocketID]; !ok {
				m.Sockets = append(m.Sockets, Socket{ID: core.SocketID})
				socketByID[core.SocketID] = len(m.Sockets) - 1
			}
		}
		for _, thread := range m.Threads {
			if _, ok := socketByID[thread.SocketID]; !ok {
				m.Sockets = append(m.Sockets, Socket{ID: thread.SocketID})
				socketByID[thread.SocketID] = len(m.Sockets) - 1
			}
		}
	}

	for _, core := range m.Cores {
		idx, ok := socketByID[core.SocketID]
		if !ok {
			m.Sockets = append(m.Sockets, Socket{ID: core.SocketID})
			idx = len(m.Sockets) - 1
			socketByID[core.SocketID] = idx
		}
		m.Sockets[idx].CoreIDs = append(m.Sockets[idx].CoreIDs, core.ID)
		m.Sockets[idx].NodeIDs = append(m.Sockets[idx].NodeIDs, core.NodeID)
		m.Sockets[idx].CPUs = append(m.Sockets[idx].CPUs, core.ThreadIDs...)
	}

	for i := range m.Sockets {
		m.Sockets[i].CPUs = SortedUniqueInts(m.Sockets[i].CPUs)
		m.Sockets[i].CoreIDs = SortedUniqueInts(m.Sockets[i].CoreIDs)
		m.Sockets[i].NodeIDs = SortedUniqueInts(m.Sockets[i].NodeIDs)
	}
}

func ensureNodeLinks(m *MachineProfile) {
	nodeByID := make(map[int]int, len(m.Nodes))
	for i := range m.Nodes {
		nodeByID[m.Nodes[i].ID] = i
	}
	for _, thread := range m.Threads {
		if thread.NodeID < 0 {
			continue
		}
		idx, ok := nodeByID[thread.NodeID]
		if !ok {
			m.Nodes = append(m.Nodes, NUMANode{ID: thread.NodeID})
			idx = len(m.Nodes) - 1
			nodeByID[thread.NodeID] = idx
		}
		m.Nodes[idx].CPUs = append(m.Nodes[idx].CPUs, thread.ID)
	}

	if len(m.Nodes) == 0 && len(m.Threads) > 0 && m.CPU.NUMANodes >= 0 {
		node := NUMANode{ID: 0}
		for _, thread := range m.Threads {
			node.CPUs = append(node.CPUs, thread.ID)
		}
		node.CPUs = SortedUniqueInts(node.CPUs)
		m.Nodes = append(m.Nodes, node)
	}

	for i := range m.Nodes {
		m.Nodes[i].CPUs = SortedUniqueInts(m.Nodes[i].CPUs)
	}
}

func computeMemoryDistribution(m *MachineProfile) {
	total := uint64(0)
	for _, node := range m.Nodes {
		total += node.MemTotalBytes
	}
	m.MemoryDistribution.Nodes = m.MemoryDistribution.Nodes[:0]
	if total == 0 {
		return
	}
	m.MemoryDistribution.TotalBytes = total
	for _, node := range m.Nodes {
		percent := float64(node.MemTotalBytes) * 100 / float64(total)
		m.MemoryDistribution.Nodes = append(m.MemoryDistribution.Nodes, NodeMemoryDistribution{
			NodeID:     node.ID,
			TotalBytes: node.MemTotalBytes,
			Percent:    percent,
		})
	}
}

func computeCPUCounts(m *MachineProfile) {
	online := make([]int, 0, len(m.Threads))
	offline := make([]int, 0)
	for _, thread := range m.Threads {
		if thread.Online {
			online = append(online, thread.ID)
		} else {
			offline = append(offline, thread.ID)
		}
	}
	if len(m.CPU.OnlineCPUs) == 0 {
		m.CPU.OnlineCPUs = online
	}
	if len(m.CPU.OfflineCPUs) == 0 {
		m.CPU.OfflineCPUs = offline
	}

	m.CPU.Sockets = len(m.Sockets)
	if len(m.Nodes) > 0 {
		m.CPU.NUMANodes = len(m.Nodes)
	}
	m.CPU.Cores = len(m.Cores)
	m.CPU.Threads = len(m.Threads)
	if m.CPU.Cores > 0 {
		m.CPU.ThreadsPerCore = m.CPU.Threads / m.CPU.Cores
	}
	if m.CPU.ThreadsPerCore < 1 {
		m.CPU.ThreadsPerCore = 1
	}
	m.CPU.SMT = m.CPU.ThreadsPerCore > 1
	m.Flags.SMT = m.CPU.SMT
	m.Flags.OfflineCPUs = SortedUniqueInts(m.CPU.OfflineCPUs)
	m.Flags.IsolatedCPUs = SortedUniqueInts(m.CPU.IsolatedCPUs)
}

func dedupeCaches(m *MachineProfile) {
	if len(m.Caches) == 0 {
		return
	}
	unique := make(map[string]Cache, len(m.Caches))
	for _, cache := range m.Caches {
		cache.SharedCPUList = SortedUniqueInts(cache.SharedCPUList)
		key := fmt.Sprintf("%d|%s|%d|%s", cache.Level, cache.Type, cache.SizeBytes, FormatIntSlice(cache.SharedCPUList))
		unique[key] = cache
	}
	m.Caches = m.Caches[:0]
	for _, cache := range unique {
		m.Caches = append(m.Caches, cache)
	}
}

func computeLLCGroups(m *MachineProfile) {
	if len(m.Caches) == 0 {
		return
	}
	maxLevel := 0
	for _, cache := range m.Caches {
		if cache.Level > maxLevel {
			maxLevel = cache.Level
		}
	}
	if maxLevel <= 0 {
		return
	}

	unique := make(map[string][]int)
	for _, cache := range m.Caches {
		if cache.Level != maxLevel {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(cache.Type))
		if kind != "" && kind != "unified" && maxLevel > 1 {
			continue
		}
		cpus := SortedUniqueInts(cache.SharedCPUList)
		if len(cpus) == 0 {
			continue
		}
		key := FormatIntSlice(cpus)
		unique[key] = cpus
	}
	if len(unique) == 0 {
		return
	}

	m.LLCGroups = m.LLCGroups[:0]
	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	keys = sortStrings(keys)
	for idx, key := range keys {
		cpus := unique[key]
		m.LLCGroups = append(m.LLCGroups, CPUGroup{
			ID:     idx,
			CPUs:   cpus,
			CPUSet: FormatCPUSet(cpus),
		})
	}
}

func ensureNonNilSlices(m *MachineProfile) {
	if m.Nodes == nil {
		m.Nodes = []NUMANode{}
	}
	if m.Sockets == nil {
		m.Sockets = []Socket{}
	}
	if m.Cores == nil {
		m.Cores = []Core{}
	}
	if m.Threads == nil {
		m.Threads = []Thread{}
	}
	if m.Caches == nil {
		m.Caches = []Cache{}
	}
	if m.LLCGroups == nil {
		m.LLCGroups = []CPUGroup{}
	}
	if m.MemoryDistribution.Nodes == nil {
		m.MemoryDistribution.Nodes = []NodeMemoryDistribution{}
	}
	if m.NumaDistance.NodeIDs == nil {
		m.NumaDistance.NodeIDs = []int{}
	}
	if m.NumaDistance.Matrix == nil {
		m.NumaDistance.Matrix = [][]int{}
	}
	if m.CPU.OnlineCPUs == nil {
		m.CPU.OnlineCPUs = []int{}
	}
	if m.CPU.OfflineCPUs == nil {
		m.CPU.OfflineCPUs = []int{}
	}
	if m.CPU.IsolatedCPUs == nil {
		m.CPU.IsolatedCPUs = []int{}
	}
	if m.Flags.IsolatedCPUs == nil {
		m.Flags.IsolatedCPUs = []int{}
	}
	if m.Flags.OfflineCPUs == nil {
		m.Flags.OfflineCPUs = []int{}
	}
	if m.Isolation.Isolated == nil {
		m.Isolation.Isolated = []int{}
	}
	if m.Isolation.NoHZFull == nil {
		m.Isolation.NoHZFull = []int{}
	}
	if m.Isolation.RCUNOCBS == nil {
		m.Isolation.RCUNOCBS = []int{}
	}
	if m.PCIeAffinity == nil {
		m.PCIeAffinity = []PCIEAffinity{}
	}
	if m.Posture.Checks == nil {
		m.Posture.Checks = []PostureCheck{}
	}
	if m.Microarch.RawFlags == nil {
		m.Microarch.RawFlags = []string{}
	}
	if strings.TrimSpace(m.Microarch.MicroarchName) == "" {
		m.Microarch.MicroarchName = "unknown"
	}
	if m.Warnings == nil {
		m.Warnings = []string{}
	}
}
