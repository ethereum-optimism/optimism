package txinclude_test

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/accounting"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

type mockEL struct {
	sendTxErrs   []error
	sendTxCalled int

	// receiptReadyCh will be closed when the receipt can be sent.
	// mockEL doesn't always close it: it may last the life of the test.
	receiptReadyCh chan struct{}
	receipt        *types.Receipt
}

func newMockEL(sendTxErrs []error, receipt *types.Receipt) *mockEL {
	return &mockEL{
		sendTxErrs:     sendTxErrs,
		receiptReadyCh: make(chan struct{}),
		receipt:        receipt,
	}
}

func (m *mockEL) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	call := m.sendTxCalled
	m.sendTxCalled++
	if call < len(m.sendTxErrs) {
		return m.sendTxErrs[call]
	}
	// Close the channel on success to make m.receipt available.
	select {
	case <-m.receiptReadyCh:
		// Already closed.
	default:
		close(m.receiptReadyCh)
	}
	return nil
}

func (m *mockEL) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-m.receiptReadyCh:
		return m.receipt, nil
	}
}

// minedNonceTooLowEL models a node where our tx has already been mined: every
// (re)submit returns ErrNonceTooLow (the nonce is consumed), while the receipt for
// our tx is available. The receipt is delayed slightly so the send-path
// ErrNonceTooLow reaches the includer before the receipt does, deterministically
// exercising the post-ErrNonceTooLow receipt lookup.
type minedNonceTooLowEL struct {
	receipt *types.Receipt
	sends   atomic.Int64
}

func (m *minedNonceTooLowEL) SendTransaction(_ context.Context, _ *types.Transaction) error {
	m.sends.Add(1)
	return core.ErrNonceTooLow
}

func (m *minedNonceTooLowEL) TransactionReceipt(ctx context.Context, _ common.Hash) (*types.Receipt, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(10 * time.Millisecond):
		return m.receipt, nil
	}
}

func newSigner(t *testing.T) txinclude.Signer {
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	return txinclude.NewPkSigner(privateKey, big.NewInt(1))
}

func TestPersistentSuccessfulTxInclusion(t *testing.T) {
	original := &types.DynamicFeeTx{
		GasFeeCap: eth.OneGWei.ToBig(),
		Gas:       21_000,
	}
	want := &txinclude.IncludedTx{
		Transaction: types.NewTx(original),
		Receipt: &types.Receipt{
			Status:            types.ReceiptStatusSuccessful,
			GasUsed:           original.Gas,
			EffectiveGasPrice: original.GasFeeCap,
		},
	}

	el := newMockEL(nil, want.Receipt)
	startingBalance := eth.OneEther
	budget := accounting.NewBudget(startingBalance)
	p := txinclude.NewPersistent(newSigner(t), el, txinclude.WithBudget(txinclude.NewTxBudget(budget)))
	got, err := p.Include(context.Background(), original)
	require.NoError(t, err)
	require.EqualExportedValues(t, want, got)
	require.Equal(t, startingBalance.Sub(eth.OneGWei.Mul(want.Receipt.GasUsed)), budget.Balance())
}

func TestPersistentFixesNonceTooLow(t *testing.T) {
	original := &types.DynamicFeeTx{
		GasFeeCap: eth.OneGWei.ToBig(),
		Gas:       21_000,
	}
	want := &txinclude.IncludedTx{
		Transaction: types.NewTx(&types.DynamicFeeTx{
			GasFeeCap: original.GasFeeCap,
			Gas:       original.Gas,
			Nonce:     original.Nonce + 2,
		}),
		Receipt: &types.Receipt{
			Status:            types.ReceiptStatusSuccessful,
			GasUsed:           original.Gas,
			EffectiveGasPrice: original.GasFeeCap,
		},
	}

	el := newMockEL([]error{core.ErrNonceTooLow, core.ErrNonceTooLow}, want.Receipt)
	startingBalance := eth.OneEther
	budget := accounting.NewBudget(startingBalance)
	// The nonce is genuinely consumed by a gap here (no receipt for our tx until a
	// send succeeds), so the disambiguating lookup must time out before advancing;
	// keep that timeout short so the test stays fast.
	p := txinclude.NewPersistent(newSigner(t), el,
		txinclude.WithBudget(txinclude.NewTxBudget(budget)),
		txinclude.WithIncludedLookupTimeout(10*time.Millisecond),
	)
	got, err := p.Include(context.Background(), original)
	require.NoError(t, err)
	require.EqualExportedValues(t, want, got)
	require.Equal(t, startingBalance.Sub(eth.OneGWei.Mul(want.Receipt.GasUsed)), budget.Balance())
}

// TestPersistentDoesNotResendMinedTx guards against over-sending: once our tx is
// mined, a resubmit returns ErrNonceTooLow, and the includer must recognize the tx
// as included (via a receipt lookup) rather than advancing the nonce and resending.
func TestPersistentDoesNotResendMinedTx(t *testing.T) {
	original := &types.DynamicFeeTx{
		GasFeeCap: eth.OneGWei.ToBig(),
		Gas:       21_000,
	}
	receipt := &types.Receipt{
		Status:            types.ReceiptStatusSuccessful,
		GasUsed:           original.Gas,
		EffectiveGasPrice: original.GasFeeCap,
	}
	el := &minedNonceTooLowEL{receipt: receipt}
	p := txinclude.NewPersistent(newSigner(t), el)
	got, err := p.Include(context.Background(), original)
	require.NoError(t, err)
	require.Equal(t, receipt, got.Receipt)
	require.Equal(t, original.Nonce, got.Transaction.Nonce(), "must not advance the nonce for an already-mined tx")
	require.Equal(t, int64(1), el.sends.Load(), "must not resubmit an already-mined tx at a new nonce")
}

func TestPersistentNoChangeOnUnderpriced(t *testing.T) {
	original := &types.DynamicFeeTx{
		GasFeeCap: eth.OneGWei.ToBig(),
		Gas:       21_000,
	}
	want := &txinclude.IncludedTx{
		Transaction: types.NewTx(original),
		Receipt: &types.Receipt{
			Status:            types.ReceiptStatusSuccessful,
			GasUsed:           original.Gas,
			EffectiveGasPrice: original.GasFeeCap,
		},
	}

	el := newMockEL([]error{txpool.ErrUnderpriced, txpool.ErrReplaceUnderpriced}, want.Receipt)
	startingBalance := eth.Ether(1)
	budget := accounting.NewBudget(startingBalance)
	p := txinclude.NewPersistent(newSigner(t), el, txinclude.WithBudget(txinclude.NewTxBudget(budget)))
	got, err := p.Include(context.Background(), original)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.EqualExportedValues(t, want, got)
	require.Equal(t, startingBalance.Sub(eth.GWei(21_000)), budget.Balance())
}

func TestPersistentContextCanceled(t *testing.T) {
	el := newMockEL(nil, nil)
	startingBalance := eth.OneEther
	budget := accounting.NewBudget(startingBalance)
	p := txinclude.NewPersistent(newSigner(t), el, txinclude.WithBudget(txinclude.NewTxBudget(budget)))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := p.Include(ctx, &types.DynamicFeeTx{
		GasFeeCap: eth.OneGWei.ToBig(),
		Gas:       21_000,
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, got)
	require.Equal(t, startingBalance, budget.Balance())
}

func TestPersistentFatalError(t *testing.T) {
	fatalErr := errors.New("the sky is falling")
	el := newMockEL([]error{fatalErr}, nil)
	startingBalance := eth.OneEther
	budget := accounting.NewBudget(startingBalance)
	p := txinclude.NewPersistent(newSigner(t), el, txinclude.WithBudget(txinclude.NewTxBudget(budget)))
	got, err := p.Include(context.Background(), &types.DynamicFeeTx{
		GasFeeCap: eth.OneGWei.ToBig(),
		Gas:       21_000,
	})
	require.ErrorIs(t, err, fatalErr)
	require.Nil(t, got)
	require.Equal(t, startingBalance, budget.Balance())
}
