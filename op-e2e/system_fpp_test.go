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
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestVerifyL2OutputRoot(t *testing.T) {
	InitParallel(t)
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

	t.Log("Sending transactions to setup existing state, prior to challenged period")
	aliceKey := cfg.Secrets.Alice
	opts, err := bind.NewKeyedTransactorWithChainID(aliceKey, cfg.L1ChainIDBig())
	require.Nil(t, err)
	SendDepositTx(t, cfg, l1Client, l2Seq, opts, func(l2Opts *DepositTxOpts) {
		l2Opts.Value = big.NewInt(100_000_000)
	})
	SendL2Tx(t, cfg, l2Seq, aliceKey, func(opts *TxOpts) {
		opts.ToAddr = &cfg.Secrets.Addresses().Bob
		opts.Value = big.NewInt(1_000)
		opts.Nonce = 1
	})
	SendWithdrawal(t, cfg, l2Seq, aliceKey, func(opts *WithdrawalTxOpts) {
		opts.Value = big.NewInt(500)
		opts.Nonce = 2
	})

	t.Log("Capture current L2 head as agreed starting point")
	l2AgreedBlock, err := l2Seq.BlockByNumber(ctx, nil)
	require.NoError(t, err, "could not retrieve l2 agreed block")
	l2Head := l2AgreedBlock.Hash()

	t.Log("Sending transactions to modify existing state, within challenged period")
	SendDepositTx(t, cfg, l1Client, l2Seq, opts, func(l2Opts *DepositTxOpts) {
		l2Opts.Value = big.NewInt(5_000)
	})
	SendL2Tx(t, cfg, l2Seq, cfg.Secrets.Bob, func(opts *TxOpts) {
		opts.ToAddr = &cfg.Secrets.Addresses().Alice
		opts.Value = big.NewInt(100)
	})
	SendWithdrawal(t, cfg, l2Seq, aliceKey, func(opts *WithdrawalTxOpts) {
		opts.Value = big.NewInt(100)
		opts.Nonce = 4
	})

	t.Log("Determine L2 claim")
	l2ClaimBlockNumber, err := l2Seq.BlockNumber(ctx)
	require.NoError(t, err, "get L2 claim block number")
	l2Output, err := rollupClient.OutputAtBlock(ctx, l2ClaimBlockNumber)
	require.NoError(t, err, "could not get expected output")
	l2Claim := l2Output.OutputRoot

	t.Log("Determine L1 head that includes all batches required for L2 claim block")
	require.NoError(t, waitForSafeHead(ctx, l2ClaimBlockNumber, rollupClient))
	l1HeadBlock, err := l1Client.BlockByNumber(ctx, nil)
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

	t.Log("Shutting down network")
	// Shutdown the nodes from the actual chain. Should now be able to run using only the pre-fetched data.
	sys.BatchSubmitter.StopIfRunning(context.Background())
	sys.L2OutputSubmitter.Stop()
	sys.L2OutputSubmitter = nil
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
