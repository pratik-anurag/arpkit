package render_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pratik-anurag/arpkit/internal/posture"
	"github.com/pratik-anurag/arpkit/internal/render"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

func TestRenderJSONGolden(t *testing.T) {
	profile := loadFixture(t, "dual_socket_numa.json")
	if err := topology.Normalize(profile); err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
	profile.Posture = posture.Compute(profile)

	out, err := render.RenderJSON(profile)
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "dual_socket_numa.json.golden")
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.Getenv("UPDATE_GOLDEN") == "1" {
			if writeErr := os.WriteFile(goldenPath, []byte(out), 0o644); writeErr != nil {
				t.Fatalf("write golden: %v", writeErr)
			}
			return
		}
		t.Fatalf("read golden: %v", err)
	}

	if out != string(golden) {
		if os.Getenv("UPDATE_GOLDEN") == "1" {
			if writeErr := os.WriteFile(goldenPath, []byte(out), 0o644); writeErr != nil {
				t.Fatalf("update golden: %v", writeErr)
			}
			return
		}
		t.Fatalf("json output mismatch\n--- got ---\n%s\n--- want ---\n%s", out, string(golden))
	}
}

func TestRenderJSONDeterministicOrdering(t *testing.T) {
	profile := loadFixture(t, "dual_socket_numa.json")
	if err := topology.Normalize(profile); err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}
	profile.Posture = posture.Compute(profile)

	out1, err := render.RenderJSON(profile)
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	reverseNodes(profile)
	reverseSockets(profile)
	reverseCores(profile)
	reverseThreads(profile)
	reverseLLCGroups(profile)

	out2, err := render.RenderJSON(profile)
	if err != nil {
		t.Fatalf("RenderJSON() second error: %v", err)
	}

	if out1 != out2 {
		t.Fatalf("json output is not deterministic")
	}
}

func reverseNodes(m *topology.MachineProfile) {
	for i, j := 0, len(m.Nodes)-1; i < j; i, j = i+1, j-1 {
		m.Nodes[i], m.Nodes[j] = m.Nodes[j], m.Nodes[i]
	}
}

func reverseSockets(m *topology.MachineProfile) {
	for i, j := 0, len(m.Sockets)-1; i < j; i, j = i+1, j-1 {
		m.Sockets[i], m.Sockets[j] = m.Sockets[j], m.Sockets[i]
	}
}

func reverseCores(m *topology.MachineProfile) {
	for i, j := 0, len(m.Cores)-1; i < j; i, j = i+1, j-1 {
		m.Cores[i], m.Cores[j] = m.Cores[j], m.Cores[i]
	}
}

func reverseThreads(m *topology.MachineProfile) {
	for i, j := 0, len(m.Threads)-1; i < j; i, j = i+1, j-1 {
		m.Threads[i], m.Threads[j] = m.Threads[j], m.Threads[i]
	}
}

func reverseLLCGroups(m *topology.MachineProfile) {
	for i, j := 0, len(m.LLCGroups)-1; i < j; i, j = i+1, j-1 {
		m.LLCGroups[i], m.LLCGroups[j] = m.LLCGroups[j], m.LLCGroups[i]
	}
}
