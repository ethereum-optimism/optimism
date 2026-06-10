package txspam

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/sync/errgroup"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
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

func NewSyncEOA(includer txinclude.Includer, plan ...txplan.Option) *SyncEOA {
	return &SyncEOA{
		plan:     txplan.Combine(plan...),
		includer: includer,
	}
}

func NewSyncEOAFromPrivKey(priv *ecdsa.PrivateKey, chainID eth.ChainID, blockTime time.Duration, el txinclude.EL) *SyncEOA {
	signer := txinclude.NewPkSigner(priv, chainID.ToBig())
	const maxConcurrentTxs = 16 // Reth's mempool limits the number of txs per account to 16.
	reliableEL := txinclude.NewReliableEL(el, blockTime)
	return NewSyncEOA(txinclude.NewLimit(txinclude.NewPersistent(signer, reliableEL), maxConcurrentTxs), func(tx *txplan.PlannedTx) {
		tx.ChainID.Set(chainID)
	})
}

// Include attempts to include the transaction specified by opts.
func (eoa *SyncEOA) Include(ctx context.Context, opts ...txplan.Option) (*txinclude.IncludedTx, error) {
	unsigned, err := txplan.NewPlannedTx(eoa.plan, txplan.Combine(opts...)).Unsigned.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("create an unsigned transaction: %w", err)
	}
	return eoa.includer.Include(ctx, unsigned)
}

// FundEOAs generates numAccounts fresh EOAs on chainID and funds each with ethPerAccount, paying
// from eoa. It returns a ready-to-use SyncEOA for every funded account. Funding transactions are
// submitted concurrently; the first failure cancels the rest and is returned.
func (eoa *SyncEOA) FundEOAs(ctx context.Context, el txinclude.EL, chainID eth.ChainID, ethPerAccount eth.ETH, numAccounts uint64, blockTime time.Duration) ([]*SyncEOA, error) {
	eoas := make([]*SyncEOA, numAccounts)
	g, ctx := errgroup.WithContext(ctx)
	for i := range numAccounts {
		g.Go(func() error {
			priv, err := crypto.GenerateKey()
			if err != nil {
				return fmt.Errorf("generate key: %w", err)
			}
			addr := crypto.PubkeyToAddress(priv.PublicKey)
			if _, err := eoa.Include(ctx, txplan.WithTo(&addr), txplan.WithValue(ethPerAccount)); err != nil {
				return fmt.Errorf("fund %s: %w", addr, err)
			}
			eoas[i] = NewSyncEOAFromPrivKey(priv, chainID, blockTime, el)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return eoas, nil
}
