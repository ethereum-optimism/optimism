package txinclude_test

import (
	"context"
	"errors"
	"math/big"
	"testing"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
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
	receipt        *optypes.Receipt
}

func newMockEL(sendTxErrs []error, receipt *optypes.Receipt) *mockEL {
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

func (m *mockEL) TransactionReceipt(ctx context.Context, hash common.Hash) (*optypes.Receipt, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-m.receiptReadyCh:
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
		Receipt: &optypes.Receipt{Receipt: types.Receipt{
			Status:            types.ReceiptStatusSuccessful,
			GasUsed:           original.Gas,
			EffectiveGasPrice: original.GasFeeCap,
		}},
	}

	el := newMockEL(nil, want.Receipt)
	p := txinclude.NewPersistent(newSigner(t), el)
	got, err := p.Include(context.Background(), original)
	require.NoError(t, err)
	require.EqualExportedValues(t, want, got)
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
		Receipt: &optypes.Receipt{Receipt: types.Receipt{
			Status:            types.ReceiptStatusSuccessful,
			GasUsed:           original.Gas,
			EffectiveGasPrice: original.GasFeeCap,
		}},
	}

	el := newMockEL([]error{core.ErrNonceTooLow, core.ErrNonceTooLow}, want.Receipt)
	p := txinclude.NewPersistent(newSigner(t), el)
	got, err := p.Include(context.Background(), original)
	require.NoError(t, err)
	require.EqualExportedValues(t, want, got)
}

func TestPersistentNoChangeOnUnderpriced(t *testing.T) {
	original := &types.DynamicFeeTx{
		GasFeeCap: eth.OneGWei.ToBig(),
		Gas:       21_000,
	}
	want := &txinclude.IncludedTx{
		Transaction: types.NewTx(original),
		Receipt: &optypes.Receipt{Receipt: types.Receipt{
			Status:            types.ReceiptStatusSuccessful,
			GasUsed:           original.Gas,
			EffectiveGasPrice: original.GasFeeCap,
		}},
	}

	el := newMockEL([]error{txpool.ErrUnderpriced, txpool.ErrReplaceUnderpriced}, want.Receipt)
	p := txinclude.NewPersistent(newSigner(t), el)
	got, err := p.Include(context.Background(), original)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.EqualExportedValues(t, want, got)
}

func TestPersistentContextCanceled(t *testing.T) {
	el := newMockEL(nil, nil)
	p := txinclude.NewPersistent(newSigner(t), el)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := p.Include(ctx, &types.DynamicFeeTx{
		GasFeeCap: eth.OneGWei.ToBig(),
		Gas:       21_000,
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, got)
}

func TestPersistentFatalError(t *testing.T) {
	fatalErr := errors.New("the sky is falling")
	el := newMockEL([]error{fatalErr}, nil)
	p := txinclude.NewPersistent(newSigner(t), el)
	got, err := p.Include(context.Background(), &types.DynamicFeeTx{
		GasFeeCap: eth.OneGWei.ToBig(),
		Gas:       21_000,
	})
	require.ErrorIs(t, err, fatalErr)
	require.Nil(t, got)
}

func TestPersistentFatalErrorReleasesNonce(t *testing.T) {
	fatalErr := errors.New("the sky is falling")
	receipt := &optypes.Receipt{Receipt: types.Receipt{Status: types.ReceiptStatusSuccessful}}
	p := txinclude.NewPersistent(newSigner(t), newMockEL([]error{fatalErr}, receipt))
	newTx := func() *types.DynamicFeeTx {
		return &types.DynamicFeeTx{GasFeeCap: eth.OneGWei.ToBig(), Gas: 21_000}
	}

	_, err := p.Include(context.Background(), newTx())
	require.ErrorIs(t, err, fatalErr)

	got, err := p.Include(context.Background(), newTx())
	require.NoError(t, err)
	require.Zero(t, got.Transaction.Nonce(), "the failed attempt's nonce must be reused")
}
