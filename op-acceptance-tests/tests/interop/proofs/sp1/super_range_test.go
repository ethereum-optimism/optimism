package sp1

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const konaSP1SuperRangeExecutorPathEnv = "KONA_SP1_SUPER_RANGE_EXECUTOR_PATH"

func TestSP1SuperRangeSimpleInteropSmoke(gt *testing.T) {
	t := devtest.SerialT(gt)
	executorPath := os.Getenv(konaSP1SuperRangeExecutorPathEnv)
	if executorPath == "" {
		t.Skip("kona-sp1 super-range executor not provided; set " + konaSP1SuperRangeExecutorPathEnv)
	}

	sys := presets.NewSimpleInterop(t)
	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	aliceB := aliceA.AsEL(sys.L2ELB)
	sys.FunderB.Fund(aliceB, eth.OneEther)

	eventLogger := aliceA.DeployEventLogger()
	initMsg := aliceA.SendRandomInitMessage(rand.New(rand.NewSource(1234)), eventLogger, 2, 10)
	execMsg := aliceB.SendExecMessage(initMsg)

	endTimestamp := sys.L2ChainB.Escape().RollupConfig().TimestampForBlock(
		bigs.Uint64Strict(execMsg.BlockNumber()),
	)
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)

	resp := sys.SuperRoots.SuperRootAtTimestamp(endTimestamp)
	t.Require().NotNil(resp.Data, "expected validated super-root data at timestamp %d", endTimestamp)
	t.Require().Len(resp.ChainIDs, 2, "expected two chains in the simple interop dependency set")
	t.Require().Contains(resp.ChainIDs, sys.L2ChainA.ChainID())
	t.Require().Contains(resp.ChainIDs, sys.L2ChainB.ChainID())
	t.Require().Contains(resp.OptimisticAtTimestamp, sys.L2ChainA.ChainID())
	t.Require().Contains(resp.OptimisticAtTimestamp, sys.L2ChainB.ChainID())

	l1Head := latestRequiredL1(resp)
	cfg := sys.L2ChainA.Escape().L2Challengers()[0].Config().CannonKona
	args := superRangeExecutorArgs(sys, cfg, l1Head, endTimestamp)

	t.Require().True(
		rustbin.RunKonaSP1SuperRange(t, t.Logger(), executorPath, t.TempDir(), args...),
		"expected kona-sp1 super-range executor to accept timestamp %d",
		endTimestamp,
	)
}

func superRangeExecutorArgs(
	sys *presets.SimpleInterop,
	cfg vm.Config,
	l1Head eth.BlockID,
	endTimestamp uint64,
) []string {
	args := []string{
		"--supernode-address", sys.SuperRoots.UserRPC(),
		"--l1-node-address", sys.L1EL.Escape().UserRPC(),
		"--l1-beacon-address", sys.L1CL.BeaconHTTPAddr(),
		"--l2-node-addresses", strings.Join([]string{
			sys.L2ELA.Escape().UserRPC(),
			sys.L2ELB.Escape().UserRPC(),
		}, ","),
		"--l1-head", l1Head.Hash.Hex(),
		"--end-timestamp", strconv.FormatUint(endTimestamp, 10),
	}
	if len(cfg.RollupConfigPaths) > 0 {
		args = append(args, "--rollup-config-paths", strings.Join(cfg.RollupConfigPaths, ","))
	}
	if cfg.L1GenesisPath != "" {
		args = append(args, "--l1-config-path", cfg.L1GenesisPath)
	}
	if cfg.DepsetConfigPath != "" {
		args = append(args, "--depset-cfg", cfg.DepsetConfigPath)
	}
	return args
}

func latestRequiredL1(resp eth.SuperRootAtTimestampResponse) eth.BlockID {
	var latest eth.BlockID
	for _, out := range resp.OptimisticAtTimestamp {
		if out.RequiredL1.Number > latest.Number {
			latest = out.RequiredL1
		}
	}
	return latest
}
