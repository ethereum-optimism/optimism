package zk

import (
	"math"
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// The live proposer keeps creating games while these tests run (including
// during time travel), so assertions only reference the specific games
// returned by WaitForZKGameCount and never assume a total game count.

func TestProposerCreatesRootZKGame(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys, _ := newProposerSystem(t)
	factory := sys.DisputeGameFactory()
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()

	game0 := factory.WaitForZKGameCount(1)

	t.Require().Equal(uint32(math.MaxUint32), game0.ParentIndex(),
		"first proposer game must be a root game using the max-uint32 parent sentinel")
	t.Require().Greater(game0.L2SequenceNumber(), anchorSequence,
		"root game must propose a sequence number beyond the anchor")
}

func TestProposerChainsSecondZKGameOnFirst(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys, _ := newProposerSystem(t)
	factory := sys.DisputeGameFactory()

	game0 := factory.WaitForZKGameCount(1)
	game1 := factory.WaitForZKGameCount(2)

	t.Require().Equal(uint32(0), game1.ParentIndex(),
		"second proposer game must chain on the first game at factory index 0")
	t.Require().Greater(game1.L2SequenceNumber(), game0.L2SequenceNumber(),
		"second game must propose a sequence number beyond its parent")
}

func TestProposerCreatedGameResolvesDefenderWon(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys, _ := newProposerSystem(t)
	factory := sys.DisputeGameFactory()
	resolver := sys.FunderL1.NewFundedEOA(eth.OneEther)

	game0 := factory.WaitForZKGameCount(1)
	advanceL1To(sys, game0.ClaimData().Deadline+1)

	// TODO(#21415): Let op-challenger resolve and close the unchallenged proposal.
	t.Require().Equal(gameTypes.GameStatusDefenderWon, game0.Resolve(resolver),
		"unchallenged proposer game must resolve in favour of the defender")
	advanceL1To(sys, game0.ResolvedAt()+uint64(zkFinalityDelay/time.Second)+1)
	game0.Close(resolver)
	sys.AnchorStateRegistry(sys.L2ChainA).WaitForAnchorRoot(game0)
}
