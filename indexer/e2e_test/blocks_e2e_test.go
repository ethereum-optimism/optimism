package e2e_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-service/client/utils"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/stretchr/testify/require"
)

func TestE2EBlockHeaders(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	l1Client := testSuite.OpSys.Clients["l1"]
	l2Client := testSuite.OpSys.Clients["sequencer"]
	l2OutputOracle, err := bindings.NewL2OutputOracleCaller(testSuite.OpCfg.L1Deployments.L2OutputOracleProxy, l1Client)
	require.NoError(t, err)

	// a minute for total setup to finish
	setupCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// wait for at least 10 L2 blocks to be created & posted on L1
	require.NoError(t, utils.WaitFor(setupCtx, time.Second, func() (bool, error) {
		l2Height, err := l2OutputOracle.LatestBlockNumber(&bind.CallOpts{Context: setupCtx})
		return l2Height != nil && l2Height.Uint64() >= 9, err
	}))

	// ensure the processors are caught up to this state
	l1Height, err := l1Client.BlockNumber(setupCtx)
	require.NoError(t, err)
	require.NoError(t, utils.WaitFor(setupCtx, time.Second, func() (bool, error) {
		l1Header := testSuite.Indexer.L1Processor.LatestProcessedHeader()
		l2Header := testSuite.Indexer.L2Processor.LatestProcessedHeader()
		return (l1Header != nil && l1Header.Number.Uint64() >= l1Height) && (l2Header != nil && l2Header.Number.Uint64() >= 9), nil
	}))

	t.Run("indexes L2 blocks", func(t *testing.T) {
		latestL2Header, err := testSuite.DB.Blocks.LatestL2BlockHeader()
		require.NoError(t, err)
		require.NotNil(t, latestL2Header)
		require.True(t, latestL2Header.Number.Int.Uint64() >= 9)

		for i := int64(0); i < 10; i++ {
			height := big.NewInt(i)

			indexedHeader, err := testSuite.DB.Blocks.L2BlockHeader(height)
			require.NoError(t, err)
			require.NotNil(t, indexedHeader)

			header, err := l2Client.HeaderByNumber(context.Background(), height)
			require.NoError(t, err)
			require.NotNil(t, indexedHeader)

			require.Equal(t, header.Number.Int64(), indexedHeader.Number.Int.Int64())
			require.Equal(t, header.Hash(), indexedHeader.Hash)
			require.Equal(t, header.ParentHash, indexedHeader.ParentHash)
			require.Equal(t, header.Time, indexedHeader.Timestamp)
		}
	})

	t.Run("indexes L2 checkpoints", func(t *testing.T) {
		latestOutput, err := testSuite.DB.Blocks.LatestCheckpointedOutput()
		require.NoError(t, err)
		require.NotNil(t, latestOutput)
		require.GreaterOrEqual(t, latestOutput.L2BlockNumber.Int.Uint64(), uint64(9))

		l2EthClient, err := node.DialEthClient(testSuite.OpSys.Nodes["sequencer"].HTTPEndpoint())
		require.NoError(t, err)

		submissionInterval := testSuite.OpCfg.DeployConfig.L2OutputOracleSubmissionInterval
		numOutputs := latestOutput.L2BlockNumber.Int.Uint64() / submissionInterval
		for i := int64(0); i < int64(numOutputs); i++ {
			blockNumber := big.NewInt((i + 1) * int64(submissionInterval))

			output, err := testSuite.DB.Blocks.OutputProposal(big.NewInt(i))
			require.NoError(t, err)
			require.NotNil(t, output)
			require.Equal(t, i, output.L2OutputIndex.Int.Int64())
			require.Equal(t, blockNumber, output.L2BlockNumber.Int)
			require.NotEmpty(t, output.L1ContractEventGUID)

			// we may as well check the integrity of the output root
			l2Block, err := l2Client.BlockByNumber(context.Background(), blockNumber)
			require.NoError(t, err)
			messagePasserStorageHash, err := l2EthClient.StorageHash(predeploys.L2ToL1MessagePasserAddr, blockNumber)
			require.NoError(t, err)

			// construct and check output root
			outputRootPreImage := [128]byte{}                                 // 4 words (first 32 are zero for version 0)
			copy(outputRootPreImage[32:64], l2Block.Root().Bytes())           // state root
			copy(outputRootPreImage[64:96], messagePasserStorageHash.Bytes()) // message passer storage root
			copy(outputRootPreImage[96:128], l2Block.Hash().Bytes())          // block hash
			require.Equal(t, crypto.Keccak256Hash(outputRootPreImage[:]), output.OutputRoot)
		}
	})

	t.Run("indexes L1 logs and associated blocks", func(t *testing.T) {
		testCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		devContracts := make([]common.Address, 0)
		testSuite.OpCfg.L1Deployments.ForEach(func(name string, address common.Address) {
			devContracts = append(devContracts, address)
		})
		logFilter := ethereum.FilterQuery{FromBlock: big.NewInt(0), ToBlock: big.NewInt(int64(l1Height)), Addresses: devContracts}
		logs, err := l1Client.FilterLogs(testCtx, logFilter) // []types.Log
		require.NoError(t, err)

		for _, log := range logs {
			contractEvent, err := testSuite.DB.ContractEvents.L1ContractEventByTxLogIndex(log.TxHash, uint64(log.Index))
			require.NoError(t, err)
			require.Equal(t, log.Topics[0], contractEvent.EventSignature)
			require.Equal(t, log.BlockHash, contractEvent.BlockHash)
			require.Equal(t, log.TxHash, contractEvent.TransactionHash)
			require.Equal(t, log.Index, uint(contractEvent.LogIndex))

			// ensure the block is also indexed
			block, err := l1Client.BlockByNumber(testCtx, big.NewInt(int64(log.BlockNumber)))
			require.NoError(t, err)

			require.Equal(t, block.Time(), contractEvent.Timestamp)

			l1BlockHeader, err := testSuite.DB.Blocks.L1BlockHeader(block.Number())
			require.NoError(t, err)
			require.Equal(t, block.Hash(), l1BlockHeader.Hash)
			require.Equal(t, block.ParentHash(), l1BlockHeader.ParentHash)
			require.Equal(t, block.Number(), l1BlockHeader.Number.Int)
			require.Equal(t, block.Time(), l1BlockHeader.Timestamp)
		}
	})
}
