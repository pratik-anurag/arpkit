package numa

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pratik-anurag/arpkit/internal/parser"
)

func TestParseDistanceLine(t *testing.T) {
	got := ParseDistanceLine("10  21  31\n")
	want := []int{10, 21, 31}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseDistanceLine()=%v want=%v", got, want)
	}
}

func TestReadDistanceMatrixFromSysfsTree(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, root, "/sys/devices/system/node/node0/distance", "10 21\n")
	mustWrite(t, root, "/sys/devices/system/node/node1/distance", "21 10\n")

	fs := parser.NewReader(root)
	got := ReadDistanceMatrix(fs, nil)

	if !reflect.DeepEqual(got.NodeIDs, []int{0, 1}) {
		t.Fatalf("node ids=%v want [0 1]", got.NodeIDs)
	}
	wantMatrix := [][]int{{10, 21}, {21, 10}}
	if !reflect.DeepEqual(got.Matrix, wantMatrix) {
		t.Fatalf("matrix=%v want=%v", got.Matrix, wantMatrix)
	}
}

func TestReadDistanceMatrixPadsMissingRows(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, root, "/sys/devices/system/node/node0/distance", "10 20\n")

	fs := parser.NewReader(root)
	got := ReadDistanceMatrix(fs, []int{0, 1})
	want := [][]int{{10, 20}, {0, 0}}
	if !reflect.DeepEqual(got.Matrix, want) {
		t.Fatalf("matrix=%v want=%v", got.Matrix, want)
	}
}

func mustWrite(t *testing.T, root string, rel string, data string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
