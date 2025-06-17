package proofs

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestProposer(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)

	dgf := sys.DisputeGameFactory()

	gameCount := dgf.GameCount()
	t.Require().Eventually(func() bool {
		newGameCount := dgf.GameCount()
		check := newGameCount > gameCount
		t.Logf("waiting for game count to increase. current=%d new=%d", gameCount, newGameCount)
		return check
	}, time.Minute*10, time.Second*5)

	newGame := dgf.GameAtIndex(gameCount)
	rootClaim := newGame.RootClaim().Value()
	l2SequenceNumber := newGame.L2SequenceNumber()

	superRoot := sys.Supervisor.FetchSuperRootAtTimestamp(l2SequenceNumber.Uint64())
	t.Require().Equal(superRoot.SuperRoot[:], rootClaim[:])
}
