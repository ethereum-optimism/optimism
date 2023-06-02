package challenger

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testlog"

	"github.com/ethereum-optimism/optimism/op-challenger/challenger/mocks"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockTraversalSubscription struct {
	errorChan chan error
}

func (m mockTraversalSubscription) Err() <-chan error {
	return m.errorChan
}

func (m mockTraversalSubscription) Unsubscribe() {}

func newLogTraversal(t *testing.T) (*logTraversal, *mocks.MinimalEthClient) {
	mockClient := mocks.NewMinimalEthClient(t)
	logTraversal := NewLogTraversal(
		mockClient,
		&ethereum.FilterQuery{},
		testlog.Logger(t, log.LvlError),
		big.NewInt(0),
	)
	return logTraversal, mockClient
}

func TestLogTraversal_Start_ReceivesNewHeads_NoTransactions(t *testing.T) {
	logTraversal, mockRpcClient := newLogTraversal(t)
	require.False(t, logTraversal.Started())

	handleLog := func(log *types.Log) error {
		return nil
	}

	sub := mockTraversalSubscription{
		errorChan: make(chan error),
	}
	var headers chan<- *types.Header
	mockRpcClient.On(
		"SubscribeNewHead",
		mock.Anything,
		mock.Anything,
	).Return(
		&sub,
		nil,
	).Run(func(args mock.Arguments) {
		headers = args.Get(1).(chan<- *types.Header)
	})

	require.NoError(t, logTraversal.Start(context.Background(), handleLog))
	require.True(t, logTraversal.Started())

	firstHeader := types.Header{
		Number: big.NewInt(1),
	}
	blockInfo := eth.HeaderBlockInfo(&firstHeader)
	mockRpcClient.On(
		"InfoByHash",
		mock.Anything,
		mock.Anything,
	).Return(
		blockInfo,
		nil,
	)

	mockRpcClient.On(
		"FetchReceipts",
		mock.Anything,
		mock.Anything,
	).Return(
		blockInfo,
		types.Receipts{},
		nil,
	)

	headers <- &firstHeader

	timeout, tCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer tCancel()
	err := e2eutils.WaitFor(timeout, 500*time.Millisecond, func() (bool, error) {
		return logTraversal.LastBlockNumber().Cmp(firstHeader.Number) == 0, nil
	})
	require.NoError(t, err)
}

func TestLogTraversal_Start_ReceivesNewHeads_Transactions(t *testing.T) {
	logTraversal, mockClient := newLogTraversal(t)
	require.False(t, logTraversal.Started())

	handleLog := func(log *types.Log) error {
		log.Address = common.HexToAddress(("0x02"))
		return nil
	}

	sub := mockTraversalSubscription{
		errorChan: make(chan error),
	}
	var headers chan<- *types.Header
	mockClient.On(
		"SubscribeNewHead",
		mock.Anything,
		mock.Anything,
	).Return(
		sub,
		nil,
	).Run(func(args mock.Arguments) {
		headers = args.Get(1).(chan<- *types.Header)
	})

	require.NoError(t, logTraversal.Start(context.Background(), handleLog))
	require.True(t, logTraversal.Started())

	firstHeader := &types.Header{
		Number: big.NewInt(1),
	}

	to := common.Address{}
	log := &types.Log{
		Address: to,
		Topics:  []common.Hash{{}},
	}

	blockInfo := eth.HeaderBlockInfo(firstHeader)
	mockClient.On(
		"InfoByHash",
		mock.Anything,
		mock.Anything,
	).Return(
		blockInfo,
		nil,
	)

	mockClient.On(
		"FetchReceipts",
		mock.Anything,
		mock.Anything,
	).Return(
		blockInfo,
		types.Receipts{{
			Logs: []*types.Log{log},
		}},
		nil,
	)

	headers <- firstHeader

	timeout, tCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer tCancel()
	err := e2eutils.WaitFor(timeout, 500*time.Millisecond, func() (bool, error) {
		return logTraversal.LastBlockNumber().Cmp(firstHeader.Number) == 0, nil
	})
	require.NoError(t, err)
}

func TestLogTraversal_Start_SubscriptionErrors(t *testing.T) {
	logTraversal, mockTraversalClient := newLogTraversal(t)
	require.False(t, logTraversal.Started())

	handleLog := func(log *types.Log) error {
		return nil
	}

	errSubscriptionFailed := errors.New("subscription failed")
	mockTraversalClient.On(
		"SubscribeNewHead",
		mock.Anything,
		mock.Anything,
	).Return(
		nil,
		errSubscriptionFailed,
	)

	require.EqualError(
		t,
		logTraversal.Start(context.Background(), handleLog),
		"subscription failed",
	)
	require.False(t, logTraversal.Started())
}

func TestLogTraversal_Quit_StopsTraversal(t *testing.T) {
	logTraversal, mockTraversalClient := newLogTraversal(t)
	require.False(t, logTraversal.Started())

	handleLog := func(log *types.Log) error {
		return nil
	}

	sub := mockTraversalSubscription{
		errorChan: make(chan error),
	}

	mockTraversalClient.On(
		"SubscribeNewHead",
		mock.Anything,
		mock.Anything,
	).Return(
		sub,
		nil,
	)

	require.NoError(t, logTraversal.Start(context.Background(), handleLog))
	require.True(t, logTraversal.Started())

	logTraversal.Quit()
	require.False(t, logTraversal.Started())
}
