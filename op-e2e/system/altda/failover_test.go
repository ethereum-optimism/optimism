package altda

import (
	"math/big"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/stretchr/testify/require"
)

// TestBatcher_FailoverToEthDA_FallbackToAltDA tests that the batcher will failover to ethDA
// if the da-server returns 503. It also tests that the batcher successfully returns to normal
// behavior of posting batches to altda once it becomes available again
// (i.e. the da-server doesn't return 503 anymore).
func TestBatcher_FailoverToEthDA_FallbackToAltDA(t *testing.T) {
	op_e2e.InitParallel(t)

	nChannelsFailover := uint64(2)

	cfg := e2esys.DefaultSystemConfig(t, e2esys.WithLogLevel(log.LevelCrit))
	cfg.DeployConfig.UseAltDA = true
	cfg.DeployConfig.DACommitmentType = "GenericCommitment"
	cfg.DeployConfig.DAChallengeWindow = 16
	cfg.DeployConfig.DAResolveWindow = 16
	cfg.DeployConfig.DABondSize = 1000000
	cfg.DeployConfig.DAResolverRefundPercentage = 0
	// Default cfg.BatcherMaxChannelDuration is 1, which means at least one channel is sent per L1 block.
	// Furthermore, by setting cfg.BatcherMaxPendingTransactions = 1,
	// we make sure the batcher posts a single commitment per L1 block.
	// This way it's easy to trigger failover and observe the commitment changing on the next L1 block.
	cfg.BatcherMaxPendingTransactions = 1
	cfg.BatcherMaxConcurrentDARequest = 1
	cfg.BatcherBatchType = 0
	// currently altda commitments can only be sent as calldata
	cfg.DataAvailabilityType = flags.CalldataType

	sys, err := cfg.Start(t)
	require.NoError(t, err, "Error starting up system")
	defer sys.Close()
	l1Client := sys.NodeClient("l1")

	startBlockL1, err := geth.WaitForBlockWithTxFromSender(cfg.DeployConfig.BatchSenderAddress, l1Client, 10)
	require.NoError(t, err)

	// Simulate altda server returning 503
	sys.FakeAltDAServer.SetPutFailoverForNRequests(nChannelsFailover)

	countEthDACommitment := uint64(0)

	// There is some nondeterministic timing behavior that affects whether the batcher has already
	// posted batches before seeing the above SetPutFailoverForNRequests behavior change.
	// Most likely, sequence of blocks will be: altDA, ethDA, ethDA, altDA, altDA, altDA.
	// 2 ethDA are expected (and checked for) because nChannelsFailover=2, so da-server will return 503 for 2 requests only,
	// and the batcher always tries altda first for a new channel, and failsover to ethDA only if altda returns 503.
	for blockNumL1 := startBlockL1.NumberU64(); blockNumL1 < startBlockL1.NumberU64()+6; blockNumL1++ {
		blockL1, err := geth.WaitForBlock(big.NewInt(0).SetUint64(blockNumL1), l1Client)
		require.NoError(t, err)
		batcherTxs, err := transactions.TransactionsBySender(blockL1, cfg.DeployConfig.BatchSenderAddress)
		require.NoError(t, err)
		require.Equal(t, 1, len(batcherTxs)) // sanity check: ensure BatcherMaxPendingTransactions=1 is working
		batcherTx := batcherTxs[0]
		if batcherTx.Data()[0] == byte(params.DerivationVersion0) {
			countEthDACommitment++
			t.Log("blockL1", blockNumL1, "batcherTxType", "ethda")
		} else if batcherTx.Data()[0] == byte(params.DerivationVersion1) {
			t.Log("blockL1", blockNumL1, "batcherTxType", "altda")
		} else {
			t.Fatalf("unexpected batcherTxType: %v", batcherTx.Data()[0])
		}
	}
	require.Equal(t, nChannelsFailover, countEthDACommitment, "Expected %v ethDA commitments, got %v", nChannelsFailover, countEthDACommitment)

}
