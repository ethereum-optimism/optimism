package review_period

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
)

// readDelaysAcrossLastUpgrade reads a chain's delays at the block before its proxies last emitted
// Upgraded and at the latest block, the same way check-review-period does on a live network.
func readDelaysAcrossLastUpgrade(t devtest.T, l1 *dsl.L1ELNode, systemConfig common.Address) (pre, post reviewperiod.ChainDelays) {
	client, err := ethclient.DialContext(t.Ctx(), l1.Escape().UserRPC())
	t.Require().NoError(err, "failed to dial the L1 RPC")
	t.Cleanup(client.Close)

	head, err := client.BlockNumber(t.Ctx())
	t.Require().NoError(err, "failed to get the L1 block number")
	post, err = reviewperiod.ReadChainDelays(t.Ctx(), l1.EthClient(), systemConfig, rpc.BlockNumber(head))
	t.Require().NoError(err, "failed to read the delays at block %d", head)

	upgrades, err := reviewperiod.UpgradedLogs(t.Ctx(), client, post, 0, head)
	t.Require().NoError(err, "failed to get the Upgraded logs")
	t.Require().NotEmpty(upgrades, "no proxy of chain %d was ever upgraded", post.L2ChainID)
	before := upgrades[len(upgrades)-1].BlockNumber - 1
	t.Logf("Comparing chain %d's delays at block %d, before its last upgrade in tx %s, and at block %d",
		post.L2ChainID, before, upgrades[len(upgrades)-1].TxHash, head)

	pre, err = reviewperiod.ReadChainDelays(t.Ctx(), l1.EthClient(), systemConfig, rpc.BlockNumber(before))
	t.Require().NoError(err, "failed to read the delays at block %d, before the last upgrade", before)
	return pre, post
}

// delaySetters binds the review period delay setters.
type delaySetters struct {
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
