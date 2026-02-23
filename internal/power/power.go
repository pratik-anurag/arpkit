package power

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/pratik-anurag/arpkit/internal/parser"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

func SnapshotLinux(fs *parser.Reader) topology.PowerInfo {
	info := topology.PowerInfo{Governor: "unknown", Driver: "unknown", TurboBoost: "unknown"}

	governors := map[string]struct{}{}
	drivers := map[string]struct{}{}

	if fs.Exists("/sys/devices/system/cpu/cpufreq") {
		entries, err := fs.ReadDir("/sys/devices/system/cpu/cpufreq")
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "policy") {
					continue
				}
				base := filepath.Join("/sys/devices/system/cpu/cpufreq", entry.Name())
				if governor, err := fs.ReadFile(filepath.Join(base, "scaling_governor")); err == nil && governor != "" {
					governors[governor] = struct{}{}
				}
				if driver, err := fs.ReadFile(filepath.Join(base, "scaling_driver")); err == nil && driver != "" {
					drivers[driver] = struct{}{}
				}
			}
		}
	}

	if len(governors) == 0 {
		if governor, err := fs.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor"); err == nil && governor != "" {
			governors[governor] = struct{}{}
		}
	}
	if len(drivers) == 0 {
		if driver, err := fs.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_driver"); err == nil && driver != "" {
			drivers[driver] = struct{}{}
		}
	}

	if len(governors) > 0 {
		info.Governor = joinSet(governors)
	}
	if len(drivers) > 0 {
		info.Driver = joinSet(drivers)
	}

	if fs.Exists("/sys/devices/system/cpu/intel_pstate/no_turbo") {
		if v, err := fs.ReadInt("/sys/devices/system/cpu/intel_pstate/no_turbo"); err == nil {
			switch v {
			case 0:
				info.TurboBoost = "enabled"
			case 1:
				info.TurboBoost = "disabled"
			default:
				info.TurboBoost = "unknown"
			}
		}
	} else if fs.Exists("/sys/devices/system/cpu/cpufreq/boost") {
		if v, err := fs.ReadInt("/sys/devices/system/cpu/cpufreq/boost"); err == nil {
			switch v {
			case 1:
				info.TurboBoost = "enabled"
			case 0:
				info.TurboBoost = "disabled"
			default:
				info.TurboBoost = "unknown"
			}
		}
	}

	return info
}

func joinSet(values map[string]struct{}) string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}
