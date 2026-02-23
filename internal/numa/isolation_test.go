package numa

import (
	"reflect"
	"testing"
)

func TestParseIsolationCmdline(t *testing.T) {
	cmdline := "BOOT_IMAGE=/vmlinuz-linux root=/dev/nvme0n1p2 rw " +
		"isolcpus=domain,managed_irq,2-7 nohz_full=2-7 rcu_nocbs=2-7,10-11"

	got := ParseIsolationCmdline(cmdline)

	if !reflect.DeepEqual(got.Isolated, []int{2, 3, 4, 5, 6, 7}) {
		t.Fatalf("isolated=%v want [2 3 4 5 6 7]", got.Isolated)
	}
	if !reflect.DeepEqual(got.NoHZFull, []int{2, 3, 4, 5, 6, 7}) {
		t.Fatalf("nohz_full=%v want [2 3 4 5 6 7]", got.NoHZFull)
	}
	if !reflect.DeepEqual(got.RCUNOCBS, []int{2, 3, 4, 5, 6, 7, 10, 11}) {
		t.Fatalf("rcu_nocbs=%v want [2 3 4 5 6 7 10 11]", got.RCUNOCBS)
	}
}

func TestUnionIsolation(t *testing.T) {
	info := ParseIsolationCmdline("isolcpus=1-3 nohz_full=3-5 rcu_nocbs=7")
	got := UnionIsolation(info)
	want := []int{1, 2, 3, 4, 5, 7}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("union=%v want=%v", got, want)
	}
}
