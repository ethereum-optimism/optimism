package upgrade

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
)

func TestPreNoInbox(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	require := t.Require()

	t.Logger().Info("Starting")

	devtest.RunParallel(t, sys.L2Networks(), func(t devtest.T, net *dsl.L2Network) {
		interopTime := net.Escape().ChainConfig().InteropTime
		t.Require().NotNil(interopTime)
		pre := net.LatestBlockBeforeTimestamp(t, *interopTime)
		el := net.Escape().L2ELNode(match.FirstL2EL)
		codeAddr := common.HexToAddress("0xC0D3C0d3C0D3C0d3c0d3c0D3c0D3C0d3C0D30022")
		implCode, err := el.EthClient().CodeAtHash(t.Ctx(), codeAddr, pre.Hash)
		require.NoError(err)
		// TODO this is not empty yet
		require.Len(implCode, 0, "needs to be empty")
		implAddrBytes, err := el.EthClient().GetStorageAt(t.Ctx(), predeploys.CrossL2InboxAddr,
			genesis.ImplementationSlot, pre.Hash.String())
		require.NoError(err)
		// TODO: this still points to 0xC0D3C0d3C0D3C0d3c0d3c0D3c0D3C0d3C0D30022
		require.Equal(common.Address{}, common.BytesToAddress(implAddrBytes[:]))
	})
	t.Logger().Info("Done")
}
