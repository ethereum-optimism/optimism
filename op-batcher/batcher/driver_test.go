package batcher

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher/mocks"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func constructDefaultBatchSubmitter(l log.Logger, l1Client L1ConfigProvider, l2Client L2ConfigProvider, rollupNode RollupNodeConfigProvider) (*BatchSubmitter, error) {
	resubmissionTimeout, err := time.ParseDuration("30s")
	if err != nil {
		return nil, err
	}

	pollInterval, err := time.ParseDuration("1s")
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
			Signer:                    nil,
		},
		From:   common.Address{},
		Rollup: &rollup.Config{},
		Channel: ChannelConfig{
			SeqWindowSize:      15,
			ChannelTimeout:     40,
			MaxChannelDuration: 1,
			SubSafetyMargin:    4,
			MaxFrameSize:       120000,
			TargetFrameSize:    100000,
			TargetNumFrames:    1,
			ApproxComprRatio:   0.4,
		},
	}

	// Construct a tx manager with a mock external tx manager
	txMgr := TransactionManager{
		batchInboxAddress: batcherConfig.Rollup.BatchInboxAddress,
		senderAddress:     batcherConfig.From,
		chainID:           batcherConfig.Rollup.L1ChainID,
		txMgr:             &mocks.ExternalTxManager{},
		l1Client:          &mocks.TxManagerProvider{},
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
	l1Client := mocks.L1ConfigProvider{}
	l2Client := mocks.L2ConfigProvider{}
	rollupNode := mocks.RollupNodeConfigProvider{}
	// rng := rand.New(rand.NewSource(1234))

	// Mocks 2 unsafe blocks (SafeL1.Number = 100, UnsafeL1.Number = 102)
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
		// Other fields aren't used in the driver
	}, nil)

	// Mock the first L2 BlockByNumber call
	oneZeroOne := new(big.Int).SetUint64(uint64(101))
	header := types.Header{
		Number: oneZeroOne,
	}
	block := types.NewBlock(&header, []*types.Transaction{}, nil, nil, trie.NewStackTrie(nil))
	l2Client.On("BlockByNumber", mock.Anything, oneZeroOne).Return(block, nil).Once()

	// Make the second L2 BlockByNumber call return a reorg error
	oneZeroTwo := new(big.Int).SetUint64(uint64(102))
	// header2 := types.Header{
	// 	Number: oneZeroTwo,
	// }
	// block2 := types.NewBlock(&header2, []*types.Transaction{}, nil, nil, trie.NewStackTrie(nil))
	l2Client.On("BlockByNumber", mock.Anything, oneZeroTwo).After(10*time.Second).Return(nil, ErrReorg).Once()

	// Create a new [BatchSubmitter]
	log := testlog.Logger(t, log.LvlCrit)
	fmt.Printf("Constructing default batch submitter...\n")
	b, err := constructDefaultBatchSubmitter(log, &l1Client, &l2Client, &rollupNode)
	require.NoError(t, err)

	l1Client.On("HeaderByNumber", mock.Anything, mock.Anything).Return(&header, nil)

	// Load blocks into the state
	// fmt.Println("Loading blocks into state...")
	// b.loadBlocksIntoState(context.Background())
	fmt.Println("Starting the driver...")
	err = b.Start()
	require.NoError(t, err)

	// Wait for the driver to run
	time.Sleep(1 * time.Second)

	// Wait for the driver to finish
	fmt.Println("Waiting for driver to finish...")
	b.wg.Wait()
}
