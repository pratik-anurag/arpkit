package pcie

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pratik-anurag/arpkit/internal/parser"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

func TestScanLinuxFromFixtureTree(t *testing.T) {
	root := filepath.Join("testdata", "sysfs")
	fs := parser.NewReader(root)

	got := ScanLinux(fs)
	want := []topology.PCIEAffinity{
		{
			BDF:        "0000:3b:00.0",
			DeviceType: "net",
			Name:       "eth0",
			Class:      "0x020000",
			NUMANode:   1,
		},
		{
			BDF:        "0000:5e:00.0",
			DeviceType: "nvme",
			Name:       "nvme0",
			Class:      "0x010802",
			NUMANode:   0,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ScanLinux() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}
