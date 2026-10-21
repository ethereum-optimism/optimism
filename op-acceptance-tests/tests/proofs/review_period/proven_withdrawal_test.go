package review_period

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
)

// gameClaimValidity binds the AnchorStateRegistry check the portal runs after proof maturity.
type gameClaimValidity struct {
	IsGameClaimValid func(game common.Address) bindings.TypedCall[bool] `sol:"isGameClaimValid"`
}

// TestWithdrawalProvenBeforeOPCMUpgradeKeepsItsProofMaturity checks that a withdrawal proven before an
// OPCM upgrade still can't finalize before its original proof maturity, and finalizes after it without
// being proven again.
func TestWithdrawalProvenBeforeOPCMUpgradeKeepsItsProofMaturity(gt *testing.T) {
	t := devtest.ParallelT(gt)
	gameType := gameTypes.PermissionedGameType
	sys := presets.NewMinimal(t,
		presets.WithTimeTravelEnabled(),
		presets.WithDeployerOptions(
			sysgo.WithFinalizationPeriodSeconds(1),
			sysgo.WithProofMaturityDelaySeconds(proofMaturityDelaySeconds),
			// Short game clocks and finality delay, so only the proof maturity blocks finalization.
			sysgo.WithDisputeGameFinalityDelaySeconds(4),
			sysgo.WithFaultGameMaxClockDuration(40),
			sysgo.WithFaultGameClockExtension(1),
			sysgo.WithPreimageOracleChallengePeriod(10),
		),
		presets.WithRespectedGameTypeOverride(gameType),
		presets.WithProposerOption(func(_ sysgo.ComponentTarget, cfg *ps.CLIConfig) {
			cfg.DisputeGameType = uint32(gameType)
		}),
		presets.WithGameTypeAdded(gameType),
	)
	bridge := sys.StandardBridge()
	proofNotOldEnough := hexutil.Encode(dsl.ErrorSelector("OptimismPortal_ProofNotOldEnough()"))

	user := sys.FunderL1.NewFundedEOA(eth.OneThirdEther)
	bridge.Deposit(eth.OneTenthEther, user) // funds the ETHLockbox
	withdrawal := bridge.InitiateWithdrawal(eth.OneHundredthEther, user.AsEL(sys.L2EL))
	sys.DisputeGameFactory().WaitForGame()
	withdrawal.Prove(user)
	sys.AdvanceTime(bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolved()

	// Before the upgrade, the game is valid and only the proof's age blocks the withdrawal.
	systemConfig := sys.L2Chain.Escape().Deployment().SystemConfigProxyAddr()
	delays, err := reviewperiod.ReadChainDelays(t.Ctx(), sys.L1EL.EthClient(), systemConfig, rpc.LatestBlockNumber)
	t.Require().NoError(err, "failed to read the review period delays")
	sys.AdvanceTime(time.Duration(delays.DisputeGameFinalityDelaySeconds+1) * time.Second)
	game := sys.DisputeGameFactory().GameAtIndex(withdrawal.ProvenDisputeGameIndex().Int64())
	asr := bindings.NewBindings[gameClaimValidity](
		bindings.WithClient(sys.L1EL.EthClient()), bindings.WithTo(delays.AnchorStateRegistry), bindings.WithTest(t))
	valid, err := contractio.Read(asr.IsGameClaimValid(game.Address), t.Ctx())
	t.Require().NoError(err, "failed to read isGameClaimValid")
	t.Require().True(valid, "the proven game's claim is not valid")
	t.Require().ErrorContains(errutil.TryAddRevertReason(withdrawal.CheckFinalizable(user)), proofNotOldEnough,
		"the proof should be too young before the upgrade")

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	t.Require().NoError(err, "failed to derive the dev keys")
	sysgo.UpgradeOPChain(t, keys, sys.L1Network.ChainID(), sys.L1EL.Escape().UserRPC(),
		sys.L2Chain.ChainID(), systemConfig, gameType)

	pre, post := readDelaysAcrossLastUpgrade(t, sys.L1EL, systemConfig)
	for _, d := range reviewperiod.CompareDelays(pre, post) {
		t.Require().False(d.Changed(), "%s changed across the OPCM upgrade: %d -> %d", d.Name, d.Before, d.After)
	}
	t.Require().ErrorContains(errutil.TryAddRevertReason(withdrawal.CheckFinalizable(user)), proofNotOldEnough,
		"the proof should still be too young after the upgrade")

	// A day after the proof, the withdrawal is past its original maturity and finalizes without a new proof.
	sys.AdvanceTime(proofMaturityDelaySeconds * time.Second)
	t.Require().NoError(withdrawal.CheckFinalizable(user), "withdrawal is not finalizable after its original proof maturity")
	withdrawal.Finalize(user)
}
