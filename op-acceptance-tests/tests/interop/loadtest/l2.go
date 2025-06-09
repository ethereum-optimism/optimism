package loadtest

import (
	"context"
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type RoundRobin[T any] struct {
	items []T
	index atomic.Uint64
}

func NewRoundRobin[T any](items []T) *RoundRobin[T] {
	return &RoundRobin[T]{
		items: items,
	}
}

func (p *RoundRobin[T]) Get() T {
	next := (p.index.Add(1) - 1) % uint64(len(p.items))
	return p.items[next]
}

type SyncEOA struct {
	Plan     txplan.Option
	Includer txinclude.Includer
}

type L2 struct {
	Config       *params.ChainConfig
	RollupConfig *rollup.Config
	EL           *dsl.L2ELNode
	EOAs         *RoundRobin[*SyncEOA]
	EventLogger  common.Address
}

func (l2 *L2) DeployEventLogger(ctx context.Context, t devtest.T) {
	tx, err := l2.Include(ctx, t, txplan.WithData(common.FromHex(bindings.EventloggerBin)))
	t.Require().NoError(err)
	l2.EventLogger = tx.Receipt.ContractAddress
}

func (l2 *L2) Include(ctx context.Context, t devtest.T, opts ...txplan.Option) (*txinclude.IncludedTx, error) {
	eoa := l2.EOAs.Get()
	unsigned, err := txplan.NewPlannedTx(eoa.Plan, txplan.Combine(opts...)).Unsigned.Eval(ctx)
	if err != nil {
		// Context cancelations and i/o timeouts can cause an error (there may be other scenarios, too).
		// Let the caller handle it.
		return nil, err
	}
	includedTx, err := eoa.Includer.Include(ctx, unsigned)
	if err != nil {
		return nil, err // Allow the caller to check for budget overdrafts, context cancelation, etc.
	}
	t.Require().Equal(ethtypes.ReceiptStatusSuccessful, includedTx.Receipt.Status)
	return includedTx, nil
}
