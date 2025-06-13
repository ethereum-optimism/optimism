package monitor

import (
	"context"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// mockClient implements both FinderClient and UpdaterClient interfaces for testing
type mockClient struct {
	infoByLabel           func(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)
	infoByNumber          func(ctx context.Context, number uint64) (eth.BlockInfo, error)
	fetchReceiptsByNumber func(ctx context.Context, number uint64) (eth.BlockInfo, types.Receipts, error)
	err                   error
}

func (m *mockClient) InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error) {
	if m.infoByLabel != nil {
		return m.infoByLabel(ctx, label)
	} else {
		i := eth.HeaderBlockInfo(&types.Header{
			Number: big.NewInt(0),
		})
		return i, m.err
	}
}

func (m *mockClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	if m.infoByNumber != nil {
		return m.infoByNumber(ctx, number)
	} else {
		i := eth.HeaderBlockInfo(&types.Header{
			Number: big.NewInt(int64(number)),
		})
		return i, m.err
	}
}

func (m *mockClient) FetchReceiptsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Receipts, error) {
	if m.fetchReceiptsByNumber != nil {
		return m.fetchReceiptsByNumber(ctx, number)
	} else {
		i := eth.HeaderBlockInfo(&types.Header{
			Number: big.NewInt(int64(number)),
		})
		return i, nil, m.err
	}
}

func mockReceiptsToCases(receipts []*types.Receipt) []*Job {
	return nil
}

func mockCallback(job *Job) {
}

func mockFinalizedCallback(chainID eth.ChainID, block eth.BlockInfo) {
}

func TestRPCFinder_StartStop(t *testing.T) {
	client := &mockClient{}
	logger := testlog.Logger(t, slog.LevelDebug)
	finder := NewFinder(eth.ChainIDFromUInt64(1), client, mockReceiptsToCases, mockCallback, mockFinalizedCallback, 1000, logger)

	require.NoError(t, finder.Start(context.Background()))
	require.NoError(t, finder.Stop())

	require.Eventually(t, func() bool {
		return finder.Stopped()
	}, time.Second, 100*time.Millisecond)
}

// TestRPCFinder_processBlock tests the processBlock method of the RPCFinder
// confirming that it checks the block for contiguity and
// calls the callback with the expected jobs if it is
func TestRPCFinder_processBlock(t *testing.T) {
	client := &mockClient{}
	logger := testlog.Logger(t, slog.LevelDebug)

	// create a single empty job regardless of the receipts
	fakeReceiptsToCases := func(receipts []*types.Receipt) []*Job {
		return []*Job{
			{},
		}
	}

	callbackInvocations := 0
	callback := func(job *Job) {
		require.Equal(t, job.LatestStatus(), jobStatusUnknown)
		callbackInvocations++
	}

	finder := NewFinder(eth.ChainIDFromUInt64(1), client, fakeReceiptsToCases, callback, mockFinalizedCallback, 1000, logger)

	receipts := []*types.Receipt{
		{
			Status: 1,
			TxHash: common.Hash{0x1},
		},
	}

	// Add a first block
	i := eth.HeaderBlockInfo(&types.Header{
		Number: big.NewInt(0),
	})
	err := finder.processBlock(i, receipts)
	require.NoError(t, err)
	require.Equal(t, 1, callbackInvocations)

	// Add a contiguous block
	j := eth.HeaderBlockInfo(&types.Header{
		Number:     big.NewInt(1),
		ParentHash: i.Hash(),
	})
	err = finder.processBlock(j, receipts)
	require.NoError(t, err)
	require.Equal(t, 2, callbackInvocations)

	// Add a non-contiguous block
	k := eth.HeaderBlockInfo(&types.Header{
		Number:     big.NewInt(2),
		ParentHash: i.Hash(), // non-contiguous
	})
	err = finder.processBlock(k, receipts)
	require.ErrorIs(t, err, ErrBlockNotContiguous)
	require.Equal(t, 2, callbackInvocations)
}

func TestRPCFinder_walkback(t *testing.T) {
	client := &mockClient{}

	a0 := eth.HeaderBlockInfo(&types.Header{
		Number: big.NewInt(0),
	})
	a1 := eth.HeaderBlockInfo(&types.Header{
		Root:       common.Hash{0x0},
		Number:     big.NewInt(1),
		ParentHash: a0.Hash(),
	})
	b1 := eth.HeaderBlockInfo(&types.Header{
		Root:       common.Hash{0x1}, // different root, different chain
		Number:     big.NewInt(1),
		ParentHash: a0.Hash(),
	})
	b2 := eth.HeaderBlockInfo(&types.Header{
		Number:     big.NewInt(2),
		ParentHash: b1.Hash(),
	})

	require.NotEqual(t, a1.Hash(), b1.Hash())

	client.infoByNumber = func(ctx context.Context, number uint64) (eth.BlockInfo, error) {
		blocks := []eth.BlockInfo{a0, b1, b2} // follows this chain, e.g. after a reorg
		if number >= uint64(len(blocks)) {
			return nil, ethereum.NotFound
		}

		return blocks[number], nil
	}

	logger := testlog.Logger(t, slog.LevelDebug)

	finder := NewFinder(eth.ChainIDFromUInt64(1), client, mockReceiptsToCases, mockCallback, mockFinalizedCallback, 1000, logger)

	finder.seenBlocks.Add(a0)
	finder.seenBlocks.Add(a1)
	finder.next = 2

	// L2 client reorg case. The finder initially saw a0,a1 but then we saw b2 (not descending from a0) which triggers a walkback.
	// After the walkback, we expect the seenBlocks to contain only a0 (the common ancestor)
	// and the next block number to be 1.
	// a0 <- a1
	//     \
	//      \
	//       b1 <- b2

	err := finder.walkback(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(1), finder.next)
	require.Equal(t, a0, finder.seenBlocks.Peek())
}

func TestRPCFinder_finality(t *testing.T) {
	client := &mockClient{}
	client.infoByLabel = func(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error) {
		if label == eth.Finalized {
			return eth.HeaderBlockInfo(&types.Header{
				Number: big.NewInt(99),
			}), nil
		}
		return nil, ethereum.NotFound
	}

	// confirm the callback is called with the correct chain and block
	testFinalizedCallback := func(chainID eth.ChainID, block eth.BlockInfo) {
		require.Equal(t, eth.ChainIDFromUInt64(1), chainID)
		require.Equal(t, uint64(99), block.NumberU64())
	}
	logger := testlog.Logger(t, slog.LevelDebug)
	finder := NewFinder(eth.ChainIDFromUInt64(1), client, mockReceiptsToCases, mockCallback, testFinalizedCallback, 1000, logger)

	finder.checkFinality(context.Background())
}
