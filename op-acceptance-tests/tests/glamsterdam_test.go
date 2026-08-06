package tests

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params/forks"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
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
