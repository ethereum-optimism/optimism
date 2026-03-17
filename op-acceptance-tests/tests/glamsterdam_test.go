package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/params/forks"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

func TestSafeHeadAdvancesAfterGlamsterdam(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// We don't want to activate at genesis since our head will automatically be safe after glamsterdam.
	// It's nice that we produce a few blocks before doing the transition to Amsterdam, to test the
	// full fork transition.
	sys := presets.NewSingleChainMultiNode(t, presets.WithDeployerOptions(sysgo.WithForkAtL1Offset(forks.Amsterdam, 15)))

	spamTxs(sys)

	// Wait for Amsterdam to activate.
	var amsterdamL1Origin eth.BlockInfo
	for {
		info, _, err := sys.L1EL.EthClient().InfoAndTxsByLabel(t.Ctx(), eth.Unsafe)
		t.Require().NoError(err)
		if info.Header().SlotNumber != nil { // Check for Amsterdam feature.
			amsterdamL1Origin = info
			break
		}
		waitForTwoSeconds(t)
	}

	// Wait for the L2 safe head's L1 origin to advance on both chains to at least the amsterdamOrigin.
	// In other words: the entire sequencer -> batcher -> verifier flow works post-Amsterdam.
	for {
		safeA := sys.L2EL.BlockRefByLabel(eth.Safe)
		safeB := sys.L2EL.BlockRefByLabel(eth.Safe)

		isAmsterdam := safeA.L1Origin.Number >= amsterdamL1Origin.NumberU64() && safeB.L1Origin.Number >= amsterdamL1Origin.NumberU64()
		if isAmsterdam {
			break
		}

		waitForTwoSeconds(t)
	}

	// Ensure that the safe heads are at least half full.
	threshold := sys.L2Chain.Escape().RollupConfig().Genesis.SystemConfig.GasLimit/2 + 1
	for {
		safeA := sys.L2EL.BlockRefByLabel(eth.Safe)
		safeB := sys.L2EL.BlockRefByLabel(eth.Safe)

		infoA, err := sys.L2EL.Escape().L2EthClient().InfoByHash(t.Ctx(), safeA.Hash)
		t.Require().NoError(err)
		infoB, err := sys.L2ELB.Escape().L2EthClient().InfoByHash(t.Ctx(), safeB.Hash)
		t.Require().NoError(err)

		hasEnoughTraffic := infoA.GasUsed() >= threshold && infoB.GasUsed() >= threshold
		if hasEnoughTraffic {
			return
		}

		waitForTwoSeconds(t)
	}
}

func waitForTwoSeconds(t devtest.T) {
	select {
	case <-time.After(time.Second * 2):
	case <-t.Ctx().Done():
		t.Require().Fail("context canceled before test could finish")
	}
}

func spamTxs(sys *presets.SingleChainMultiNode) {
	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	eoas := loadtest.FundEOAs(sys.T, eth.HundredEther, 100, l2BlockTime, sys.L2EL, sys.Wallet, sys.FaucetL2)
	eoasRR := loadtest.NewRoundRobin(eoas)
	spammer := loadtest.SpammerFunc(func(t devtest.T) error {
		_, err := eoasRR.Get().Include(t, txplan.WithTo(&predeploys.L1BlockAddr), txplan.WithGasLimit(50_000))
		return err
	})
	schedule := loadtest.NewBurst(l2BlockTime, loadtest.WithBaseRPS(200))
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
