package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pratik-anurag/arpkit/internal/parser"
	"github.com/pratik-anurag/arpkit/internal/topology"
	"github.com/pratik-anurag/arpkit/internal/util"
)

type DiagramConfig struct {
	Width        int
	Wide         bool
	ColorEnabled bool
}

type diagramLayout int

const (
	layoutCompact diagramLayout = iota
	layoutStacked
	layoutSideBySide
)

type diagramNode struct {
	ID      int
	MemSize uint64
	CPUs    []int
	Sockets []diagramSocket
}

type diagramSocket struct {
	ID    int
	CPUs  []int
	Cores []diagramCore
}

type diagramCore struct {
	LocalID int
	Threads []int
}

type tokenLine struct {
	plain string
	text  string
	cores int
}

func RenderDiagram(m *topology.MachineProfile, cfg DiagramConfig) string {
	if m == nil || len(m.Cores) == 0 {
		return "n/a"
	}

	width := cfg.Width
	if width <= 0 {
		width = defaultTermWidth
	}
	if width < 44 {
		width = 44
	}

	layout := layoutForWidth(width)
	ansi := newANSI(cfg.ColorEnabled)
	nodes := buildDiagramNodes(m)
	if len(nodes) == 0 {
		return "n/a"
	}

	all := make([]string, 0, len(nodes)*10)
	for i, node := range nodes {
		nodeLines := renderNodeBox(node, width, layout, cfg.Wide, ansi)
		for _, line := range nodeLines {
			all = append(all, clampLine(line, width))
		}
		if i != len(nodes)-1 {
			all = append(all, "")
		}
	}
	return strings.Join(all, "\n")
}

func layoutForWidth(width int) diagramLayout {
	switch {
	case width >= 120:
		return layoutSideBySide
	case width >= 80:
		return layoutStacked
	default:
		return layoutCompact
	}
}

func renderNodeBox(node diagramNode, width int, layout diagramLayout, wide bool, ansi ansiStyle) []string {
	palette := paletteForNode(node.ID)
	title := buildNodeTitle(node)
	innerWidth := width - 2

	socketBlocks := make([][]string, 0, len(node.Sockets))
	if len(node.Sockets) == 0 {
		socketBlocks = append(socketBlocks, drawBox(innerWidth, "sockets", []string{"none"}))
	} else {
		switch {
		case layout == layoutSideBySide && len(node.Sockets) > 1:
			gap := 2
			colWidth := (innerWidth - gap) / 2
			if colWidth < 28 {
				for _, socket := range node.Sockets {
					socketBlocks = append(socketBlocks, renderSocketBox(socket, innerWidth, layoutStacked, wide, palette, ansi))
				}
			} else {
				row := make([][]string, 0, 2)
				for i, socket := range node.Sockets {
					row = append(row, renderSocketBox(socket, colWidth, layoutSideBySide, wide, palette, ansi))
					if len(row) == 2 || i == len(node.Sockets)-1 {
						socketBlocks = append(socketBlocks, composeHorizontal(row, gap))
						row = row[:0]
					}
				}
			}
		default:
			for _, socket := range node.Sockets {
				socketBlocks = append(socketBlocks, renderSocketBox(socket, innerWidth, layout, wide, palette, ansi))
			}
		}
	}

	nodeBody := make([]string, 0, 16)
	for i, block := range socketBlocks {
		nodeBody = append(nodeBody, block...)
		if i != len(socketBlocks)-1 {
			nodeBody = append(nodeBody, strings.Repeat(" ", innerWidth))
		}
	}

	box := drawBox(width, title, nodeBody)
	box = colorizeBorders(box, func(s string) string { return ansi.border(palette.Border, s) })
	box[0] = strings.Replace(box[0], title, ansi.title(palette.Title, title), 1)
	return box
}

func renderSocketBox(socket diagramSocket, width int, layout diagramLayout, wide bool, palette nodePalette, ansi ansiStyle) []string {
	title := fmt.Sprintf("S%d", socket.ID)
	contentWidth := width - 2
	body := renderSocketBody(socket, contentWidth, layout, wide, palette, ansi)
	box := drawBox(width, title, body)
	box = colorizeBorders(box, func(s string) string { return ansi.socket(palette.Socket, s) })
	box[0] = strings.Replace(box[0], title, ansi.title(palette.Socket, title), 1)
	return box
}

func renderSocketBody(socket diagramSocket, width int, layout diagramLayout, wide bool, palette nodePalette, ansi ansiStyle) []string {
	if len(socket.Cores) == 0 {
		return []string{"none"}
	}

	if wide {
		lines := make([]string, 0, len(socket.Cores))
		for _, core := range socket.Cores {
			line := formatCoreToken(core, width, false)
			lines = append(lines, styleCoreLine(line, palette, ansi))
		}
		return lines
	}

	if layout == layoutCompact {
		tokens := compactTokens(socket.Cores, width, palette, ansi)
		maxLines := 3
		if width < 56 {
			maxLines = 2
		}
		lines, shown := packTokenLines(tokens, width, maxLines)
		hidden := 0
		for i := shown; i < len(tokens); i++ {
			hidden += tokens[i].cores
		}
		if hidden > 0 {
			lines = append(lines, ansi.dim(fmt.Sprintf("+%d more", hidden)))
		}
		return lines
	}

	tokens := make([]tokenLine, 0, len(socket.Cores))
	for _, core := range socket.Cores {
		plain := formatCoreToken(core, width, false)
		tokens = append(tokens, tokenLine{plain: plain, text: styleCoreLine(plain, palette, ansi), cores: 1})
	}

	maxLines := 4
	if layout == layoutSideBySide {
		maxLines = 3
	}
	lines, shown := packTokenLines(tokens, width, maxLines)
	if shown < len(tokens) {
		hidden := len(tokens) - shown
		lines = append(lines, ansi.dim(fmt.Sprintf("+%d more", hidden)))
	}
	return lines
}

func compactTokens(cores []diagramCore, width int, palette nodePalette, ansi ansiStyle) []tokenLine {
	if len(cores) == 0 {
		return nil
	}
	groupSize := 4
	if width < 44 {
		groupSize = 8
	} else if width < 56 {
		groupSize = 6
	}

	out := make([]tokenLine, 0, (len(cores)+groupSize-1)/groupSize)
	for i := 0; i < len(cores); i += groupSize {
		end := i + groupSize
		if end > len(cores) {
			end = len(cores)
		}
		group := cores[i:end]
		threads := make([]int, 0, len(group)*2)
		for _, core := range group {
			threads = append(threads, core.Threads...)
		}
		threads = topology.SortedUniqueInts(threads)
		startID := group[0].LocalID
		endID := group[len(group)-1].LocalID

		cpus := parser.FormatCPUList(threads)
		plain := ""
		if startID == endID {
			plain = fmt.Sprintf("C%d [%s]", startID, cpus)
		} else {
			plain = fmt.Sprintf("C%d-%d [%s]", startID, endID, cpus)
		}
		if len([]rune(plain)) > width {
			plain = fmt.Sprintf("C%d..C%d (%dt)", startID, endID, len(threads))
		}
		if len([]rune(plain)) > width {
			plain = truncatePlain(plain, width)
		}
		out = append(out, tokenLine{plain: plain, text: styleCoreLine(plain, palette, ansi), cores: len(group)})
	}
	return out
}

func packTokenLines(tokens []tokenLine, width int, maxLines int) ([]string, int) {
	if len(tokens) == 0 {
		return []string{"none"}, 0
	}
	if width < 8 {
		width = 8
	}
	if maxLines <= 0 {
		maxLines = len(tokens) + 1
	}

	lines := make([]string, 0, maxLines)
	linePlain := ""
	lineText := ""
	shown := 0
	for i, token := range tokens {
		if linePlain == "" {
			linePlain = token.plain
			lineText = token.text
			shown = i + 1
			continue
		}

		candidatePlain := linePlain + "  " + token.plain
		if len([]rune(candidatePlain)) <= width {
			linePlain = candidatePlain
			lineText = lineText + "  " + token.text
			shown = i + 1
			continue
		}

		lines = append(lines, lineText)
		if len(lines) >= maxLines {
			return lines, shown
		}

		linePlain = token.plain
		lineText = token.text
		shown = i + 1
	}

	if lineText != "" {
		lines = append(lines, lineText)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return lines, shown
}

func formatCoreToken(core diagramCore, width int, compact bool) string {
	cpus := parser.FormatCPUList(core.Threads)
	plain := fmt.Sprintf("C%d [%s]", core.LocalID, cpus)
	if compact && len([]rune(plain)) > width {
		plain = fmt.Sprintf("C%d (%dt)", core.LocalID, len(core.Threads))
	}
	if len([]rune(plain)) > width {
		plain = fmt.Sprintf("C%d (%dt)", core.LocalID, len(core.Threads))
	}
	if len([]rune(plain)) > width {
		plain = truncatePlain(plain, width)
	}
	return plain
}

func styleCoreLine(line string, palette nodePalette, ansi ansiStyle) string {
	if !ansi.enabled {
		return line
	}
	parts := strings.SplitN(line, " ", 2)
	if len(parts) == 1 {
		return ansi.key(parts[0])
	}
	return ansi.key(parts[0]) + " " + ansi.cpu(palette.CPU, strings.TrimSpace(parts[1]))
}

func buildNodeTitle(node diagramNode) string {
	mem := "n/a"
	if node.MemSize > 0 {
		mem = util.HumanBytes(node.MemSize)
	}
	cpus := parser.FormatCPUList(node.CPUs)
	if cpus == "" {
		cpus = "n/a"
	}
	if node.ID < 0 {
		return fmt.Sprintf("NUMA N/A (mem %s | cpus %s)", mem, cpus)
	}
	return fmt.Sprintf("NUMA %d (mem %s | cpus %s)", node.ID, mem, cpus)
}

func buildDiagramNodes(m *topology.MachineProfile) []diagramNode {
	if m == nil {
		return nil
	}
	hasNUMA := len(m.Nodes) > 0 && m.CPU.NUMANodes > 0

	type nodeBuilder struct {
		ID      int
		MemSize uint64
		CPUs    []int
		Sockets map[int]*diagramSocket
	}

	builders := make(map[int]*nodeBuilder)
	if hasNUMA {
		for _, node := range m.Nodes {
			cpus := append([]int(nil), node.CPUs...)
			builders[node.ID] = &nodeBuilder{
				ID:      node.ID,
				MemSize: node.MemTotalBytes,
				CPUs:    cpus,
				Sockets: make(map[int]*diagramSocket),
			}
		}
	} else {
		all := append([]int(nil), m.CPU.OnlineCPUs...)
		if len(all) == 0 {
			for _, thread := range m.Threads {
				all = append(all, thread.ID)
			}
		}
		builders[-1] = &nodeBuilder{
			ID:      -1,
			MemSize: m.MemoryDistribution.TotalBytes,
			CPUs:    all,
			Sockets: make(map[int]*diagramSocket),
		}
	}

	threadsByCore := make(map[int][]int, len(m.Cores))
	for _, thread := range m.Threads {
		threadsByCore[thread.CoreID] = append(threadsByCore[thread.CoreID], thread.ID)
	}
	for coreID, ids := range threadsByCore {
		threadsByCore[coreID] = topology.SortedUniqueInts(ids)
	}

	cores := append([]topology.Core(nil), m.Cores...)
	sort.Slice(cores, func(i, j int) bool {
		if cores[i].NodeID != cores[j].NodeID {
			return cores[i].NodeID < cores[j].NodeID
		}
		if cores[i].SocketID != cores[j].SocketID {
			return cores[i].SocketID < cores[j].SocketID
		}
		if cores[i].LocalID != cores[j].LocalID {
			return cores[i].LocalID < cores[j].LocalID
		}
		return cores[i].ID < cores[j].ID
	})

	for _, core := range cores {
		nodeID := core.NodeID
		if !hasNUMA || nodeID < 0 {
			nodeID = -1
		}
		builder, ok := builders[nodeID]
		if !ok {
			builder = &nodeBuilder{ID: nodeID, Sockets: make(map[int]*diagramSocket)}
			builders[nodeID] = builder
		}
		socket, ok := builder.Sockets[core.SocketID]
		if !ok {
			socket = &diagramSocket{ID: core.SocketID}
			builder.Sockets[core.SocketID] = socket
		}
		threads := append([]int(nil), threadsByCore[core.ID]...)
		socket.Cores = append(socket.Cores, diagramCore{LocalID: core.LocalID, Threads: threads})
		socket.CPUs = append(socket.CPUs, threads...)
		builder.CPUs = append(builder.CPUs, threads...)
	}

	nodes := make([]diagramNode, 0, len(builders))
	for _, builder := range builders {
		socketIDs := make([]int, 0, len(builder.Sockets))
		for socketID := range builder.Sockets {
			socketIDs = append(socketIDs, socketID)
		}
		sort.Ints(socketIDs)

		sockets := make([]diagramSocket, 0, len(socketIDs))
		for _, socketID := range socketIDs {
			socket := builder.Sockets[socketID]
			socket.CPUs = topology.SortedUniqueInts(socket.CPUs)
			sort.Slice(socket.Cores, func(i, j int) bool { return socket.Cores[i].LocalID < socket.Cores[j].LocalID })
			sockets = append(sockets, *socket)
		}

		nodes = append(nodes, diagramNode{
			ID:      builder.ID,
			MemSize: builder.MemSize,
			CPUs:    topology.SortedUniqueInts(builder.CPUs),
			Sockets: sockets,
		})
	}

	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return nodes
}
