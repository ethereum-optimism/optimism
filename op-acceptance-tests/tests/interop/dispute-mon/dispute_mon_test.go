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
	game := sys.DisputeGameFactory().StartCannonKonaGame(proposer, proofs.WithRootClaim(common.HexToHash("0xdead")))
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
		proofs.WithSuperRootFrom(eth.Bytes32(common.HexToHash("0xdead"))),
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

func TestDisputeMonitorReportsClaimTiming(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(
		t,
		presets.WithTimeTravelEnabled(),
		presets.WithGameTypeAdded(gameTypes.CannonKonaGameType),
		presets.WithoutHonestProposer(),
	)

	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	game := sys.DisputeGameFactory().StartCannonKonaGame(proposer)
	mon := sys.StartDisputeMon()

	mon.VerifyState(
		disputemon.GameCount(gameTypes.CannonKonaGameType, 1),
		disputemon.FailedGames(0),
		disputemon.UnresolvedClaimsInFirstHalf(1),
		disputemon.UnresolvedClaimsInSecondHalf(0),
		disputemon.ExpiredClaims(0),
		disputemon.UnexpiredClaims(1),
		disputemon.ResolvableClaims(0),
	)

	sys.AdvanceTime(game.MaxClockDuration() + time.Second)
	mon.VerifyState(
		disputemon.UnresolvedClaimsInFirstHalf(0),
		disputemon.UnresolvedClaimsInSecondHalf(1),
		disputemon.ExpiredClaims(1),
		disputemon.UnexpiredClaims(0),
		disputemon.ResolvableClaims(1),
	)
}

func TestDisputeMonitorReportsResolvedGameAccounting(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(
		t,
		presets.WithTimeTravelEnabled(),
		presets.WithGameTypeAdded(gameTypes.CannonKonaGameType),
		presets.WithRespectedGameTypeOverride(gameTypes.CannonKonaGameType),
		presets.WithoutHonestProposer(),
	)

	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	game := sys.DisputeGameFactory().StartCannonKonaGame(proposer)
	sys.AdvanceTime(game.MaxClockDuration() + time.Second)
	game.ResolveClaim(proposer, 0)
	game.Resolve(proposer)
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)

	mon := sys.StartDisputeMon()
	mon.VerifyState(
		disputemon.GameCount(gameTypes.CannonKonaGameType, 1),
		disputemon.FailedGames(0),

		disputemon.CompletedBeforeMaxDuration(1),
		disputemon.ResolvedClaims(1),
		disputemon.ResolvedClaimsInFirstHalf(0),
		disputemon.ResolvedClaimsInSecondHalf(1),
		disputemon.ResolvableClaims(0),

		disputemon.ExactNonWithdrawableCredits(1),
		disputemon.NoWithdrawalRequests(game),
		disputemon.FullyCollateralized(game, game.RootClaim().Bond()),
	)
}

func TestDisputeMonitorReportsHonestActorLoss(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(
		t,
		presets.WithTimeTravelEnabled(),
		presets.WithGameTypeAdded(gameTypes.CannonKonaGameType),
		presets.WithRespectedGameTypeOverride(gameTypes.CannonKonaGameType),
		presets.WithoutHonestProposer(),
	)

	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	counterer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	game := sys.DisputeGameFactory().StartCannonKonaGame(proposer, proofs.WithRootClaim(common.HexToHash("0xdead")))
	rootClaim := game.RootClaim()
	rootBond := new(big.Int).Set(rootClaim.Bond())
	counterClaim := rootClaim.Attack(counterer, common.HexToHash("0xbeef"))

	sys.AdvanceTime(game.MaxClockDuration() + time.Second)
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
