package frontend

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type EngineBackend interface {
	apis.EngineAPI
}

type EngineFrontend struct {
	b EngineBackend
}

func NewEngineFrontend(b EngineBackend) *EngineFrontend {
	return &EngineFrontend{b: b}
}

// BlobsBundleV1 is the engine-API blobs bundle. The sync tester never produces blobs (OP Stack
// payloads carry none), so the bundle it serves is always empty — but its arrays must be present
// and non-null for spec-strict clients.
type BlobsBundleV1 struct {
	Commitments []hexutil.Bytes `json:"commitments"`
	Proofs      []hexutil.Bytes `json:"proofs"`
	Blobs       []hexutil.Bytes `json:"blobs"`
}

// GetPayloadResponse is the spec-shaped engine_getPayloadV2+ response.
//
// The sync tester stores built payloads as eth.ExecutionPayloadEnvelope, which carries only the
// payload and the parent beacon block root — the gossip envelope shape, not the getPayload
// response the execution-apis spec defines. Go clients never noticed the difference (they decode
// into that same envelope and ignore what is absent), but spec-strict clients hard-fail on the
// missing required fields: alloy's payload envelopes require blockValue, blobsBundle,
// shouldOverrideBuilder, parentBeaconBlockRoot and (V4+) executionRequests, so a kona-node
// sealing a block against the sync tester failed the fetch outright ("missing field
// `blockValue`") and could never advance a chain it had to build. The per-version wrappers below
// fill the spec shape in; the payload itself is served untouched.
type GetPayloadResponse struct {
	ExecutionPayload      *eth.ExecutionPayload `json:"executionPayload"`
	BlockValue            hexutil.Big           `json:"blockValue"`
	BlobsBundle           *BlobsBundleV1        `json:"blobsBundle,omitempty"`
	ShouldOverrideBuilder *bool                 `json:"shouldOverrideBuilder,omitempty"`
	ParentBeaconBlockRoot *common.Hash          `json:"parentBeaconBlockRoot,omitempty"`
	ExecutionRequests     *[]hexutil.Bytes      `json:"executionRequests,omitempty"`
}

// specPayloadResponse wraps a stored envelope into the spec response shape.
//
// The block value is served as zero: the sync tester relays canonical blocks rather than
// building them, so there is no fee value to report, and zero is a valid quantity for the
// required field. `withBlobs` adds the V3+ fields, `withRequests` the V4+ one (always an empty
// list on the OP Stack).
func specPayloadResponse(env *eth.ExecutionPayloadEnvelope, withBlobs, withRequests bool) *GetPayloadResponse {
	resp := &GetPayloadResponse{ExecutionPayload: env.ExecutionPayload}
	if withBlobs {
		resp.BlobsBundle = &BlobsBundleV1{
			Commitments: []hexutil.Bytes{},
			Proofs:      []hexutil.Bytes{},
			Blobs:       []hexutil.Bytes{},
		}
		shouldOverride := false
		resp.ShouldOverrideBuilder = &shouldOverride
		parentBeaconBlockRoot := common.Hash{}
		if env.ParentBeaconBlockRoot != nil {
			parentBeaconBlockRoot = *env.ParentBeaconBlockRoot
		}
		resp.ParentBeaconBlockRoot = &parentBeaconBlockRoot
	}
	if withRequests {
		requests := []hexutil.Bytes{}
		resp.ExecutionRequests = &requests
	}
	return resp
}

// GetPayloadV1 predates the envelope response shape; it keeps relaying the stored envelope,
// which is what the Go clients that would ever call V1 decode.
func (e *EngineFrontend) GetPayloadV1(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return e.b.GetPayloadV1(ctx, payloadID)
}

func (e *EngineFrontend) GetPayloadV2(ctx context.Context, payloadID eth.PayloadID) (*GetPayloadResponse, error) {
	env, err := e.b.GetPayloadV2(ctx, payloadID)
	if err != nil {
		return nil, err
	}
	return specPayloadResponse(env, false, false), nil
}

func (e *EngineFrontend) GetPayloadV3(ctx context.Context, payloadID eth.PayloadID) (*GetPayloadResponse, error) {
	env, err := e.b.GetPayloadV3(ctx, payloadID)
	if err != nil {
		return nil, err
	}
	return specPayloadResponse(env, true, false), nil
}

func (e *EngineFrontend) GetPayloadV4(ctx context.Context, payloadID eth.PayloadID) (*GetPayloadResponse, error) {
	env, err := e.b.GetPayloadV4(ctx, payloadID)
	if err != nil {
		return nil, err
	}
	return specPayloadResponse(env, true, true), nil
}

func (e *EngineFrontend) GetPayloadV5(ctx context.Context, payloadID eth.PayloadID) (*GetPayloadResponse, error) {
	env, err := e.b.GetPayloadV5(ctx, payloadID)
	if err != nil {
		return nil, err
	}
	return specPayloadResponse(env, true, true), nil
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

func (e *EngineFrontend) ExchangeCapabilities(ctx context.Context, args []string) []string {
	return e.b.ExchangeCapabilities(ctx, args)
}
