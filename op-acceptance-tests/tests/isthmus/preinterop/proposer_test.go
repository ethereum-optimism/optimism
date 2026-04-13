package preinterop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

func TestProposer(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSimpleInteropPreinterop(t)

	dgf := sys.DisputeGameFactory()

	newGame := dgf.WaitForGame()
	rootClaim := newGame.RootClaim().Value()
	l2SequenceNumber := newGame.L2SequenceNumber()
	sys.SuperRoots.AssertSuperRootAtTimestamp(l2SequenceNumber, rootClaim)
}
