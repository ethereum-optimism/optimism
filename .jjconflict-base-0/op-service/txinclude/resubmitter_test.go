package txinclude_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type mockSender struct {
	errs  []error
	calls uint64
	txs   []*types.Transaction
}

func (m *mockSender) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	call := m.calls
	m.calls++
	m.txs = append(m.txs, tx)
	if call < uint64(len(m.errs)) {
		return m.errs[call]
	}
	return nil
}

func TestResubmitterSuccessfulTransaction(t *testing.T) {
	inner := &mockSender{}
	resubmitter := txinclude.NewResubmitter(inner, time.Millisecond)
	var wg sync.WaitGroup
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg.Add(1)
	go func() {
		defer wg.Done()
		tx := types.NewTx(&types.DynamicFeeTx{Nonce: 3})
		require.ErrorIs(t, resubmitter.SendTransaction(ctx, tx), context.Canceled)
		require.NotEmpty(t, inner.txs)
		require.Equal(t, inner.txs[0], tx)
	}()
}

func TestResubmitterFatalErrors(t *testing.T) {
	inner := &mockSender{
		// Just test a subset of them.
		errs: []error{txpool.ErrInvalidSender, txpool.ErrReplaceUnderpriced, txpool.ErrGasLimit},
	}
	resubmitter := txinclude.NewResubmitter(inner, time.Millisecond)
	for _, want := range inner.errs {
		require.Equal(t, want, resubmitter.SendTransaction(context.Background(), types.NewTx(&types.DynamicFeeTx{})))
	}
}
