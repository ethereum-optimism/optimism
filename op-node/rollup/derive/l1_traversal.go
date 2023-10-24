package derive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L1 Traversal fetches the next L1 block and exposes it through the progress API

type L1BlockRefByNumberFetcher interface {
	L1BlockRefByNumber(context.Context, uint64) (eth.L1BlockRef, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
}

type L1Traversal struct {
	block    eth.L1BlockRef
	done     bool
	l1Blocks L1BlockRefByNumberFetcher
	log      log.Logger
	sysCfg   eth.SystemConfig
	cfg      *rollup.Config
}

var _ ResettableStage = (*L1Traversal)(nil)

func NewL1Traversal(log log.Logger, cfg *rollup.Config, l1Blocks L1BlockRefByNumberFetcher) *L1Traversal {
	return &L1Traversal{
		log:      log,
		l1Blocks: l1Blocks,
		cfg:      cfg,
	}
}

func (l1t *L1Traversal) Origin() eth.L1BlockRef {
	return l1t.block
}

// NextL1Block returns the next block. It does not advance, but it can only be
// called once before returning io.EOF
func (l1t *L1Traversal) NextL1Block(_ context.Context) (eth.L1BlockRef, error) {
	if !l1t.done {
		l1t.done = true
		return l1t.block, nil
	} else {
		return eth.L1BlockRef{}, io.EOF
	}
}

// AdvanceL1Block advances the internal state of L1 Traversal
func (l1t *L1Traversal) AdvanceL1Block(ctx context.Context) error {
	origin := l1t.block
	nextL1Origin, err := l1t.l1Blocks.L1BlockRefByNumber(ctx, origin.Number+1)
	if errors.Is(err, ethereum.NotFound) {
		l1t.log.Debug("can't find next L1 block info (yet)", "number", origin.Number+1, "origin", origin)
		return io.EOF
	} else if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to find L1 block info by number, at origin %s next %d: %w", origin, origin.Number+1, err))
	}
	if l1t.block.Hash != nextL1Origin.ParentHash {
		return NewResetError(fmt.Errorf("detected L1 reorg from %s to %s with conflicting parent %s", l1t.block, nextL1Origin, nextL1Origin.ParentID()))
	}

	// Parse L1 receipts of the given block and update the L1 system configuration
	_, receipts, err := l1t.l1Blocks.FetchReceipts(ctx, nextL1Origin.Hash)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to fetch receipts of L1 block %s (parent: %s) for L1 sysCfg update: %w", nextL1Origin, origin, err))
	}
	if err := UpdateSystemConfigWithL1Receipts(&l1t.sysCfg, receipts, l1t.cfg); err != nil {
		// the sysCfg changes should always be formatted correctly.
		return NewCriticalError(fmt.Errorf("failed to update L1 sysCfg with receipts from block %s: %w", origin, err))
	}

	l1t.block = nextL1Origin
	l1t.done = false
	return nil
}

// Reset sets the internal L1 block to the supplied base.
func (l1t *L1Traversal) Reset(ctx context.Context, base eth.L1BlockRef, cfg eth.SystemConfig) error {
	l1t.block = base
	l1t.done = false
	l1t.sysCfg = cfg
	l1t.log.Info("completed reset of derivation pipeline", "origin", base)
	return io.EOF
}

func (l1c *L1Traversal) SystemConfig() eth.SystemConfig {
	return l1c.sysCfg
}
