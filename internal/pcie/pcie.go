package pcie

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pratik-anurag/arpkit/internal/parser"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

func ScanLinux(fs *parser.Reader) []topology.PCIEAffinity {
	if !fs.Exists("/sys/bus/pci/devices") {
		return nil
	}
	entries, err := fs.ReadDir("/sys/bus/pci/devices")
	if err != nil {
		return nil
	}

	out := make([]topology.PCIEAffinity, 0)
	for _, entry := range entries {
		name := entry.Name()
		devicePath := filepath.Join("/sys/bus/pci/devices", name)
		classText, err := fs.ReadFile(filepath.Join(devicePath, "class"))
		if err != nil {
			continue
		}
		classText = strings.TrimSpace(strings.ToLower(classText))
		classCode, err := parseHex(classText)
		if err != nil {
			continue
		}

		deviceType := ""
		base := classCode >> 16
		sub := (classCode >> 8) & 0xff
		switch {
		case base == 0x02:
			deviceType = "net"
		case base == 0x01 && sub == 0x08:
			deviceType = "nvme"
		default:
			continue
		}

		numaNode := -1
		if v, err := fs.ReadInt(filepath.Join(devicePath, "numa_node")); err == nil {
			numaNode = v
		}

		names := []string{}
		switch deviceType {
		case "net":
			names = readNames(fs, filepath.Join(devicePath, "net"))
		case "nvme":
			names = readNames(fs, filepath.Join(devicePath, "nvme"))
		}
		if len(names) == 0 {
			names = []string{name}
		}

		for _, devName := range names {
			out = append(out, topology.PCIEAffinity{
				BDF:        name,
				DeviceType: deviceType,
				Name:       devName,
				Class:      classText,
				NUMANode:   numaNode,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].DeviceType != out[j].DeviceType {
			return out[i].DeviceType < out[j].DeviceType
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].BDF < out[j].BDF
	})
	return out
}

func readNames(fs *parser.Reader, path string) []string {
	if !fs.Exists(path) {
		return nil
	}
	entries, err := fs.ReadDir(path)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Name())
	}
	sort.Strings(out)
	return out
}

func parseHex(value string) (uint64, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "0x")
	return strconv.ParseUint(value, 16, 64)
}
