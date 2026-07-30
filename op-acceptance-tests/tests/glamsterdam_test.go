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
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

func TestSafeHeadsAdvanceAcrossGlamsterdam(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t,
		glamsterdamL1Geth(t),
		presets.WithDeployerOptions(
			sysgo.WithForkAtL1Genesis(forks.BPO5),
			sysgo.WithForkAtL1Offset(forks.Amsterdam, 30),
		),
	)
	spamGlamsterdamTxs(sys)

	l1Config := sys.L1Network.Escape().ChainConfig()
	t.Require().NotNil(l1Config.AmsterdamTime)
	l1Genesis := sys.L1EL.BlockRefByNumber(0)
	t.Require().Greater(*l1Config.AmsterdamTime, l1Genesis.Time,
		"Glamsterdam must activate after L1 genesis to exercise the fork transition")

	postForkL1 := sys.L1EL.WaitForTime(*l1Config.AmsterdamTime)
	postForkHeader, err := sys.L1EL.EthClient().HeaderByHash(t.Ctx(), postForkL1.Hash)
	t.Require().NoError(err)
	t.Require().NotNil(postForkHeader.SlotNumber, "post-Glamsterdam L1 block must include a slot number")
	t.Require().NotNil(postForkHeader.BlockAccessListHash,
		"post-Glamsterdam L1 block must include a block access list hash")

	dsl.CheckAll(t,
		sys.L2EL.L1OriginReachedFn(eth.Safe, postForkL1.Number, 120),
		sys.L2ELB.L1OriginReachedFn(eth.Safe, postForkL1.Number, 120),
	)

	threshold := sys.L2Chain.Escape().RollupConfig().Genesis.SystemConfig.GasLimit/2 + 1
	for name, el := range map[string]*dsl.L2ELNode{
		"sequencer": sys.L2EL,
		"verifier":  sys.L2ELB,
	} {
		el.WaitForLabel(eth.Safe, func(info eth.BlockInfo) (bool, error) {
			if info.GasUsed() >= threshold {
				t.Logger().Info("Found half-full post-Glamsterdam safe block",
					"node", name, "block", eth.ToBlockID(info), "gasUsed", info.GasUsed(), "threshold", threshold)
				return true, nil
			}
			return false, nil
		})
	}
}

func glamsterdamL1Geth(t devtest.T) presets.Option {
	out, err := exec.CommandContext(t.Ctx(), "mise", "which", "geth").Output()
	t.Require().NoError(err, "resolve mise-installed Glamsterdam geth")
	return presets.WithL1Geth(strings.TrimSpace(string(out)))
}

func spamGlamsterdamTxs(sys *presets.SingleChainMultiNode) {
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
