package review_period

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
)

// requireDelaysUnchangedAcrossLastUpgrade reads a chain's delays at the block before its proxies last
// emitted Upgraded and at the latest block, the same way check-review-period does on a live network, and
// requires them to be equal. It returns the latest delays and the block of the last upgrade.
func requireDelaysUnchangedAcrossLastUpgrade(t devtest.T, l1 *dsl.L1ELNode, systemConfig common.Address, upgrade string) (reviewperiod.ChainDelays, uint64) {
	client, err := ethclient.DialContext(t.Ctx(), l1.Escape().UserRPC())
	t.Require().NoError(err, "failed to dial the L1 RPC")
	t.Cleanup(client.Close)

	head, err := client.BlockNumber(t.Ctx())
	t.Require().NoError(err, "failed to get the L1 block number")
	post, err := reviewperiod.ReadChainDelays(t.Ctx(), l1.EthClient(), systemConfig, rpc.BlockNumber(head))
	t.Require().NoError(err, "failed to read the delays at block %d", head)

	upgrades, err := reviewperiod.UpgradedLogs(t.Ctx(), client, post, 0, head)
	t.Require().NoError(err, "failed to get the Upgraded logs")
	t.Require().NotEmpty(upgrades, "no proxy of chain %d was ever upgraded", post.L2ChainID)
	last := upgrades[len(upgrades)-1]
	t.Logf("Comparing chain %d's delays at block %d, before its last upgrade in tx %s, and at block %d",
		post.L2ChainID, last.BlockNumber-1, last.TxHash, head)

	pre, err := reviewperiod.ReadChainDelays(t.Ctx(), l1.EthClient(), systemConfig, rpc.BlockNumber(last.BlockNumber-1))
	t.Require().NoError(err, "failed to read the delays at block %d, before the last upgrade", last.BlockNumber-1)
	for _, d := range reviewperiod.CompareDelays(pre, post) {
		t.Require().False(d.Changed(), "%s of chain %d changed across %s: %d -> %d", d.Name, post.L2ChainID, upgrade, d.Before, d.After)
	}
	return post, last.BlockNumber
}

// newPermissionedMinimal returns a minimal chain whose proposer creates permissioned games, with short game
// clocks and finality delay so a game resolves and finalizes within seconds.
func newPermissionedMinimal(t devtest.T, deployerOpts ...sysgo.DeployerOption) *presets.Minimal {
	gameType := gameTypes.PermissionedGameType
	// The game requires max(2*clockExtension, clockExtension+preimageOracleChallengePeriod) <= maxClockDuration.
	opts := append([]sysgo.DeployerOption{
		sysgo.WithDisputeGameFinalityDelaySeconds(4),
		sysgo.WithFaultGameMaxClockDuration(40),
		sysgo.WithFaultGameClockExtension(1),
		sysgo.WithPreimageOracleChallengePeriod(10),
	}, deployerOpts...)
	return presets.NewMinimal(t,
		presets.WithTimeTravelEnabled(),
		presets.WithDeployerOptions(opts...),
		presets.WithRespectedGameTypeOverride(gameType),
		presets.WithProposerOption(func(_ sysgo.ComponentTarget, cfg *ps.CLIConfig) {
			cfg.DisputeGameType = uint32(gameType)
		}),
		presets.WithGameTypeAdded(gameType),
	)
}

// upgradeMidTest runs an OPCM upgrade on the chain and requires it to leave the delays and the respected
// game type unchanged.
func upgradeMidTest(t devtest.T, sys *presets.Minimal) {
	before := sys.L1EL.BlockRefByLabel(eth.Unsafe).Number
	sysgo.RerunOPCMUpgrade(t, sys.L1EL.Escape(), sys.L2Chain.Escape(), gameTypes.PermissionedGameType)
	_, upgradeBlock := requireDelaysUnchangedAcrossLastUpgrade(t, sys.L1EL, sys.L2Chain.Escape().Deployment().SystemConfigProxyAddr(), "the OPCM upgrade")
	t.Require().Greater(upgradeBlock, before, "the last upgrade is not the one this test ran")
	sys.StandardBridge().VerifyRespectedGameType(gameTypes.PermissionedGameType)
}

// delaySetters binds the review period delay setters.
type delaySetters struct {
	SetProofMaturityDelaySeconds       func(seconds *big.Int) bindings.TypedCall[any] `sol:"setProofMaturityDelaySeconds"`
	SetDisputeGameFinalityDelaySeconds func(seconds *big.Int) bindings.TypedCall[any] `sol:"setDisputeGameFinalityDelaySeconds"`
}

// delayOwner is the L1 ProxyAdmin owner, the only account allowed to change the delays.
type delayOwner struct {
	t  devtest.T
	l1 *dsl.L1ELNode
	*dsl.EOA
}

// newDelayOwner returns the L1 ProxyAdmin owner, funded to send the setter calls.
func newDelayOwner(t devtest.T, l1 *dsl.L1ELNode, l1ChainID eth.ChainID, l2 *dsl.L2Network, funder *dsl.FunderEOA) *delayOwner {
	priv := l2.Escape().Keys().Secret(devkeys.ChainOperatorKeys(l1ChainID.ToBig())(devkeys.L1ProxyAdminOwnerRole))
	owner := dsl.NewKey(t, priv).User(l1)
	funder.FundAtLeast(owner, eth.OneEther)
	return &delayOwner{t: t, l1: l1, EOA: owner}
}

func (o *delayOwner) setters(addr common.Address) delaySetters {
	return bindings.NewBindings[delaySetters](bindings.WithClient(o.l1.EthClient()), bindings.WithTo(addr), bindings.WithTest(o.t))
}

// SetFinalityDelay sets the dispute game finality delay on the chain's AnchorStateRegistry.
func (o *delayOwner) SetFinalityDelay(d reviewperiod.ChainDelays, seconds uint64) {
	_, err := contractio.Write(o.setters(d.AnchorStateRegistry).SetDisputeGameFinalityDelaySeconds(new(big.Int).SetUint64(seconds)), o.t.Ctx(), o.Plan())
	o.t.Require().NoError(err, "failed to set the dispute game finality delay to %d", seconds)
	o.t.Logf("Set the dispute game finality delay of chain %d to %d", d.L2ChainID, seconds)
}

// SetProofMaturityDelay sets the proof maturity delay on the chain's OptimismPortal2.
func (o *delayOwner) SetProofMaturityDelay(d reviewperiod.ChainDelays, seconds uint64) {
	_, err := contractio.Write(o.setters(d.OptimismPortal).SetProofMaturityDelaySeconds(new(big.Int).SetUint64(seconds)), o.t.Ctx(), o.Plan())
	o.t.Require().NoError(err, "failed to set the proof maturity delay to %d", seconds)
	o.t.Logf("Set the proof maturity delay of chain %d to %d", d.L2ChainID, seconds)
}

// guardianActions binds the guardian's AnchorStateRegistry actions.
type guardianActions struct {
	BlacklistDisputeGame func(game common.Address) bindings.TypedCall[any] `sol:"blacklistDisputeGame"`
}

// guardian is the Superchain guardian, which can blacklist dispute games.
type guardian struct {
	t  devtest.T
	l1 *dsl.L1ELNode
	*dsl.EOA
}

// newGuardian returns the Superchain guardian, funded to send transactions.
func newGuardian(t devtest.T, l1 *dsl.L1ELNode, l1ChainID eth.ChainID, l2 *dsl.L2Network, funder *dsl.FunderEOA) *guardian {
	priv := l2.Escape().Keys().Secret(devkeys.SuperchainOperatorKeys(l1ChainID.ToBig())(devkeys.SuperchainConfigGuardianKey))
	g := dsl.NewKey(t, priv).User(l1)
	funder.FundAtLeast(g, eth.OneEther)
	return &guardian{t: t, l1: l1, EOA: g}
}

// BlacklistGame blacklists a dispute game on the chain's AnchorStateRegistry.
func (g *guardian) BlacklistGame(d reviewperiod.ChainDelays, game common.Address) {
	asr := bindings.NewBindings[guardianActions](bindings.WithClient(g.l1.EthClient()), bindings.WithTo(d.AnchorStateRegistry), bindings.WithTest(g.t))
	_, err := contractio.Write(asr.BlacklistDisputeGame(game), g.t.Ctx(), g.Plan())
	g.t.Require().NoError(err, "failed to blacklist dispute game %s", game)
	g.t.Logf("Blacklisted dispute game %s", game)
}
