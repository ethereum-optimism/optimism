package batcher

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// publicationCursor skips projection positions that have already been derived,
// including deposit-only fallback blocks. It never changes private safety refs.
// Read the head afresh on every poll: an L1 reorg can reopen skipped positions.
func (l *BatchSubmitter) publicationCursor(ctx context.Context, status *eth.SyncStatus) (eth.L2BlockRef, PublicProjectionBlock, bool, error) {
	cursor := status.LocalSafeL2
	if l.PublicProjection == nil {
		return cursor, PublicProjectionBlock{}, false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, l.Config.NetworkTimeout)
	defer cancel()
	head, err := l.PublicProjection.SafeBlock(ctx)
	if err != nil {
		return cursor, PublicProjectionBlock{}, false, err
	}
	if head == nil || head.Hash == (common.Hash{}) {
		return cursor, PublicProjectionBlock{}, false, fmt.Errorf("missing safe projection head")
	}
	canonicalHead, err := l.PublicProjection.BlockByNumber(ctx, head.Number)
	if err != nil {
		return cursor, PublicProjectionBlock{}, false, err
	}
	if canonicalHead == nil || canonicalHead.Number != head.Number || canonicalHead.Hash != head.Hash {
		return cursor, PublicProjectionBlock{}, false, fmt.Errorf("projection rollup and execution heads disagree at %d", head.Number)
	}
	id := *head
	l.channelMgrMutex.Lock()
	previous := l.projectionHead
	l.channelMgrMutex.Unlock()
	reset := previous != (eth.BlockID{}) && id.Number < previous.Number
	if previous != (eth.BlockID{}) && !reset && previous.Number != id.Number {
		canonical, err := l.PublicProjection.BlockByNumber(ctx, previous.Number)
		if err != nil {
			return cursor, id, false, err
		}
		if canonical == nil || canonical.Number != previous.Number {
			return cursor, id, false, fmt.Errorf("missing previous projection position %d", previous.Number)
		}
		reset = canonical.Hash != previous.Hash
	}
	if previous != (eth.BlockID{}) && previous.Number == id.Number {
		reset = previous.Hash != id.Hash
	}
	if head.Number > status.UnsafeL2.Number {
		return cursor, id, reset, fmt.Errorf("private head %d has not caught up to projection %d", status.UnsafeL2.Number, head.Number)
	}
	if head.Number != cursor.Number {
		source, err := l.EndpointProvider.PayloadSource(ctx)
		if err != nil {
			return cursor, id, reset, err
		}
		payload, err := source.PayloadByNumber(ctx, head.Number)
		if err != nil {
			return cursor, id, reset, err
		}
		if payload == nil || payload.ExecutionPayload == nil || uint64(payload.ExecutionPayload.BlockNumber) != head.Number {
			return cursor, id, reset, fmt.Errorf("missing private block at projection position %d", head.Number)
		}
		cursor, err = derive.PayloadToBlockRef(l.RollupConfig, payload.ExecutionPayload)
		if err != nil {
			return cursor, id, reset, err
		}
	}
	return cursor, id, reset, nil
}

// These helpers run under channelMgrMutex, like the channel encoder. Discard
// prepared receipts/writes when their blocks are pruned without being encoded.
func (l *BatchSubmitter) forgetPreparedBlocks(blocks []SizedBlock) {
	if encoder, ok := l.BlockEnricher.(*PrivateInteropEncoder); ok {
		hashes := make([]common.Hash, len(blocks))
		for i, block := range blocks {
			hashes[i] = block.Hash()
		}
		encoder.forget(hashes)
	}
}

func (l *BatchSubmitter) clearChannelState(origin eth.BlockID) {
	if encoder, ok := l.BlockEnricher.(*PrivateInteropEncoder); ok {
		encoder.mu.Lock()
		clear(encoder.prepared)
		encoder.mu.Unlock()
	}
	l.channelMgr.Clear(origin)
}
