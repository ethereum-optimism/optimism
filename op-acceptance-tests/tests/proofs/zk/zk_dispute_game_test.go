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
	challenger, _ := fundedActors(sys)

	// The honest proposer creates the valid root proposal.
	game := factory.WaitForZKGameAtIndex(0)
	t.Require().Equal(uint32(math.MaxUint32), game.ParentIndex())
	t.Require().Equal(proofs.ZKProposalUnchallenged, game.ProposalStatus())

	// A third party grief-challenges the valid proposal; the honest challenger does not challenge it.
	challengedClaim := game.Challenge(challenger)
	t.Require().Equal(challenger.Address(), challengedClaim.Challenger)

	// The kona-sp1-proposer detects the challenge and defends its own game;
	// the proof commits to the submitting signer.
	game.WaitForProposalStatus(proofs.ZKProposalChallengedAndValidProofProvided)
	t.Require().Equal(zkProposerAddress(t, sys), game.ClaimData().Prover)

	// The proven-valid proposal resolves and anchors. The live proposer keeps
	// chaining and anchoring later games, so assert the anchor reached at
	// least this game's sequence (descendants can only anchor if this game
	// resolved in the defender's favor).
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
	advanceL1To(&sys.SingleChainInterop, game.ResolvedAt()+uint64(zkFinalityDelay/time.Second)+1)
	sys.AnchorStateRegistry(sys.L2ChainA).WaitForAnchorRootAtLeast(game)
}

// TestProposerDefendsForeignValidGame pins the prestate-based defense set end
// to end: the proposer proves, resolves, and claims the prover credit on a
// challenged valid game it did NOT create.
func TestProposerDefendsForeignValidGame(gt *testing.T) {
	t := devtest.SerialT(gt)
	// The honest challenger resolves games and claims credit on the
	// proposer's behalf; disable it so every assertion below binds to the
	// proposer.
	sys := newSystem(t, presets.WithoutHonestChallenger())
	factory := sys.DisputeGameFactory()
	proposerAddr := zkProposerAddress(t, sys)
	weth := factory.DelayedWETH(factory.ZKGameImpl().Args.Weth)
	creator, challenger := fundedActors(sys)

	// A foreign EOA creates a valid game (no super-root override: the DSL
	// derives the claim from the supernode), and a second EOA challenges it.
	// The timestamp is placed OFF the honest proposer's fixed proposal grid
	// (one second past its first game): an on-grid anchor-rooted game would
	// collide with the proposer's own creation on the factory UUID
	// (identical claim and extraData) and revert with GameAlreadyExists.
	proposerGame := factory.WaitForZKGameAtIndex(0)
	foreignTimestamp := proposerGame.L2SequenceNumber() + 1
	factory.WaitForSafeSuperRootAfter(foreignTimestamp)
	game := factory.StartZKGame(creator, proofs.WithL2SequenceNumber(foreignTimestamp))
	challengerBond := game.ChallengerBond()
	challengedClaim := game.Challenge(challenger)
	t.Require().Equal(challenger.Address(), challengedClaim.Challenger)

	// The proposer's defense set is prestate-based, not creator-based: it
	// proves the foreign game and its resolution task resolves it; the test
	// never calls prove or resolve itself.
	game.WaitForProposalStatus(proofs.ZKProposalChallengedAndValidProofProvided)
	t.Require().Equal(proposerAddr, game.ClaimData().Prover)
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
	advanceL1To(&sys.SingleChainInterop, game.ResolvedAt()+uint64(zkFinalityDelay/time.Second)+1)

	// The proposer's claim task unlocks its prover credit - the challenger's
	// bond - into DelayedWETH. Asserting the unlocked amount is race-free:
	// the payout cannot run until L1 time advances past the WETH delay.
	var withdrawal proofs.ZKWithdrawal
	t.Require().Eventuallyf(func() bool {
		withdrawal = weth.Withdrawal(game.Address, proposerAddr)
		return withdrawal.Amount.Sign() > 0
	}, 2*time.Minute, time.Second, "proposer did not unlock its prover credit")
	t.Require().Equal(challengerBond, eth.WeiBig(withdrawal.Amount),
		"prover credit must equal the challenger bond")

	advanceL1To(&sys.SingleChainInterop, withdrawal.MaturesAt(weth.Delay()))
	game.WaitForClaimedCredit(proposerAddr)
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
