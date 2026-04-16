package selector

import (
	"math"
	"testing"
)

func TestComputeSchedule_NoDependencies(t *testing.T) {
	items := []ExecutionItem{
		{ID: "a", RunCost: 100},
		{ID: "b", RunCost: 200},
		{ID: "c", RunCost: 150},
	}
	sched := ComputeSchedule(items, 8)
	if len(sched.Layers) != 1 {
		t.Errorf("expected 1 layer, got %d", len(sched.Layers))
	}
	if math.Abs(sched.WallClock-200) > 0.01 {
		t.Errorf("expected wall clock 200, got %.0f", sched.WallClock)
	}
	if math.Abs(sched.TotalCPU-450) > 0.01 {
		t.Errorf("expected total CPU 450, got %.0f", sched.TotalCPU)
	}
}

func TestComputeSchedule_WithPrerequisites(t *testing.T) {
	items := []ExecutionItem{
		{ID: "build", RunCost: 180},
		{ID: "test-l1", RunCost: 1200, Prerequisites: []string{"build"}},
		{ID: "test-l2", RunCost: 600, Prerequisites: []string{"build"}},
	}
	sched := ComputeSchedule(items, 8)
	if len(sched.Layers) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(sched.Layers))
	}
	if math.Abs(sched.WallClock-1380) > 0.01 {
		t.Errorf("expected wall clock 1380, got %.0f", sched.WallClock)
	}
}

func TestComputeSchedule_ParallelismLimit(t *testing.T) {
	// 4 independent items, all at the same prereq depth. A slot-limited
	// dispatcher with 2 slots produces one scheduling layer whose
	// makespan is bounded below by the LPT estimate:
	//   max(longest_item, total_cpu / slots) = max(300, 750/2) = 375
	items := []ExecutionItem{
		{ID: "a", RunCost: 100},
		{ID: "b", RunCost: 200},
		{ID: "c", RunCost: 150},
		{ID: "d", RunCost: 300},
	}
	sched := ComputeSchedule(items, 2)
	if len(sched.Layers) != 1 {
		t.Errorf("expected 1 layer (no prereqs), got %d", len(sched.Layers))
	}
	if math.Abs(sched.WallClock-375) > 0.01 {
		t.Errorf("expected wall clock 375 (LPT bound at parallelism=2), got %.2f", sched.WallClock)
	}
	if math.Abs(sched.TotalCPU-750) > 0.01 {
		t.Errorf("expected total CPU 750, got %.2f", sched.TotalCPU)
	}
}

func TestComputeSchedule_ParallelismSaturated(t *testing.T) {
	// When one item dominates, wall clock = its duration regardless of
	// parallelism. LPT bound: max(1000, 1300/4) = 1000.
	items := []ExecutionItem{
		{ID: "big", RunCost: 1000},
		{ID: "a", RunCost: 100},
		{ID: "b", RunCost: 100},
		{ID: "c", RunCost: 100},
	}
	sched := ComputeSchedule(items, 4)
	if math.Abs(sched.WallClock-1000) > 0.01 {
		t.Errorf("expected wall clock 1000 (dominated by big), got %.2f", sched.WallClock)
	}
}

func TestComputeSchedule_Chain(t *testing.T) {
	items := []ExecutionItem{
		{ID: "a", RunCost: 100},
		{ID: "b", RunCost: 200, Prerequisites: []string{"a"}},
		{ID: "c", RunCost: 300, Prerequisites: []string{"b"}},
	}
	sched := ComputeSchedule(items, 8)
	if len(sched.Layers) != 3 {
		t.Errorf("expected 3 layers, got %d", len(sched.Layers))
	}
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

func TestComputeSchedule_UniqueIDs(t *testing.T) {
	// Two items with the same CheckTypeID but different IDs
	items := []ExecutionItem{
		{ID: "forge-test:L1", CheckTypeID: "forge-test", RunCost: 1200},
		{ID: "forge-test:L2", CheckTypeID: "forge-test", RunCost: 600},
		{ID: "forge-build", CheckTypeID: "forge-build", RunCost: 180},
	}
	// L1 and L2 depend on build
	items[0].Prerequisites = []string{"forge-build"}
	items[1].Prerequisites = []string{"forge-build"}

	sched := ComputeSchedule(items, 8)
	if len(sched.Layers) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(sched.Layers))
	}
	// Layer 1: build (180s), Layer 2: L1+L2 in parallel (1200s)
	if math.Abs(sched.WallClock-1380) > 0.01 {
		t.Errorf("expected wall clock 1380, got %.0f", sched.WallClock)
	}
}
