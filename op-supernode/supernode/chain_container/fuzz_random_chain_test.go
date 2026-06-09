package chain_container

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	"github.com/stretchr/testify/require"
)

// FuzzRandomChainReadPath generates a chain set from the input and drives the
// chain-container read seams (the interop ingestion path), asserting the model
// answers consistently and never panics.
func FuzzRandomChainReadPath(f *testing.F) {
	f.Add([]byte("seed-1"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		ctx := context.Background()
		m := NewRandomChainManager(data)
		m.Generate()

		for _, rc := range m.Chains() {
			require.NoError(t, rc.Start(ctx))
		}

		for _, rc := range m.Chains() {
			cc, err := m.ChainContainer(rc.chainID)
			require.NoError(t, err)
			require.Equal(t, rc.chainID, cc.ID())

			ss, err := cc.SyncStatus(ctx)
			require.NoError(t, err)
			require.Equal(t, rc.l2[rc.safe].Ref.Hash, ss.LocalSafeL2.Hash)

			require.Equal(t, genL2BlockTime, int(cc.BlockTime()))

			for n := range rc.l2 {
				blk := rc.l2[n]

				out, err := cc.OutputV0AtBlockNumber(ctx, uint64(n))
				require.NoError(t, err)
				require.Equal(t, blk.Ref.Hash, out.BlockHash)

				ts, err := cc.BlockNumberToTimestamp(ctx, uint64(n))
				require.NoError(t, err)
				require.Equal(t, blk.Ref.Time, ts)

				info, rcpts, err := cc.FetchReceipts(ctx, blk.Ref.ID())
				require.NoError(t, err)
				require.Equal(t, blk.Ref.Hash, info.Hash())
				require.Len(t, rcpts, 1) // Receipts packs all logs into one receipt
				require.Len(t, rcpts[0].Logs, len(blk.ExecMsgs))
				for _, lg := range rcpts[0].Logs {
					_, derr := processors.DecodeExecutingMessageLog(lg)
					require.NoError(t, derr)
				}
			}

			// Timestamp <-> block-number round-trips at the safe head.
			safeTS, err := cc.BlockNumberToTimestamp(ctx, rc.safe)
			require.NoError(t, err)
			num, err := cc.TimestampToBlockNumber(ctx, safeTS)
			require.NoError(t, err)
			require.Equal(t, rc.safe, num)

			fin, err := cc.ELFinalizedHead(ctx)
			require.NoError(t, err)
			require.Equal(t, rc.l2[rc.finalized].Ref.Hash, fin.Hash)

			firstTS, err := cc.FirstSafeHeadTimestamp(ctx)
			require.NoError(t, err)
			require.Equal(t, rc.l2[rc.safeDB[0].L2.Number].Ref.Time, firstTS)

			safeBlk := rc.l2[rc.safe]
			l2id, _, err := cc.OptimisticAt(ctx, safeBlk.Ref.Time)
			require.NoError(t, err)
			require.Equal(t, safeBlk.Ref.Hash, l2id.Hash)
		}
	})
}
