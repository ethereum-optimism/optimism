package batcher

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	opfees "github.com/ethereum-optimism/optimism/op-core/fees"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	derivetest "github.com/ethereum-optimism/optimism/op-node/rollup/derive/test"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestSizedBlockSkipsDeposits pins that the size estimates count every
// batch-submitted transaction, of every type and including a post-exec one,
// and skip the deposits, which are derived from L1.
func TestSizedBlockSkipsDeposits(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	payload := derivetest.RandomL2Payload(rng, 12, defaultTestRollupConfig.L2ChainID, 0)
	batched := append([]eth.Data(nil), payload.Transactions[1:]...) // [0] is the L1-info deposit

	userDeposit, err := (&optypes.DepositTx{SourceHash: common.Hash{1}, Gas: 21_000}).MarshalBinary()
	require.NoError(t, err)
	postExec, err := (&optypes.PostExecTx{Data: []byte{0xc2, 0x80, 0x80}}).MarshalBinary()
	require.NoError(t, err)
	payload.Transactions = append(payload.Transactions, userDeposit, postExec)
	batched = append(batched, postExec)

	wantRaw, wantDA := uint64(70), uint64(70)
	for _, tx := range batched {
		wantRaw += uint64(len(tx)) + 2
		wantDA += bigs.Uint64Strict(opfees.NewRollupCostData(tx).EstimatedDASize())
	}
	sized := ToSizedBlock(payload)
	require.Equal(t, wantRaw, sized.RawSize())
	require.Equal(t, wantDA, sized.EstimatedDABytes())
}
