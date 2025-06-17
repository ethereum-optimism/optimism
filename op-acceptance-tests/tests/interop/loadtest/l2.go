package loadtest

import (
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
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
	plan     txplan.Option
	includer txinclude.Includer
}

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

type L2 struct {
	Config      *params.ChainConfig
	BlockTime   time.Duration
	EL          *dsl.L2ELNode
	EOAs        *RoundRobin[*SyncEOA]
	EventLogger common.Address
	Wallet      *dsl.HDWallet
}

func (l2 *L2) DeployEventLogger(t devtest.T) {
	tx, err := l2.Include(t, txplan.WithData(common.FromHex(bindings.EventloggerBin)))
	t.Require().NoError(err)
	l2.EventLogger = tx.Receipt.ContractAddress
}

// Include includes the transaction on l2. It guarantees that the returned transaction was executed
// successfully when the error is non-nil.
func (l2 *L2) Include(t devtest.T, opts ...txplan.Option) (*txinclude.IncludedTx, error) {
	includedTx, err := l2.EOAs.Get().Include(t, opts...)
	if err != nil {
		return nil, err
	}
	t.Require().Equal(ethtypes.ReceiptStatusSuccessful, includedTx.Receipt.Status)
	return includedTx, nil
}
