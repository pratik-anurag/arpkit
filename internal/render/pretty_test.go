package render_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pratik-anurag/arpkit/internal/posture"
	"github.com/pratik-anurag/arpkit/internal/render"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

func TestPrettyGoldenLayouts(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		wide   bool
		golden string
	}{
		{name: "default_80", width: 80, wide: false, golden: "pretty_default_80.golden"},
		{name: "wide_140", width: 140, wide: true, golden: "pretty_wide_140.golden"},
		{name: "narrow_60", width: 60, wide: false, golden: "pretty_narrow_60.golden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := loadFixture(t, "dual_socket_single_numa.json")
			if err := topology.Normalize(profile); err != nil {
				t.Fatalf("Normalize() error: %v", err)
			}
			profile.Posture = posture.Compute(profile)

			out, err := render.RenderPretty(profile, render.Options{
				ColorMode: "never",
				Profile:   "verbose",
				Version:   "test",
				Width:     tt.width,
				Wide:      tt.wide,
			})
			if err != nil {
				t.Fatalf("RenderPretty() error: %v", err)
			}

			goldenPath := filepath.Join("testdata", tt.golden)
			golden, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			if out != string(golden) {
				if os.Getenv("UPDATE_GOLDEN") == "1" {
					if writeErr := os.WriteFile(goldenPath, []byte(out), 0o644); writeErr != nil {
						t.Fatalf("update golden: %v", writeErr)
					}
					return
				}
				t.Fatalf("pretty output mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", tt.name, out, string(golden))
			}
		})
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
