package txinclude_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
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

// TestResubmitterWrappedErrors verifies that wrapped errors are properly recognized
// using errors.Is, which is the improvement over the previous string-based matching.
func TestResubmitterWrappedErrors(t *testing.T) {
	testCases := []struct {
		name          string
		wrappedErr    error
		expectedErr   error
		shouldBeFatal bool
	}{
		{
			name:          "wrapped nonce too low",
			wrappedErr:    fmt.Errorf("transaction failed: %w", core.ErrNonceTooLow),
			expectedErr:   core.ErrNonceTooLow,
			shouldBeFatal: true,
		},
		{
			name:          "wrapped insufficient funds",
			wrappedErr:    fmt.Errorf("execution reverted: %w", core.ErrInsufficientFunds),
			expectedErr:   core.ErrInsufficientFunds,
			shouldBeFatal: true,
		},
		{
			name:          "wrapped already known (non-fatal)",
			wrappedErr:    fmt.Errorf("mempool error: %w", txpool.ErrAlreadyKnown),
			expectedErr:   context.Canceled, // Should continue resubmitting until context canceled
			shouldBeFatal: false,
		},
		{
			name:          "wrapped replace underpriced",
			wrappedErr:    fmt.Errorf("replacement transaction: %w", txpool.ErrReplaceUnderpriced),
			expectedErr:   txpool.ErrReplaceUnderpriced,
			shouldBeFatal: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inner := &mockSender{
				errs: []error{tc.wrappedErr},
			}
			resubmitter := txinclude.NewResubmitter(inner, time.Millisecond)

			if tc.shouldBeFatal {
				// Fatal errors should return immediately
				err := resubmitter.SendTransaction(context.Background(), types.NewTx(&types.DynamicFeeTx{}))
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedErr)
				require.Equal(t, uint64(1), inner.calls, "should only call inner sender once for fatal errors")
			} else {
				// Non-fatal errors should continue resubmitting
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancel()

				err := resubmitter.SendTransaction(ctx, types.NewTx(&types.DynamicFeeTx{}))
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.Greater(t, inner.calls, uint64(1), "should call inner sender multiple times for non-fatal errors")
			}
		})
	}
}

// TestResubmitterDoubleWrappedErrors tests deeply nested error wrapping
func TestResubmitterDoubleWrappedErrors(t *testing.T) {
	deeplyWrappedErr := fmt.Errorf("outer error: %w",
		fmt.Errorf("middle error: %w",
			fmt.Errorf("inner error: %w", core.ErrNonceTooLow)))

	inner := &mockSender{
		errs: []error{deeplyWrappedErr},
	}
	resubmitter := txinclude.NewResubmitter(inner, time.Millisecond)

	err := resubmitter.SendTransaction(context.Background(), types.NewTx(&types.DynamicFeeTx{}))
	require.Error(t, err)
	require.ErrorIs(t, err, core.ErrNonceTooLow, "should recognize deeply wrapped errors")
	require.Equal(t, uint64(1), inner.calls, "should only call inner sender once")
}

// TestResubmitterUnrecognizedError verifies behavior with unknown errors
func TestResubmitterUnrecognizedError(t *testing.T) {
	unknownErr := errors.New("some unknown network error")

	inner := &mockSender{
		errs: []error{unknownErr, unknownErr}, // Return same error twice
	}
	resubmitter := txinclude.NewResubmitter(inner, time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := resubmitter.SendTransaction(ctx, types.NewTx(&types.DynamicFeeTx{}))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Greater(t, inner.calls, uint64(1), "should retry on unrecognized errors")
}
