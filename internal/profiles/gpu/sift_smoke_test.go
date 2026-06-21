package gpu

import (
	"strings"
	"testing"
)

// Each fleet node carries only its own devices (realistic-2026 topology).
func TestSiftDevicesForNode(t *testing.T) {
	want := map[int]int{0: 4, 1: 2, 2: 4, 3: 4, 4: 4} // fleet node -> device count
	for node, n := range want {
		d, err := siftDevicesForNode(node, realisticFleetYAML)
		if err != nil {
			t.Fatalf("node %d: %v", node, err)
		}
		if len(d) != n {
			t.Errorf("node %d: got %d devices, want %d", node, len(d), n)
		}
	}
	d0, err := siftDevicesForNode(0, realisticFleetYAML)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(d0[0].Name, "inferentia2-") {
		t.Errorf("node 0 first device = %s, want inferentia2-*", d0[0].Name)
	}
}

// The topology-witness fleet puts 8 H100s on fleet node 0, split across two
// islands (4 + 4) — the setup the same-island gang e2e needs.
func TestSiftDevicesTopologyWitness(t *testing.T) {
	d, err := siftDevicesForNode(0, topologyWitnessFleetYAML)
	if err != nil {
		t.Fatalf("node 0: %v", err)
	}
	if len(d) != 8 {
		t.Fatalf("node 0: got %d devices, want 8", len(d))
	}
	islandCount := map[int64]int{}
	for _, dev := range d {
		if !strings.HasPrefix(dev.Name, "h100-") {
			t.Errorf("device %s, want h100-*", dev.Name)
		}
		attr, ok := dev.Attributes["island"]
		if !ok || attr.IntValue == nil {
			t.Fatalf("device %s missing island attribute", dev.Name)
		}
		islandCount[*attr.IntValue]++
	}
	if len(islandCount) != 2 || islandCount[0] != 4 || islandCount[1] != 4 {
		t.Errorf("island distribution = %v, want {0:4, 1:4}", islandCount)
	}
}

// SIFT_SCENARIO selects which embedded fleet the driver publishes.
func TestScenarioFleetYAMLSelect(t *testing.T) {
	if got := scenarioFleetYAML(); string(got) != string(realisticFleetYAML) {
		t.Error("default scenario should be realistic-2026")
	}
	t.Setenv("SIFT_SCENARIO", "topology-witness")
	if got := scenarioFleetYAML(); string(got) != string(topologyWitnessFleetYAML) {
		t.Error("SIFT_SCENARIO=topology-witness should select the topology-witness fleet")
	}
}
