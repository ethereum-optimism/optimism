package verifier

import (
	"context"
	"math/big"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// TestPendingBlockIsLatest tests that we serve the latest block as pending block
func TestPendingBlockIsLatest(t *testing.T) {
	op_e2e.InitParallel(t)

	cfg := e2esys.DefaultSystemConfig(t)
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")

	l2Seq := sys.NodeClient("sequencer")

	t.Run("block", func(t *testing.T) {
		l2SeqSrc := sys.NodeSourceClient("sequencer")
		for i := 0; i < 10; i++ {
			pending, err := l2SeqSrc.PayloadByLabel(context.Background(), eth.BlockLabel(rpc.PendingBlockNumber.String()))
			require.NoError(t, err)
			latest, err := l2SeqSrc.PayloadByLabel(context.Background(), eth.Unsafe)
			require.NoError(t, err)
			if pending.ExecutionPayload.BlockNumber == latest.ExecutionPayload.BlockNumber {
				require.Equal(t, pending.ExecutionPayload.BlockHash, latest.ExecutionPayload.BlockHash, "pending must exactly match latest block")
				return
			}
			// re-try until we have the same number, as the requests are not an atomic bundle, and the sequencer may create a block.
		}
		t.Fatal("failed to get pending block with same number as latest block")
	})
	t.Run("header", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			pending, err := l2Seq.HeaderByNumber(context.Background(), big.NewInt(rpc.PendingBlockNumber.Int64()))
			require.NoError(t, err)
			latest, err := l2Seq.HeaderByNumber(context.Background(), nil)
			require.NoError(t, err)
			if bigs.Uint64Strict(pending.Number) == bigs.Uint64Strict(latest.Number) {
				require.Equal(t, pending.Hash(), latest.Hash(), "pending must exactly match latest header")
				return
			}
			// re-try until we have the same number, as the requests are not an atomic bundle, and the sequencer may create a block.
		}
		t.Fatal("failed to get pending header with same number as latest header")
	})
}
