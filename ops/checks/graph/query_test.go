package graph

import (
	"math"
	"testing"
)

func buildTestGraph(t *testing.T) *Graph {
	t.Helper()
	g := NewGraph()

	// Source nodes: A -> B -> C
	_ = g.AddNode(&Node{ID: "src-a", Kind: KindSource, Name: "package A"})
	_ = g.AddNode(&Node{ID: "src-b", Kind: KindSource, Name: "package B"})
	_ = g.AddNode(&Node{ID: "src-c", Kind: KindSource, Name: "package C"})

	// Check nodes
	_ = g.AddNode(&Node{ID: "chk-a", Kind: KindCheck, Name: "test A"})
	_ = g.AddNode(&Node{ID: "chk-b", Kind: KindCheck, Name: "test B"})
	_ = g.AddNode(&Node{ID: "chk-c", Kind: KindCheck, Name: "test C"})

	// Prerequisite: build must run before chk-a
	_ = g.AddNode(&Node{ID: "build", Kind: KindCheck, Name: "build"})
	_ = g.AddEdge(&Edge{From: "build", To: "chk-a", Kind: EdgePrerequisite, Source: SourceManual, Confidence: 1, Strength: 1})

	// Import edges: A imports B, B imports C
	_ = g.AddEdge(&Edge{From: "src-a", To: "src-b", Kind: EdgeImports, Source: SourceStatic, Confidence: 1.0, Strength: 0.8})
	_ = g.AddEdge(&Edge{From: "src-b", To: "src-c", Kind: EdgeImports, Source: SourceStatic, Confidence: 1.0, Strength: 0.5})

	// tested_by edges
	_ = g.AddEdge(&Edge{From: "src-a", To: "chk-a", Kind: EdgeTestedBy, Source: SourceStatic, Confidence: 1.0, Strength: 0.9})
	_ = g.AddEdge(&Edge{From: "src-b", To: "chk-b", Kind: EdgeTestedBy, Source: SourceStatic, Confidence: 1.0, Strength: 0.9})
	_ = g.AddEdge(&Edge{From: "src-c", To: "chk-c", Kind: EdgeTestedBy, Source: SourceStatic, Confidence: 1.0, Strength: 0.9})

	return g
}

func TestReachableChecks_DirectDependency(t *testing.T) {
	g := buildTestGraph(t)

	results := ReachableChecks(g, []string{"src-a"}, 0.01)

	found := make(map[string]float64)
	for _, r := range results {
		found[r.CheckID] = r.Signal
	}

	// Direct: src-a -> chk-a with strength 0.9
	if signal, ok := found["chk-a"]; !ok {
		t.Error("expected chk-a to be reachable")
	} else if math.Abs(signal-0.9) > 0.001 {
		t.Errorf("expected chk-a signal ~0.9, got %f", signal)
	}
}

func TestReachableChecks_TransitiveDependency(t *testing.T) {
	g := buildTestGraph(t)

	results := ReachableChecks(g, []string{"src-a"}, 0.01)

	found := make(map[string]float64)
	for _, r := range results {
		found[r.CheckID] = r.Signal
	}

	// Transitive: src-a -> src-b (0.8) -> chk-b (0.9) = 0.72
	if signal, ok := found["chk-b"]; !ok {
		t.Error("expected chk-b to be reachable")
	} else if math.Abs(signal-0.72) > 0.001 {
		t.Errorf("expected chk-b signal ~0.72, got %f", signal)
	}

	// 2-hop transitive: src-a -> src-b (0.8) -> src-c (0.5) -> chk-c (0.9) = 0.36
	if signal, ok := found["chk-c"]; !ok {
		t.Error("expected chk-c to be reachable")
	} else if math.Abs(signal-0.36) > 0.001 {
		t.Errorf("expected chk-c signal ~0.36, got %f", signal)
	}
}

func TestReachableChecks_MinSignalCutoff(t *testing.T) {
	g := buildTestGraph(t)

	// With high min signal, distant checks are excluded
	results := ReachableChecks(g, []string{"src-a"}, 0.5)

	found := make(map[string]bool)
	for _, r := range results {
		found[r.CheckID] = true
	}

	if !found["chk-a"] {
		t.Error("expected chk-a (signal=0.9) to be reachable with minSignal=0.5")
	}
	if !found["chk-b"] {
		t.Error("expected chk-b (signal=0.72) to be reachable with minSignal=0.5")
	}
	if found["chk-c"] {
		t.Error("expected chk-c (signal=0.36) to be filtered with minSignal=0.5")
	}
}

func TestReachableChecks_NonexistentNode(t *testing.T) {
	g := buildTestGraph(t)
	results := ReachableChecks(g, []string{"nonexistent"}, 0.01)
	if len(results) != 0 {
		t.Errorf("expected 0 results for nonexistent node, got %d", len(results))
	}
}

func TestPrerequisites(t *testing.T) {
	g := buildTestGraph(t)
	prereqs := Prerequisites(g, "chk-a")
	if len(prereqs) != 1 {
		t.Fatalf("expected 1 prerequisite, got %d", len(prereqs))
	}
	if prereqs[0] != "build" {
		t.Errorf("expected prerequisite 'build', got %q", prereqs[0])
	}
}

func TestPrerequisites_TransitiveChain(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "step1", Kind: KindCheck, Name: "step1"})
	_ = g.AddNode(&Node{ID: "step2", Kind: KindCheck, Name: "step2"})
	_ = g.AddNode(&Node{ID: "step3", Kind: KindCheck, Name: "step3"})

	// step1 -> step2 -> step3 (prerequisite chain)
	_ = g.AddEdge(&Edge{From: "step1", To: "step2", Kind: EdgePrerequisite, Source: SourceManual, Confidence: 1, Strength: 1})
	_ = g.AddEdge(&Edge{From: "step2", To: "step3", Kind: EdgePrerequisite, Source: SourceManual, Confidence: 1, Strength: 1})

	prereqs := Prerequisites(g, "step3")
	if len(prereqs) != 2 {
		t.Fatalf("expected 2 prerequisites, got %d: %v", len(prereqs), prereqs)
	}
	// Should be in topological order: step1 before step2
	if prereqs[0] != "step1" || prereqs[1] != "step2" {
		t.Errorf("expected [step1, step2], got %v", prereqs)
	}
}

func TestPrerequisites_NoneFound(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "standalone", Kind: KindCheck, Name: "standalone"})
	prereqs := Prerequisites(g, "standalone")
	if len(prereqs) != 0 {
		t.Errorf("expected 0 prerequisites, got %d", len(prereqs))
	}
}
