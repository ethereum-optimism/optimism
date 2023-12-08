package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type EngineClientConfig struct {
	L2ClientConfig
}

func EngineClientDefaultConfig(config *rollup.Config) *EngineClientConfig {
	return &EngineClientConfig{
		// engine is trusted, no need to recompute responses etc.
		L2ClientConfig: *L2ClientDefaultConfig(config, true),
	}
}

// EngineClient extends L2Client with engine API bindings.
type EngineClient struct {
	*L2Client
}

func NewEngineClient(client client.RPC, log log.Logger, metrics caching.Metrics, config *EngineClientConfig) (*EngineClient, error) {
	l2Client, err := NewL2Client(client, log, metrics, &config.L2ClientConfig)
	if err != nil {
		return nil, err
	}

	return &EngineClient{
		L2Client: l2Client,
	}, nil
}

// ForkchoiceUpdate updates the forkchoice on the execution client. If attributes is not nil, the engine client will also begin building a block
// based on attributes after the new head block and return the payload ID.
//
// The RPC may return three types of errors:
// 1. Processing error: ForkchoiceUpdatedResult.PayloadStatusV1.ValidationError or other non-success PayloadStatusV1,
// 2. `error` as eth.InputError: the forkchoice state or attributes are not valid.
// 3. Other types of `error`: temporary RPC errors, like timeouts.
func (s *EngineClient) ForkchoiceUpdate(ctx context.Context, fc *eth.ForkchoiceState, attributes *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	llog := s.log.New("state", fc)       // local logger
	tlog := llog.New("attr", attributes) // trace logger
	tlog.Trace("Sharing forkchoice-updated signal")
	fcCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var result eth.ForkchoiceUpdatedResult
	err := s.client.CallContext(fcCtx, &result, "engine_forkchoiceUpdatedV2", fc, attributes)
	if err == nil {
		tlog.Trace("Shared forkchoice-updated signal")
		if attributes != nil { // block building is optional, we only get a payload ID if we are building a block
			tlog.Trace("Received payload id", "payloadId", result.PayloadID)
		}
		return &result, nil
	} else {
		llog.Warn("Failed to share forkchoice-updated signal", "err", err)
		if rpcErr, ok := err.(rpc.Error); ok {
			code := eth.ErrorCode(rpcErr.ErrorCode())
			switch code {
			case eth.InvalidForkchoiceState, eth.InvalidPayloadAttributes:
				return nil, eth.InputError{
					Inner: err,
					Code:  code,
				}
			default:
				return nil, fmt.Errorf("unrecognized rpc error: %w", err)
			}
		}
		return nil, err
	}
}

// NewPayload executes a full block on the execution engine.
// This returns a PayloadStatusV1 which encodes any validation/processing error,
// and this type of error is kept separate from the returned `error` used for RPC errors, like timeouts.
func (s *EngineClient) NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e := s.log.New("block_hash", payload.BlockHash)
	e.Trace("sending payload for execution")

	execCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var result eth.PayloadStatusV1
	err := s.client.CallContext(execCtx, &result, "engine_newPayloadV2", payload)
	e.Trace("Received payload execution result", "status", result.Status, "latestValidHash", result.LatestValidHash, "message", result.ValidationError)
	if err != nil {
		e.Error("Payload execution failed", "err", err)
		return nil, fmt.Errorf("failed to execute payload: %w", err)
	}
	return &result, nil
}

// GetPayload gets the execution payload associated with the PayloadId.
// There may be two types of error:
// 1. `error` as eth.InputError: the payload ID may be unknown
// 2. Other types of `error`: temporary RPC errors, like timeouts.
func (s *EngineClient) GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error) {
	e := s.log.New("payload_id", payloadId)
	e.Trace("getting payload")
	var result eth.ExecutionPayloadEnvelope
	err := s.client.CallContext(ctx, &result, "engine_getPayloadV2", payloadId)
	if err != nil {
		e.Warn("Failed to get payload", "payload_id", payloadId, "err", err)
		if rpcErr, ok := err.(rpc.Error); ok {
			code := eth.ErrorCode(rpcErr.ErrorCode())
			switch code {
			case eth.UnknownPayload:
				return nil, eth.InputError{
					Inner: err,
					Code:  code,
				}
			default:
				return nil, fmt.Errorf("unrecognized rpc error: %w", err)
			}
		}
		return nil, err
	}
	e.Trace("Received payload")
	return result.ExecutionPayload, nil
}

func (s *EngineClient) SignalSuperchainV1(ctx context.Context, recommended, required params.ProtocolVersion) (params.ProtocolVersion, error) {
	var result params.ProtocolVersion
	err := s.client.CallContext(ctx, &result, "engine_signalSuperchainV1", &catalyst.SuperchainSignal{
		Recommended: recommended,
		Required:    required,
	})
	return result, err
}
