package indexer

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

const (
	DefaultConnectionTimeout        = 5 * time.Second
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

	fmt.Printf("🎁 New block=%d hash=%s\n", number, blockHash)

	if number < f.cfg.ConfDepth {
		return nil
	}
	endHeight := number - f.cfg.ConfDepth + 1

	minNextHeight := lowest + f.cfg.ConfDepth
	if minNextHeight > number {
		fmt.Printf("    🌵 Fork block=%d hash=%s\n", number, blockHash)
		return nil
	}
	startHeight := lowest + 1

	// Clamp to max batch size
	if startHeight+f.cfg.MaxBatchSize < endHeight+1 {
		endHeight = startHeight + f.cfg.MaxBatchSize - 1
	}

	nHeaders := endHeight - startHeight + 1
	if nHeaders > 1 {
		fmt.Printf("    🏗  Loading block batch start=%d end=%d\n",
			startHeight, endHeight)
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
				fmt.Printf("    ❌ Unable to load block=%d err=%v\n",
					height, err)
				return
			}

			headers[ii] = header
		}(i)
	}
	wg.Wait()

	fmt.Printf("    🔍 Verifying block range start=%d end=%d\n",
		startHeight, endHeight)

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
				fmt.Printf("    ⚠️  Parent hash of block=%d hash=%s does not "+
					"connect to block=%d hash=%s", header.Number.Uint64(),
					header.Hash(), prevHeader.Number.Uint64(), prevHeader.Hash())
				headers = headers[:i]
				break
			}
		}

		fmt.Printf("    ✅ Confirmed block=%d hash=%s\n",
			header.Number.Uint64(), header.Hash())
	}

	return headers
}

func NewConfirmedHeaderSelector(cfg HeaderSelectorConfig) *ConfirmedHeaderSelector {
	if cfg.ConfDepth == 0 {
		panic("ConfDepth must be greater than zero")
	}
	if cfg.MaxBatchSize == 0 {
		panic("MaxBatchSize must be greater than zero")
	}

	return &ConfirmedHeaderSelector{
		cfg: cfg,
	}
}
