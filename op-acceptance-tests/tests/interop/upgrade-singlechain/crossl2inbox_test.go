package upgrade

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestPostInbox(gt *testing.T) {
	gt.Skip("Skipping Interop Acceptance Test")
	t := devtest.ParallelT(gt)
	offset := uint64(30)
	sys := presets.NewSingleChainInterop(t, presets.WithDeployerOptions(
		func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
			for _, l2Cfg := range builder.L2s() {
				l2Cfg.WithForkAtOffset(forks.Interop, &offset)
			}
		},
	))
	devtest.RunParallel(t, sys.L2Networks(), func(t devtest.T, net *dsl.L2Network) {
		require := t.Require()
		el := net.PrimaryEL()

		activationBlock := net.AwaitActivation(t, forks.Interop)
		require.NotZero(activationBlock, "must not activate interop at genesis")

		pre := activationBlock.Number - 1

		verifyNoCrossL2InboxAtBlock := func(blockNum uint64) {
			net.PublicRPC().WaitForBlockNumber(blockNum)
			implAddrBytes, err := el.EthClient().GetStorageAt(t.Ctx(), predeploys.CrossL2InboxAddr,
				genesis.ImplementationSlot, hexutil.Uint64(blockNum).String())
			require.NoError(err)
			implAddr := common.BytesToAddress(implAddrBytes[:])
			require.Equal(common.Address{}, implAddr, "Should not have CrossL2Inbox implementation")
		}

		verifyNoCrossL2InboxAtBlock(pre)
		verifyNoCrossL2InboxAtBlock(activationBlock.Number)
		verifyNoCrossL2InboxAtBlock(activationBlock.Number + 1)
	})
}
