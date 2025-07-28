package driver

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/common"
)

var _ L2Chain = (*Multiplexer)(nil)

type Multiplexer struct {
	l2Sources []*sources.EngineClient
}

func NewMultiplexer(l2Sources []*sources.EngineClient) *Multiplexer {
	return &Multiplexer{
		l2Sources: l2Sources,
	}
}

func (m *Multiplexer) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	return m.l2Sources[0].L2BlockRefByLabel(ctx, label)
}

func (m *Multiplexer) L2BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L2BlockRef, error) {
	return m.l2Sources[0].L2BlockRefByHash(ctx, hash)
}

func (m *Multiplexer) L2BlockRefByNumber(ctx context.Context, number uint64) (eth.L2BlockRef, error) {
	return m.l2Sources[0].L2BlockRefByNumber(ctx, number)
}

func (m *Multiplexer) GetPayload(ctx context.Context, payloadInfo eth.PayloadInfo) (*eth.ExecutionPayloadEnvelope, error) {
	return m.l2Sources[0].GetPayload(ctx, payloadInfo)
}

func (m *Multiplexer) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return m.l2Sources[0].ForkchoiceUpdate(ctx, state, attr)
}

func (m *Multiplexer) NewPayload(ctx context.Context, payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash) (*eth.PayloadStatusV1, error) {
	return m.l2Sources[0].NewPayload(ctx, payload, parentBeaconBlockRoot)
}

func (m *Multiplexer) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error) {
	return m.l2Sources[0].PayloadByHash(ctx, hash)
}

func (m *Multiplexer) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	return m.l2Sources[0].PayloadByNumber(ctx, number)
}

func (m *Multiplexer) SystemConfigByL2Hash(ctx context.Context, hash common.Hash) (eth.SystemConfig, error) {
	return m.l2Sources[0].SystemConfigByL2Hash(ctx, hash)
}
