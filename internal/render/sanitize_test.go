package render_test

import (
	"strings"
	"testing"

	"github.com/pratik-anurag/arpkit/internal/render"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

func TestRenderPrettyEscapesControlBytes(t *testing.T) {
	profile := &topology.MachineProfile{
		Metadata: topology.Metadata{
			OS:     "Fixture\x1b[31m Linux",
			Kernel: "6.8.0",
			Arch:   "amd64",
		},
		CPU: topology.CPUInfo{
			Architecture: "amd64",
			Vendor:       "Acme\tCPU",
			ModelName:    "Xeon\nHidden",
		},
		Microarch: topology.MicroarchInfo{
			MicroarchName: "Skylake",
			Vendor:        "Acme\tCPU",
		},
		PCIeAffinity: []topology.PCIEAffinity{
			{Name: "eth0\tblue", DeviceType: "net", NUMANode: 0},
		},
		Posture: topology.Posture{
			Checks: []topology.PostureCheck{
				{Name: "Power governor", Status: "warn", Reason: "powersave\x1b[2J"},
			},
			SMT: "SMT on\rboom",
		},
		Warnings: []string{"warn\rmsg"},
	}

	out, err := render.RenderPretty(profile, render.Options{
		ColorMode: "never",
		Profile:   "verbose",
		Width:     160,
		Debug:     true,
	})
	if err != nil {
		t.Fatalf("RenderPretty() error: %v", err)
	}

	for _, raw := range []string{"\x1b", "\r", "\t"} {
		if strings.Contains(out, raw) {
			t.Fatalf("output contains raw control byte %q\n%s", raw, out)
		}
	}
	for _, want := range []string{
		`Fixture\x1b[31m Linux`,
		`Xeon\nHidden`,
		`eth0\tblue`,
		`warn\rmsg`,
		`powersave\x1b[2J`,
		`SMT on\rboom`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\n%s", want, out)
		}
	}
}
