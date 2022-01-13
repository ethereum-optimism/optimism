package indexer

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	DefaultConnectionTimeout        = 20 * time.Second
	DefaultConfDepth         uint64 = 20
	DefaultMaxBatchSize      uint64 = 100
)

type HeaderSelector interface {
	NewHead(context.Context, uint64, *types.Header, HeaderBackend) []*types.Header
}

type HeaderBackend interface {
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
}

type HeaderSelectorConfig struct {
	ConfDepth    uint64
	MaxBatchSize uint64
}

type ConfirmedHeaderSelector struct {
	cfg HeaderSelectorConfig
}

func (f *ConfirmedHeaderSelector) NewHead(
	ctx context.Context,
	lowest uint64,
	header *types.Header,
	backend HeaderBackend,
) []*types.Header {

	number := header.Number.Uint64()
	blockHash := header.Hash()

	log.Info("New block", "block", number, "hash", blockHash)

	if number < f.cfg.ConfDepth {
		return nil
	}
	endHeight := number - f.cfg.ConfDepth + 1

	minNextHeight := lowest + f.cfg.ConfDepth
	if minNextHeight > number {
		log.Info("Fork block=%d hash=%s", number, blockHash)
		return nil
	}
	startHeight := lowest + 1

	// Clamp to max batch size
	if startHeight+f.cfg.MaxBatchSize < endHeight+1 {
		endHeight = startHeight + f.cfg.MaxBatchSize - 1
	}

	nHeaders := endHeight - startHeight + 1
	if nHeaders > 1 {
		log.Info("Loading block batch ",
			"startHeight", startHeight, "endHeight", endHeight)
	}

	headers := make([]*types.Header, nHeaders)
	var wg sync.WaitGroup
	for i := uint64(0); i < nHeaders; i++ {
		wg.Add(1)
		go func(ii uint64) {
			defer wg.Done()

			ctxt, cancel := context.WithTimeout(ctx, DefaultConnectionTimeout)
			defer cancel()

			height := startHeight + ii
			bigHeight := new(big.Int).SetUint64(height)
			header, err := backend.HeaderByNumber(ctxt, bigHeight)
			if err != nil {
				log.Error("Unable to load block ", "block", height, "err", err)
				return
			}

			headers[ii] = header
		}(i)
	}
	wg.Wait()

	log.Info("Verifying block range ",
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
					"block", header.Number.Uint64, "hash", header.Hash(),
					"prev", prevHeader.Number.Uint64(), "hash", prevHeader.Hash())
				headers = headers[:i]
				break
			}
		}

		log.Info("Confirmed block ",
			"block", header.Number.Uint64(), "hash", header.Hash())
	}

	return headers
}

func NewConfirmedHeaderSelector(cfg HeaderSelectorConfig) (*ConfirmedHeaderSelector,
	error) {
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
