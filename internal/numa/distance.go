package numa

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pratik-anurag/arpkit/internal/parser"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

func ReadDistanceMatrix(fs *parser.Reader, nodeIDs []int) topology.NumaDistance {
	ids := append([]int(nil), nodeIDs...)
	if len(ids) == 0 {
		ids = discoverNodeIDs(fs)
	}
	ids = topology.SortedUniqueInts(ids)
	if len(ids) == 0 {
		return topology.NumaDistance{NodeIDs: []int{}, Matrix: [][]int{}}
	}

	matrix := make([][]int, 0, len(ids))
	for _, nodeID := range ids {
		text, err := fs.ReadFile(fmt.Sprintf("/sys/devices/system/node/node%d/distance", nodeID))
		if err != nil {
			matrix = append(matrix, make([]int, len(ids)))
			continue
		}
		row := ParseDistanceLine(text)
		if len(row) < len(ids) {
			padding := make([]int, len(ids)-len(row))
			row = append(row, padding...)
		}
		if len(row) > len(ids) {
			row = row[:len(ids)]
		}
		matrix = append(matrix, row)
	}

	return topology.NumaDistance{NodeIDs: ids, Matrix: matrix}
}

func ParseDistanceLine(text string) []int {
	fields := strings.Fields(strings.TrimSpace(text))
	out := make([]int, 0, len(fields))
	for _, field := range fields {
		v, err := strconv.Atoi(field)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	return out
}

func discoverNodeIDs(fs *parser.Reader) []int {
	entries, err := fs.ReadDir("/sys/devices/system/node")
	if err != nil {
		return nil
	}
	ids := make([]int, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "node") {
			continue
		}
		id, err := strconv.Atoi(strings.TrimPrefix(name, "node"))
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}
