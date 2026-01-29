package proofs

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

	superRoot := eth.SuperRoot(sys.SuperRoots.SuperV1AtTimestamp(l2SequenceNumber))
	t.Require().Equal(superRoot[:], rootClaim[:])
}
