package sysgo

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

type engineClient struct {
	inner *rpc.Client
}

func dialEngine(ctx context.Context, endpoint string, jwtSecret [32]byte) (*engineClient, error) {
	engineCl, err := gethrpc.DialOptions(ctx, endpoint, rpc.WithHTTPAuth(gn.NewJWTAuth(jwtSecret)))
	if err != nil {
		return nil, err
	}
	return &engineClient{
		inner: engineCl,
	}, nil
}

var _ geth.EngineAPI = (*engineClient)(nil)

func (e *engineClient) forkchoiceUpdated(fs engine.ForkchoiceStateV1, pa *engine.PayloadAttributes, method string) (engine.ForkChoiceResponse, error) {
	var x engine.ForkChoiceResponse
	if err := e.inner.CallContext(context.Background(), &x, method, fs, pa); err != nil {
		return engine.ForkChoiceResponse{}, err
	}
	return x, nil
}

func (e *engineClient) ForkchoiceUpdatedV2(fs engine.ForkchoiceStateV1, pa *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
	return e.forkchoiceUpdated(fs, pa, "engine_forkchoiceUpdatedV2")
}

func (e *engineClient) ForkchoiceUpdatedV3(fs engine.ForkchoiceStateV1, pa *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
	return e.forkchoiceUpdated(fs, pa, "engine_forkchoiceUpdatedV3")
}

func (e *engineClient) getPayload(id engine.PayloadID, method string) (*engine.ExecutionPayloadEnvelope, error) {
	var result engine.ExecutionPayloadEnvelope
	if err := e.inner.CallContext(context.Background(), &result, method, id); err != nil {
		return nil, err
	}
	return &result, nil
}

func (e *engineClient) GetPayloadV2(id engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	return e.getPayload(id, "engine_getPayloadV2")
}

func (e *engineClient) GetPayloadV3(id engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	return e.getPayload(id, "engine_getPayloadV3")
}

func (e *engineClient) GetPayloadV4(id engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	return e.getPayload(id, "engine_getPayloadV4")
}

func (e *engineClient) GetPayloadV5(id engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	return e.getPayload(id, "engine_getPayloadV5")
}

func (e *engineClient) NewPayloadV2(data engine.ExecutableData) (engine.PayloadStatusV1, error) {
	var result engine.PayloadStatusV1
	if err := e.inner.CallContext(context.Background(), &result, "engine_newPayloadV2", data); err != nil {
		return engine.PayloadStatusV1{}, err
	}
	return result, nil
}

func (e *engineClient) NewPayloadV3(data engine.ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash) (engine.PayloadStatusV1, error) {
	var result engine.PayloadStatusV1
	if err := e.inner.CallContext(context.Background(), &result, "engine_newPayloadV3", data, versionedHashes, beaconRoot); err != nil {
		return engine.PayloadStatusV1{}, err
	}
	return result, nil
}

func (e *engineClient) NewPayloadV4(data engine.ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash, executionRequests []hexutil.Bytes) (engine.PayloadStatusV1, error) {
	var result engine.PayloadStatusV1
	if err := e.inner.CallContext(context.Background(), &result, "engine_newPayloadV4", data, versionedHashes, beaconRoot, executionRequests); err != nil {
		return engine.PayloadStatusV1{}, err
	}
	return result, nil
}
