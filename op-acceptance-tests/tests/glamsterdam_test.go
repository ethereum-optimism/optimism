package tests

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/forks"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batcherflags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

func TestSafeHeadAdvancesAcrossGlamsterdam(gt *testing.T) {
	t := devtest.ParallelT(gt)
	prepareGlamsterdamOpReth(t)
	sys := presets.NewMinimal(t,
		glamsterdamL1Geth(t),
		presets.WithDeployerOptions(
			sysgo.WithForkAtL1Genesis(forks.BPO5),
			// Leave enough time for the devstack to start, fund the load generators, and
			// produce a loaded block before activating Glamsterdam.
			sysgo.WithForkAtL1Offset(forks.Amsterdam, 120),
		),
	)
	spamGlamsterdamTxs(sys)

	l1Config := sys.L1Network.Escape().ChainConfig()
	t.Require().NotNil(l1Config.AmsterdamTime)
	l1Genesis := sys.L1EL.BlockRefByNumber(0)
	t.Require().Greater(*l1Config.AmsterdamTime, l1Genesis.Time,
		"Glamsterdam must activate after L1 genesis to exercise the fork transition")

	threshold := sys.L2Chain.Escape().RollupConfig().Genesis.SystemConfig.GasLimit/2 + 1
	sys.L2EL.WaitForGasUsed(eth.Unsafe, threshold, 2*time.Minute)
	preForkL1 := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	t.Require().Less(preForkL1.Time, *l1Config.AmsterdamTime,
		"load must begin before Glamsterdam activates")

	postForkL1 := sys.L1EL.WaitForTime(*l1Config.AmsterdamTime)
	postForkHeader, err := sys.L1EL.EthClient().HeaderByHash(t.Ctx(), postForkL1.Hash)
	t.Require().NoError(err)
	t.Require().NotNil(postForkHeader.SlotNumber, "post-Glamsterdam L1 block must include a slot number")
	t.Require().NotNil(postForkHeader.BlockAccessListHash,
		"post-Glamsterdam L1 block must include a block access list hash")

	sys.L2EL.WaitL1OriginReached(eth.Safe, postForkL1.Number, 120)
	sys.L2EL.WaitForGasUsed(eth.Safe, threshold, 2*time.Minute)
}

func TestAutoDASwitchesFromCalldataToBlobsAtGlamsterdam(gt *testing.T) {
	t := devtest.ParallelT(gt)
	prepareGlamsterdamOpReth(t)
	sys := presets.NewMinimal(t,
		glamsterdamL1Geth(t),
		presets.WithDeployerOptions(
			// Activate the stable test blob schedule before setting the large excess blob gas.
			sysgo.WithForkAtL1Genesis(forks.BPO5),
			// Leave enough time to submit a pre-Amsterdam batch before exercising the fork.
			sysgo.WithForkAtL1Offset(forks.Amsterdam, 60),
			withGlamsterdamAutoDABlobFee,
		),
		presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *batcher.CLIConfig) {
			cfg.Stopped = true
			cfg.DataAvailabilityType = batcherflags.AutoType
			cfg.ThrottleConfig.LowerThreshold = 0
			// Bound the execution-gas price while leaving headroom for geth's tip estimate.
			// The configured blob base fee is cheaper only when the Amsterdam calldata
			// floor is included throughout this range.
			cfg.TxMgrConfig.MinBaseFeeGwei = 1
			cfg.TxMgrConfig.MaxBaseFeeGwei = 1
			cfg.TxMgrConfig.MinTipCapGwei = 1
			cfg.TxMgrConfig.MaxTipCapGwei = 1.4
		}),
	)

	l1Config := sys.L1Network.Escape().ChainConfig()
	t.Require().NotNil(l1Config.AmsterdamTime)
	l1Genesis := sys.L1EL.BlockRefByNumber(0)
	t.Require().Greater(*l1Config.AmsterdamTime, l1Genesis.Time,
		"Glamsterdam must activate after L1 genesis to exercise both pricing regimes")

	preForkStart := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	t.Require().Less(preForkStart.Time, *l1Config.AmsterdamTime)
	sys.L2Batcher.Start()
	preForkBatchTx := sys.L2Chain.WaitForBatchTransaction(preForkStart.Number)
	// The provider initializes and falls back to blobs, so observing calldata also proves that
	// the gas-price and L1-header queries succeeded instead of silently taking a fallback path.
	t.Require().Equal(uint8(gethtypes.DynamicFeeTxType), preForkBatchTx.Type(),
		"auto DA must choose calldata under the pre-Amsterdam floor schedule")
	preForkL1 := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	t.Require().Less(preForkL1.Time, *l1Config.AmsterdamTime,
		"the calldata batch must be included before Glamsterdam activates")
	sys.L2Batcher.Stop()

	postForkL1 := sys.L1EL.WaitForTime(*l1Config.AmsterdamTime)
	postForkHeader, err := sys.L1EL.EthClient().HeaderByHash(t.Ctx(), postForkL1.Hash)
	t.Require().NoError(err)
	t.Require().NotNil(postForkHeader.BlockAccessListHash,
		"post-Glamsterdam L1 block must include a block access list hash")

	sys.L2Batcher.Start()
	postForkBatchTx := sys.L2Chain.WaitForBatchTransaction(postForkL1.Number)
	t.Require().Equal(uint8(gethtypes.BlobTxType), postForkBatchTx.Type(),
		"auto DA must switch to blobs when the Amsterdam calldata floor makes them cheaper")
}

func TestGlamsterdamP2PUnsafeBlockBecomesSafe(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNode(t,
		glamsterdamL1Geth(t),
		presets.WithDeployerOptions(sysgo.WithForkAtL1Genesis(forks.Amsterdam)),
	)

	l1Config := sys.L1Network.Escape().ChainConfig()
	t.Require().NotNil(l1Config.AmsterdamTime)
	l1Genesis := sys.L1EL.BlockRefByNumber(0)
	t.Require().LessOrEqual(*l1Config.AmsterdamTime, l1Genesis.Time,
		"Glamsterdam must be active at L1 genesis")

	alice := sys.FunderL2.NewFundedEOA(eth.OneTenthEther)
	bob := sys.Wallet.NewEOA(sys.L2EL)
	sys.L2Batcher.Stop()

	transfer := alice.Transfer(bob.Address(), eth.OneWei)
	receipt := transfer.Included.Value()
	target := bigs.Uint64Strict(receipt.BlockNumber)
	sequencerBlock := sys.L2EL.BlockRefByNumber(target)
	t.Require().Equal(receipt.BlockHash, sequencerBlock.Hash)

	t.Require().NoError(sys.L2ELB.ReachedFn(eth.Unsafe, target, 30)())
	verifierBlock := sys.L2ELB.BlockRefByNumber(target)
	t.Require().Equal(sequencerBlock.ID(), verifierBlock.ID(),
		"verifier must receive the unbatched block over P2P")

	sys.L2Batcher.Start()
	dsl.CheckAll(t,
		sys.L2EL.ReachedFn(eth.Safe, target, 120),
		sys.L2ELB.ReachedFn(eth.Safe, target, 120),
	)
	t.Require().NoError(sys.L2CLB.MatchedFn(sys.L2CL, safety.CrossSafe, 30)(),
		"sequencer and verifier must match after the P2P block becomes safe")
	t.Require().Equal(sequencerBlock.ID(), sys.L2EL.BlockRefByNumber(target).ID())
	t.Require().Equal(sequencerBlock.ID(), sys.L2ELB.BlockRefByNumber(target).ID())

	stoppedUnsafeHash := sys.L2CL.StopSequencer()
	t.Require().NoError(sys.L2CLB.MatchedFn(sys.L2CL, safety.LocalUnsafe, 30)(),
		"verifier must catch up to the sequencer's final unsafe head")
	sequencerUnsafe := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	verifierUnsafe := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	t.Require().Equal(stoppedUnsafeHash, sequencerUnsafe.Hash)
	t.Require().Equal(sequencerUnsafe.ID(), verifierUnsafe.ID(),
		"sequencer and verifier must finish on the same unsafe chain")
}

func withGlamsterdamAutoDABlobFee(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
	// With one 120,000-byte calldata tx, one 130,044-byte blob, and a 2 gwei
	// execution-gas price (1 gwei base fee + 1 gwei tip), a 100 gwei blob base fee gives:
	//
	//   pre-Amsterdam calldata: (21,000 + 40*120,000)*2 / 120,000 = 80.35 gwei/byte
	//   pre-Amsterdam blob:     (21,000*2 + 131,072*100) / 130,044 = 101.11 gwei/byte
	//   Amsterdam calldata:     (15,000 + 64*120,000)*2 / 120,000 = 128.25 gwei/byte
	//   Amsterdam blob:         (15,000*2 + 131,072*100) / 130,044 = 101.02 gwei/byte
	//
	// Solving the per-byte equalities for blob base fee F gives the break-even points:
	//
	//   pre-Amsterdam: F = (80.35*130,044 - 21,000*2) / 131,072 = 79.399 gwei
	//   Amsterdam:     F = (128.25*130,044 - 15,000*2) / 131,072 = 127.015 gwei
	//
	// Thus calldata is cheaper before Amsterdam, while blobs are cheaper after it. The configured
	// tip range allows a 2-2.4 gwei execution-gas price; at the upper bound the corresponding
	// break-even points are 95.279 and 152.418 gwei, so 100 gwei still distinguishes the rules.
	//
	// A large update fraction keeps the fee in that range while the devstack starts. An empty block
	// reduces excess blob gas by 14*131,072, which lowers the fee by only about 0.183%. It takes
	// 126 consecutive empty blocks (about 12m36s at the six-second L1 block time) to fall below
	// 79.4 gwei. Even the 2.4 gwei lower boundary takes 27 empty blocks to cross, while Amsterdam
	// activates after 10; no other component submits blobs before the batcher starts.
	const (
		blobBaseFeeUpdateFraction = uint64(1_000_000_000)
		genesisExcessBlobGas      = uint64(25_328_436_000)
	)
	stableBlobConfig := &params.BlobConfig{
		Target:         params.DefaultBPO4BlobConfig.Target,
		Max:            params.DefaultBPO4BlobConfig.Max,
		UpdateFraction: blobBaseFeeUpdateFraction,
	}
	builder.L1().
		WithL1BlobSchedule(&params.BlobScheduleConfig{
			Cancun:    params.DefaultCancunBlobConfig,
			Prague:    params.DefaultPragueBlobConfig,
			Osaka:     params.DefaultOsakaBlobConfig,
			BPO1:      params.DefaultBPO1BlobConfig,
			BPO2:      params.DefaultBPO2BlobConfig,
			BPO3:      params.DefaultBPO3BlobConfig,
			BPO4:      params.DefaultBPO4BlobConfig,
			BPO5:      stableBlobConfig,
			Amsterdam: stableBlobConfig,
		}).
		WithExcessBlobGas(genesisExcessBlobGas)
}

func glamsterdamL1Geth(t devtest.T) presets.Option {
	out, err := exec.CommandContext(t.Ctx(), "mise", "which", "geth").Output()
	t.Require().NoError(err, "resolve mise-installed Glamsterdam geth")
	return presets.WithL1Geth(strings.TrimSpace(string(out)))
}

func prepareGlamsterdamOpReth(t devtest.T) {
	_, err := (rustbin.Spec{
		SrcDir:  "rust",
		Package: "op-reth",
		Binary:  "op-reth",
	}).EnsureExists(t.Ctx(), t.Logger())
	t.Require().NoError(err, "prepare op-reth before starting the L1 fork activation clock")
}

func spamGlamsterdamTxs(sys *presets.Minimal) {
	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	eoas := loadtest.FundEOAs(sys.T, eth.ThousandEther, 2, l2BlockTime, sys.L2EL, sys.Wallet, sys.FunderL2)

	// Deploy a four-byte infinite-loop runtime. Calls consume their gas limit while adding almost
	// no batch data, so the load schedule fills blocks without delaying derivation itself.
	const gasBurnerInitCode = "0x6004600c60003960046000f35b600056"
	deployed, err := eoas[0].Include(sys.T,
		txplan.WithData(common.FromHex(gasBurnerInitCode)),
		txplan.WithGasLimit(100_000),
	)
	sys.T.Require().NoError(err)
	gasBurnerAddr := deployed.Receipt.ContractAddress
	sys.T.Require().NotEqual(common.Address{}, gasBurnerAddr)

	eoasRR := loadtest.NewRoundRobin(eoas)
	spammer := loadtest.SpammerFunc(func(t devtest.T) error {
		_, err := eoasRR.Get().Include(t,
			txplan.WithTo(&gasBurnerAddr),
			txplan.WithGasLimit(16_000_000),
		)
		return err
	})
	schedule := loadtest.NewBurst(l2BlockTime, loadtest.WithBaseRPS(2))

	ctx, cancel := context.WithCancel(sys.T.Ctx())
	var wg sync.WaitGroup
	wg.Add(1)
	sys.T.Cleanup(func() {
		cancel()
		wg.Wait()
	})
	go func() {
		defer wg.Done()
		schedule.Run(sys.T.WithCtx(ctx), spammer)
	}()
}
