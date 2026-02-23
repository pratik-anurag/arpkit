package numa

import (
	"regexp"
	"strings"

	"github.com/pratik-anurag/arpkit/internal/parser"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

var cpuSetTokenRE = regexp.MustCompile(`\d+(?:-\d+(?::\d+)?)?`)

func ParseIsolationCmdline(cmdline string) topology.IsolationInfo {
	out := topology.IsolationInfo{
		Isolated: []int{},
		NoHZFull: []int{},
		RCUNOCBS: []int{},
	}
	if strings.TrimSpace(cmdline) == "" {
		return out
	}
	for _, field := range strings.Fields(cmdline) {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		cpus := parseKernelCPUSet(parts[1])
		switch key {
		case "isolcpus":
			out.Isolated = append(out.Isolated, cpus...)
		case "nohz_full":
			out.NoHZFull = append(out.NoHZFull, cpus...)
		case "rcu_nocbs":
			out.RCUNOCBS = append(out.RCUNOCBS, cpus...)
		}
	}
	out.Isolated = topology.SortedUniqueInts(out.Isolated)
	out.NoHZFull = topology.SortedUniqueInts(out.NoHZFull)
	out.RCUNOCBS = topology.SortedUniqueInts(out.RCUNOCBS)
	return out
}

func parseKernelCPUSet(value string) []int {
	tokens := cpuSetTokenRE.FindAllString(value, -1)
	if len(tokens) == 0 {
		return nil
	}
	cpus, err := parser.ParseCPUList(strings.Join(tokens, ","))
	if err != nil {
		return nil
	}
	return cpus
}

func UnionIsolation(info topology.IsolationInfo) []int {
	out := make([]int, 0, len(info.Isolated)+len(info.NoHZFull)+len(info.RCUNOCBS))
	out = append(out, info.Isolated...)
	out = append(out, info.NoHZFull...)
	out = append(out, info.RCUNOCBS...)
	return topology.SortedUniqueInts(out)
}
