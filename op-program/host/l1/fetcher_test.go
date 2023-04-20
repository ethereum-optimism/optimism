package l1

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	cll1 "github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// Needs to implement the Oracle interface
var _ cll1.Oracle = (*FetchingL1Oracle)(nil)

// Want to be able to use an L1Client as the data source
var _ Source = (*sources.L1Client)(nil)

func TestHeaderByHash(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		expected := &testutils.MockBlockInfo{}
		source := &stubSource{nextInfo: expected}
		oracle := newFetchingOracle(t, source)

		actual := oracle.HeaderByBlockHash(expected.Hash())
		require.Equal(t, expected, actual)
	})

	t.Run("UnknownBlock", func(t *testing.T) {
		oracle := newFetchingOracle(t, &stubSource{})
		hash := common.HexToHash("0x4455")
		require.PanicsWithError(t, fmt.Errorf("unknown block: %s", hash).Error(), func() {
			oracle.HeaderByBlockHash(hash)
		})
	})

	t.Run("Error", func(t *testing.T) {
		err := errors.New("kaboom")
		source := &stubSource{nextErr: err}
		oracle := newFetchingOracle(t, source)

		hash := common.HexToHash("0x8888")
		require.PanicsWithError(t, fmt.Errorf("retrieve block %s: %w", hash, err).Error(), func() {
			oracle.HeaderByBlockHash(hash)
		})
	})
}

func TestTransactionsByHash(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		expectedInfo := &testutils.MockBlockInfo{}
		expectedTxs := types.Transactions{
			&types.Transaction{},
		}
		source := &stubSource{nextInfo: expectedInfo, nextTxs: expectedTxs}
		oracle := newFetchingOracle(t, source)

		info, txs := oracle.TransactionsByBlockHash(expectedInfo.Hash())
		require.Equal(t, expectedInfo, info)
		require.Equal(t, expectedTxs, txs)
	})

	t.Run("UnknownBlock_NoInfo", func(t *testing.T) {
		oracle := newFetchingOracle(t, &stubSource{})
		hash := common.HexToHash("0x4455")
		require.PanicsWithError(t, fmt.Errorf("unknown block: %s", hash).Error(), func() {
			oracle.TransactionsByBlockHash(hash)
		})
	})

	t.Run("UnknownBlock_NoTxs", func(t *testing.T) {
		oracle := newFetchingOracle(t, &stubSource{nextInfo: &testutils.MockBlockInfo{}})
		hash := common.HexToHash("0x4455")
		require.PanicsWithError(t, fmt.Errorf("unknown block: %s", hash).Error(), func() {
			oracle.TransactionsByBlockHash(hash)
		})
	})

	t.Run("Error", func(t *testing.T) {
		err := errors.New("kaboom")
		source := &stubSource{nextErr: err}
		oracle := newFetchingOracle(t, source)

		hash := common.HexToHash("0x8888")
		require.PanicsWithError(t, fmt.Errorf("retrieve transactions for block %s: %w", hash, err).Error(), func() {
			oracle.TransactionsByBlockHash(hash)
		})
	})
}

func TestReceiptsByHash(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		expectedInfo := &testutils.MockBlockInfo{}
		expectedRcpts := types.Receipts{
			&types.Receipt{},
		}
		source := &stubSource{nextInfo: expectedInfo, nextRcpts: expectedRcpts}
		oracle := newFetchingOracle(t, source)

		info, rcpts := oracle.ReceiptsByBlockHash(expectedInfo.Hash())
		require.Equal(t, expectedInfo, info)
		require.Equal(t, expectedRcpts, rcpts)
	})

	t.Run("UnknownBlock_NoInfo", func(t *testing.T) {
		oracle := newFetchingOracle(t, &stubSource{})
		hash := common.HexToHash("0x4455")
		require.PanicsWithError(t, fmt.Errorf("unknown block: %s", hash).Error(), func() {
			oracle.ReceiptsByBlockHash(hash)
		})
	})

	t.Run("UnknownBlock_NoTxs", func(t *testing.T) {
		oracle := newFetchingOracle(t, &stubSource{nextInfo: &testutils.MockBlockInfo{}})
		hash := common.HexToHash("0x4455")
		require.PanicsWithError(t, fmt.Errorf("unknown block: %s", hash).Error(), func() {
			oracle.ReceiptsByBlockHash(hash)
		})
	})

	t.Run("Error", func(t *testing.T) {
		err := errors.New("kaboom")
		source := &stubSource{nextErr: err}
		oracle := newFetchingOracle(t, source)

		hash := common.HexToHash("0x8888")
		require.PanicsWithError(t, fmt.Errorf("retrieve receipts for block %s: %w", hash, err).Error(), func() {
			oracle.ReceiptsByBlockHash(hash)
		})
	})
}

func newFetchingOracle(t *testing.T, source Source) *FetchingL1Oracle {
	return NewFetchingL1Oracle(context.Background(), testlog.Logger(t, log.LvlDebug), source)
}

type stubSource struct {
	nextInfo  eth.BlockInfo
	nextTxs   types.Transactions
	nextRcpts types.Receipts
	nextErr   error
}

func (s stubSource) InfoByHash(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, error) {
	return s.nextInfo, s.nextErr
}

func (s stubSource) InfoAndTxsByHash(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	return s.nextInfo, s.nextTxs, s.nextErr
}

func (s stubSource) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	return s.nextInfo, s.nextRcpts, s.nextErr
}
