package derive

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimistic-specs/opnode/l2"
	"github.com/ethereum/go-ethereum/common"
)

type BlockPreparer interface {
	GetPayload(ctx context.Context, payloadId l2.PayloadID) (*l2.ExecutionPayload, error)
	ForkchoiceUpdated(ctx context.Context, state *l2.ForkchoiceState, attr *l2.PayloadAttributes) (l2.ForkchoiceUpdatedResult, error)
}

// BlockOutputs uses the engine API to derive a full L2 block from the block inputs.
// The fcState does not affect the block production, but may inform the engine of finality and head changes to sync towards before block computation.
func BlockOutputs(ctx context.Context, engine BlockPreparer, l2Parent common.Hash, l2Finalized common.Hash, attributes *l2.PayloadAttributes) (*l2.ExecutionPayload, error) {
	fcState := &l2.ForkchoiceState{
		HeadBlockHash:      l2Parent, // no difference yet between Head and Safe, no data ahead of L1 yet.
		SafeBlockHash:      l2Parent,
		FinalizedBlockHash: l2Finalized,
	}
	fcResult, err := engine.ForkchoiceUpdated(ctx, fcState, attributes)
	if err != nil {
		return nil, fmt.Errorf("engine failed to process forkchoice update for block derivation: %v", err)
	} else if fcResult.Status != l2.UpdateSuccess {
		return nil, fmt.Errorf("engine not in sync, failed to derive block, status: %s", fcResult.Status)
	}

	payload, err := engine.GetPayload(ctx, *fcResult.PayloadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payload: %v", err)
	}
	return payload, nil
}
