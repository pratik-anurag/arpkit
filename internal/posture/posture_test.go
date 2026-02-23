package posture

import (
	"reflect"
	"testing"

	"github.com/pratik-anurag/arpkit/internal/topology"
)

func TestComputeDeterministicScore(t *testing.T) {
	profile := &topology.MachineProfile{
		CPU: topology.CPUInfo{
			SMT:            true,
			ThreadsPerCore: 2,
		},
		NumaDistance: topology.NumaDistance{
			NodeIDs: []int{1, 0},
			Matrix: [][]int{
				{10, 20},
				{20, 10},
			},
		},
		MemoryDistribution: topology.MemoryDistribution{
			TotalBytes: 128 << 30,
			Nodes: []topology.NodeMemoryDistribution{
				{NodeID: 1, TotalBytes: 64 << 30, Percent: 50},
				{NodeID: 0, TotalBytes: 64 << 30, Percent: 50},
			},
		},
		Isolation: topology.IsolationInfo{
			Isolated: []int{2, 3},
		},
		Power: topology.PowerInfo{
			Governor: "performance",
		},
	}

	got1 := Compute(profile)
	got2 := Compute(profile)
	if !reflect.DeepEqual(got1, got2) {
		t.Fatalf("posture output is not deterministic:\nfirst=%+v\nsecond=%+v", got1, got2)
	}
	if got1.Score != 10.0 {
		t.Fatalf("score=%.1f want=10.0", got1.Score)
	}
	if len(got1.Checks) != 4 {
		t.Fatalf("checks=%d want=4", len(got1.Checks))
	}
}

func TestComputeWarnPaths(t *testing.T) {
	profile := &topology.MachineProfile{
		CPU: topology.CPUInfo{
			SMT:            false,
			ThreadsPerCore: 1,
		},
		NumaDistance: topology.NumaDistance{
			NodeIDs: []int{0, 1},
			Matrix: [][]int{
				{10, 60},
				{59, 10},
			},
		},
		MemoryDistribution: topology.MemoryDistribution{
			TotalBytes: 96 << 30,
			Nodes: []topology.NodeMemoryDistribution{
				{NodeID: 0, TotalBytes: 72 << 30, Percent: 75},
				{NodeID: 1, TotalBytes: 24 << 30, Percent: 25},
			},
		},
		Power: topology.PowerInfo{
			Governor: "powersave",
		},
	}

	got := Compute(profile)
	if got.Score >= 5.0 {
		t.Fatalf("score=%.1f expected a degraded score", got.Score)
	}
	if got.SMT != "SMT off" {
		t.Fatalf("smt note=%q want SMT off", got.SMT)
	}
}
