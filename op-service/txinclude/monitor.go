package txinclude

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Monitor is a ReceiptGetter that will continue looking for a receipt even when
// it doesn't find it right away.
type Monitor struct {
	inner     ReceiptGetter
	blockTime time.Duration
}

var _ ReceiptGetter = (*Monitor)(nil)

func NewMonitor(inner ReceiptGetter, blockTime time.Duration) *Monitor {
	return &Monitor{
		inner:     inner,
		blockTime: blockTime,
	}
}

var transientErrs = []error{
	ethereum.NotFound,
	errors.New("transaction indexing in progress"), // Not exported from geth.
}

func (m *Monitor) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	for {
		receipt, err := m.inner.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if !slices.ContainsFunc(transientErrs, func(transientErr error) bool {
			// TODO(13408): we should not need to use strings.Contains.
			return strings.Contains(err.Error(), transientErr.Error())
		}) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.blockTime):
		}
	}
}
