package depreqres

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/depreqres/common"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestUnsafeChainNotStalling_DisabledReqRespSync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t, common.ReqRespSyncDisabledOpts(sync.ELSync)...)
	// We don't want the safe head to move, as this can also progress the unsafe head
	sys.L2Batcher.Stop()
	l := t.Logger()

	l.Info("Confirm that the CL nodes are progressing the unsafe chain")
	delta := uint64(3)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, delta, 30),
		sys.L2CLB.AdvancedFn(types.LocalUnsafe, delta, 30),
	)

	l.Info("Disconnect L2CL from L2CLB, and vice versa")
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLB)

	l.Info("Confirm that the CL nodes are disconnected")
	sys.L2CL.IsP2PDisconnected(sys.L2CLB)
	sys.L2CLB.IsP2PDisconnected(sys.L2CL)

	l.Info("Confirm that the unsafe chain for L2CLB cannot advance")
	numAttempts := 10
	sys.L2CL.AdvancedUnsafe(delta, numAttempts) // this is the sequencer, it builds the unsafe chain
	sys.L2CLB.NotAdvancedUnsafe(numAttempts)

	l.Info("Re-connect L2CL to L2CLB")
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	l.Info("Confirm that the unsafe chain for L2CLB can advance")
	sys.L2CLB.Advanced(types.LocalUnsafe, delta, numAttempts)
	sys.L2ELB.Advanced(eth.Unsafe, delta)
}
