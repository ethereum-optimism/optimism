package prefetcher

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRetryingL1Source(t *testing.T) {
	ctx := context.Background()
	hash := common.Hash{0xab}
	info := &testutils.MockBlockInfo{InfoHash: hash}
	// The mock really doesn't like returning nil for a eth.BlockInfo so return a value we expect to be ignored instead
	wrongInfo := &testutils.MockBlockInfo{InfoHash: common.Hash{0x99}}
	txs := types.Transactions{
		&types.Transaction{},
	}
	rcpts := types.Receipts{
		&types.Receipt{},
	}

	t.Run("InfoByHash Success", func(t *testing.T) {
		source, mock := createL1Source(t)
		defer mock.AssertExpectations(t)
		mock.ExpectInfoByHash(hash, info, nil)

		result, err := source.InfoByHash(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, info, result)
	})

	t.Run("InfoByHash Error", func(t *testing.T) {
		source, mock := createL1Source(t)
		defer mock.AssertExpectations(t)
		expectedErr := errors.New("boom")
		mock.ExpectInfoByHash(hash, wrongInfo, expectedErr)
		mock.ExpectInfoByHash(hash, info, nil)

		result, err := source.InfoByHash(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, info, result)
	})

	t.Run("InfoAndTxsByHash Success", func(t *testing.T) {
		source, mock := createL1Source(t)
		defer mock.AssertExpectations(t)
		mock.ExpectInfoAndTxsByHash(hash, info, txs, nil)

		actualInfo, actualTxs, err := source.InfoAndTxsByHash(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, info, actualInfo)
		require.Equal(t, txs, actualTxs)
	})

	t.Run("InfoAndTxsByHash Error", func(t *testing.T) {
		source, mock := createL1Source(t)
		defer mock.AssertExpectations(t)
		expectedErr := errors.New("boom")
		mock.ExpectInfoAndTxsByHash(hash, wrongInfo, nil, expectedErr)
		mock.ExpectInfoAndTxsByHash(hash, info, txs, nil)

		actualInfo, actualTxs, err := source.InfoAndTxsByHash(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, info, actualInfo)
		require.Equal(t, txs, actualTxs)
	})

	t.Run("FetchReceipts Success", func(t *testing.T) {
		source, mock := createL1Source(t)
		defer mock.AssertExpectations(t)
		mock.ExpectFetchReceipts(hash, info, rcpts, nil)

		actualInfo, actualRcpts, err := source.FetchReceipts(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, info, actualInfo)
		require.Equal(t, rcpts, actualRcpts)
	})

	t.Run("FetchReceipts Error", func(t *testing.T) {
		source, mock := createL1Source(t)
		defer mock.AssertExpectations(t)
		expectedErr := errors.New("boom")
		mock.ExpectFetchReceipts(hash, wrongInfo, nil, expectedErr)
		mock.ExpectFetchReceipts(hash, info, rcpts, nil)

		actualInfo, actualRcpts, err := source.FetchReceipts(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, info, actualInfo)
		require.Equal(t, rcpts, actualRcpts)
	})
}

func createL1Source(t *testing.T) (*RetryingL1Source, *testutils.MockL1Source) {
	logger := testlog.Logger(t, log.LvlDebug)
	mock := &testutils.MockL1Source{}
	source := NewRetryingL1Source(logger, mock)
	// Avoid sleeping in tests by using a fixed backoff strategy with no delay
	source.strategy = backoff.Fixed(0)
	return source, mock
}

func TestRetryingL2Source(t *testing.T) {
	ctx := context.Background()
	hash := common.Hash{0xab}
	info := &testutils.MockBlockInfo{InfoHash: hash}
	// The mock really doesn't like returning nil for a eth.BlockInfo so return a value we expect to be ignored instead
	wrongInfo := &testutils.MockBlockInfo{InfoHash: common.Hash{0x99}}
	txs := types.Transactions{
		&types.Transaction{},
	}
	data := []byte{1, 2, 3, 4, 5}

	t.Run("InfoAndTxsByHash Success", func(t *testing.T) {
		source, mock := createL2Source(t)
		defer mock.AssertExpectations(t)
		mock.ExpectInfoAndTxsByHash(hash, info, txs, nil)

		actualInfo, actualTxs, err := source.InfoAndTxsByHash(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, info, actualInfo)
		require.Equal(t, txs, actualTxs)
	})

	t.Run("InfoAndTxsByHash Error", func(t *testing.T) {
		source, mock := createL2Source(t)
		defer mock.AssertExpectations(t)
		expectedErr := errors.New("boom")
		mock.ExpectInfoAndTxsByHash(hash, wrongInfo, nil, expectedErr)
		mock.ExpectInfoAndTxsByHash(hash, info, txs, nil)

		actualInfo, actualTxs, err := source.InfoAndTxsByHash(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, info, actualInfo)
		require.Equal(t, txs, actualTxs)
	})

	t.Run("NodeByHash Success", func(t *testing.T) {
		source, mock := createL2Source(t)
		defer mock.AssertExpectations(t)
		mock.ExpectNodeByHash(hash, data, nil)

		actual, err := source.NodeByHash(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, data, actual)
	})

	t.Run("NodeByHash Error", func(t *testing.T) {
		source, mock := createL2Source(t)
		defer mock.AssertExpectations(t)
		expectedErr := errors.New("boom")
		mock.ExpectNodeByHash(hash, nil, expectedErr)
		mock.ExpectNodeByHash(hash, data, nil)

		actual, err := source.NodeByHash(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, data, actual)
	})

	t.Run("CodeByHash Success", func(t *testing.T) {
		source, mock := createL2Source(t)
		defer mock.AssertExpectations(t)
		mock.ExpectCodeByHash(hash, data, nil)

		actual, err := source.CodeByHash(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, data, actual)
	})

	t.Run("CodeByHash Error", func(t *testing.T) {
		source, mock := createL2Source(t)
		defer mock.AssertExpectations(t)
		expectedErr := errors.New("boom")
		mock.ExpectCodeByHash(hash, nil, expectedErr)
		mock.ExpectCodeByHash(hash, data, nil)

		actual, err := source.CodeByHash(ctx, hash)
		require.NoError(t, err)
		require.Equal(t, data, actual)
	})
}

func createL2Source(t *testing.T) (*RetryingL2Source, *MockL2Source) {
	logger := testlog.Logger(t, log.LvlDebug)
	mock := &MockL2Source{}
	source := NewRetryingL2Source(logger, mock)
	// Avoid sleeping in tests by using a fixed backoff strategy with no delay
	source.strategy = backoff.Fixed(0)
	return source, mock
}

type MockL2Source struct {
	mock.Mock
}

func (m *MockL2Source) InfoAndTxsByHash(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	out := m.Mock.MethodCalled("InfoAndTxsByHash", blockHash)
	return out[0].(eth.BlockInfo), out[1].(types.Transactions), *out[2].(*error)
}

func (m *MockL2Source) NodeByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	out := m.Mock.MethodCalled("NodeByHash", hash)
	return out[0].([]byte), *out[1].(*error)
}

func (m *MockL2Source) CodeByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	out := m.Mock.MethodCalled("CodeByHash", hash)
	return out[0].([]byte), *out[1].(*error)
}

func (m *MockL2Source) ExpectInfoAndTxsByHash(blockHash common.Hash, info eth.BlockInfo, txs types.Transactions, err error) {
	m.Mock.On("InfoAndTxsByHash", blockHash).Once().Return(info, txs, &err)
}

func (m *MockL2Source) ExpectNodeByHash(hash common.Hash, node []byte, err error) {
	m.Mock.On("NodeByHash", hash).Once().Return(node, &err)
}

func (m *MockL2Source) ExpectCodeByHash(hash common.Hash, code []byte, err error) {
	m.Mock.On("CodeByHash", hash).Once().Return(code, &err)
}

var _ L2Source = (*MockL2Source)(nil)
