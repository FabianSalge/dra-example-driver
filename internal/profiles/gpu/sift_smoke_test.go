package gpu

import (
	"strings"
	"testing"
)

// Each fleet node carries only its own devices (realistic-2026 topology).
func TestSiftDevicesForNode(t *testing.T) {
	want := map[int]int{0: 4, 1: 2, 2: 4, 3: 4, 4: 4} // fleet node -> device count
	for node, n := range want {
		d, err := siftDevicesForNode(node)
		if err != nil {
			t.Fatalf("node %d: %v", node, err)
		}
		if len(d) != n {
			t.Errorf("node %d: got %d devices, want %d", node, len(d), n)
		}
	}
	d0, err := siftDevicesForNode(0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(d0[0].Name, "inferentia2-") {
		t.Errorf("node 0 first device = %s, want inferentia2-*", d0[0].Name)
	}
}
