/*
 * Sift customization of dra-example-driver: publish a heterogeneous accelerator
 * fleet (from a Sift scenario) as ResourceSlices, instead of identical mock GPUs.
 * The matching/attribute logic lives in the pure github.com/FabianSalge/sift core
 * (golden rule); this file only converts its neutral output into resourceapi.
 */

package gpu

import (
	"bytes"
	_ "embed"
	"fmt"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	"github.com/FabianSalge/sift/config"
	"github.com/FabianSalge/sift/dra"
)

//go:embed realistic-2026.yaml
var fleetYAML []byte

// siftDevices loads the embedded Sift scenario fleet and renders each device as a
// resourceapi.Device.
func siftDevices() ([]resourceapi.Device, error) {
	fleet, err := config.LoadFleet(bytes.NewReader(fleetYAML))
	if err != nil {
		return nil, fmt.Errorf("loading sift fleet: %w", err)
	}
	devices := make([]resourceapi.Device, 0, len(fleet))
	for _, d := range fleet {
		devices = append(devices, toResourceDevice(dra.Describe(d)))
	}
	return devices, nil
}

// toResourceDevice converts the neutral dra.Device into a resourceapi.Device:
// attribute maps split by value kind, plus memory as a capacity quantity.
func toResourceDevice(nd dra.Device) resourceapi.Device {
	attrs := make(map[resourceapi.QualifiedName]resourceapi.DeviceAttribute, len(nd.StringAttrs)+len(nd.BoolAttrs)+len(nd.IntAttrs))
	for k, v := range nd.StringAttrs {
		attrs[resourceapi.QualifiedName(k)] = resourceapi.DeviceAttribute{StringValue: ptr.To(v)}
	}
	for k, v := range nd.BoolAttrs {
		attrs[resourceapi.QualifiedName(k)] = resourceapi.DeviceAttribute{BoolValue: ptr.To(v)}
	}
	for k, v := range nd.IntAttrs {
		attrs[resourceapi.QualifiedName(k)] = resourceapi.DeviceAttribute{IntValue: ptr.To(v)}
	}
	return resourceapi.Device{
		Name:       nd.Name,
		Attributes: attrs,
		Capacity: map[resourceapi.QualifiedName]resourceapi.DeviceCapacity{
			"memory": {Value: *resource.NewQuantity(int64(nd.MemoryGB*(1<<30)), resource.BinarySI)},
		},
	}
}
