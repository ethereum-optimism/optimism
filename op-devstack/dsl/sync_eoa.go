package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// SyncEOA plans transactions for an account and submits them through a
// txinclude.Includer. Nonce management, resubmission, and concurrency behavior
// are provided by the supplied includer.
type SyncEOA struct {
	plan     txplan.Option
	includer txinclude.Includer
}

// NewSyncEOA creates a SyncEOA backed by includer.
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
