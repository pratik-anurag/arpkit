package render

import "strings"

type Options struct {
	ColorMode  string
	ColorTheme string
	IsTTY      bool
	Width      int

	Profile string
	Only    map[string]struct{}

	NoDiagram bool
	NoPill    bool
	Compact   bool
	Wide      bool
	Unicode   bool
	Mem       bool
	Microarch bool
	Distance  bool
	PCIe      bool
	Posture   bool
	Debug     bool
	Version   string
}

func (o Options) ColorEnabled() bool {
	switch o.ColorMode {
	case "always":
		return true
	case "never":
		return false
	default:
		return o.IsTTY
	}
}

func ParseOnly(value string) map[string]struct{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	out := make(map[string]struct{})
	for _, part := range strings.Split(value, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		out[p] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (o Options) HasOnly() bool {
	return len(o.Only) > 0
}

func (o Options) sectionEnabled(section string) bool {
	switch section {
	case "microarch":
		if o.Microarch {
			return true
		}
	case "distance":
		if o.Distance {
			return true
		}
	case "pcie":
		if o.PCIe {
			return true
		}
	case "posture":
		if o.Posture {
			return true
		}
	}
	if len(o.Only) > 0 {
		_, ok := o.Only[section]
		return ok
	}
	for _, s := range defaultSections(o.Profile, o.Mem) {
		if s == section {
			return true
		}
	}
	return false
}

func defaultSections(profile string, forceMem bool) []string {
	switch profile {
	case "min":
		if forceMem {
			return []string{"summary", "mem"}
		}
		return []string{"summary"}
	case "verbose":
		return []string{"summary", "freq", "topology", "cache", "microarch", "power", "posture", "distance", "llc", "isolation", "pcie", "mem", "memtop", "notes"}
	default:
		sections := []string{"summary", "freq", "topology", "cache", "microarch", "power", "posture"}
		if forceMem {
			sections = append(sections, "mem")
		}
		return sections
	}
}
