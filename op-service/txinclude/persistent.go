package txinclude

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
)

// Persistent is an Includer that persists transactions to an execution layer.
// It owns the account used by its signer: callers must not submit transactions
// from that account through another path while the Persistent is in use.
type Persistent struct {
	cfg    *persistentConfig
	signer Signer
	el     EL
}

var _ Includer = (*Persistent)(nil)

type persistentConfig struct {
	nonces *nonceManager
}

type PersistentOption func(cfg *persistentConfig)

// WithStartNonce instructs the Persistent includer to use nonce as a starting nonce. The default
// is zero.
func WithStartNonce(nonce uint64) PersistentOption {
	return func(cfg *persistentConfig) {
		cfg.nonces = newNonceManager(nonce)
	}
}

// NewPersistent creates a Persistent Includer. The returned includer exclusively
// owns the account used by signer and manages all of its transaction nonces.
//
// It assumes el is reliable:
//   - el.SendTransaction guarantees mempool inclusion without the possibility of eviction.
//   - el.TransactionReceipt will return a valid receipt if one eventually exists.
func NewPersistent(signer Signer, el EL, opts ...PersistentOption) *Persistent {
	cfg := &persistentConfig{
		nonces: newNonceManager(0),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &Persistent{
		cfg:    cfg,
		signer: signer,
		el:     el,
	}
}

// Include attempts to persist tx to p's EL. It fails on an error from the context, signer, or
// EL, except that it takes the next nonce on a too-low nonce and retries an underpriced
// transaction unchanged (fee bumping is TODO(16368)).
func (p *Persistent) Include(ctx context.Context, tx types.TxData) (*IncludedTx, error) {
	// These channels could also go at the top of p.try, but we reuse them across
	// p.try calls for efficiency.
	errCh := make(chan error)
	defer close(errCh)
	includedCh := make(chan *IncludedTx)
	defer close(includedCh)

	unincluded := newUnincludedTx(tx)
	unincluded.SetNonce(p.cfg.nonces.Next())
	state := &tryState{Transaction: unincluded}

	var included *IncludedTx
	var err error

	for {
		included, state, err = p.try(ctx, state, includedCh, errCh)
		if err != nil {
			return nil, err
		}
		if state != nil {
			continue
		}
		return included, nil
	}
}

type tryState struct {
	Transaction *unincludedTx
}

// try attempts to include the state.Transaction.
//
// There are three stopping conditions; one corresponding to each return value:
//
//  1. state.Transaction is included: the IncludedTx is returned.
//  2. a recoverable error is encountered: state is updated for another try.
//  3. a fatal error is encountered: the error is returned after some state is cleaned up.
func (p *Persistent) try(parentCtx context.Context, state *tryState, includedCh chan *IncludedTx, errCh chan error) (*IncludedTx, *tryState, error) {
	var wg sync.WaitGroup
	defer wg.Wait()
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	// Sign.
	signed, err := p.signer.Sign(ctx, state.Transaction.Opaque())
	if err != nil {
		return nil, nil, fmt.Errorf("sign tx: %w", err)
	}

	// Submit.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := p.el.SendTransaction(ctx, signed); err != nil {
			safeSend(ctx, errCh, fmt.Errorf("send transaction: %w", err))
		}
	}()

	// Monitor.
	wg.Add(1)
	go func() {
		defer wg.Done()
		receipt, err := p.el.TransactionReceipt(ctx, signed.Hash())
		if err != nil {
			safeSend(ctx, errCh, fmt.Errorf("get transaction receipt: %w", err))
			return
		}
		safeSend(ctx, includedCh, &IncludedTx{
			Transaction: signed,
			Receipt:     receipt,
		})
	}()

	select {
	case included := <-includedCh:
		return included, nil, nil

	case <-ctx.Done():
		p.cfg.nonces.InsertGap(signed.Nonce())
		return nil, nil, ctx.Err()

	case err := <-errCh:
		switch {
		case errors.Is(err, core.ErrNonceTooLow):
			state.Transaction.SetNonce(p.cfg.nonces.Next())
			return nil, state, nil
		case errors.Is(err, txpool.ErrReplaceUnderpriced) || errors.Is(err, txpool.ErrUnderpriced):
			// TODO(16368): bump fees.
			return nil, state, nil
		default:
			p.cfg.nonces.InsertGap(signed.Nonce())
			return nil, nil, err
		}
	}
}

func safeSend[T any](ctx context.Context, ch chan<- T, v T) {
	select {
	case ch <- v:
	case <-ctx.Done():
	}
}

const unsetErrStr = "tx is unset"

type unincludedTx struct {
	legacyTx     *types.LegacyTx
	accessListTx *types.AccessListTx
	dynamicFeeTx *types.DynamicFeeTx
	blobTx       *types.BlobTx
	setCodeTx    *types.SetCodeTx
}

func newUnincludedTx(data types.TxData) *unincludedTx {
	tx := &unincludedTx{}
	switch t := data.(type) {
	case *types.LegacyTx:
		tx.legacyTx = t
	case *types.AccessListTx:
		tx.accessListTx = t
	case *types.DynamicFeeTx:
		tx.dynamicFeeTx = t
	case *types.BlobTx:
		tx.blobTx = t
	case *types.SetCodeTx:
		tx.setCodeTx = t
	default:
		panic(fmt.Errorf("unrecognized tx type: %T", data))
	}
	return tx
}

func (tx *unincludedTx) SetNonce(nonce uint64) {
	switch {
	case tx.legacyTx != nil:
		tx.legacyTx.Nonce = nonce
	case tx.accessListTx != nil:
		tx.accessListTx.Nonce = nonce
	case tx.dynamicFeeTx != nil:
		tx.dynamicFeeTx.Nonce = nonce
	case tx.blobTx != nil:
		tx.blobTx.Nonce = nonce
	case tx.setCodeTx != nil:
		tx.setCodeTx.Nonce = nonce
	default:
		panic(unsetErrStr)
	}
}

func (tx *unincludedTx) Opaque() *types.Transaction {
	switch {
	case tx.legacyTx != nil:
		return types.NewTx(tx.legacyTx)
	case tx.accessListTx != nil:
		return types.NewTx(tx.accessListTx)
	case tx.dynamicFeeTx != nil:
		return types.NewTx(tx.dynamicFeeTx)
	case tx.blobTx != nil:
		return types.NewTx(tx.blobTx)
	case tx.setCodeTx != nil:
		return types.NewTx(tx.setCodeTx)
	default:
		panic(unsetErrStr)
	}
}
