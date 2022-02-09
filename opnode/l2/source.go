package l2

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type RPC interface {
	ExecutePayload(ctx context.Context, payload *ExecutionPayload) (*ExecutePayloadResult, error)
	ForkchoiceUpdated(ctx context.Context, state *ForkchoiceState, attr *PayloadAttributes) (ForkchoiceUpdatedResult, error)
}

// ExecutePayload executes the payload and parses the return status into a useful error code
func ExecutePayload(ctx context.Context, rpc RPC, payload *ExecutionPayload) error {
	execCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	execRes, err := rpc.ExecutePayload(execCtx, payload)
	if err != nil {
		return fmt.Errorf("failed to execute payload: %v", err)
	}
	switch execRes.Status {
	case ExecutionValid:
		return nil
	case ExecutionSyncing:
		return fmt.Errorf("failed to execute payload %s, node is syncing, latest valid hash is %s", payload.ID(), execRes.LatestValidHash)
	case ExecutionInvalid:
		return fmt.Errorf("execution payload %s was INVALID! Latest valid hash is %s, ignoring bad block: %q", payload.ID(), execRes.LatestValidHash, execRes.ValidationError)
	default:
		return fmt.Errorf("unknown execution status on %s: %q, ", payload.ID(), string(execRes.Status))
	}
}

// ForkchoiceUpdate updates the forkchoive for L2 and parses the return status into a useful error code
func ForkchoiceUpdate(ctx context.Context, rpc RPC, l2BlockHash common.Hash, l2Finalized common.Hash) error {
	postState := &ForkchoiceState{
		HeadBlockHash:      l2BlockHash, // no difference yet between Head and Safe, no data ahead of L1 yet.
		SafeBlockHash:      l2BlockHash,
		FinalizedBlockHash: l2Finalized,
	}

	fcCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	fcRes, err := rpc.ForkchoiceUpdated(fcCtx, postState, nil)
	if err != nil {
		return fmt.Errorf("failed to update forkchoice: %v", err)
	}
	switch fcRes.Status {
	case UpdateSyncing:
		return fmt.Errorf("updated forkchoice, but node is syncing: %v", err)
	case UpdateSuccess:
		return nil
	default:
		return fmt.Errorf("unknown forkchoice status on %s: %q, ", l2BlockHash, string(fcRes.Status))
	}
}
