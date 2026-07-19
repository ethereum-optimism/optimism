package disputemon

import (
	"math/big"
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
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

	sys.StartDisputeMon().VerifyHealthySuperPermissionedGame()
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
	mon.VerifyInvalidProposalForecast(gameTypes.CannonKonaGameType)
}

func TestDisputeMonitorReportsIncorrectResolvedGame(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainInterop(t, presets.WithoutHonestProposer())

	game := sys.DisputeGameFactory().StartSuperPermissionedGame(
		proofs.WithSuperRootFrom(eth.Bytes32(invalidRootClaim())),
	)
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)

	sys.StartDisputeMon().VerifyIncorrectResolvedGame()
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
	mon := startCannonKonaDisputeMon(t, sys)

	sys.AdvanceTime(game.MaxClockDuration() + time.Second)
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
	mon.VerifyResolvedGameAccounting(game)
}

func TestDisputeMonitorReportsHonestActorLoss(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newCannonKonaDisputeSystem(t)

	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	game := sys.DisputeGameFactory().StartCannonKonaGame(proposer, proofs.WithRootClaim(invalidRootClaim()))
	rootClaim := game.RootClaim()
	rootBond := new(big.Int).Set(rootClaim.Bond())
	mon := presets.StartDisputeMon(
		t,
		sys.L1EL,
		sys.L2Chain.DisputeGameFactoryProxyAddr(),
		presets.WithDisputeMonRollupNodes(sys.L2CL),
		presets.WithDisputeMonHonestActors(proposer.Address()),
	)

	rootClaim.WaitForCounterClaim()
	sys.AdvanceTime(game.MaxClockDuration() + time.Second)
	game.WaitForGameStatus(gameTypes.GameStatusChallengerWon)
	mon.VerifyHonestActorLoss(proposer.Address(), rootBond)
}

func newCannonKonaDisputeSystem(t devtest.T) *presets.Minimal {
	return presets.NewMinimal(
		t,
		presets.WithTimeTravelEnabled(),
		presets.WithGameTypeAdded(gameTypes.CannonKonaGameType),
		presets.WithRespectedGameTypeOverride(gameTypes.CannonKonaGameType),
		presets.WithoutHonestProposer(),
	)
}

func startCannonKonaDisputeMon(t devtest.T, sys *presets.Minimal) *presets.DisputeMon {
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
