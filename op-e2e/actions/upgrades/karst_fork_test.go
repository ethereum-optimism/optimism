package upgrades

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestKarstNetworkUpgradeTransactions(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, helpers.DefaultRollupTestParams())
	l := testlog.Logger(t, log.LvlDebug)

	dp.DeployConfig.ActivateForkAtOffset(forks.Karst, 4)
	require.NoError(t, dp.DeployConfig.Check(l))

	sd := e2eutils.Setup(t, dp, helpers.DefaultAlloc)
	_, _, _, sequencer, engine, _, _, _ := helpers.SetupReorgTestActors(t, dp, sd, l)
	ethCl := engine.EthClient()
	ctx := context.Background()

	sequencer.ActL2PipelineFull(t)

	sequencer.ActBuildL2ToFork(t, forks.Karst)

	latestBlock, err := ethCl.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, sequencer.L2Unsafe().Number, latestBlock.NumberU64())

	// Verify correct NUT count: L1 info tx + all NUTs from embedded bundle.
	nutTxs, nutGas, err := derive.UpgradeTransactions(forks.Karst)
	require.NoError(t, err)
	transactions := latestBlock.Transactions()
	require.Equal(t, 1+len(nutTxs), len(transactions),
		"activation block must contain L1 info tx + all Karst NUTs")

	// All NUT receipts must succeed — iNUTB-005.
	var failedNUTs []int
	for i := 1; i < len(transactions); i++ {
		receipt, err := ethCl.TransactionReceipt(ctx, transactions[i].Hash())
		require.NoError(t, err)
		t.Logf("NUT %d: status=%d gasUsed=%d gasLimit=%d", i, receipt.Status, receipt.GasUsed, transactions[i].Gas())
		if receipt.Status != types.ReceiptStatusSuccessful {
			failedNUTs = append(failedNUTs, i)
		}
		require.NotEmpty(t, transactions[i].Data(), "upgrade tx must have calldata")
	}
	require.Empty(t, failedNUTs, "NUT txs with status=0 (failed): %v", failedNUTs)

	// Block gas limit must be expanded by total NUT gas — iUBGL-001 / iUBGL-003.
	prevBlock, err := ethCl.BlockByNumber(ctx, new(big.Int).Sub(latestBlock.Number(), big.NewInt(1)))
	require.NoError(t, err)
	require.Equal(t, prevBlock.GasLimit()+nutGas, latestBlock.GasLimit(),
		"upgrade block gas limit must be expanded by sum of NUT gas limits")

	require.True(t, sd.RollupCfg.IsKarst(latestBlock.Time()))
}
