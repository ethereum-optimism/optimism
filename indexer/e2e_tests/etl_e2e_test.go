package e2e_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/stretchr/testify/require"
)

func TestE2EETL(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	l2OutputOracle, err := bindings.NewL2OutputOracle(testSuite.OpCfg.L1Deployments.L2OutputOracleProxy, testSuite.L1Client)
	require.NoError(t, err)

	// wait for at least 10 L2 blocks posted on L1
	require.NoError(t, wait.For(context.Background(), time.Second, func() (bool, error) {
		l2Height, err := l2OutputOracle.LatestBlockNumber(&bind.CallOpts{Context: context.Background()})
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
