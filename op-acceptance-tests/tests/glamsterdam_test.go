package tests

import (
	"testing"

	"github.com/ethereum/go-ethereum/params/forks"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestSafeHeadsAdvanceAcrossGlamsterdam(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t,
		presets.WithDeployerOptions(
			sysgo.WithForkAtL1Genesis(forks.BPO5),
			sysgo.WithForkAtL1Offset(forks.Amsterdam, 15),
		),
	)

	l1Config := sys.L1Network.Escape().ChainConfig()
	t.Require().NotNil(l1Config.AmsterdamTime)
	l1Genesis := sys.L1EL.BlockRefByNumber(0)
	t.Require().Greater(*l1Config.AmsterdamTime, l1Genesis.Time,
		"Glamsterdam must activate after L1 genesis to exercise the fork transition")

	postForkL1 := sys.L1EL.WaitForTime(*l1Config.AmsterdamTime)
	dsl.CheckAll(t,
		sys.L2EL.L1OriginReachedFn(eth.Safe, postForkL1.Number, 120),
		sys.L2ELB.L1OriginReachedFn(eth.Safe, postForkL1.Number, 120),
	)
}
