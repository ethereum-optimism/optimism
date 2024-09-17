package da

import (
	"context"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestSystemE2EDencunAtGenesis tests if L2 finalizes when blobs are present on L1
func TestSystemE2EDencunAtGenesisWithBlobs(t *testing.T) {
	op_e2e.InitParallel(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := e2esys.DefaultSystemConfig(t)
	cfg.DeployConfig.L1CancunTimeOffset = new(hexutil.Uint64)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")

	// send a blob-containing txn on l1
	ethPrivKey := sys.Cfg.Secrets.Alice
	txData := transactions.CreateEmptyBlobTx(true, sys.Cfg.L1ChainIDBig().Uint64())
	tx := types.MustSignNewTx(ethPrivKey, types.LatestSignerForChainID(cfg.L1ChainIDBig()), txData)
	// send blob-containing txn
	sendCtx, sendCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer sendCancel()

	l1Client := sys.NodeClient("l1")
	err = l1Client.SendTransaction(sendCtx, tx)
	require.NoError(t, err, "Sending L1 empty blob tx")
	// Wait for transaction on L1
	blockContainsBlob, err := wait.ForReceiptOK(ctx, l1Client, tx.Hash())
	require.Nil(t, err, "Waiting for blob tx on L1")
	// end sending blob-containing txns on l1
	l2Client := sys.NodeClient("sequencer")
	finalizedBlock, err := geth.WaitForL1OriginOnL2(sys.RollupConfig, blockContainsBlob.BlockNumber.Uint64(), l2Client, 30*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L1 origin of blob tx on L2")
	finalizationTimeout := 30 * time.Duration(cfg.DeployConfig.L1BlockTime) * time.Second
	_, err = geth.WaitForBlockToBeSafe(finalizedBlock.Header().Number, l2Client, finalizationTimeout)
	require.Nil(t, err, "Waiting for safety of L2 block")
}
