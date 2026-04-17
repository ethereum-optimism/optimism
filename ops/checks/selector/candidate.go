package selector

import (
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// SignalContribution records one piece of evidence that fed a Candidate's
// aggregate signal. Keeping per-source contributions lets the explain
// command surface *why* a check was selected, and lets the optimizer
// reason about evidence quality later (e.g. down-weight stale sources).
type SignalContribution struct {
	Source       graph.EdgeSource `json:"source"`                 // coverage, static, ci_history, ai, manual
	EdgeKind     graph.EdgeKind   `json:"edge_kind,omitempty"`    // tested_by, imports, observed_correlation, ...
	Contribution float64          `json:"contribution"`           // signal this source contributed before aggregation
	Raw          map[string]any   `json:"raw,omitempty"`          // source-specific detail (hit_lines, total_changed, trigger, ...)
}

// Candidate is a single thing the selector *could* run — a (check, scope,
// profile) triple with an aggregated signal and the provenance that
// produced it.
//
// Phase 1 (Resolve) emits one Candidate per scope per profile for
// scopeable checks with coverage data, one unscoped Candidate per
// check for binary/trigger/blast-radius matches, and one Candidate per
// scope (profile="") when falling back to import-based reachability.
//
// Phase 2 (Optimize) reads Candidates and policy and produces an
// execution plan. Phase 2 does not touch the graph, the diff, or the
// catalog's trigger rules — all of that is Phase 1's responsibility.
type Candidate struct {
	CheckID    string               // catalog check type ID, e.g. "forge-test"
	Scope      string               // single scope arg ("" = run the whole check)
	Profile    string               // catalog profile name ("" = no profile env)
	Signal     float64              // aggregated relevance [0, 1]
	Provenance []SignalContribution // evidence that produced Signal

	// Prerequisites carries check IDs whose output this Candidate
	// consumes — in addition to the catalog's check-level prereqs
	// (e.g. forge-test always requires forge-build). These are
	// discovered at walk time from `produces` edges: if the reverse
	// walk from a changed source file passes through a node that
	// some check produces, that producer must run first for the
	// consumer's scope to be valid (e.g. gen-go-bindings must run
	// before go-test on the bindings' consumers).
	Prerequisites []string
}
