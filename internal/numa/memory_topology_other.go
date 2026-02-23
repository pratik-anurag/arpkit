//go:build !linux

package numa

import (
	"github.com/pratik-anurag/arpkit/internal/parser"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

func DetectMemoryTopology(fs *parser.Reader) topology.MemoryTopology {
	return topology.MemoryTopology{Known: false, DIMMsPopulated: -1, DIMMsTotal: -1, Channels: -1}
}
