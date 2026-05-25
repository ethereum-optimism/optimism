package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params/forks"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-glamsterdam/glamsterdamtest"
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

	spamTxsMultiNode(sys)

	pollInterval := 2 * time.Second

	// Wait for Amsterdam to activate on L1.
	l1Fetch := func(ctx context.Context) (*types.Header, error) {
		return sys.L1EL.Escape().EthClient().HeaderByLabel(ctx, eth.Unsafe)
	}
	amsterdamHdr, err := glamsterdamtest.WaitForAmsterdamOnL1(t.Ctx(), t.Logger(), l1Fetch, pollInterval)
	t.Require().NoError(err)
	target := amsterdamHdr.Number.Uint64()

	// Both verifiers (A and B) must reach Amsterdam on their L2 safe head's L1 origin.
	_, err = glamsterdamtest.WaitForSafeHeadPastL1(t.Ctx(), t.Logger(), sys.L2CL.Escape().RollupAPI(), target, pollInterval)
	t.Require().NoError(err)
	_, err = glamsterdamtest.WaitForSafeHeadPastL1(t.Ctx(), t.Logger(), sys.L2CLB.Escape().RollupAPI(), target, pollInterval)
	t.Require().NoError(err)

	// Ensure that the safe heads on both verifiers are at least half full.
	threshold := sys.L2Chain.Escape().RollupConfig().Genesis.SystemConfig.GasLimit/2 + 1
	gasA := func(ctx context.Context, hash common.Hash) (uint64, error) {
		info, err := sys.L2EL.Escape().L2EthClient().InfoByHash(ctx, hash)
		if err != nil {
			return 0, err
		}
		return info.GasUsed(), nil
	}
	gasB := func(ctx context.Context, hash common.Hash) (uint64, error) {
		info, err := sys.L2ELB.Escape().L2EthClient().InfoByHash(ctx, hash)
		if err != nil {
			return 0, err
		}
		return info.GasUsed(), nil
	}
	for {
		safeA := sys.L2EL.BlockRefByLabel(eth.Safe)
		safeB := sys.L2ELB.BlockRefByLabel(eth.Safe)
		errA := glamsterdamtest.CheckSafeHeadTraffic(t.Ctx(), t.Logger(), gasA, safeA.Hash, threshold)
		errB := glamsterdamtest.CheckSafeHeadTraffic(t.Ctx(), t.Logger(), gasB, safeB.Hash, threshold)
		if errA == nil && errB == nil {
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

func spamTxsMultiNode(sys *presets.SingleChainMultiNode) {
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
