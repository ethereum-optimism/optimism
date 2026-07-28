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
	"github.com/ethereum/go-ethereum"
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

// acceptedThenNonceTooLowEL models an EL that accepts a transaction once, then
// reports ErrNonceTooLow when the resubmitter sends the same transaction again.
// The receipt remains unavailable until the test observes a further resubmission.
type acceptedThenNonceTooLowEL struct {
	receipt      *types.Receipt
	receiptReady chan struct{}
	retried      chan struct{}
	sends        atomic.Int64
}

func newAcceptedThenNonceTooLowEL(receipt *types.Receipt) *acceptedThenNonceTooLowEL {
	return &acceptedThenNonceTooLowEL{
		receipt:      receipt,
		receiptReady: make(chan struct{}),
		retried:      make(chan struct{}),
	}
}

func (m *acceptedThenNonceTooLowEL) SendTransaction(_ context.Context, _ *types.Transaction) error {
	call := m.sends.Add(1)
	if call == 1 {
		return nil
	}
	if call == 3 {
		close(m.retried)
	}
	return core.ErrNonceTooLow
}

func (m *acceptedThenNonceTooLowEL) TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error) {
	select {
	case <-m.receiptReady:
		return m.receipt, nil
	default:
		return nil, ethereum.NotFound
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
	p := txinclude.NewPersistent(newSigner(t), el, txinclude.WithBudget(txinclude.NewTxBudget(budget)))
	got, err := p.Include(context.Background(), original)
	require.NoError(t, err)
	require.EqualExportedValues(t, want, got)
	require.Equal(t, startingBalance.Sub(eth.OneGWei.Mul(want.Receipt.GasUsed)), budget.Balance())
}

func TestPersistentResubmitsSameTxAfterNonceTooLow(t *testing.T) {
	original := &types.DynamicFeeTx{
		GasFeeCap: eth.OneGWei.ToBig(),
		Gas:       21_000,
	}
	receipt := &types.Receipt{
		Status:            types.ReceiptStatusSuccessful,
		GasUsed:           original.Gas,
		EffectiveGasPrice: original.GasFeeCap,
	}
	el := newAcceptedThenNonceTooLowEL(receipt)
	p := txinclude.NewPersistent(newSigner(t), txinclude.NewReliableEL(el, time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		included *txinclude.IncludedTx
		err      error
	}
	resultCh := make(chan result, 1)
	go func() {
		included, err := p.Include(ctx, original)
		resultCh <- result{included: included, err: err}
	}()

	select {
	case <-el.retried:
		close(el.receiptReady)
	case <-time.After(time.Second):
		t.Fatal("transaction was not resubmitted after ErrNonceTooLow")
	}

	select {
	case res := <-resultCh:
		require.NoError(t, res.err)
		require.Equal(t, receipt, res.included.Receipt)
		require.Equal(t, original.Nonce, res.included.Transaction.Nonce(), "must keep resubmitting the same nonce")
		require.GreaterOrEqual(t, el.sends.Load(), int64(3))
	case <-time.After(time.Second):
		t.Fatal("transaction receipt was not returned")
	}
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
