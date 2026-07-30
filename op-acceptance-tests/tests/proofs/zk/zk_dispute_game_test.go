package zk

import (
	"math"
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

func TestDeploymentUsesSuperAggregationVKey(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSystem(t)
	vkey := loadSuperAggregationVKey(t)
	factory := sys.DisputeGameFactory()

	factory.VerifyGameImplAbsent(gameTypes.SuperCannonKonaGameType)
	zk := factory.ZKGameImpl()
	t.Require().NotEqual(common.Address{}, zk.Address)
	t.Require().Equal(vkey, zk.Args.AbsolutePrestate)
	t.Require().Equal(uint64(zkChallengeDuration/time.Second), zk.Args.MaxChallengeDuration)
	t.Require().Equal(uint64(zkProveDuration/time.Second), zk.Args.MaxProveDuration)
	t.Require().Positive(zk.Args.ChallengerBond.Sign())
	t.Require().NotEqual(common.Address{}, zk.Args.AnchorStateRegistry)
	t.Require().NotEqual(common.Address{}, zk.Args.Weth)
	l1Head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	code, err := sys.L1EL.EthClient().CodeAtHash(t.Ctx(), zk.Args.Verifier, l1Head.Hash)
	t.Require().NoError(err)
	t.Require().NotEmpty(code, "mock verifier must have deployed code")
}

func TestChallengedValidProposalAnchors(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSystem(t)
	factory := sys.DisputeGameFactory()
	challenger, prover := fundedActors(sys)

	// The honest proposer creates the valid root proposal.
	game := factory.WaitForZKGameAtIndex(0)
	t.Require().Equal(uint32(math.MaxUint32), game.ParentIndex())
	t.Require().Equal(proofs.ZKProposalUnchallenged, game.ProposalStatus())

	// A third party grief-challenges the valid proposal; the honest challenger does not challenge it.
	challengedClaim := game.Challenge(challenger)
	t.Require().Equal(challenger.Address(), challengedClaim.Challenger)
	// TODO(#21463): Submit the proof via the kona-sp1-proposer once the defend path lands.
	provedClaim := game.Prove(prover, []byte("mock-sp1-super-aggregation-proof"))
	t.Require().Equal(proofs.ZKProposalChallengedAndValidProofProvided, proofs.ZKProposalStatus(provedClaim.Status))
	t.Require().Equal(prover.Address(), provedClaim.Prover)

	// The proven-valid proposal resolves and anchors. The live proposer keeps
	// chaining and anchoring later games, so assert the anchor reached at
	// least this game's sequence (descendants can only anchor if this game
	// resolved in the defender's favor).
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
	advanceL1To(&sys.SingleChainInterop, game.ResolvedAt()+uint64(zkFinalityDelay/time.Second)+1)
	sys.AnchorStateRegistry(sys.L2ChainA).WaitForAnchorRootAtLeast(game)
}

func fundedActors(sys *presets.SimpleInterop) (*dsl.EOA, *dsl.EOA) {
	actors := sys.FunderL1.NewFundedEOAs(2, eth.OneEther)
	return actors[0], actors[1]
}

func advanceL1To(sys *presets.SingleChainInterop, timestamp uint64) {
	current := sys.L1EL.BlockRefByLabel(eth.Unsafe).Time
	if current < timestamp {
		sys.AdvanceTime(time.Duration(timestamp-current) * time.Second)
	}
	sys.L1EL.WaitForTime(timestamp)
}
