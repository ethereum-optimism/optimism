package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources/caching"
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
// May return an error in ForkChoiceResult, but the error is marshalled into the error return
func (s *EngineClient) ForkchoiceUpdate(ctx context.Context, fc *eth.ForkchoiceState, attributes *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	e := s.log.New("state", fc, "attr", attributes)
	e.Trace("Sharing forkchoice-updated signal")
	fcCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var result eth.ForkchoiceUpdatedResult
	err := s.client.CallContext(fcCtx, &result, "engine_forkchoiceUpdatedV1", fc, attributes)
	if err == nil {
		e.Trace("Shared forkchoice-updated signal")
		if attributes != nil {
			e.Trace("Received payload id", "payloadId", result.PayloadID)
		}
		return &result, nil
	} else {
		e = e.New("err", err)
		if rpcErr, ok := err.(rpc.Error); ok {
			code := eth.ErrorCode(rpcErr.ErrorCode())
			e.Warn("Unexpected error code in forkchoice-updated response", "code", code)
		} else {
			e.Error("Failed to share forkchoice-updated signal")
		}
		return nil, err
	}
}

// ExecutePayload executes a built block on the execution engine and returns an error if it was not successful.
func (s *EngineClient) NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	e := s.log.New("block_hash", payload.BlockHash)
	e.Trace("sending payload for execution")

	execCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var result eth.PayloadStatusV1
	err := s.client.CallContext(execCtx, &result, "engine_newPayloadV1", payload)
	e.Trace("Received payload execution result", "status", result.Status, "latestValidHash", result.LatestValidHash, "message", result.ValidationError)
	if err != nil {
		e.Error("Payload execution failed", "err", err)
		return nil, fmt.Errorf("failed to execute payload: %v", err)
	}
	return &result, nil
}

// GetPayload gets the execution payload associated with the PayloadId
func (s *EngineClient) GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error) {
	e := s.log.New("payload_id", payloadId)
	e.Trace("getting payload")
	var result eth.ExecutionPayload
	err := s.client.CallContext(ctx, &result, "engine_getPayloadV1", payloadId)
	if err != nil {
		e = e.New("payload_id", payloadId, "err", err)
		if rpcErr, ok := err.(rpc.Error); ok {
			code := eth.ErrorCode(rpcErr.ErrorCode())
			if code != eth.UnavailablePayload {
				e.Warn("unexpected error code in get-payload response", "code", code)
			} else {
				e.Warn("unavailable payload in get-payload request")
			}
		} else {
			e.Error("failed to get payload")
		}
		return nil, err
	}
	e.Trace("Received payload")
	return &result, nil
}
