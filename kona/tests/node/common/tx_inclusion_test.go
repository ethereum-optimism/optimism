package node

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	node_utils "github.com/op-rs/kona/node/utils"
)

func TestL2TransactionInclusion(gt *testing.T) {
	t := devtest.SerialT(gt)
	out := node_utils.NewMixedOpKona(t)

	originNode := out.L2ELSequencerNodes()[0]
	funder := dsl.NewFunder(out.Wallet, out.Faucet, originNode)

	user := funder.NewFundedEOA(eth.OneEther)
	to := out.Wallet.NewEOA(originNode)
	toInitialBalance := to.GetBalance()
	tx := user.Transfer(to.Address(), eth.HalfEther)

	inclusionBlock, err := tx.IncludedBlock.Eval(t.Ctx())
	if err != nil {
		gt.Fatal("transaction receipt not found", "error", err)
	}

	// Ensure the block containing the transaction has propagated to the rest of the network.
	for _, node := range out.L2ELNodes() {
		block := node.WaitForBlockNumber(inclusionBlock.Number)
		blockID := block.Hash()
		blockNumber := block.NumberU64()

		// It's possible that the block has already been included, and `WaitForBlockNumber` returns a block
		// at a taller height.
		if blockNumber > inclusionBlock.Number {
			blockID = node.BlockRefByNumber(inclusionBlock.Number).Hash
		}

		// Ensure that the block ID matches the expected inclusion block hash.
		if blockID != inclusionBlock.Hash {
			gt.Fatal("transaction not included in block", "node", node.String(), "expectedBlockHash", inclusionBlock.Hash, "actualBlockHash", blockID)
		}

		// Ensure that the recipient's balance has been updated in the eyes of the EL node.
		to.AsEL(node).VerifyBalanceExact(toInitialBalance.Add(eth.HalfEther))
	}
}
