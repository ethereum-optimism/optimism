package txinclude

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// mockReceiptGetter implements ReceiptGetter for testing
type mockReceiptGetter struct {
	receipt *types.Receipt
	errs    []error
	calls   uint64
}

func (m *mockReceiptGetter) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	call := m.calls
	m.calls++
	if call < uint64(len(m.errs)) {
		return nil, m.errs[call]
	}
	return m.receipt, nil
}

func TestMonitorReceiptFound(t *testing.T) {
	inner := &mockReceiptGetter{
		receipt: &types.Receipt{},
	}
	monitor := NewMonitor(inner, time.Millisecond)
	receipt, err := monitor.TransactionReceipt(context.Background(), inner.receipt.TxHash)
	require.NoError(t, err)
	require.Equal(t, inner.receipt, receipt)
}

func TestMonitorTransientError(t *testing.T) {
	inner := &mockReceiptGetter{
		errs:    []error{ethereum.NotFound},
		receipt: &types.Receipt{},
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
