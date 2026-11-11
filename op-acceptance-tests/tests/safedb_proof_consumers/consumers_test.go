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

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

func TestProposerPublishesSuperRoot(gt *testing.T) {
	t := devtest.ParallelT(gt)
	require := t.Require()
	system := presets.NewMinimal(t,
		presets.WithDeployerOptions(
			sysgo.WithJovianAtGenesis,
			sysgo.WithUseInterop(),
			sysgo.WithDevFeatureEnabled(devfeatures.SuperRootGamesMigrationFlag),
		),
	)

	// The genesis block's required L1 is fixed at 0 without a SafeDB read, so wait for a game past it.
	game := system.DisputeGameFactory().WaitForSuperGameFrom(system.L2Chain.TimestampForBlockNum(1))
	game.VerifyGameType(gameTypes.SuperPermissionedGameType)
	response := dsl.NewOpNodeSuperRoots(system.L2CL).AwaitSuperRootMatching(game.L2SequenceNumber(), game.RootClaimValue())

	chainOutput, ok := response.OptimisticAtTimestamp[system.L2Chain.ChainID()]
	require.True(ok, "super root response must include the L2 chain")
	block := system.L2EL.BlockRefByHash(chainOutput.Output.BlockHash)
	require.Equal(system.L2CL.OutputAtBlock(block.Number).OutputRoot, chainOutput.OutputRoot)
	requireFirstL1WithSafeHead(t, system.L2CL, block.Number, response.Data.VerifiedRequiredL1)
}

func TestDisputeMonitorAgreesWithSafeOutputRoot(gt *testing.T) {
	t := devtest.ParallelT(gt)
	require := t.Require()
	system := newOutputRootSystem(t)
	proposer := system.FunderL1.NewFundedEOA(eth.OneEther)
	game := system.DisputeGameFactory().StartCannonKonaGame(proposer, proofs.WithL2SequenceNumber(3))

	monitor := system.StartDisputeMon()
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.CannonKonaGameType, 1),
		disputemon.FailedGames(0),
		disputemon.AgreedRoots(1),
		disputemon.DisagreedRoots(0),
	)

	// dispute-mon also agrees when it cannot read the SafeDB, so check the record it relies on.
	l1Head := system.L1EL.BlockRefByHash(game.L1Head())
	safeHead := system.L2CL.SafeHeadRecordAtL1Block(l1Head.Number)
	require.NotNil(safeHead, "SafeDB must have a record at the game's L1 head %d", l1Head.Number)
	require.GreaterOrEqual(safeHead.SafeHead.Number, game.L2SequenceNumber())
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

// requireFirstL1WithSafeHead asserts that requiredL1 is the first L1 block at which the
// SafeDB records l2Block as safe.
func requireFirstL1WithSafeHead(t devtest.T, cl *dsl.L2CLNode, l2Block uint64, requiredL1 eth.BlockID) {
	require := t.Require()
	require.NotZero(requiredL1.Number, "required L1 for L2 block %d", l2Block)

	atRequired := cl.SafeHeadRecordAtL1Block(requiredL1.Number)
	require.NotNil(atRequired, "SafeDB must have a record at the required L1 block %d", requiredL1.Number)
	require.Equal(requiredL1, atRequired.L1Block, "required L1 must be a SafeDB record")
	require.GreaterOrEqual(atRequired.SafeHead.Number, l2Block)

	beforeRequired := cl.SafeHeadRecordAtL1Block(requiredL1.Number - 1)
	if beforeRequired == nil {
		return
	}
	require.Less(beforeRequired.SafeHead.Number, l2Block, "L2 block %d was already safe before the required L1 block", l2Block)
}
