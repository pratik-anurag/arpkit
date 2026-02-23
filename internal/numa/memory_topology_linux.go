//go:build linux

package numa

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pratik-anurag/arpkit/internal/parser"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

var channelRE = regexp.MustCompile(`(?i)channel\s*([a-z0-9]+)`)

func DetectMemoryTopology(fs *parser.Reader) topology.MemoryTopology {
	mt := topology.MemoryTopology{Known: false, DIMMsPopulated: -1, DIMMsTotal: -1, Channels: -1}

	if !fs.Exists("/sys/devices/system/edac/mc") {
		return mt
	}
	mcEntries, err := fs.ReadDir("/sys/devices/system/edac/mc")
	if err != nil {
		return mt
	}

	totalSlots := 0
	populated := 0
	channels := map[string]struct{}{}

	for _, mc := range mcEntries {
		if !mc.IsDir() || !strings.HasPrefix(mc.Name(), "mc") {
			continue
		}
		mcPath := filepath.Join("/sys/devices/system/edac/mc", mc.Name())
		entries, err := fs.ReadDir(mcPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if !strings.HasPrefix(entry.Name(), "dimm") && !strings.HasPrefix(entry.Name(), "csrow") {
				continue
			}
			totalSlots++
			dimmPath := filepath.Join(mcPath, entry.Name())

			if loc, err := fs.ReadFile(filepath.Join(dimmPath, "location")); err == nil {
				if match := channelRE.FindStringSubmatch(loc); len(match) == 2 {
					channels[strings.ToUpper(match[1])] = struct{}{}
				}
			}

			if sizeText, err := fs.ReadFile(filepath.Join(dimmPath, "size")); err == nil {
				sizeText = strings.TrimSpace(sizeText)
				if sizeText != "" {
					if v, parseErr := strconv.ParseUint(sizeText, 10, 64); parseErr == nil && v > 0 {
						populated++
					}
				}
			}
		}
	}

	if totalSlots == 0 {
		return mt
	}

	mt.Known = true
	mt.DIMMsTotal = totalSlots
	mt.DIMMsPopulated = populated
	if len(channels) > 0 {
		mt.Channels = len(channels)
	}
	return mt
}
