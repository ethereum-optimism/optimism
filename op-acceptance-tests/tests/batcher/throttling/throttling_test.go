package throttling

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batcherConfig "github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

const blockSizeLimit = 5_000

// TestDABlockThrottling verifies that the execution client respects the block size limit set via
// miner_setMaxDASize. It spams transactions to saturate block space and asserts that blocks are
// filled to near capacity without exceeding the limit.
func TestDABlockThrottling(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t, presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
		// Enable throttling with step controller for predictable behavior.
		cfg.ThrottleConfig.LowerThreshold = 99 // > 0 enables the throttling loop.
		cfg.ThrottleConfig.UpperThreshold = 100
		cfg.ThrottleConfig.ControllerType = batcherConfig.StepControllerType

		cfg.ThrottleConfig.BlockSizeLowerLimit = blockSizeLimit - 1
		cfg.ThrottleConfig.BlockSizeUpperLimit = blockSizeLimit

		cfg.PollInterval = 500 * time.Millisecond // Fast poll for quicker test feedback.
	}))

	spamTxs(sys)

	const minFullSize = blockSizeLimit * 95 / 100

	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	var consecutiveFull uint64
	for {
		select {
		case <-time.Tick(l2BlockTime):
			_, txs, err := sys.L2EL.Escape().EthClient().InfoAndTxsByLabel(t.Ctx(), eth.Unsafe)
			t.Require().NoError(err)

			var calldataSize uint64
			for _, tx := range txs {
				if tx.IsDepositTx() {
					continue
				}
				calldataSize += bigs.Uint64Strict(tx.RollupCostData().EstimatedDASize())
			}
			t.Require().LessOrEqual(calldataSize, uint64(blockSizeLimit))

			if calldataSize >= minFullSize {
				consecutiveFull++
			} else {
				consecutiveFull = 0
			}

			if consecutiveFull == 3 {
				return
			}
		case <-t.Ctx().Done():
			t.Require().Fail("Never saw three consecutive blocks near the max size")
		}
	}
}

func spamTxs(sys *presets.Minimal) {
	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	eoas := loadtest.FundEOAs(sys.T, eth.HundredEther, 50, l2BlockTime, sys.L2EL, sys.Wallet, sys.FaucetL2)
	eoasRR := loadtest.NewRoundRobin(eoas)
	spammer := loadtest.SpammerFunc(func(t devtest.T) error {
		_, err := eoasRR.Get().Include(t, txplan.WithTo(&predeploys.L1BlockAddr), txplan.WithData(make([]byte, 0)), txplan.WithGasLimit(70_000))
		return err
	})
	schedule := loadtest.NewBurst(l2BlockTime, loadtest.WithBaseRPS(50))

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
