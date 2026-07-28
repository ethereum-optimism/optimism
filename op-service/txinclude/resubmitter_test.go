package txinclude_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum/go-ethereum/core"
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

type acceptedThenNonceTooLowSender struct {
	calls   atomic.Uint64
	retried chan struct{}
}

func (s *acceptedThenNonceTooLowSender) SendTransaction(context.Context, *types.Transaction) error {
	call := s.calls.Add(1)
	if call == 1 {
		return nil
	}
	if call == 3 {
		close(s.retried)
	}
	return core.ErrNonceTooLow
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

func TestResubmitterRetriesNonceTooLowAfterSuccessfulSubmission(t *testing.T) {
	inner := &acceptedThenNonceTooLowSender{retried: make(chan struct{})}
	resubmitter := txinclude.NewResubmitter(inner, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- resubmitter.SendTransaction(ctx, types.NewTx(&types.DynamicFeeTx{}))
	}()

	select {
	case <-inner.retried:
		cancel()
	case <-time.After(time.Second):
		cancel()
		t.Fatal("resubmitter did not retry after ErrNonceTooLow")
	}
	require.ErrorIs(t, <-errCh, context.Canceled)
	require.GreaterOrEqual(t, inner.calls.Load(), uint64(3))
}

func TestResubmitterFatalErrors(t *testing.T) {
	inner := &mockSender{
		// Just test a subset of them, including an initial ErrNonceTooLow.
		errs: []error{core.ErrNonceTooLow, txpool.ErrInvalidSender, txpool.ErrReplaceUnderpriced, txpool.ErrGasLimit},
	}
	resubmitter := txinclude.NewResubmitter(inner, time.Millisecond)
	for _, want := range inner.errs {
		require.Equal(t, want, resubmitter.SendTransaction(context.Background(), types.NewTx(&types.DynamicFeeTx{})))
	}
}
