package safedb_proof_consumers

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/disputemon"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

func TestProposerPublishesSuperRoot(gt *testing.T) {
	t := devtest.ParallelT(gt)
	system := presets.NewMinimal(t,
		presets.WithDeployerOptions(
			sysgo.WithJovianAtGenesis,
			sysgo.WithDevFeatureEnabled(devfeatures.OptimismPortalInteropFlag),
			sysgo.WithDevFeatureEnabled(devfeatures.SuperRootGamesMigrationFlag),
		),
	)

	game := system.DisputeGameFactory().WaitForGame()
	game.VerifyGameType(gameTypes.SuperPermissionedGameType)
	dsl.NewOpNodeSuperRoots(system.L2CL).AssertSuperRootAtTimestamp(game.L2SequenceNumber(), game.RootClaimValue())
}

func TestDisputeMonitorClassifiesOutputRoots(gt *testing.T) {
	for _, testCase := range []struct {
		name string

		agreedRoots    int
		disagreedRoots int
		gameOptions    []proofs.GameOpt
	}{
		{name: "valid", agreedRoots: 1},
		{name: "invalid", disagreedRoots: 1, gameOptions: []proofs.GameOpt{proofs.WithRootClaim(common.HexToHash("0xdead"))}},
	} {
		gt.Run(testCase.name, func(gt *testing.T) {
			t := devtest.ParallelT(gt)
			system := newOutputRootSystem(t)
			proposer := system.FunderL1.NewFundedEOA(eth.OneEther)
			gameOptions := append([]proofs.GameOpt{proofs.WithL2SequenceNumber(3)}, testCase.gameOptions...)
			system.DisputeGameFactory().StartCannonKonaGame(proposer, gameOptions...)

			monitor := system.StartDisputeMon()
			monitor.VerifyState(
				disputemon.GameCount(gameTypes.CannonKonaGameType, 1),
				disputemon.FailedGames(0),
				disputemon.AgreedRoots(testCase.agreedRoots),
				disputemon.DisagreedRoots(testCase.disagreedRoots),
			)
		})
	}
}

func TestDisputeMonitorRejectsOutputRootNotSafeAtGameL1Head(gt *testing.T) {
	t := devtest.ParallelT(gt)
	system := newOutputRootSystem(t)
	proposer := system.FunderL1.NewFundedEOA(eth.OneEther)
	system.L2CL.Reached(safety.LocalSafe, 3, 30)
	system.L2Batcher.Stop()
	system.L2CL.Advanced(safety.LocalUnsafe, 3, 30)
	blockNumber := system.L2CL.HeadBlockRef(safety.LocalUnsafe).Number

	game := system.DisputeGameFactory().StartCannonKonaGame(proposer,
		proofs.WithL2SequenceNumber(blockNumber),
		proofs.WithFutureProposal(),
	)
	system.L2Batcher.Start()
	system.L2CL.Reached(safety.LocalSafe, blockNumber, 30)

	monitor := system.StartDisputeMon()
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.CannonKonaGameType, 1),
		disputemon.FailedGames(0),
		disputemon.AgreedRoots(0),
		disputemon.DisagreedRoots(1),
		disputemon.InvalidProposalObserved(game),
	)
}

func newOutputRootSystem(t devtest.T) *presets.Minimal {
	return presets.NewMinimalNoFaultProofs(t,
		presets.WithDeployerOptions(sysgo.WithJovianAtGenesis),
		presets.WithGameTypeAdded(gameTypes.CannonKonaGameType),
	)
}
