package batcher

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	opfees "github.com/ethereum-optimism/optimism/op-core/fees"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	derivetest "github.com/ethereum-optimism/optimism/op-node/rollup/derive/test"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

// TestSizedBlockParity pins that the payload-based size estimates equal the
// values computed from the typed go-ethereum block: tx.Size() equals the
// canonical encoding length, and the rollup cost data over the raw bytes
// equals opfees.TxRollupCostData over the typed transaction.
func TestSizedBlockParity(t *testing.T) {
	rng := rand.New(rand.NewSource(0x5121))
	for i := 0; i < 8; i++ {
		block := derivetest.RandomL2BlockWithChainId(rng, rng.Intn(16), defaultTestRollupConfig.L2ChainID)
		sized := mustSizedBlockFromGeth(block)

		wantRaw := uint64(70)
		wantDA := uint64(70)
		for _, tx := range block.Transactions() {
			if optypes.IsDepositTx(tx) {
				continue
			}
			wantRaw += tx.Size() + 2
			wantDA += bigs.Uint64Strict(opfees.TxRollupCostData(tx).EstimatedDASize())
		}
		require.Equal(t, wantRaw, sized.RawSize(), "raw size, block %d", i)
		require.Equal(t, wantDA, sized.EstimatedDABytes(), "estimated DA bytes, block %d", i)
	}
}
