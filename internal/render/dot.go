package render

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/pratik-anurag/arpkit/internal/topology"
)

func RenderDOT(m *topology.MachineProfile) (string, error) {
	if m == nil {
		return "", fmt.Errorf("nil machine profile")
	}
	m.Sort()

	buf := bytes.NewBuffer(make([]byte, 0, 4096))
	buf.WriteString("digraph arpkit {\n")
	buf.WriteString("  rankdir=LR;\n")
	buf.WriteString("  node [shape=box, fontsize=10];\n")

	hasNUMA := len(m.Nodes) > 0 && m.CPU.NUMANodes > 0
	if !hasNUMA {
		buf.WriteString("  root [label=\"NUMA N/A\", shape=oval];\n")
	}

	for _, node := range m.Nodes {
		fmt.Fprintf(buf, "  numa_%d [label=\"NUMA%d\", shape=oval];\n", node.ID, node.ID)
	}

	for _, socket := range m.Sockets {
		fmt.Fprintf(buf, "  socket_%d [label=\"Socket %d\"];\n", socket.ID, socket.ID)
		if hasNUMA && len(socket.NodeIDs) > 0 {
			nodeIDs := append([]int(nil), socket.NodeIDs...)
			sort.Ints(nodeIDs)
			fmt.Fprintf(buf, "  numa_%d -> socket_%d;\n", nodeIDs[0], socket.ID)
		} else {
			fmt.Fprintf(buf, "  root -> socket_%d;\n", socket.ID)
		}
	}

	for _, core := range m.Cores {
		fmt.Fprintf(buf, "  core_%d [label=\"Core %d\"];\n", core.ID, core.LocalID)
		fmt.Fprintf(buf, "  socket_%d -> core_%d;\n", core.SocketID, core.ID)
	}

	for _, thread := range m.Threads {
		fmt.Fprintf(buf, "  cpu_%d [label=\"CPU %d\"];\n", thread.ID, thread.ID)
		fmt.Fprintf(buf, "  core_%d -> cpu_%d;\n", thread.CoreID, thread.ID)
	}

	buf.WriteString("}\n")
	return buf.String(), nil
}
