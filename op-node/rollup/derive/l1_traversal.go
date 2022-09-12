package derive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
)

type L1BlockRefByNumberFetcher interface {
	L1BlockRefByNumber(context.Context, uint64) (eth.L1BlockRef, error)
}

// L1Traversal controls advancing the L1 block into the pipeline.
// It is advance by an external entity & exposes it's internal state
// through the `NextL1Block` function.
//
// After creation, L1Traversal must be reset. Then after any reset or advance,
// `NextL1Block` must be called prior to the next advance.
type L1Traversal struct {
	log     log.Logger
	fetcher L1BlockRefByNumberFetcher

	done  bool
	block eth.L1BlockRef
}

var _ PullStage = (*L1Traversal)(nil)

func NewL1Traversal(log log.Logger, l1Blocks L1BlockRefByNumberFetcher) *L1Traversal {
	return &L1Traversal{
		log:     log,
		fetcher: l1Blocks,
		done:    false,
	}
}

// Advance forces the L1 Traversal to attempt to advance to the next L1 block.
// It retuns one of four sentinel errors:
// - nil upon success
// - io.EOF upon an ethereum.NotFound error when fetching the next block
// - Temporary error upon other errors fetching from L1
// - Reset Error upon detecting an L1 reorg
func (l1t *L1Traversal) Advance(ctx context.Context) error {
	if !l1t.done {
		panic("dev error: advancing L1 Traversal without having consumed previous block")
	}
	origin := l1t.block
	nextL1Origin, err := l1t.fetcher.L1BlockRefByNumber(ctx, origin.Number+1)
	if errors.Is(err, ethereum.NotFound) {
		l1t.log.Debug("can't find next L1 block info (yet)", "number", origin.Number+1, "origin", origin)
		return io.EOF
	} else if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to find L1 block info by number, at origin %s next %d: %w", origin, origin.Number+1, err))
	}
	if origin.Hash != nextL1Origin.ParentHash {
		return NewResetError(fmt.Errorf("detected L1 reorg from %s to %s with conflicting parent %s", origin, nextL1Origin, nextL1Origin.ParentID()))
	}
	return nil
}

// NextL1Block returns the next L1 block the first time this function is called
// and then io.EOF after successive calls. Calling `Advance` resets the internal
// state such that this function may be able to retun the next block.
func (l1t *L1Traversal) NextL1Block(ctx context.Context) (eth.L1BlockRef, error) {
	if l1t.done {
		return eth.L1BlockRef{}, io.EOF
	} else {
		fmt.Println("called nextL1block")
		l1t.done = true
		return l1t.block, nil
	}
}

func (l1t *L1Traversal) Reset(ctx context.Context, base eth.L1BlockRef) (eth.L1BlockRef, error) {
	l1t.block = base
	l1t.done = false
	l1t.log.Info("completed reset of derivation pipeline", "origin", base)
	return base, io.EOF
}
