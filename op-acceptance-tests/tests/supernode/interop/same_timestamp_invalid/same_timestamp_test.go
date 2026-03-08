package same_timestamp_invalid

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// TestSupernodeSameTimestampExecMessage: Chain B executes Chain A's init at same timestamp - VALID
func TestSupernodeSameTimestampExecMessage(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0).ForSameTimestampTesting(t)
	rng := rand.New(rand.NewSource(99999))

	pairA := sys.PrepareInitA(rng, 0)

	sys.IncludeAndValidate(
		[]*txplan.PlannedTx{pairA.SubmitInit()},
		[]*txplan.PlannedTx{pairA.SubmitExecTo(sys.Bob)},
		false, false, // neither replaced
	)
}

// TestSupernodeSameTimestampInvalidTransitive: Bad log index causes transitive invalidation
func TestSupernodeSameTimestampInvalidTransitive(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0).ForSameTimestampTesting(t)
	rng := rand.New(rand.NewSource(77777))

	pairA := sys.PrepareInitA(rng, 0)
	pairB := sys.PrepareInitB(rng, 0)

	sys.IncludeAndValidate(
		[]*txplan.PlannedTx{pairA.SubmitInit(), pairB.SubmitExecTo(sys.Alice)},
		[]*txplan.PlannedTx{pairB.SubmitInit(), pairA.SubmitInvalidExecTo(sys.Bob)},
		true, true, // both replaced (B invalid, A transitive)
	)
}

// TestSupernodeSameTimestampCycle: Mutual exec messages create cycle - both replaced
func TestSupernodeSameTimestampCycle(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0).ForSameTimestampTesting(t)
	rng := rand.New(rand.NewSource(55555))

	pairA := sys.PrepareInitA(rng, 0)
	pairB := sys.PrepareInitB(rng, 0)

	sys.IncludeAndValidate(
		[]*txplan.PlannedTx{pairA.SubmitInit(), pairB.SubmitExecTo(sys.Alice)},
		[]*txplan.PlannedTx{pairB.SubmitInit(), pairA.SubmitExecTo(sys.Bob)},
		true, true, // both replaced (cycle detected)
	)
}
