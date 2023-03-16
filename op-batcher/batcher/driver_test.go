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

func constructDefaultBatchSubmitter(l log.Logger, mockTxMgr ExternalTxManager, l1Client L1ConfigProvider, l2Client L2ConfigProvider, rollupNode RollupNodeConfigProvider) (*BatchSubmitter, error) {
	resubmissionTimeout, err := time.ParseDuration("30s")
	if err != nil {
		return nil, err
	}

	pollInterval, err := time.ParseDuration("10s")
	if err != nil {
		return nil, err
	}

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
			// Set the max frame size to 1 so that we can test sending transactions
			MaxFrameSize:     1,
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

	if err != nil {
		return nil, err
	}
	return &b, nil
}

// TestDriverLoadBlocksIntoState ensures that the [BatchSubmitter] can load blocks into the state.
func TestDriverLoadBlocksIntoState(t *testing.T) {
	// Setup the batch submitter
	l1Client := mocks.L1ConfigProvider{}
	l2Client := mocks.L2ConfigProvider{}
	rollupNode := mocks.RollupNodeConfigProvider{}
	txMgr := mocks.ExternalTxManager{}
	log := testlog.Logger(t, log.LvlCrit)
	fmt.Printf("Constructing default batch submitter...\n")
	b, err := constructDefaultBatchSubmitter(log, &txMgr, &l1Client, &l2Client, &rollupNode)
	require.NoError(t, err)

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
			Number:     101,
			Hash:       common.Hash{},
			ParentHash: common.Hash{},
			Time:       0,
		},
	}, nil).Once()

	// Mock the first L2 BlockByNumber call
	oneZeroOne := new(big.Int).SetUint64(uint64(101))
	header := types.Header{
		Number: oneZeroOne,
	}
	l1Block := types.NewBlock(&types.Header{
		BaseFee:    big.NewInt(10),
		Difficulty: common.Big0,
		Number:     big.NewInt(100),
	}, nil, nil, nil, trie.NewStackTrie(nil))
	l1InfoTx, err := derive.L1InfoDeposit(0, l1Block, eth.SystemConfig{}, false)
	require.NoError(t, err)
	l1Client.On("HeaderByNumber", mock.Anything, mock.Anything).Return(&header, nil).Once()

	txs := []*types.Transaction{types.NewTx(l1InfoTx)}
	for i := 0; i < 500_000; i++ {
		txData := make([]byte, 32)
		_, _ = rand.Read(txData)
		tx := types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), txData)
		txs = append(txs, tx)
	}
	block := types.NewBlock(&header, txs, nil, nil, trie.NewStackTrie(nil))
	l2Client.On("BlockByNumber", mock.Anything, oneZeroOne).Return(block, nil).Once()

	// Mock internal calls for the batch transaction manager's sending transaction
	l1Client.On("SuggestGasTipCap", mock.Anything).Return(big.NewInt(10), nil).Once()
	l1Client.On("HeaderByNumber", mock.Anything, mock.Anything).Return(&types.Header{
		BaseFee: big.NewInt(10),
	}, nil).Once()
	l1Client.On("NonceAt", mock.Anything, mock.Anything, mock.Anything).Return(uint64(0), nil).Once().Run(func(args mock.Arguments) {
		// At this point, the batch submitter should have a pending transaction
		// in the state.
		fmt.Printf("Batch submitter pending transactions: %v\n", b.state.pendingTransactions)
		require.Equal(t, 1, len(b.state.pendingTransactions))

		// The number of frames should also be > 1, ie the batch submitter is not finished.
		require.Greater(t, b.state.pendingChannel.NumFrames(), 1)
		require.False(t, b.state.pendingChannelIsFullySubmitted())
	})

	// Block on the send call to allow
	// .WaitUntil(time.After(100*time.Second))
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
			Number:     101,
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
	}, nil).Once()

	// Make the second L2 BlockByNumber call return a reorg error
	oneZeroTwo := new(big.Int).SetUint64(uint64(102))
	header = types.Header{
		Number: oneZeroTwo,
	}
	l2Client.On("BlockByNumber", mock.Anything, oneZeroTwo).Return(nil, ErrReorg).Once()
	l1Client.On("HeaderByNumber", mock.Anything, mock.Anything).Return(&header, nil).Once()
	rollupNode.On("SyncStatus", mock.Anything).WaitUntil(time.After(10*time.Second)).Return(nil, nil).Once().Run(func(args mock.Arguments) {
		fmt.Println("Batch submitter called sync status...")
	})
	l1Client.On("HeaderByNumber", mock.Anything, mock.Anything).Return(&header, nil).Once().Run(func(args mock.Arguments) {
		fmt.Println("Batch submitter called header by number...")
		b.wg.Done()
	})

	// Start the batch submitter
	err = b.Start()
	require.NoError(t, err)
	b.wg.Wait()
	require.Equal(t, 0, len(b.state.pendingTransactions))
}
