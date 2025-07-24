package upgrade

import (
	"testing"

	"github.com/HashKeyChain/verse/op-chain-ops/genesis"
	"github.com/HashKeyChain/verse/op-devstack/devtest"
	"github.com/HashKeyChain/verse/op-devstack/dsl"
	"github.com/HashKeyChain/verse/op-devstack/presets"
	"github.com/HashKeyChain/verse/op-devstack/stack/match"
	"github.com/HashKeyChain/verse/op-node/rollup"
	"github.com/HashKeyChain/verse/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestPostInbox(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainInterop(t)
	devtest.RunParallel(t, sys.L2Networks(), func(t devtest.T, net *dsl.L2Network) {
		require := t.Require()
		el := net.Escape().L2ELNode(match.FirstL2EL)

		activationBlock := net.AwaitActivation(t, rollup.Interop)
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
