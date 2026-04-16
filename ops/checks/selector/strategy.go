package selector

import (
	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/policy"
)

// Stage is a development-lifecycle stage, aliased from policy so
// selector consumers don't need to import policy for the common case
// of passing a stage value through.
type Stage = policy.StageConfig

// Optimizer turns Candidates (Phase 1 output) into an execution plan.
//
// The optimizer receives only the candidate table, the stage, and the
// catalog — no graph, no diff. All evidence is already aggregated into
// per-candidate Signal and Provenance. Policy is captured at optimizer
// construction time.
type Optimizer interface {
	Optimize(
		candidates []Candidate,
		stage Stage,
		cat *catalog.Catalog,
	) (*Result, error)
}
