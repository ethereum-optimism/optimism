package upgrade

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestPostInbox(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainInterop(t)
	t.Gate().Len(sys.L2Networks(), 1, "only applies with a single network")
	devtest.RunParallel(t, sys.L2Networks(), func(t devtest.T, net *dsl.L2Network) {
		require := t.Require()
		el := net.Escape().L2ELNode(match.FirstL2EL)

		activationBlock := net.AwaitActivation(t, rollup.Interop)
		require.NotZero(activationBlock, "must not activate interop at genesis")

		pre := activationBlock.Number - 1

		// Should not have CrossL2Inbox implementation before activation
		implAddrBytes, err := el.EthClient().GetStorageAt(t.Ctx(), predeploys.CrossL2InboxAddr,
			genesis.ImplementationSlot, hexutil.Uint64(pre).String())
		require.NoError(err)
		implAddr := common.BytesToAddress(implAddrBytes[:])
		require.Equal(common.Address{}, implAddr, "Should not have CrossL2Inbox implementation")

		// Should not have CrossL2Inbox implementation after activation
		implAddrBytes, err = el.EthClient().GetStorageAt(t.Ctx(), predeploys.CrossL2InboxAddr,
			genesis.ImplementationSlot, activationBlock.Hash.String())
		require.NoError(err)
		implAddr = common.BytesToAddress(implAddrBytes[:])
		require.Equal(common.Address{}, implAddr, "Should not have CrossL2Inbox implementation")
	})
}
