package gpu

import "testing"

func TestSiftDevicesSmoke(t *testing.T) {
	d, err := siftDevices()
	if err != nil {
		t.Fatalf("siftDevices error: %v", err)
	}
	if len(d) == 0 {
		t.Fatal("siftDevices returned 0 devices")
	}
	t.Logf("got %d devices; first=%s attrs=%v", len(d), d[0].Name, d[0].Attributes)
}
