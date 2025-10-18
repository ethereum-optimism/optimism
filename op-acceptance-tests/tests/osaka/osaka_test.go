package osaka

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// configureDevstackEnvVars sets the appropriate env vars to use a mise-installed geth binary for
// the L1 EL. This is useful in Osaka acceptance tests since op-geth does not include full Osaka
// support. This is meant to run before presets.DoMain in a TestMain function. It will log to
// stdout. ResetDevstackEnvVars should be used to reset the environment variables when TestMain
// exits.
//
// Note that this is a no-op if either [sysgo.DevstackL1ELKindVar] or [sysgo.GethExecPathEnvVar]
// are set.
//
// The returned callback resets any modified environment variables.
func configureDevstackEnvVars() func() {
	if _, ok := os.LookupEnv(sysgo.DevstackL1ELKindEnvVar); ok {
		return func() {}
	}
	if _, ok := os.LookupEnv(sysgo.GethExecPathEnvVar); ok {
		return func() {}
	}

	cmd := exec.Command("mise", "which", "geth")
	buf := bytes.NewBuffer([]byte{})
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to find mise-installed geth: %v\n", err)
		return func() {}
	}
	execPath := strings.TrimSpace(buf.String())
	fmt.Println("Found mise-installed geth:", execPath)
	_ = os.Setenv(sysgo.GethExecPathEnvVar, execPath)
	_ = os.Setenv(sysgo.DevstackL1ELKindEnvVar, "geth")
	return func() {
		_ = os.Unsetenv(sysgo.GethExecPathEnvVar)
		_ = os.Unsetenv(sysgo.DevstackL1ELKindEnvVar)
	}
}

func TestMain(m *testing.M) {
	resetEnvVars := configureDevstackEnvVars()
	defer resetEnvVars()

	presets.DoMain(m, stack.MakeCommon(stack.Combine[*sysgo.Orchestrator](
		sysgo.DefaultMinimalSystem(&sysgo.DefaultMinimalSystemIDs{}),
		sysgo.WithDeployerOptions(func(_ devtest.P, _ devkeys.Keys, builder intentbuilder.Builder) {
			_, l1Config := builder.WithL1(sysgo.DefaultL1ID)
			l1Config.WithOsakaOffset(0)
			l1Config.WithBPO1Offset(0)
			l1Config.WithL1BlobSchedule(&params.BlobScheduleConfig{
				Cancun: params.DefaultCancunBlobConfig,
				Osaka:  params.DefaultOsakaBlobConfig,
				Prague: params.DefaultPragueBlobConfig,
				BPO1:   params.DefaultBPO1BlobConfig,
			})
		}),
		sysgo.WithBatcherOption(func(_ stack.L2BatcherID, cfg *batcher.CLIConfig) {
			cfg.DataAvailabilityType = flags.BlobsType
			cfg.TxMgrConfig.CellProofTime = 0 // Force cell proofs to be used
		}),
	)))
}

func TestBatcherUsesNewSidecarFormatAfterOsaka(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	t.Log("Waiting for Osaka to activate")
	t.Require().NotNil(sys.L1Network.Escape().ChainConfig().OsakaTime)
	sys.L1EL.WaitForTime(*sys.L1Network.Escape().ChainConfig().OsakaTime)
	t.Log("Osaka activated")

	// 1. Wait for the sequencer to build a block after Osaka is activated. This avoids a race
	//    condition where the unsafe head has been posted as part of a blob, but has not been
	//    marked as "safe" yet.
	sys.L2EL.WaitForBlock()

	// 2. Wait for the batcher to include target in a batch and post it to L1. Because the batch is
	//    posted after Osaka has activated, it means the batcher must have successfully used the
	//    new format.
	target := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	blockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	for range time.Tick(blockTime) {
		if sys.L2EL.BlockRefByLabel(eth.Safe).Number >= target.Number {
			// If the safe head is ahead of the target height and the target block is part of the
			// canonical chain, then the target block is safe.
			_, err := sys.L2EL.Escape().EthClient().BlockRefByHash(t.Ctx(), target.Hash)
			t.Require().NoError(err)
			return
		}
	}
}

func TestBlobBaseFeeIsCorrectAfterBPOFork(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	t.Log("Waiting for BPO1 to activate")
	t.Require().NotNil(sys.L1Network.Escape().ChainConfig().BPO1Time)
	sys.L1EL.WaitForTime(*sys.L1Network.Escape().ChainConfig().BPO1Time)
	t.Log("BPO1 activated")

	sys.L1EL.WaitForBlock()
	l1BlockTime := sys.L1EL.EstimateBlockTime()
	l1ChainConfig := sys.L1Network.Escape().ChainConfig()

	spamBlobs(t, sys) // Raise the blob base fee to make blob parameter changes visible.

	// Wait for the blob base fee to rise above 1 so the blob parameter changes will be visible.
	for range time.Tick(l1BlockTime) {
		info, _, err := sys.L1EL.EthClient().InfoAndTxsByLabel(t.Ctx(), eth.Unsafe)
		t.Require().NoError(err)
		if calcBlobBaseFee(l1ChainConfig, info).Cmp(big.NewInt(1)) > 0 {
			break
		}
		t.Logf("Waiting for blob base fee to rise above 1")
	}

	l2UnsafeRef := sys.L2CL.SyncStatus().UnsafeL2

	// Get the L1 blob base fee.
	l1OriginInfo, err := sys.L1EL.EthClient().InfoByHash(t.Ctx(), l2UnsafeRef.L1Origin.Hash)
	t.Require().NoError(err)
	l1BlobBaseFee := calcBlobBaseFee(l1ChainConfig, l1OriginInfo)

	// Get the L2 blob base fee from the system deposit tx.
	info, txs, err := sys.L2EL.Escape().EthClient().InfoAndTxsByHash(t.Ctx(), l2UnsafeRef.Hash)
	t.Require().NoError(err)
	blockInfo, err := derive.L1BlockInfoFromBytes(sys.L2Chain.Escape().RollupConfig(), info.Time(), txs[0].Data())
	t.Require().NoError(err)
	l2BlobBaseFee := blockInfo.BlobBaseFee

	t.Require().Equal(l1BlobBaseFee, l2BlobBaseFee)
}

func spamBlobs(t devtest.T, sys *presets.Minimal) {
	l1BlockTime := sys.L1EL.EstimateBlockTime()
	l1ChainConfig := sys.L1Network.Escape().ChainConfig()

	eoa := sys.FunderL1.NewFundedEOA(eth.OneEther.Mul(5))
	signer := txinclude.NewPkSigner(eoa.Key().Priv(), sys.L1Network.ChainID().ToBig())
	l1ETHClient := sys.L1EL.EthClient()
	syncEOA := loadtest.NewSyncEOA(txinclude.NewPersistent(signer, struct {
		*txinclude.Monitor
		*txinclude.Resubmitter
	}{
		txinclude.NewMonitor(l1ETHClient, l1BlockTime),
		txinclude.NewResubmitter(l1ETHClient, l1BlockTime),
	}), eoa.Plan())

	var blob eth.Blob
	_, err := rand.Read(blob[:])
	t.Require().NoError(err)
	// get the field-elements into a valid range
	for i := range 4096 {
		blob[32*i] &= 0b0011_1111
	}

	const maxBlobTxsPerAccountInMempool = 16 // Private policy param in geth.
	spammer := loadtest.SpammerFunc(func(t devtest.T) error {
		_, err := syncEOA.Include(t, txplan.WithBlobs([]*eth.Blob{&blob}, l1ChainConfig), txplan.WithTo(&common.Address{}))
		return err
	})
	txsPerSlot := min(l1ChainConfig.BlobScheduleConfig.BPO1.Max*3/4, maxBlobTxsPerAccountInMempool)
	schedule := loadtest.NewConstant(l1BlockTime, loadtest.WithBaseRPS(uint64(txsPerSlot)))

	ctx, cancel := context.WithCancel(t.Ctx())
	var wg sync.WaitGroup
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
	wg.Add(1)
	go func() {
		defer wg.Done()
		schedule.Run(t.WithCtx(ctx), spammer)
	}()
}

func calcBlobBaseFee(cfg *params.ChainConfig, info eth.BlockInfo) *big.Int {
	return eip4844.CalcBlobFee(cfg, &types.Header{
		// It's unfortunate that we can't build a proper header from a BlockInfo.
		// We do our best to work around deficiencies in the BlockInfo implementation here.
		Time:          info.Time(),
		ExcessBlobGas: info.ExcessBlobGas(),
	})
}
