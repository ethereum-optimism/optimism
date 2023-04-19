package op_e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	opp "github.com/ethereum-optimism/optimism/op-program/host"
	oppconf "github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestVerifyL2OutputRoot(t *testing.T) {
	parallel(t)
	ctx := context.Background()

	cfg := DefaultSystemConfig(t)
	// We don't need a verifier - just the sequencer is enough
	delete(cfg.Nodes, "verifier")

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	log := testlog.Logger(t, log.LvlInfo)
	log.Info("genesis", "l2", sys.RollupConfig.Genesis.L2, "l1", sys.RollupConfig.Genesis.L1, "l2_time", sys.RollupConfig.Genesis.L2Time)

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	rollupRPCClient, err := rpc.DialContext(context.Background(), sys.RollupNodes["sequencer"].HTTPEndpoint())
	require.Nil(t, err)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))

	// TODO (CLI-3855): Actually perform some tx to set up a more complex chain.

	// Wait for the safe head to reach block 10
	require.NoError(t, waitForSafeHead(ctx, 10, rollupClient))

	// Use block 5 as the agreed starting block on L2
	l2AgreedBlock, err := l2Seq.BlockByNumber(ctx, big.NewInt(5))
	require.NoError(t, err, "could not retrieve l2 genesis")
	l2Head := l2AgreedBlock.Hash() // Agreed starting L2 block

	// Get the expected output at block 10
	l2ClaimBlockNumber := uint64(10)
	l2Output, err := rollupClient.OutputAtBlock(ctx, l2ClaimBlockNumber)
	require.NoError(t, err, "could not get expected output")
	l2Claim := l2Output.OutputRoot

	// Find the current L1 head
	l1BlockNumber, err := l1Client.BlockNumber(ctx)
	require.NoError(t, err, "get l1 head block number")
	l1HeadBlock, err := l1Client.BlockByNumber(ctx, new(big.Int).SetUint64(l1BlockNumber))
	require.NoError(t, err, "get l1 head block")
	l1Head := l1HeadBlock.Hash()

	preimageDir := t.TempDir()
	fppConfig := oppconf.NewConfig(sys.RollupConfig, sys.L2GenesisCfg.Config, l1Head, l2Head, common.Hash(l2Claim), l2ClaimBlockNumber)
	fppConfig.L1URL = sys.NodeEndpoint("l1")
	fppConfig.L2URL = sys.NodeEndpoint("sequencer")
	fppConfig.DataDir = preimageDir

	// Check the FPP confirms the expected output
	t.Log("Running fault proof in fetching mode")
	err = opp.FaultProofProgram(log, fppConfig)
	require.NoError(t, err)

	// Shutdown the nodes from the actual chain. Should now be able to run using only the pre-fetched data.
	for _, node := range sys.Nodes {
		require.NoError(t, node.Close())
	}

	t.Log("Running fault proof in offline mode")
	// Should be able to rerun in offline mode using the pre-fetched images
	fppConfig.L1URL = ""
	fppConfig.L2URL = ""
	err = opp.FaultProofProgram(log, fppConfig)
	require.NoError(t, err)

	// Check that a fault is detected if we provide an incorrect claim
	t.Log("Running fault proof with invalid claim")
	fppConfig.L2Claim = common.Hash{0xaa}
	err = opp.FaultProofProgram(log, fppConfig)
	require.ErrorIs(t, err, opp.ErrClaimNotValid)
}

func waitForSafeHead(ctx context.Context, safeBlockNum uint64, rollupClient *sources.RollupClient) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		seqStatus, err := rollupClient.SyncStatus(ctx)
		if err != nil {
			return err
		}
		if seqStatus.SafeL2.Number >= safeBlockNum {
			return nil
		}
	}
}
