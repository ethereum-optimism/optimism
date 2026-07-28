package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// SyncEOA sends transactions from a single account through a txinclude.Includer,
// which manages nonces and resubmission. Because the includer serializes nonce
// assignment, many transactions can be submitted concurrently and each awaited to
// inclusion. It underpins concurrent funding (see FunderEOA) and the load tests.
type SyncEOA struct {
	plan     txplan.Option
	includer txinclude.Includer
}

// NewSyncEOA creates a SyncEOA whose nonce and concurrency behavior comes from includer.
func NewSyncEOA(includer txinclude.Includer, plan txplan.Option) *SyncEOA {
	return &SyncEOA{
		plan:     plan,
		includer: includer,
	}
}

// Include attempts to include the transaction specified by opts.
func (eoa *SyncEOA) Include(t devtest.T, opts ...txplan.Option) (*txinclude.IncludedTx, error) {
	unsigned, err := txplan.NewPlannedTx(eoa.plan, txplan.Combine(opts...)).Unsigned.Eval(t.Ctx())
	if err != nil {
		return nil, err
	}
	return eoa.includer.Include(t.Ctx(), unsigned)
}
