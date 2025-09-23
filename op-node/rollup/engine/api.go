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

func (ec *EngineController) OpenBlock(ctx context.Context, parent eth.BlockID, attrs *eth.PayloadAttributes) (eth.PayloadInfo, error) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	_, err := ec.engine.L2BlockRefByHash(ctx, parent.Hash)
	if err != nil {
		return eth.PayloadInfo{}, fmt.Errorf("failed to retrieve parent block %s from engine: %w", parent, err)
	}

	if err := ec.initializeUnknowns(ctx); err != nil {
		return eth.PayloadInfo{}, fmt.Errorf("failed to initialize forkchoice pre-state: %w", err)
	}

	fc := eth.ForkchoiceState{
		HeadBlockHash:      parent.Hash,
		SafeBlockHash:      ec.safeHead.Hash,
		FinalizedBlockHash: ec.finalizedHead.Hash,
	}
	id, errTyp, err := startPayload(ctx, ec.engine, fc, attrs)
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

func (ec *EngineController) CancelBlock(ctx context.Context, id eth.PayloadInfo) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	_, err := ec.engine.GetPayload(ctx, id)
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

func (ec *EngineController) SealBlock(ctx context.Context, id eth.PayloadInfo) (*eth.ExecutionPayloadEnvelope, error) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	envelope, err := ec.engine.GetPayload(ctx, id)
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

func (ec *EngineController) CommitBlock(ctx context.Context, signed *opsigner.SignedExecutionPayloadEnvelope) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	envelope := signed.Envelope
	ref, err := derive.PayloadToBlockRef(ec.rollupCfg, envelope.ExecutionPayload)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	status, err := ec.engine.NewPayload(ctx, envelope.ExecutionPayload, envelope.ParentBeaconBlockRoot)
	if err != nil {
		return fmt.Errorf("failed to insert payload: %w", err)
	}

	switch status.Status {
	case eth.ExecutionInvalid, eth.ExecutionInvalidBlockHash:
		return &rpc.JsonError{
			Code:    apis.BuildErrCodeInvalidInput,
			Message: fmt.Sprintf("execution invalid: %v", err),
		}
	case eth.ExecutionValid:
		break
	}

	ec.SetUnsafeHead(ref)
	ec.emitter.Emit(ctx, UnsafeUpdateEvent{Ref: ref})
	if err := ec.tryUpdateEngine(ctx); err != nil {
		return fmt.Errorf("failed to update engine forkchoice: %w", err)
	}
	return nil
}
