package graph

import (
	"testing"
)

// TestCheckPrerequisites_TransitiveViaArtifact — forge-build produces
// forge-artifacts consumed by interfaces-check, which produces
// interfaces consumed by forge-test. CheckPrerequisites(forge-test)
// returns {forge-build, interfaces-check} in lex order.
func TestCheckPrerequisites_TransitiveViaArtifact(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "check:forge-build", Kind: KindCheck})
	_ = g.AddNode(&Node{ID: "check:interfaces-check", Kind: KindCheck})
	_ = g.AddNode(&Node{ID: "check:forge-test", Kind: KindCheck})
	_ = g.AddNode(&Node{ID: "sol:src/Foo.sol", Kind: KindSource})
	_ = g.AddNode(&Node{ID: "artifact:forge-artifacts", Kind: KindArtifact})
	_ = g.AddNode(&Node{ID: "artifact:interfaces", Kind: KindArtifact})

	mk := func(from, to string, kind EdgeKind) {
		_ = g.AddEdge(&Edge{From: from, To: to, Kind: kind, Source: SourceStatic, Confidence: 1, Strength: 1})
	}
	mk("check:forge-build", "sol:src/Foo.sol", EdgeConsumes)
	mk("check:forge-build", "artifact:forge-artifacts", EdgeProduces)
	mk("check:interfaces-check", "artifact:forge-artifacts", EdgeConsumes)
	mk("check:interfaces-check", "artifact:interfaces", EdgeProduces)
	mk("check:forge-test", "artifact:interfaces", EdgeConsumes)
	mk("check:forge-test", "artifact:forge-artifacts", EdgeConsumes)

	got := CheckPrerequisites(g, "check:forge-test")
	if len(got) != 2 {
		t.Fatalf("expected 2 prereqs, got %d: %v", len(got), got)
	}
	// Lex sort: forge-build < interfaces-check.
	if got[0] != "forge-build" || got[1] != "interfaces-check" {
		t.Errorf("expected [forge-build, interfaces-check], got %v", got)
	}
}

func TestCheckPrerequisites_NoneFound(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "check:standalone", Kind: KindCheck})
	got := CheckPrerequisites(g, "check:standalone")
	if len(got) != 0 {
		t.Errorf("expected 0 prereqs for a check with no consumes edges, got %v", got)
	}
}

// TestCheckPrerequisites_DeterministicOrder — the same graph must
// return the same order every call, regardless of map iteration
// order.
func TestCheckPrerequisites_DeterministicOrder(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "check:consumer", Kind: KindCheck})
	_ = g.AddNode(&Node{ID: "artifact:a", Kind: KindArtifact})
	for _, id := range []string{"zzz", "aaa", "mmm", "bbb"} {
		_ = g.AddNode(&Node{ID: "check:" + id, Kind: KindCheck})
		_ = g.AddEdge(&Edge{From: "check:" + id, To: "artifact:a", Kind: EdgeProduces, Source: SourceStatic, Confidence: 1, Strength: 1})
	}
	_ = g.AddEdge(&Edge{From: "check:consumer", To: "artifact:a", Kind: EdgeConsumes, Source: SourceStatic, Confidence: 1, Strength: 1})

	first := CheckPrerequisites(g, "check:consumer")
	want := []string{"aaa", "bbb", "mmm", "zzz"}
	if len(first) != len(want) {
		t.Fatalf("len mismatch: got %v, want %v", first, want)
	}
	for i := range want {
		if first[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, first[i], want[i])
		}
	}
	// Run 10 more times; order must be identical.
	for i := 0; i < 10; i++ {
		again := CheckPrerequisites(g, "check:consumer")
		for j := range want {
			if again[j] != want[j] {
				t.Fatalf("run %d: non-deterministic order: got %v, want %v", i, again, want)
			}
		}
	}
}
