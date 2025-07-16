package frontend

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type EngineBackend interface {
	GetPayloadV1(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayload, error)
	GetPayloadV2(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error)
	GetPayloadV3(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error)
	GetPayloadV4(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error)

	ForkchoiceUpdatedV1(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	ForkchoiceUpdatedV2(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	ForkchoiceUpdatedV3(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)

	NewPayloadV1(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error)
	NewPayloadV2(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error)
	NewPayloadV3(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash) (*eth.PayloadStatusV1, error)
	NewPayloadV4(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash, executionRequests []hexutil.Bytes) (*eth.PayloadStatusV1, error)
}

type EngineFrontend struct {
	b EngineBackend
}

func NewEngineFrontend(b EngineBackend) *EngineFrontend {
	return &EngineFrontend{b: b}
}

func (e *EngineFrontend) GetPayloadV1(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayload, error) {
	return e.b.GetPayloadV1(ctx, payloadID)
}

func (e *EngineFrontend) GetPayloadV2(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return e.b.GetPayloadV2(ctx, payloadID)
}

func (e *EngineFrontend) GetPayloadV3(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return e.b.GetPayloadV3(ctx, payloadID)
}

func (e *EngineFrontend) GetPayloadV4(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return e.b.GetPayloadV4(ctx, payloadID)
}

func (e *EngineFrontend) ForkchoiceUpdatedV1(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return e.b.ForkchoiceUpdatedV1(ctx, state, attr)
}

func (e *EngineFrontend) ForkchoiceUpdatedV2(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return e.b.ForkchoiceUpdatedV2(ctx, state, attr)
}

func (e *EngineFrontend) ForkchoiceUpdatedV3(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return e.b.ForkchoiceUpdatedV3(ctx, state, attr)
}

func (e *EngineFrontend) NewPayloadV1(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	return e.b.NewPayloadV1(ctx, payload)
}

func (e *EngineFrontend) NewPayloadV2(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	return e.b.NewPayloadV2(ctx, payload)
}

func (e *EngineFrontend) NewPayloadV3(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash) (*eth.PayloadStatusV1, error) {
	return e.b.NewPayloadV3(ctx, payload, versionedHashes, beaconRoot)
}

func (e *EngineFrontend) NewPayloadV4(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash, executionRequests []hexutil.Bytes) (*eth.PayloadStatusV1, error) {
	return e.b.NewPayloadV4(ctx, payload, versionedHashes, beaconRoot, executionRequests)
}
