package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
)

// RollupAPI is the API we serve as rollup-node to interact with the execution engine and forkchoice state.
type RollupAPI interface {
	apis.BuildAPI
	apis.CommitAPI
}

var _ RollupAPI = (*EngineController)(nil)

func (e *EngineController) OpenBlock(ctx context.Context, parent eth.BlockID, attrs *eth.PayloadAttributes) (eth.PayloadInfo, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	_, err := e.engine.L2BlockRefByHash(ctx, parent.Hash)
	if err != nil {
		return eth.PayloadInfo{}, fmt.Errorf("failed to retrieve parent block %s from engine: %w", parent, err)
	}

	if err := e.initializeUnknowns(ctx); err != nil {
		return eth.PayloadInfo{}, fmt.Errorf("failed to initialize forkchoice pre-state: %w", err)
	}

	fc := eth.ForkchoiceState{
		HeadBlockHash:      parent.Hash,
		SafeBlockHash:      e.SafeL2Head().Hash,
		FinalizedBlockHash: e.FinalizedHead().Hash,
	}
	id, errTyp, err := e.startPayload(ctx, fc, attrs)
	if err != nil {
		switch errTyp {
		case BlockInsertTemporaryErr:
			// RPC errors are not persistent block processing errors
			return eth.PayloadInfo{}, &rpc.JsonError{
				Code:    apis.BuildErrCodeTemporary,
				Message: fmt.Sprintf("temporarily cannot insert new safe block: %v", err),
			}
		case BlockInsertPrestateErr:
			return eth.PayloadInfo{}, &rpc.JsonError{
				Code:    apis.BuildErrCodePrestate,
				Message: fmt.Sprintf("need reset to resolve pre-state problem: %v", err),
			}
		case BlockInsertPayloadErr:
			return eth.PayloadInfo{}, &rpc.JsonError{
				Code:    apis.BuildErrCodePrestate,
				Message: fmt.Sprintf("invalid payload attributes: %v", err),
			}
		default:
			return eth.PayloadInfo{}, &rpc.JsonError{
				Code:    apis.BuildErrCodeOther,
				Message: fmt.Sprintf("unknown error type %d: %v", errTyp, err),
			}
		}
	}
	return eth.PayloadInfo{
		ID:        id,
		Timestamp: uint64(attrs.Timestamp),
	}, nil
}

func (e *EngineController) CancelBlock(ctx context.Context, id eth.PayloadInfo) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.engine.GetPayload(ctx, id)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) && eth.ErrorCode(rpcErr.ErrorCode()) == eth.UnknownPayload {
			return &rpc.JsonError{ // unwrap error, to serve opstack RPC
				Code:    apis.BuildErrCodeUnknownPayload,
				Message: "unknown payload",
			}
		}
		return &rpc.JsonError{
			Code:    apis.BuildErrCodeOther,
			Message: fmt.Sprintf("failed to cancel payload: %v", err),
		}
	}
	return nil
}

func (e *EngineController) SealBlock(ctx context.Context, id eth.PayloadInfo) (*eth.ExecutionPayloadEnvelope, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	envelope, err := e.engine.GetPayload(ctx, id)
	if err != nil {
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) && eth.ErrorCode(rpcErr.ErrorCode()) == eth.UnknownPayload {
			return nil, &rpc.JsonError{ // unwrap error, to serve opstack RPC
				Code:    apis.BuildErrCodeUnknownPayload,
				Message: "unknown payload",
			}
		}
		return nil, &rpc.JsonError{
			Code:    apis.BuildErrCodeOther,
			Message: fmt.Sprintf("failed to seal payload: %v", err),
		}
	}
	return envelope, nil
}

func (e *EngineController) CommitBlock(ctx context.Context, signed *opsigner.SignedExecutionPayloadEnvelope) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	envelope := signed.Envelope
	ref, err := derive.PayloadToBlockRef(e.rollupCfg, envelope.ExecutionPayload)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	status, err := e.engine.NewPayload(ctx, envelope.ExecutionPayload, envelope.ParentBeaconBlockRoot)
	if err != nil {
		return fmt.Errorf("failed to insert payload: %w", err)
	}

	switch status.Status {
	case eth.ExecutionValid, eth.ExecutionSyncing, eth.ExecutionAccepted:
		// Proceed to the forkchoice update. SYNCING and ACCEPTED are tolerated here to stay
		// consistent with insertUnsafePayload, which also continues on a non-VALID-but-not-invalid
		// status so the engine can be driven towards the new head.
	default:
		// Anything else (INVALID, INVALID_BLOCK_HASH, INVALID_TERMINAL_BLOCK, or a status this
		// version does not recognize) must not become the unsafe head.
		return &rpc.JsonError{
			Code:    apis.BuildErrCodeInvalidInput,
			Message: eth.NewPayloadErr(envelope.ExecutionPayload, status).Error(),
		}
	}

	e.SetUnsafeHead(ref)
	e.emitter.Emit(ctx, UnsafeUpdateEvent{Ref: ref})
	if err := e.tryUpdateEngineInternal(ctx); err != nil {
		return fmt.Errorf("failed to update engine forkchoice: %w", err)
	}
	return nil
}
