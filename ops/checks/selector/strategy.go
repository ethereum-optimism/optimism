package selector

import "github.com/ethereum-optimism/optimism/ops/checks/catalog"

// Optimizer turns Candidates (Phase 1 output) into an execution plan.
//
// The optimizer receives only the candidate table, the stage, and the
// catalog — no graph, no diff. All evidence is already aggregated into
// per-candidate Signal and Provenance. This separation makes Phase 2
// pure policy: the same Candidates + stage + policy should always
// produce the same plan, regardless of how the signals were gathered.
type Optimizer interface {
	Optimize(
		candidates []Candidate,
		stage Stage,
		cat *catalog.Catalog,
	) (*Result, error)
}
