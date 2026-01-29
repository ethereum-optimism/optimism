package preinterop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestProposer(gt *testing.T) {
	gt.Skip("TODO(#16166): Re-enable once the supervisor endpoint supports super roots before interop")
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)

	dgf := sys.DisputeGameFactory()

	newGame := dgf.WaitForGame()
	rootClaim := newGame.RootClaim().Value()
	l2SequenceNumber := newGame.L2SequenceNumber()

	superV1 := sys.SuperRoots.SuperV1AtTimestamp(l2SequenceNumber)
	superRoot := eth.SuperRoot(superV1)
	t.Require().Equal(superRoot[:], rootClaim[:])
}
