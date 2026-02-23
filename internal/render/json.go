package render

import (
	"encoding/json"
	"fmt"

	"github.com/pratik-anurag/arpkit/internal/topology"
)

func RenderJSON(m *topology.MachineProfile) (string, error) {
	if m == nil {
		return "", fmt.Errorf("nil machine profile")
	}
	m.Sort()
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}
