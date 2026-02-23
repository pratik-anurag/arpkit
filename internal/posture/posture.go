package posture

import (
	"fmt"
	"math"
	"strings"

	"github.com/pratik-anurag/arpkit/internal/topology"
)

const (
	statusPass    = "pass"
	statusWarn    = "warn"
	statusUnknown = "unknown"
)

func Compute(m *topology.MachineProfile) topology.Posture {
	if m == nil {
		return topology.Posture{Score: 0, Checks: []topology.PostureCheck{}}
	}

	checks := []topology.PostureCheck{
		checkNUMABalance(m),
		checkMemorySymmetry(m),
		checkIsolation(m),
		checkPowerGovernor(m),
	}

	score := 5.0
	for _, check := range checks {
		switch check.Name {
		case "NUMA balance":
			score += weightedDelta(check.Status, 2.0, -1.0)
		case "Memory symmetry":
			score += weightedDelta(check.Status, 1.5, -1.0)
		case "Isolation":
			score += weightedDelta(check.Status, 0.8, 0)
		case "Power governor":
			score += weightedDelta(check.Status, 0.8, -0.8)
		}
	}

	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}
	score = math.Round(score*10) / 10

	smtNote := "SMT off"
	if m.CPU.SMT {
		smtNote = fmt.Sprintf("SMT on (%d threads/core)", max(m.CPU.ThreadsPerCore, 1))
	}

	return topology.Posture{
		Score:  score,
		Checks: checks,
		SMT:    smtNote,
	}
}

func checkNUMABalance(m *topology.MachineProfile) topology.PostureCheck {
	if len(m.NumaDistance.NodeIDs) <= 1 {
		return topology.PostureCheck{Name: "NUMA balance", Status: statusPass, Reason: "single NUMA node"}
	}
	matrix := m.NumaDistance.Matrix
	n := len(m.NumaDistance.NodeIDs)
	if len(matrix) != n {
		return topology.PostureCheck{Name: "NUMA balance", Status: statusUnknown, Reason: "distance matrix unavailable"}
	}
	maxDistance := 0
	for i := 0; i < n; i++ {
		if len(matrix[i]) != n {
			return topology.PostureCheck{Name: "NUMA balance", Status: statusUnknown, Reason: "distance matrix unavailable"}
		}
		for j := 0; j < n; j++ {
			if matrix[i][j] != matrix[j][i] {
				return topology.PostureCheck{Name: "NUMA balance", Status: statusWarn, Reason: "distance matrix not symmetric"}
			}
			if i != j && matrix[i][j] > maxDistance {
				maxDistance = matrix[i][j]
			}
		}
	}
	if maxDistance <= 30 {
		return topology.PostureCheck{Name: "NUMA balance", Status: statusPass, Reason: fmt.Sprintf("max node distance %d", maxDistance)}
	}
	return topology.PostureCheck{Name: "NUMA balance", Status: statusWarn, Reason: fmt.Sprintf("high inter-node distance (%d)", maxDistance)}
}

func checkMemorySymmetry(m *topology.MachineProfile) topology.PostureCheck {
	if len(m.MemoryDistribution.Nodes) <= 1 {
		return topology.PostureCheck{Name: "Memory symmetry", Status: statusPass, Reason: "single NUMA memory pool"}
	}
	if m.MemoryDistribution.TotalBytes == 0 {
		return topology.PostureCheck{Name: "Memory symmetry", Status: statusUnknown, Reason: "memory distribution unavailable"}
	}
	minPct := m.MemoryDistribution.Nodes[0].Percent
	maxPct := m.MemoryDistribution.Nodes[0].Percent
	for _, node := range m.MemoryDistribution.Nodes[1:] {
		if node.Percent < minPct {
			minPct = node.Percent
		}
		if node.Percent > maxPct {
			maxPct = node.Percent
		}
	}
	delta := maxPct - minPct
	if delta <= 15 {
		return topology.PostureCheck{Name: "Memory symmetry", Status: statusPass, Reason: fmt.Sprintf("node spread %.1f%%", delta)}
	}
	return topology.PostureCheck{Name: "Memory symmetry", Status: statusWarn, Reason: fmt.Sprintf("node spread %.1f%%", delta)}
}

func checkIsolation(m *topology.MachineProfile) topology.PostureCheck {
	hasIsolation := len(m.Isolation.Isolated) > 0 || len(m.Isolation.NoHZFull) > 0 || len(m.Isolation.RCUNOCBS) > 0
	if !hasIsolation {
		return topology.PostureCheck{Name: "Isolation", Status: statusUnknown, Reason: "no kernel isolation parameters"}
	}
	parts := make([]string, 0, 3)
	if len(m.Isolation.Isolated) > 0 {
		parts = append(parts, "isolcpus")
	}
	if len(m.Isolation.NoHZFull) > 0 {
		parts = append(parts, "nohz_full")
	}
	if len(m.Isolation.RCUNOCBS) > 0 {
		parts = append(parts, "rcu_nocbs")
	}
	return topology.PostureCheck{Name: "Isolation", Status: statusPass, Reason: "configured: " + strings.Join(parts, ",")}
}

func checkPowerGovernor(m *topology.MachineProfile) topology.PostureCheck {
	gov := strings.ToLower(strings.TrimSpace(m.Power.Governor))
	if gov == "" || gov == "unknown" || gov == "n/a" {
		return topology.PostureCheck{Name: "Power governor", Status: statusUnknown, Reason: "governor unavailable"}
	}
	if strings.Contains(gov, "performance") {
		return topology.PostureCheck{Name: "Power governor", Status: statusPass, Reason: gov}
	}
	if strings.Contains(gov, "powersave") {
		return topology.PostureCheck{Name: "Power governor", Status: statusWarn, Reason: gov}
	}
	return topology.PostureCheck{Name: "Power governor", Status: statusUnknown, Reason: gov}
}

func weightedDelta(status string, passDelta float64, warnDelta float64) float64 {
	switch status {
	case statusPass:
		return passDelta
	case statusWarn:
		return warnDelta
	default:
		return 0
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
