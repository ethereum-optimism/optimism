package l1

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	cll1 "github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// Needs to implement the Oracle interface
var _ cll1.Oracle = (*FetchingL1Oracle)(nil)

// Want to be able to use an L1Client as the data source
var _ Source = (*ethclient.Client)(nil)

func TestHeaderByHash(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		expected := &types.Header{}
		source := &stubSource{nextHeader: expected}
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
	rng := rand.New(rand.NewSource(1234))

	t.Run("Success", func(t *testing.T) {
		expectedBlock, _ := testutils.RandomBlock(rng, 3)
		expectedTxs := expectedBlock.Transactions()
		source := &stubSource{nextBlock: expectedBlock, nextTxs: expectedTxs}
		oracle := newFetchingOracle(t, source)

		header, txs := oracle.TransactionsByBlockHash(expectedBlock.Hash())
		require.Equal(t, expectedBlock.Header(), header)
		require.Equal(t, expectedTxs, txs)
	})

	t.Run("UnknownBlock_NoHeader", func(t *testing.T) {
		oracle := newFetchingOracle(t, &stubSource{})
		hash := common.HexToHash("0x4455")
		require.PanicsWithError(t, fmt.Errorf("unknown block: %s", hash).Error(), func() {
			oracle.TransactionsByBlockHash(hash)
		})
	})

	t.Run("UnknownBlock_NoTxs", func(t *testing.T) {
		oracle := newFetchingOracle(t, &stubSource{nextHeader: &types.Header{}})
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
	rng := rand.New(rand.NewSource(1234))

	t.Run("Success", func(t *testing.T) {
		expectedBlock, _ := testutils.RandomBlock(rng, 2)
		expectedTxs := expectedBlock.Transactions()

		expectedRcpts := types.Receipts{
			&types.Receipt{TxHash: expectedTxs[0].Hash()},
			&types.Receipt{TxHash: expectedTxs[1].Hash()},
		}
		source := &stubSource{
			nextBlock: expectedBlock,
			nextTxs:   expectedTxs,
			rcpts: map[common.Hash]*types.Receipt{
				expectedTxs[0].Hash(): expectedRcpts[0],
				expectedTxs[1].Hash(): expectedRcpts[1],
			}}
		oracle := newFetchingOracle(t, source)

		header, rcpts := oracle.ReceiptsByBlockHash(expectedBlock.Hash())
		require.Equal(t, expectedBlock.Header(), header)
		require.Equal(t, expectedRcpts, rcpts)
	})

	t.Run("UnknownBlock_NoHeader", func(t *testing.T) {
		oracle := newFetchingOracle(t, &stubSource{})
		hash := common.HexToHash("0x4455")
		require.PanicsWithError(t, fmt.Errorf("unknown block: %s", hash).Error(), func() {
			oracle.ReceiptsByBlockHash(hash)
		})
	})

	t.Run("UnknownBlock_NoTxs", func(t *testing.T) {
		oracle := newFetchingOracle(t, &stubSource{nextHeader: &types.Header{}})
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
		require.PanicsWithError(t, fmt.Errorf("retrieve transactions for block %s: %w", hash, err).Error(), func() {
			oracle.ReceiptsByBlockHash(hash)
		})
	})
}

func newFetchingOracle(t *testing.T, source Source) *FetchingL1Oracle {
	return NewFetchingL1Oracle(context.Background(), testlog.Logger(t, log.LvlDebug), source)
}

type stubSource struct {
	nextHeader *types.Header
	nextBlock  *types.Block
	nextTxs    types.Transactions
	rcpts      map[common.Hash]*types.Receipt
	nextErr    error
}

func (s stubSource) HeaderByHash(ctx context.Context, blockHash common.Hash) (*types.Header, error) {
	return s.nextHeader, s.nextErr
}

func (s stubSource) BlockByHash(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return s.nextBlock, s.nextErr
}

func (s stubSource) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return s.rcpts[txHash], s.nextErr
}
