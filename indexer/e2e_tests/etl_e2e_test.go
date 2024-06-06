package e2e_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/bindings"
	bindingspreview "github.com/ethereum-optimism/optimism/op-node/bindings/preview"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/stretchr/testify/require"
)

func TestE2EL1ETLInactivityWindow(t *testing.T) {
	withInactivityWindow := func(cfg *config.Config) *config.Config {
		cfg.Chain.ETLAllowedInactivityWindowSeconds = 1

		// Passing the inactivity window will index the latest header
		// in the batch. Make the batch size 1 so all blocks are indexed
		cfg.Chain.L1HeaderBufferSize = 1
		return cfg
	}

	testSuite := createE2ETestSuite(t, withInactivityWindow)

	// wait for 10 L1 blocks to be posted
	require.NoError(t, wait.For(context.Background(), time.Second, func() (bool, error) {
		l1Header := testSuite.Indexer.BridgeProcessor.LastL1Header
		return l1Header != nil && l1Header.Number.Uint64() >= 10, nil
	}))

	// each block is indexed
	for height := int64(0); height < int64(10); height++ {
		header, err := testSuite.DB.Blocks.L1BlockHeaderWithFilter(database.BlockHeader{Number: big.NewInt(height)})
		require.NoError(t, err)
		require.NotNil(t, header)
		require.Equal(t, header.Number.Uint64(), uint64(height))
	}
}

func TestE2EETL(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	l2OutputOracle, err := bindings.NewL2OutputOracle(testSuite.OpCfg.L1Deployments.L2OutputOracleProxy, testSuite.L1Client)
	require.NoError(t, err)

	disputeGameFactory, err := bindings.NewDisputeGameFactoryCaller(testSuite.OpCfg.L1Deployments.DisputeGameFactoryProxy, testSuite.L1Client)
	require.NoError(t, err)

	optimismPortal, err := bindingspreview.NewOptimismPortal2Caller(testSuite.OpCfg.L1Deployments.OptimismPortalProxy, testSuite.L1Client)
	require.NoError(t, err)

	// wait for at least 10 L2 blocks posted on L1
	require.NoError(t, wait.For(context.Background(), time.Second, func() (bool, error) {
		var l2Height *big.Int
		var err error
		if e2eutils.UseFaultProofs() {
			gameCount, err := disputeGameFactory.GameCount(&bind.CallOpts{Context: context.Background()})
			require.NoError(t, err)
			if gameCount.Cmp(big.NewInt(0)) == 0 {
				return false, nil
			}

			latestGame, err := withdrawals.FindLatestGame(context.Background(), disputeGameFactory, optimismPortal)
			require.NoError(t, err)
			l2Height = new(big.Int).SetBytes(latestGame.ExtraData[0:32])
		} else {
			l2Height, err = l2OutputOracle.LatestBlockNumber(&bind.CallOpts{Context: context.Background()})
		}
		return l2Height != nil && l2Height.Uint64() >= 9, err
	}))

	// ensure we've indexed up to this state
	l1Height, err := testSuite.L1Client.BlockNumber(context.Background())
	require.NoError(t, err)
	require.NoError(t, wait.For(context.Background(), 100*time.Millisecond, func() (bool, error) {
		l1Header, err := testSuite.DB.Blocks.L1LatestBlockHeader()
		require.NoError(t, err)

		l2Header, err := testSuite.DB.Blocks.L2LatestBlockHeader()
		require.NoError(t, err)

		return (l1Header != nil && l1Header.Number.Uint64() >= l1Height) && (l2Header != nil && l2Header.Number.Uint64() >= 9), nil
	}))

	t.Run("indexes all L2 blocks", func(t *testing.T) {
		latestL2Header, err := testSuite.DB.Blocks.L2LatestBlockHeader()
		require.NoError(t, err)
		require.NotNil(t, latestL2Header)
		require.True(t, latestL2Header.Number.Uint64() >= 9)

		for i := int64(0); i < 10; i++ {
			height := big.NewInt(i)

			indexedHeader, err := testSuite.DB.Blocks.L2BlockHeaderWithFilter(database.BlockHeader{Number: height})
			require.NoError(t, err)
			require.NotNil(t, indexedHeader)

			header, err := testSuite.L2Client.HeaderByNumber(context.Background(), height)
			require.NoError(t, err)
			require.NotNil(t, indexedHeader)

			require.Equal(t, header.Number.Int64(), indexedHeader.Number.Int64())
			require.Equal(t, header.Hash(), indexedHeader.Hash)
			require.Equal(t, header.ParentHash, indexedHeader.ParentHash)
			require.Equal(t, header.Time, indexedHeader.Timestamp)

			// ensure the right rlp encoding is stored. checking the hashes
			// suffices as it is based on the rlp bytes of the header
			require.Equal(t, header.Hash(), indexedHeader.RLPHeader.Hash())
		}
	})

	t.Run("indexes L1 blocks with accompanying contract event", func(t *testing.T) {
		l1Contracts := []common.Address{}
		testSuite.OpCfg.L1Deployments.ForEach(func(name string, addr common.Address) { l1Contracts = append(l1Contracts, addr) })
		logFilter := ethereum.FilterQuery{FromBlock: big.NewInt(0), ToBlock: big.NewInt(int64(l1Height)), Addresses: l1Contracts}
		logs, err := testSuite.L1Client.FilterLogs(context.Background(), logFilter) // []types.Log
		require.NoError(t, err)

		for i := range logs {
			log := logs[i]
			contractEvent, err := testSuite.DB.ContractEvents.L1ContractEventWithFilter(database.ContractEvent{TransactionHash: log.TxHash, LogIndex: uint64(log.Index)})
			require.NoError(t, err)
			require.Equal(t, log.Topics[0], contractEvent.EventSignature)
			require.Equal(t, log.BlockHash, contractEvent.BlockHash)
			require.Equal(t, log.Address, contractEvent.ContractAddress)
			require.Equal(t, log.TxHash, contractEvent.TransactionHash)
			require.Equal(t, log.Index, uint(contractEvent.LogIndex))

			// ensure the right rlp encoding of the contract log is stored
			logRlp, err := rlp.EncodeToBytes(&log)
			require.NoError(t, err)
			contractEventRlp, err := rlp.EncodeToBytes(contractEvent.RLPLog)
			require.NoError(t, err)
			require.ElementsMatch(t, logRlp, contractEventRlp)

			// ensure the block is also indexed
			block, err := testSuite.L1Client.BlockByNumber(context.Background(), big.NewInt(int64(log.BlockNumber)))
			require.NoError(t, err)
			require.Equal(t, block.Time(), contractEvent.Timestamp)
			require.Equal(t, block.Hash(), contractEvent.BlockHash)

			l1BlockHeader, err := testSuite.DB.Blocks.L1BlockHeader(block.Hash())
			require.NoError(t, err)
			require.Equal(t, block.Hash(), l1BlockHeader.Hash)
			require.Equal(t, block.ParentHash(), l1BlockHeader.ParentHash)
			require.Equal(t, block.Number().Uint64(), l1BlockHeader.Number.Uint64())
			require.Equal(t, block.Time(), l1BlockHeader.Timestamp)

			// ensure the right rlp encoding is stored. checking the hashes
			// suffices as it is based on the rlp bytes of the header
			require.Equal(t, block.Hash(), l1BlockHeader.RLPHeader.Hash())
		}
	})
}
