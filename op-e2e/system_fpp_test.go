package op_e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-program/client/driver"
	opp "github.com/ethereum-optimism/optimism/op-program/host"
	oppconf "github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestVerifyL2OutputRoot(t *testing.T) {
	testVerifyL2OutputRoot(t, false)
}

func TestVerifyL2OutputRootDetached(t *testing.T) {
	testVerifyL2OutputRoot(t, true)
}

func TestVerifyL2OutputRootEmptyBlock(t *testing.T) {
	testVerifyL2OutputRootEmptyBlock(t, false)
}

func TestVerifyL2OutputRootEmptyBlockDetached(t *testing.T) {
	testVerifyL2OutputRootEmptyBlock(t, true)
}

// TestVerifyL2OutputRootEmptyBlock asserts that the program can verify the output root of an empty block
// induced by missing batches.
// Setup is as follows:
// - create initial conditions and agreed l2 state
// - stop the batch submitter to induce empty blocks
// - wait for the seq window to expire so we can observe empty blocks
// - select an empty block as our claim
// - reboot the batch submitter
// - update the state root via a tx
// - run program
func testVerifyL2OutputRootEmptyBlock(t *testing.T, detached bool) {
	InitParallel(t)
	ctx := context.Background()

	cfg := DefaultSystemConfig(t)
	// We don't need a verifier - just the sequencer is enough
	delete(cfg.Nodes, "verifier")
	// Use a small sequencer window size to avoid test timeout while waiting for empty blocks
	// But not too small to ensure that our claim and subsequent state change is published
	cfg.DeployConfig.SequencerWindowSize = 16

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

	// Avoids flaky test by avoiding reorgs at epoch 0
	t.Log("Wait for safe head to advance once for setup")
	ss, err := l2Seq.BlockByNumber(ctx, big.NewInt(int64(rpc.SafeBlockNumber)))
	require.NoError(t, err)
	require.NoError(t, waitForSafeHead(ctx, ss.NumberU64()+cfg.DeployConfig.SequencerWindowSize+1, rollupClient))

	t.Log("Sending transactions to setup existing state, prior to challenged period")
	aliceKey := cfg.Secrets.Alice
	receipt := SendL2Tx(t, cfg, l2Seq, aliceKey, func(opts *TxOpts) {
		opts.ToAddr = &cfg.Secrets.Addresses().Bob
		opts.Value = big.NewInt(1_000)
	})
	require.NoError(t, waitForSafeHead(ctx, receipt.BlockNumber.Uint64(), rollupClient))

	t.Logf("Capture current L2 head as agreed starting point. l2Head=%x l2BlockNumber=%v", receipt.BlockHash, receipt.BlockNumber)
	l2Head := receipt.BlockHash

	t.Log("=====Stopping batch submitter=====")
	err = sys.BatchSubmitter.Stop(ctx)
	require.NoError(t, err, "could not stop batch submitter")

	// Wait for the sequencer to catch up with the current L1 head so we know all submitted batches are processed
	t.Log("Wait for sequencer to catch up with last submitted batch")
	l1HeadNum, err := l1Client.BlockNumber(ctx)
	require.NoError(t, err)
	_, err = waitForL1OriginOnL2(l1HeadNum, l2Seq, 30*time.Second)
	require.NoError(t, err)

	// Get the current safe head now that the batcher is stopped
	safeBlock, err := l2Seq.BlockByNumber(ctx, big.NewInt(int64(rpc.SafeBlockNumber)))
	require.NoError(t, err)

	// Wait for safe head to start advancing again when the sequencing window elapses, for at least three blocks
	t.Log("Wait for safe head to advance after sequencing window elapses")
	require.NoError(t, waitForSafeHead(ctx, safeBlock.NumberU64()+3, rollupClient))

	// Use the 2nd empty block as our L2 claim block
	t.Log("Determine L2 claim")
	l2ClaimBlock, err := l2Seq.BlockByNumber(ctx, big.NewInt(int64(safeBlock.NumberU64()+2)))
	require.NoError(t, err, "get L2 claim block number")
	l2ClaimBlockNumber := l2ClaimBlock.Number().Uint64()
	l2Output, err := rollupClient.OutputAtBlock(ctx, l2ClaimBlockNumber)
	require.NoError(t, err, "could not get expected output")
	l2Claim := l2Output.OutputRoot

	t.Log("=====Restarting batch submitter=====")
	err = sys.BatchSubmitter.Start()
	require.NoError(t, err, "could not start batch submitter")

	t.Log("Add a transaction to the next batch after sequence of empty blocks")
	receipt = SendL2Tx(t, cfg, l2Seq, aliceKey, func(opts *TxOpts) {
		opts.ToAddr = &cfg.Secrets.Addresses().Bob
		opts.Value = big.NewInt(1_000)
		opts.Nonce = 1
	})
	require.NoError(t, waitForSafeHead(ctx, receipt.BlockNumber.Uint64(), rollupClient))

	t.Log("Determine L1 head that includes batch after sequence of empty blocks")
	l1HeadBlock, err := l1Client.BlockByNumber(ctx, nil)
	require.NoError(t, err, "get l1 head block")
	l1Head := l1HeadBlock.Hash()

	testFaultProofProgramScenario(t, ctx, sys, &FaultProofProgramTestScenario{
		L1Head:             l1Head,
		L2Head:             l2Head,
		L2Claim:            common.Hash(l2Claim),
		L2ClaimBlockNumber: l2ClaimBlockNumber,
		Detached:           detached,
	})
}

func testVerifyL2OutputRoot(t *testing.T, detached bool) {
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

	testFaultProofProgramScenario(t, ctx, sys, &FaultProofProgramTestScenario{
		L1Head:             l1Head,
		L2Head:             l2Head,
		L2Claim:            common.Hash(l2Claim),
		L2ClaimBlockNumber: l2ClaimBlockNumber,
		Detached:           detached,
	})
}

type FaultProofProgramTestScenario struct {
	L1Head             common.Hash
	L2Head             common.Hash
	L2Claim            common.Hash
	L2ClaimBlockNumber uint64
	Detached           bool
}

// testFaultProofProgramScenario runs the fault proof program in several contexts, given a test scenario.
func testFaultProofProgramScenario(t *testing.T, ctx context.Context, sys *System, s *FaultProofProgramTestScenario) {
	preimageDir := t.TempDir()
	fppConfig := oppconf.NewConfig(sys.RollupConfig, sys.L2GenesisCfg.Config, s.L1Head, s.L2Head, common.Hash(s.L2Claim), s.L2ClaimBlockNumber)
	fppConfig.L1URL = sys.NodeEndpoint("l1")
	fppConfig.L2URL = sys.NodeEndpoint("sequencer")
	fppConfig.DataDir = preimageDir
	if s.Detached {
		// When running in detached mode we need to compile the client executable since it will be called directly.
		fppConfig.ExecCmd = BuildOpProgramClient(t)
	}

	// Check the FPP confirms the expected output
	t.Log("Running fault proof in fetching mode")
	log := testlog.Logger(t, log.LvlInfo)
	err := opp.FaultProofProgram(ctx, log, fppConfig)
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
	err = opp.FaultProofProgram(ctx, log, fppConfig)
	require.NoError(t, err)

	// Check that a fault is detected if we provide an incorrect claim
	t.Log("Running fault proof with invalid claim")
	fppConfig.L2Claim = common.Hash{0xaa}
	err = opp.FaultProofProgram(ctx, log, fppConfig)
	if s.Detached {
		require.Error(t, err, "exit status 1")
	} else {
		require.ErrorIs(t, err, driver.ErrClaimNotValid)
	}
}

func waitForSafeHead(ctx context.Context, safeBlockNum uint64, rollupClient *sources.RollupClient) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
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
