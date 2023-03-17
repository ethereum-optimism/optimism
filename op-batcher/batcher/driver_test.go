package batcher

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher/mocks"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func constructDefaultBatchSubmitter(l log.Logger, mockTxMgr ExternalTxManager, l1Client L1DataProvider, l2Client L2DataProvider, rollupNode RollupNodeConfigProvider) *BatchSubmitter {
	resubmissionTimeout := 30 * time.Second
	pollInterval := 5 * time.Second
	batcherConfig := Config{
		log:          l,
		L1Client:     l1Client,
		L2Client:     l2Client,
		RollupNode:   rollupNode,
		PollInterval: pollInterval,
		TxManagerConfig: txmgr.Config{
			ResubmissionTimeout:       resubmissionTimeout,
			ReceiptQueryInterval:      time.Second,
			NumConfirmations:          1,
			SafeAbortNonceTooLowCount: 3,
			From:                      common.Address{},
			Signer: func(ctx context.Context, from common.Address, tx *types.Transaction) (*types.Transaction, error) {
				return tx, nil
			},
		},
		From:   common.Address{},
		Rollup: &rollup.Config{},
		Channel: ChannelConfig{
			SeqWindowSize:      15,
			ChannelTimeout:     40,
			MaxChannelDuration: 1,
			SubSafetyMargin:    4,
			// Set the max frame size to 24 so that we can test sending transactions
			// The fixed overhead for frame size is 23, so we must be larger or else
			// the uint64 will underflow, causing the frame size to be essentially unbound
			MaxFrameSize:     24,
			TargetFrameSize:  1,
			TargetNumFrames:  1,
			ApproxComprRatio: 0.4,
		},
	}

	// Construct a tx manager with a mock external tx manager
	txMgr := TransactionManager{
		batchInboxAddress: batcherConfig.Rollup.BatchInboxAddress,
		senderAddress:     batcherConfig.From,
		chainID:           batcherConfig.Rollup.L1ChainID,
		txMgr:             mockTxMgr,
		l1Client:          l1Client,
		signerFn:          batcherConfig.TxManagerConfig.Signer,
		log:               l,
	}

	b := BatchSubmitter{
		Config: batcherConfig,
		txMgr:  &txMgr,
		state:  NewChannelManager(l, batcherConfig.Channel),
	}

	return &b
}

// TestDriverLoadBlocksIntoState ensures that the [BatchSubmitter] can load blocks into the state.
func TestDriverLoadBlocksIntoState(t *testing.T) {
	// Setup the batch submitter
	l1Client := mocks.L1DataProvider{}
	l2Client := mocks.L2DataProvider{}
	rollupNode := mocks.RollupNodeConfigProvider{}
	txMgr := mocks.ExternalTxManager{}
	log := testlog.Logger(t, log.LvlCrit)
	b := constructDefaultBatchSubmitter(log, &txMgr, &l1Client, &l2Client, &rollupNode)

	// The first block range will only be the first block.
	// This allows the batch submitter to construct a pending transaction
	// and then have the second block range be the second block
	// which will re-org the pending transaction and clear state.
	rollupNode.On("SyncStatus", mock.Anything).Return(&eth.SyncStatus{
		HeadL1: eth.L1BlockRef{
			Number:     100,
			Hash:       common.Hash{},
			ParentHash: common.Hash{},
			Time:       0,
		},
		SafeL2: eth.L2BlockRef{
			Number:     100,
			Hash:       common.Hash{},
			ParentHash: common.Hash{},
			Time:       0,
		},
		UnsafeL2: eth.L2BlockRef{
			Number:     102,
			Hash:       common.Hash{},
			ParentHash: common.Hash{},
			Time:       0,
		},
	}, nil).Once().Run(func(args mock.Arguments) {
		// In the first sync status call, nothing should be set
		require.Equal(t, uint64(0), b.lastStoredBlock.Number)
	})

	// Mock the first L2 BlockByNumber call
	oneZeroOne := new(big.Int).SetUint64(uint64(101))
	header := types.Header{
		Number: oneZeroOne,
	}
	l1Block := types.NewBlock(&types.Header{
		BaseFee:    big.NewInt(10),
		Difficulty: common.Big0,
		Number:     big.NewInt(101),
	}, nil, nil, nil, trie.NewStackTrie(nil))
	l1InfoTx, err := derive.L1InfoDeposit(0, l1Block, eth.SystemConfig{}, false)
	require.NoError(t, err)
	txs := []*types.Transaction{types.NewTx(l1InfoTx)}
	for i := 0; i < 100_000; i++ {
		txData := make([]byte, 32)
		_, _ = rand.Read(txData)
		tx := types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), txData)
		txs = append(txs, tx)
	}
	block := types.NewBlock(&header, txs, nil, nil, trie.NewStackTrie(nil))
	oneZeroTwo := new(big.Int).SetUint64(uint64(102))
	header2 := types.Header{
		ParentHash: block.Hash(),
		Number:     oneZeroTwo,
	}
	block2 := types.NewBlock(&header2, txs, nil, nil, trie.NewStackTrie(nil))
	l2Client.On("BlockByNumber", mock.Anything, oneZeroOne).Return(block, nil).WaitUntil(time.After(2 * time.Second)).Once().Run(func(args mock.Arguments) {
		// The last stored block should be set by the [SyncStatus] call
		require.Equal(t, uint64(100), b.lastStoredBlock.Number)
	})
	l2Client.On("BlockByNumber", mock.Anything, oneZeroTwo).Return(block2, nil).Once().Run(func(args mock.Arguments) {
		// The last stored block should be set by the [SyncStatus] call
		require.Equal(t, uint64(101), b.lastStoredBlock.Number)
	})

	// Mock the [HeaderByNumber] call.
	// This is triggered by the internal l1tip call in the batch submitter block loop.
	// When this runs, the batch submitter should have already loaded blocks into state.
	l1Client.On("HeaderByNumber", mock.Anything, mock.Anything).Return(&header, nil).Once().Run(func(args mock.Arguments) {
		require.Equal(t, uint64(102), b.lastStoredBlock.Number)
	})

	// In between the header by number call and sending the transaction to the tx manager,
	// The batch submitter's block loop should have collected the tx data from state.
	// This will set the state's (Channel Manager's) pending transactions, blocks,

	// Mock internal calls for the batch transaction manager's sending transaction
	l1Client.On("SuggestGasTipCap", mock.Anything).Return(big.NewInt(10), nil).Once()
	l1Client.On("HeaderByNumber", mock.Anything, mock.Anything).Return(&types.Header{
		BaseFee: big.NewInt(10),
	}, nil).Once()
	l1Client.On("NonceAt", mock.Anything, mock.Anything, mock.Anything).Return(uint64(0), nil).Once().Run(func(args mock.Arguments) {
		// require.Greater(t, b.state.pendingChannel.NumFrames(), 1)
		require.False(t, b.state.pendingChannelIsFullySubmitted())

		// At this point, the channel manager should have a pending transaction
		require.Equal(t, 1, len(b.state.pendingTransactions))
	})

	// Block on the send call to allow
	txMgr.On("Send", mock.Anything, mock.Anything).Return(&types.Receipt{
		TxHash:      common.Hash{},
		BlockNumber: big.NewInt(1),
	}, nil).Once()

	// Once the polling interval fires the tick again,
	// new unsafe L2 blocks will be loaded into the state.
	// So we need to mock the sync status again with the new unsafe L2 block.
	rollupNode.On("SyncStatus", mock.Anything).Return(&eth.SyncStatus{
		HeadL1: eth.L1BlockRef{
			Number:     100,
			Hash:       common.Hash{},
			ParentHash: common.Hash{},
			Time:       0,
		},
		SafeL2: eth.L2BlockRef{
			Number:     102,
			Hash:       common.Hash{},
			ParentHash: common.Hash{},
			Time:       0,
		},
		UnsafeL2: eth.L2BlockRef{
			Number:     103,
			Hash:       common.Hash{},
			ParentHash: common.Hash{},
			Time:       0,
		},
	}, nil).Once()

	// Make the second L2 BlockByNumber call return a reorg error
	oneZeroThree := new(big.Int).SetUint64(uint64(103))
	header = types.Header{
		Number: oneZeroThree,
	}
	var testPendingChannel *channelBuilder
	l2Client.On("BlockByNumber", mock.Anything, oneZeroThree).Return(nil, ErrReorg).Once().Run(func(args mock.Arguments) {
		// The pending channel should still have frames left to submit for the current channel
		require.False(t, b.state.pendingChannelIsFullySubmitted())
		require.True(t, b.state.pendingChannel.HasFrame())
		require.Greater(t, b.state.pendingChannel.NumFrames(), 0)
		testPendingChannel = b.state.pendingChannel
	})
	l1Client.On("HeaderByNumber", mock.Anything, mock.Anything).Return(&header, nil).Once().Run(func(args mock.Arguments) {
		b.cancel()
		b.loadBlocksIntoState(context.Background())
		b.wg.Done()
	})
	rollupNode.On("SyncStatus", mock.Anything).Return(nil, fmt.Errorf("no-op")).Once()
	l1Client.On("HeaderByNumber", mock.Anything, mock.Anything).Return(&header, nil).Once()

	// Start the batch submitter and wait for the loop to close
	err = b.Start()
	require.NoError(t, err)
	b.wg.Wait()

	// The batch submitter should have wiped state
	require.Nil(t, b.state.pendingChannel)
	require.Equal(t, 0, len(b.state.pendingTransactions))
	// But there should be a pending channel with these frames:
	require.Greater(t, testPendingChannel.NumFrames(), 0)
}
