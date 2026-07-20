package disputemon

import (
	"math/big"
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/disputemon"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

func TestDisputeMonitorReportsHealthySuperPermissionedGame(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainInterop(t, presets.WithoutHonestProposer())

	game := sys.DisputeGameFactory().StartSuperPermissionedGame()
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)

	mon := sys.StartDisputeMon()
	mon.VerifyDisputeMonHealthy()
	mon.VerifyState(
		disputemon.GameCount(gameTypes.SuperPermissionedGameType, 1),
		disputemon.FailedGames(0),
		disputemon.AgreedRoots(1),
		disputemon.DisagreedRoots(0),
	)
}

func TestDisputeMonitorForecastsInvalidProposalForChallenger(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(t, presets.WithGameTypeAdded(gameTypes.CannonKonaGameType))

	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	game := sys.DisputeGameFactory().StartCannonKonaGame(proposer, proofs.WithRootClaim(invalidRootClaim()))
	game.WaitForGameStatus(gameTypes.GameStatusInProgress)

	mon := presets.StartDisputeMon(
		t,
		sys.L1EL,
		sys.L2Chain.DisputeGameFactoryProxyAddr(),
		presets.WithDisputeMonRollupNodes(sys.L2CL),
	)
	mon.VerifyState(
		disputemon.GameCount(gameTypes.CannonKonaGameType, 1),
		disputemon.FailedGames(0),
		disputemon.IncorrectDefenderAhead(1),
		disputemon.InvalidProposalObserved(game),
	)
}

func TestDisputeMonitorReportsIncorrectResolvedGame(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainInterop(t, presets.WithoutHonestProposer())

	game := sys.DisputeGameFactory().StartSuperPermissionedGame(
		proofs.WithSuperRootFrom(eth.Bytes32(invalidRootClaim())),
	)
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)

	mon := sys.StartDisputeMon()
	mon.VerifyState(
		disputemon.GameCount(gameTypes.SuperPermissionedGameType, 1),
		disputemon.FailedGames(0),
		disputemon.IncorrectDefenderWins(1),
		disputemon.InvalidProposalObserved(game),
	)
}

func TestDisputeMonitorReportsReferenceNodeFailure(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(t, presets.WithGameTypeAdded(gameTypes.CannonKonaGameType))

	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	sys.DisputeGameFactory().StartCannonKonaGame(proposer)

	mon := presets.StartDisputeMon(
		t,
		sys.L1EL,
		sys.L2Chain.DisputeGameFactoryProxyAddr(),
		presets.WithDisputeMonRollupNodes(sys.L2CL),
	)
	baseline := mon.VerifyHealthyOutputFetch(gameTypes.CannonKonaGameType)

	sys.L2CL.Stop()
	mon.VerifyReferenceNodeFailure(baseline)
}

func TestDisputeMonitorReportsResolvedGameAccounting(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newCannonKonaDisputeSystem(t)

	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	game := sys.DisputeGameFactory().StartCannonKonaGame(proposer)
	advanceL1Time(sys, game.MaxClockDuration()+time.Second)
	game.ResolveClaim(proposer, 0)
	game.Resolve(proposer)
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)

	startCannonKonaDisputeMon(t, sys).VerifyState(
		disputemon.GameCount(gameTypes.CannonKonaGameType, 1),
		disputemon.FailedGames(0),
		disputemon.CompletedBeforeMaxDuration(1),
		disputemon.ResolvedClaimsInFirstHalf(1),
		disputemon.ResolvableClaims(0),
		disputemon.ExpectedNonWithdrawableCredits(1),
		disputemon.ExcessCredits(0),
		disputemon.DeficientNonWithdrawableCredits(0),
		disputemon.MatchingWithdrawalRequests(game, 0),
		disputemon.DivergentWithdrawalRequests(game, 0),
		disputemon.SufficientCollateral(game, game.RootClaim().Bond()),
		disputemon.NoInsufficientCollateral(game),
	)
}

func TestDisputeMonitorReportsHonestActorLoss(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newCannonKonaDisputeSystem(t)

	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	counterer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	game := sys.DisputeGameFactory().StartCannonKonaGame(proposer, proofs.WithRootClaim(invalidRootClaim()))
	rootClaim := game.RootClaim()
	rootBond := new(big.Int).Set(rootClaim.Bond())
	counterClaim := rootClaim.Attack(counterer, common.HexToHash("0xbeef"))

	advanceL1Time(sys, game.MaxClockDuration()+time.Second)
	game.ResolveClaim(counterer, counterClaim.Index)
	game.ResolveClaim(counterer, rootClaim.Index)
	game.Resolve(counterer)
	game.WaitForGameStatus(gameTypes.GameStatusChallengerWon)

	mon := presets.StartDisputeMon(
		t,
		sys.L1EL,
		sys.L2Chain.DisputeGameFactoryProxyAddr(),
		presets.WithDisputeMonRollupNodes(sys.L2CL),
		presets.WithDisputeMonHonestActors(proposer.Address()),
	)
	mon.VerifyState(
		disputemon.GameCount(gameTypes.CannonKonaGameType, 1),
		disputemon.HonestActorInvalidClaims(proposer.Address(), 1),
		disputemon.HonestActorLostBonds(proposer.Address(), rootBond),
		disputemon.CorrectChallengerWins(1),
	)
}

func newCannonKonaDisputeSystem(t devtest.T) *presets.Minimal {
	return presets.NewMinimalNoFaultProofs(
		t,
		presets.WithTimeTravelEnabled(),
		presets.WithGameTypeAdded(gameTypes.CannonKonaGameType),
		presets.WithRespectedGameTypeOverride(gameTypes.CannonKonaGameType),
		presets.WithoutHonestProposer(),
	)
}

func advanceL1Time(sys *presets.Minimal, duration time.Duration) {
	target := sys.L1EL.BlockRefByLabel(eth.Unsafe).Time + uint64(duration/time.Second)
	sys.AdvanceTime(duration)
	sys.L1EL.WaitForTime(target)
}

func startCannonKonaDisputeMon(t devtest.T, sys *presets.Minimal) *disputemon.DisputeMon {
	return presets.StartDisputeMon(
		t,
		sys.L1EL,
		sys.L2Chain.DisputeGameFactoryProxyAddr(),
		presets.WithDisputeMonRollupNodes(sys.L2CL),
	)
}

func invalidRootClaim() common.Hash {
	return common.HexToHash("0xdead")
}
