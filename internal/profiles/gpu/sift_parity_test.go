package gpu

import (
	"bytes"
	"context"
	"testing"

	"k8s.io/dynamic-resource-allocation/cel"

	"github.com/FabianSalge/sift/allocator"
	"github.com/FabianSalge/sift/config"
	"github.com/FabianSalge/sift/dra"
)

// TestWorkloadSelectorParity is the ADR-0018 conformance check: the CEL the
// allocator emits (dra.WorkloadSelector) must select EXACTLY the devices that
// allocator.Feasible accepts — evaluated with the same DRA CEL engine the
// kube-scheduler runs, against the same ResourceSlice devices the driver
// publishes (toResourceDevice(dra.Describe(d))). If they ever diverge, the
// in-cluster scheduler and the reference model have drifted (golden rule break).
//
// Parity covers the HARD filter only — the sole thing CEL expresses. Soft
// cost-weighted scoring does not map to CEL and stays in the allocator/benchmark
// (ADR-0018, ADR-0024).
func TestWorkloadSelectorParity(t *testing.T) {
	// Bare attributes are published under the driver name as their domain, so the
	// emitted CEL and the cel.Device.Driver must use it (gpu.example.com).
	const domain = ProfileName + ".example.com"

	fleet, err := config.LoadFleet(bytes.NewReader(realisticFleetYAML))
	if err != nil {
		t.Fatalf("loading fleet: %v", err)
	}

	// A fixture engineered so every device is feasible for some workloads and
	// infeasible for others — parity is exercised in both directions, including
	// the memory boundary (h100=80GB vs floors of 80 and 100) and precision sets
	// no GPU has (int8) vs only one device has (fp4).
	workloads := []allocator.Workload{
		{Name: "train-bf16-80", Kind: allocator.KindTrain, MinMemoryGB: 80, RequiredPrecisions: []allocator.Precision{allocator.PrecisionBF16}},
		{Name: "infer-int8-32", Kind: allocator.KindInfer, MinMemoryGB: 32, RequiredPrecisions: []allocator.Precision{allocator.PrecisionINT8}},
		{Name: "infer-fp8-100", Kind: allocator.KindInfer, MinMemoryGB: 100, RequiredPrecisions: []allocator.Precision{allocator.PrecisionFP8}},
		{Name: "train-fp4-150", Kind: allocator.KindTrain, MinMemoryGB: 150, RequiredPrecisions: []allocator.Precision{allocator.PrecisionFP4}},
		{Name: "infer-multi-prec-80", Kind: allocator.KindInfer, MinMemoryGB: 80, RequiredPrecisions: []allocator.Precision{allocator.PrecisionBF16, allocator.PrecisionFP8}},
		{Name: "infer-loose", Kind: allocator.KindInfer},
		{Name: "train-impossible-mem", Kind: allocator.KindTrain, MinMemoryGB: 500, RequiredPrecisions: []allocator.Precision{allocator.PrecisionBF16}},
		{Name: "gang-train-bf16-80", Kind: allocator.KindTrain, MinMemoryGB: 80, RequiredPrecisions: []allocator.Precision{allocator.PrecisionBF16}, DeviceCount: 4, SameIsland: true},
	}

	compiler := cel.GetCompiler(cel.Features{})
	ctx := context.Background()

	for _, w := range workloads {
		req := dra.WorkloadSelector(w, domain)
		result := compiler.CompileCELExpression(req.Expression, cel.Options{})
		if result.Error != nil {
			t.Fatalf("workload %q: compiling %q: %v", w.Name, req.Expression, result.Error)
		}

		feasibleCount := 0
		for _, d := range fleet {
			rd := toResourceDevice(dra.Describe(d))
			match, _, err := result.DeviceMatches(ctx, cel.Device{
				Driver:     domain,
				Attributes: rd.Attributes,
				Capacity:   rd.Capacity,
			})
			if err != nil {
				t.Fatalf("workload %q, device %q: evaluating %q: %v", w.Name, d.ID, req.Expression, err)
			}
			want := allocator.Feasible(d, w)
			if match != want {
				t.Errorf("parity mismatch: workload %q device %q: CEL=%v allocator.Feasible=%v\n  expr: %s",
					w.Name, d.ID, match, want, req.Expression)
			}
			if want {
				feasibleCount++
			}
		}
		t.Logf("%-22s feasible on %d/%d devices  |  %s", w.Name, feasibleCount, len(fleet), req.Expression)
	}
}
