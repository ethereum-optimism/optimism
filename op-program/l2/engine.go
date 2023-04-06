package l2

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-program/l2/engineapi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrNotFound = errors.New("not found")
)

type OracleEngine struct {
	api       *engineapi.L2EngineAPI
	backend   engineapi.EngineBackend
	rollupCfg *rollup.Config
}

func NewOracleEngine(rollupCfg *rollup.Config, logger log.Logger, backend engineapi.EngineBackend) *OracleEngine {
	engineAPI := engineapi.NewL2EngineAPI(logger, backend)
	return &OracleEngine{
		api:       engineAPI,
		backend:   backend,
		rollupCfg: rollupCfg,
	}
}

func (o OracleEngine) GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error) {
	return o.api.GetPayloadV1(ctx, payloadId)
}

func (o OracleEngine) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return o.api.ForkchoiceUpdatedV1(ctx, state, attr)
}

func (o OracleEngine) NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	return o.api.NewPayloadV1(ctx, payload)
}

func (o OracleEngine) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayload, error) {
	block := o.backend.GetBlockByHash(hash)
	if block == nil {
		return nil, ErrNotFound
	}
	return eth.BlockAsPayload(block)
}

func (o OracleEngine) PayloadByNumber(ctx context.Context, n uint64) (*eth.ExecutionPayload, error) {
	hash := o.backend.GetCanonicalHash(n)
	if hash == (common.Hash{}) {
		return nil, ErrNotFound
	}
	return o.PayloadByHash(ctx, hash)
}

func (o OracleEngine) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	var header *types.Header
	switch label {
	case eth.Unsafe:
		header = o.backend.CurrentHeader()
	case eth.Safe:
		header = o.backend.CurrentSafeBlock()
	case eth.Finalized:
		header = o.backend.CurrentFinalBlock()
	default:
		return eth.L2BlockRef{}, fmt.Errorf("unknown label: %v", label)
	}
	if header == nil {
		return eth.L2BlockRef{}, ErrNotFound
	}
	block := o.backend.GetBlockByHash(header.Hash())
	if block == nil {
		return eth.L2BlockRef{}, ErrNotFound
	}
	return derive.L2BlockToBlockRef(block, &o.rollupCfg.Genesis)
}

func (o OracleEngine) L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error) {
	block := o.backend.GetBlockByHash(l2Hash)
	if block == nil {
		return eth.L2BlockRef{}, ErrNotFound
	}
	return derive.L2BlockToBlockRef(block, &o.rollupCfg.Genesis)
}

func (o OracleEngine) SystemConfigByL2Hash(ctx context.Context, hash common.Hash) (eth.SystemConfig, error) {
	payload, err := o.PayloadByHash(ctx, hash)
	if err != nil {
		return eth.SystemConfig{}, err
	}
	return derive.PayloadToSystemConfig(payload, o.rollupCfg)
}
