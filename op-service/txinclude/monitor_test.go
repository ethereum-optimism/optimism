package txinclude

import (
	"context"
	"errors"
	"testing"
	"time"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// mockReceiptGetter implements ReceiptGetter for testing
type mockReceiptGetter struct {
	receipt *optypes.Receipt
	errs    []error
	calls   uint64
}

func (m *mockReceiptGetter) TransactionReceipt(ctx context.Context, hash common.Hash) (*optypes.Receipt, error) {
	call := m.calls
	m.calls++
	if call < uint64(len(m.errs)) {
		return nil, m.errs[call]
	}
	return m.receipt, nil
}

func TestMonitorReceiptFound(t *testing.T) {
	inner := &mockReceiptGetter{
		receipt: &optypes.Receipt{},
	}
	monitor := NewMonitor(inner, time.Millisecond)
	receipt, err := monitor.TransactionReceipt(context.Background(), inner.receipt.TxHash)
	require.NoError(t, err)
	require.Equal(t, inner.receipt, receipt)
}

func TestMonitorTransientError(t *testing.T) {
	inner := &mockReceiptGetter{
		errs: []error{
			ethereum.NotFound,
			errors.New("transaction indexing in progress"),
			errors.New("transaction indexing is in progress"),
		},
		receipt: &optypes.Receipt{},
	}
	receipt, err := NewMonitor(inner, time.Millisecond).TransactionReceipt(context.Background(), inner.receipt.TxHash)
	require.NoError(t, err)
	require.Equal(t, inner.receipt, receipt)
}

func TestMonitorFatalError(t *testing.T) {
	want := errors.New("connection refused")
	inner := &mockReceiptGetter{
		errs: []error{want},
	}
	hash := common.Hash{}
	receipt, err := NewMonitor(inner, time.Millisecond).TransactionReceipt(context.Background(), hash)
	require.ErrorIs(t, want, err)
	require.Nil(t, receipt)
}
