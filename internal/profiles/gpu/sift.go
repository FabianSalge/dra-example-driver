/*
 * Sift customization of dra-example-driver: publish a heterogeneous accelerator
 * fleet (from a Sift scenario) as ResourceSlices, instead of identical mock GPUs.
 * The matching/attribute logic lives in the pure github.com/FabianSalge/sift core
 * (golden rule); this file only converts its neutral output into resourceapi.
 *
 * Topology is mirrored onto real nodes: each kube node advertises only the fleet
 * node it represents, named by its sift.dev/fleet-node label.
 */

package gpu

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strconv"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"

	"github.com/FabianSalge/sift/config"
	"github.com/FabianSalge/sift/dra"
)

//go:embed realistic-2026.yaml
var fleetYAML []byte

// fleetNodeLabel names which fleet node a kube node represents.
const fleetNodeLabel = "sift.dev/fleet-node"

// siftDevices renders the devices this kube node should advertise: the scenario
// fleet filtered to the fleet node named by this node's sift.dev/fleet-node label.
func siftDevices(nodeName string) ([]resourceapi.Device, error) {
	fleetNode, err := fleetNodeForKubeNode(nodeName)
	if err != nil {
		return nil, err
	}
	return siftDevicesForNode(fleetNode)
}

// siftDevicesForNode loads the embedded fleet and renders the devices on the given
// fleet node. Pure (no cluster), so it is unit-testable.
func siftDevicesForNode(fleetNode int) ([]resourceapi.Device, error) {
	fleet, err := config.LoadFleet(bytes.NewReader(fleetYAML))
	if err != nil {
		return nil, fmt.Errorf("loading sift fleet: %w", err)
	}
	var devices []resourceapi.Device
	for _, d := range fleet {
		if d.Node != fleetNode {
			continue
		}
		devices = append(devices, toResourceDevice(dra.Describe(d)))
	}
	return devices, nil
}

// fleetNodeForKubeNode reads the sift.dev/fleet-node label off this node (the
// driver already has nodes:get RBAC and NODE_NAME via the downward API).
func fleetNodeForKubeNode(nodeName string) (int, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return 0, fmt.Errorf("in-cluster config: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return 0, fmt.Errorf("kube client: %w", err)
	}
	node, err := cs.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("get node %q: %w", nodeName, err)
	}
	label, ok := node.Labels[fleetNodeLabel]
	if !ok {
		return 0, fmt.Errorf("node %q missing label %s", nodeName, fleetNodeLabel)
	}
	return strconv.Atoi(label)
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
