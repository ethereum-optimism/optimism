package review_period

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-review-period/reviewperiod"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
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
