package topology_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pratik-anurag/arpkit/internal/topology"
)

func TestNormalizeSingleSocket(t *testing.T) {
	profile := loadFixture(t, "single_socket.json")
	if err := topology.Normalize(profile); err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}

	if profile.CPU.Sockets != 1 {
		t.Fatalf("sockets=%d want=1", profile.CPU.Sockets)
	}
	if profile.CPU.NUMANodes != 1 {
		t.Fatalf("numa=%d want=1", profile.CPU.NUMANodes)
	}
	if profile.CPU.Cores != 2 {
		t.Fatalf("cores=%d want=2", profile.CPU.Cores)
	}
	if profile.CPU.Threads != 4 {
		t.Fatalf("threads=%d want=4", profile.CPU.Threads)
	}
	if !profile.CPU.SMT {
		t.Fatal("expected SMT on")
	}
	if len(profile.MemoryDistribution.Nodes) != 1 || profile.MemoryDistribution.Nodes[0].Percent != 100 {
		t.Fatalf("unexpected memory distribution: %+v", profile.MemoryDistribution.Nodes)
	}
}

func TestNormalizeDualSocketNUMA(t *testing.T) {
	profile := loadFixture(t, "dual_socket_numa.json")
	if err := topology.Normalize(profile); err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
	if profile.CPU.Sockets != 2 {
		t.Fatalf("sockets=%d want=2", profile.CPU.Sockets)
	}
	if profile.CPU.NUMANodes != 2 {
		t.Fatalf("numa=%d want=2", profile.CPU.NUMANodes)
	}
	if profile.MemoryDistribution.TotalBytes == 0 {
		t.Fatal("expected memory total to be computed")
	}
	if profile.Nodes[0].ID != 0 || profile.Nodes[1].ID != 1 {
		t.Fatalf("node order not deterministic: %+v", profile.Nodes)
	}
}

func TestNormalizeOfflineAndIsolated(t *testing.T) {
	profile := loadFixture(t, "offline_cpus.json")
	if err := topology.Normalize(profile); err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
	if len(profile.CPU.OfflineCPUs) != 2 {
		t.Fatalf("offline cpus=%v want [4 5]", profile.CPU.OfflineCPUs)
	}
	if len(profile.Flags.IsolatedCPUs) != 2 {
		t.Fatalf("isolated cpus flags=%v want [4 5]", profile.Flags.IsolatedCPUs)
	}
}

func TestNormalizeComputesLLCGroups(t *testing.T) {
	profile := &topology.MachineProfile{
		Cores: []topology.Core{
			{ID: 0, SocketID: 0, NodeID: 0, LocalID: 0},
			{ID: 1, SocketID: 0, NodeID: 0, LocalID: 1},
			{ID: 2, SocketID: 1, NodeID: 1, LocalID: 0},
			{ID: 3, SocketID: 1, NodeID: 1, LocalID: 1},
		},
		Threads: []topology.Thread{
			{ID: 0, CoreID: 0, CoreLocalID: 0, SocketID: 0, NodeID: 0, Online: true},
			{ID: 1, CoreID: 0, CoreLocalID: 0, SocketID: 0, NodeID: 0, Online: true},
			{ID: 2, CoreID: 1, CoreLocalID: 1, SocketID: 0, NodeID: 0, Online: true},
			{ID: 3, CoreID: 1, CoreLocalID: 1, SocketID: 0, NodeID: 0, Online: true},
			{ID: 4, CoreID: 2, CoreLocalID: 0, SocketID: 1, NodeID: 1, Online: true},
			{ID: 5, CoreID: 2, CoreLocalID: 0, SocketID: 1, NodeID: 1, Online: true},
			{ID: 6, CoreID: 3, CoreLocalID: 1, SocketID: 1, NodeID: 1, Online: true},
			{ID: 7, CoreID: 3, CoreLocalID: 1, SocketID: 1, NodeID: 1, Online: true},
		},
		Caches: []topology.Cache{
			{Level: 1, Type: "data", SizeBytes: 32 * 1024, SharedCPUList: []int{0, 1}},
			{Level: 1, Type: "data", SizeBytes: 32 * 1024, SharedCPUList: []int{2, 3}},
			{Level: 1, Type: "data", SizeBytes: 32 * 1024, SharedCPUList: []int{4, 5}},
			{Level: 1, Type: "data", SizeBytes: 32 * 1024, SharedCPUList: []int{6, 7}},
			{Level: 3, Type: "unified", SizeBytes: 32 * 1024 * 1024, SharedCPUList: []int{0, 1, 2, 3}},
			{Level: 3, Type: "unified", SizeBytes: 32 * 1024 * 1024, SharedCPUList: []int{4, 5, 6, 7}},
		},
	}

	if err := topology.Normalize(profile); err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
	if len(profile.LLCGroups) != 2 {
		t.Fatalf("llc groups=%d want=2", len(profile.LLCGroups))
	}
	if !reflect.DeepEqual(profile.LLCGroups[0].CPUs, []int{0, 1, 2, 3}) {
		t.Fatalf("group0 cpus=%v want [0 1 2 3]", profile.LLCGroups[0].CPUs)
	}
	if profile.LLCGroups[0].CPUSet != "0-3" {
		t.Fatalf("group0 cpuset=%q want 0-3", profile.LLCGroups[0].CPUSet)
	}
	if !reflect.DeepEqual(profile.LLCGroups[1].CPUs, []int{4, 5, 6, 7}) {
		t.Fatalf("group1 cpus=%v want [4 5 6 7]", profile.LLCGroups[1].CPUs)
	}
	if profile.LLCGroups[1].CPUSet != "4-7" {
		t.Fatalf("group1 cpuset=%q want 4-7", profile.LLCGroups[1].CPUSet)
	}
}

func TestSortNUMADistanceOrdering(t *testing.T) {
	profile := &topology.MachineProfile{
		NumaDistance: topology.NumaDistance{
			NodeIDs: []int{2, 0, 1},
			Matrix: [][]int{
				{10, 40, 30},
				{40, 10, 20},
				{30, 20, 10},
			},
		},
	}

	profile.Sort()
	if !reflect.DeepEqual(profile.NumaDistance.NodeIDs, []int{0, 1, 2}) {
		t.Fatalf("node ids=%v want [0 1 2]", profile.NumaDistance.NodeIDs)
	}

	want := [][]int{
		{10, 20, 40},
		{20, 10, 30},
		{40, 30, 10},
	}
	if !reflect.DeepEqual(profile.NumaDistance.Matrix, want) {
		t.Fatalf("matrix=%v want=%v", profile.NumaDistance.Matrix, want)
	}
}

func loadFixture(t *testing.T, name string) *topology.MachineProfile {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "fixtures", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var m topology.MachineProfile
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", name, err)
	}
	return &m
}
