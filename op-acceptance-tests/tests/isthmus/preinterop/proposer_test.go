package preinterop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestProposer(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)

	dgf := sys.DisputeGameFactory()

	newGame := dgf.WaitForGame()
	rootClaim := newGame.RootClaim().Value()
	l2SequenceNumber := newGame.L2SequenceNumber()

	response := sys.SuperRoots.SuperRootAtTimestamp(l2SequenceNumber)
	t.Require().NotNilf(response.Data, "super root does not exist at time %d", l2SequenceNumber)
	superRoot := eth.SuperRoot(response.Data.Super)
	t.Require().Equal(superRoot[:], rootClaim[:])
}
