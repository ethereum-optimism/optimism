package selector

import (
	"math"
	"testing"
)

func TestComputeSchedule_NoDependencies(t *testing.T) {
	selections := []Selection{
		{CheckID: "a", RunCost: 100},
		{CheckID: "b", RunCost: 200},
		{CheckID: "c", RunCost: 150},
	}

	sched := ComputeSchedule(selections, 8)

	// All 3 checks have no prerequisites, so they run in 1 layer
	if len(sched.Layers) != 1 {
		t.Errorf("expected 1 layer, got %d", len(sched.Layers))
	}
	// Wall clock = max(100, 200, 150) = 200
	if math.Abs(sched.WallClock-200) > 0.01 {
		t.Errorf("expected wall clock 200, got %.0f", sched.WallClock)
	}
	// Total CPU = 100 + 200 + 150 = 450
	if math.Abs(sched.TotalCPU-450) > 0.01 {
		t.Errorf("expected total CPU 450, got %.0f", sched.TotalCPU)
	}
}

func TestComputeSchedule_WithPrerequisites(t *testing.T) {
	// build (180s) must run first, then test-l1 (1200s) and test-l2 (600s) in parallel
	selections := []Selection{
		{CheckID: "check:build", RunCost: 180},
		{CheckID: "check:test-l1", RunCost: 1200, Prerequisites: []string{"check:build"}},
		{CheckID: "check:test-l2", RunCost: 600, Prerequisites: []string{"check:build"}},
	}

	sched := ComputeSchedule(selections, 8)

	// Layer 0: build, Layer 1: test-l1 + test-l2
	if len(sched.Layers) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(sched.Layers))
	}

	// Wall clock = 180 + max(1200, 600) = 1380
	if math.Abs(sched.WallClock-1380) > 0.01 {
		t.Errorf("expected wall clock 1380, got %.0f", sched.WallClock)
	}

	// Total CPU = 180 + 1200 + 600 = 1980
	if math.Abs(sched.TotalCPU-1980) > 0.01 {
		t.Errorf("expected total CPU 1980, got %.0f", sched.TotalCPU)
	}
}

func TestComputeSchedule_ParallelismLimit(t *testing.T) {
	// 4 independent checks, parallelism limit of 2
	selections := []Selection{
		{CheckID: "a", RunCost: 100},
		{CheckID: "b", RunCost: 200},
		{CheckID: "c", RunCost: 150},
		{CheckID: "d", RunCost: 300},
	}

	sched := ComputeSchedule(selections, 2)

	// Should split into 2 sub-layers of 2 checks each
	if len(sched.Layers) != 2 {
		t.Errorf("expected 2 layers (parallelism=2), got %d", len(sched.Layers))
	}
}

func TestComputeSchedule_Chain(t *testing.T) {
	// Sequential chain: a -> b -> c
	selections := []Selection{
		{CheckID: "a", RunCost: 100},
		{CheckID: "b", RunCost: 200, Prerequisites: []string{"a"}},
		{CheckID: "c", RunCost: 300, Prerequisites: []string{"b"}},
	}

	sched := ComputeSchedule(selections, 8)

	// 3 layers, fully sequential
	if len(sched.Layers) != 3 {
		t.Errorf("expected 3 layers, got %d", len(sched.Layers))
	}
	// Wall clock = 100 + 200 + 300 = 600
	if math.Abs(sched.WallClock-600) > 0.01 {
		t.Errorf("expected wall clock 600, got %.0f", sched.WallClock)
	}
}

func TestComputeSchedule_Empty(t *testing.T) {
	sched := ComputeSchedule(nil, 8)
	if sched.WallClock != 0 {
		t.Errorf("expected 0 wall clock, got %.0f", sched.WallClock)
	}
}
