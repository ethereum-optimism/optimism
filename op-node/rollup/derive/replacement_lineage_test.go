package derive

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestReplacedBlockLineageSelection reproduces the derivation asymmetry observed on
// devnet interop-reorg-5 (chain 420120192, blocks 13460 / 14245): after an interop
// invalidation replaces block N with a deposits-only block, the SAME span batch
// yields DIFFERENT continuations depending on where a node's derivation walk is
// anchored relative to N:
//
//   - anchored AT the replacement (l2SafeHead = replaced N): the span passes the
//     prefix parent_check against N's ancestor, and GetSingularBatches past-skips
//     block N entirely — its content is never re-validated — so the span TAIL
//     (N+1..) is re-applied on top of the replacement (popNextBatch re-stamps
//     ParentHash from the actual parent). The node keeps this span's lineage.
//
//   - anchored BELOW N (l2SafeHead = N-1): block N is RE-PROPOSED. In production
//     the supernode's SuperAuthority denylist then rejects it at payload
//     insertion (op-node/rollup/engine/payload_process.go), triggering a
//     deposits-only replacement that discards the remainder of the channel — so
//     the NEXT channel's lineage wins instead.
//
// Two nodes processing identical L1 data therefore end on different canonical
// chains, and the abandoned lineage dead-ends permanently once the batcher moves
// on. This test pins the derive-level half of that mechanism.
func TestReplacedBlockLineageSelection(t *testing.T) {
	zero := uint64(0)
	cfg := &rollup.Config{
		Genesis:           rollup.Genesis{L2Time: 1000},
		BlockTime:         1,
		SeqWindowSize:     3600,
		MaxSequencerDrift: 600,
		DeltaTime:         &zero,
		HoloceneTime:      &zero,
	}
	chainID := big.NewInt(420)

	l1A := eth.L1BlockRef{Hash: common.HexToHash("0xaaaa01"), Number: 100, Time: 1002}
	l1B := eth.L1BlockRef{Hash: common.HexToHash("0xaaaa02"), Number: 101, ParentHash: l1A.Hash, Time: 1014}

	parentHash := common.HexToHash("0x44444444444444444444444444444444444444444444444444444444deadbeef")
	l2Parent := eth.L2BlockRef{ // block N-1 = 4, common ancestor
		Hash:     parentHash,
		Number:   4,
		Time:     1004,
		L1Origin: eth.BlockID{Hash: l1A.Hash, Number: l1A.Number},
	}
	// Block N=5 as replaced by the interop invalidation: deposits-only, with a hash
	// that the span batch's original chain never contained.
	replacement := eth.L2BlockRef{
		Hash:       common.HexToHash("0x5555555555555555555555555555555555555555555555555555555500197d39"),
		Number:     5,
		ParentHash: parentHash,
		Time:       1005,
		L1Origin:   eth.BlockID{Hash: l1A.Hash, Number: l1A.Number},
	}

	// One span batch covering N..N+3 (5..8), parent_check = hash(4).
	// This models channel A's batch 18 (13456-13465) relative to replaced 13460.
	var singulars []*SingularBatch
	for i := uint64(5); i <= 8; i++ {
		singulars = append(singulars, &SingularBatch{
			ParentHash:   parentHash, // only the first block's parent is encoded (parent_check)
			EpochNum:     rollup.Epoch(l1A.Number),
			EpochHash:    l1A.Hash,
			Timestamp:    1000 + i,
			Transactions: nil,
		})
	}
	span := initializedSpanBatch(singulars, cfg.Genesis.L2Time, chainID)
	batch := BatchWithL1InclusionBlock{L1InclusionBlock: l1B, Batch: span}

	logger := testlog.Logger(t, log.LevelError)
	l2Client := testutils.MockL2Client{}
	var nilErr error
	l2Client.Mock.On("L2BlockRefByNumber", l2Parent.Number).Return(l2Parent, &nilErr)

	ctx := context.Background()

	t.Run("anchored_at_replacement_past_skips_and_keeps_span_tail", func(t *testing.T) {
		// Walk anchored at the replacement block: the Holocene prefix check (the
		// production BatchStage path — unlike the legacy full check it never
		// fetches overlap payloads) consults only the pre-overlap ancestor
		// (block 4), so the span is ACCEPTED even though its own block 5 was
		// invalidated and replaced.
		validity, _ := checkSpanBatchPrefix(ctx, cfg, logger, []eth.L1BlockRef{l1A, l1B}, replacement, span, batch.L1InclusionBlock, &l2Client)
		require.EqualValues(t, BatchAccept, validity,
			"span overlapping a replaced block must still pass prefix checks via the pre-overlap ancestor")

		got, err := span.GetSingularBatches([]eth.L1BlockRef{l1A, l1B}, replacement)
		require.NoError(t, err)
		var ts []uint64
		for _, sb := range got {
			ts = append(ts, sb.Timestamp)
		}
		// Block 5 (the replaced block) is past-skipped — its span content is never
		// re-validated against the replacement — and the tail 6..8 will be re-applied
		// on top of the replacement (popNextBatch stamps ParentHash from the actual
		// parent, i.e. the replacement hash).
		require.Equal(t, []uint64{1006, 1007, 1008}, ts,
			"tail must be re-applied across the replaced block without re-validating it")
	})

	t.Run("anchored_below_replacement_reproposes_denied_block", func(t *testing.T) {
		// Walk anchored below the replacement (fresh sync / deep reset anchor):
		// the SAME span re-proposes block 5. In production this is the path where
		// the SuperAuthority denylist fires at payload insertion, the block is
		// replaced deposits-only, the channel remainder is discarded, and the next
		// channel's lineage wins.
		validity, _ := checkSpanBatchPrefix(ctx, cfg, logger, []eth.L1BlockRef{l1A, l1B}, l2Parent, span, batch.L1InclusionBlock, &l2Client)
		require.EqualValues(t, BatchAccept, validity)

		got, err := span.GetSingularBatches([]eth.L1BlockRef{l1A, l1B}, l2Parent)
		require.NoError(t, err)
		var ts []uint64
		for _, sb := range got {
			ts = append(ts, sb.Timestamp)
		}
		require.Equal(t, []uint64{1005, 1006, 1007, 1008}, ts,
			"walk from below must re-propose the invalidated block")
	})
}
