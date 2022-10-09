package l2

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/services/util"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	DefaultConnectionTimeout        = 20 * time.Second
	DefaultConfDepth         uint64 = 20
	DefaultMaxBatchSize             = 50
)

type HeaderSelectorConfig struct {
	ConfDepth    uint64
	MaxBatchSize uint64
}

type ConfirmedHeaderSelector struct {
	cfg HeaderSelectorConfig
}

func HeadersByRange(ctx context.Context, client *rpc.Client, startHeight uint64, count int) ([]*types.Header, error) {
	height := startHeight
	batchElems := make([]rpc.BatchElem, count)
	for i := 0; i < count; i++ {
		batchElems[i] = rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args: []interface{}{
				util.ToBlockNumArg(new(big.Int).SetUint64(height + uint64(i))),
				false,
			},
			Result: new(types.Header),
			Error:  nil,
		}
	}

	if err := client.BatchCallContext(ctx, batchElems); err != nil {
		return nil, err
	}

	out := make([]*types.Header, count)
	for i := 0; i < len(batchElems); i++ {
		if batchElems[i].Error != nil {
			return nil, batchElems[i].Error
		}
		out[i] = batchElems[i].Result.(*types.Header)
	}

	return out, nil
}

func (f *ConfirmedHeaderSelector) NewHead(
	ctx context.Context,
	lowest uint64,
	header *types.Header,
	client *rpc.Client,
) ([]*types.Header, error) {

	number := header.Number.Uint64()
	blockHash := header.Hash()

	logger.Info("New block", "block", number, "hash", blockHash)

	if number < f.cfg.ConfDepth {
		return nil, nil
	}
	endHeight := number - f.cfg.ConfDepth + 1

	minNextHeight := lowest + f.cfg.ConfDepth
	if minNextHeight > number {
		log.Info("Fork block=%d hash=%s", number, blockHash)
		return nil, nil
	}
	startHeight := lowest + 1

	// Clamp to max batch size
	if startHeight+f.cfg.MaxBatchSize < endHeight+1 {
		endHeight = startHeight + f.cfg.MaxBatchSize - 1
	}

	nHeaders := int(endHeight - startHeight + 1)
	if nHeaders > 1 {
		logger.Info("Loading blocks",
			"startHeight", startHeight, "endHeight", endHeight)
	}

	headers := make([]*types.Header, 0)
	height := startHeight
	left := nHeaders - len(headers)
	for left > 0 {
		count := DefaultMaxBatchSize
		if count > left {
			count = left
		}

		logger.Info("Loading block batch",
			"height", height, "count", count)

		ctxt, cancel := context.WithTimeout(ctx, DefaultConnectionTimeout)
		fetched, err := HeadersByRange(ctxt, client, height, count)
		cancel()
		if err != nil {
			return nil, err
		}

		headers = append(headers, fetched...)
		left = nHeaders - len(headers)
		height += uint64(count)
	}

	logger.Debug("Verifying block range ",
		"startHeight", startHeight, "endHeight", endHeight)

	for i, header := range headers {
		// Trim the returned headers if any of the lookups failed.
		if header == nil {
			headers = headers[:i]
			break
		}

		// Assert that each header builds on the parent before it, trim if there
		// are any discontinuities.
		if i > 0 {
			prevHeader := headers[i-1]
			if prevHeader.Hash() != header.ParentHash {
				log.Error("Parent hash does not connect to ",
					"block", header.Number.Uint64(), "hash", header.Hash(),
					"prev", prevHeader.Number.Uint64(), "hash", prevHeader.Hash())
				headers = headers[:i]
				break
			}
		}

		log.Debug("Confirmed block ",
			"block", header.Number.Uint64(), "hash", header.Hash())
	}

	return headers, nil
}

func NewConfirmedHeaderSelector(cfg HeaderSelectorConfig) (*ConfirmedHeaderSelector, error) {
	if cfg.ConfDepth == 0 {
		return nil, errors.New("ConfDepth must be greater than zero")
	}
	if cfg.MaxBatchSize == 0 {
		return nil, errors.New("MaxBatchSize must be greater than zero")
	}

	return &ConfirmedHeaderSelector{
		cfg: cfg,
	}, nil
}
