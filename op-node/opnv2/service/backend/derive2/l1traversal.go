package derive2

import (
	"context"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L1Traversal implements the legacy L1 traversal derivation stage interface,
// while cleaning up state.
// It allows for the next L1 origin to be prepared at any point, so it can be prefetched later.
type L1Traversal struct {
	logger log.Logger
	cfg    *rollup.Config

	l1Fetcher derive.L1ReceiptsFetcher

	// The L1 origin that the pipeline is already consuming.
	// This is installed in every stage during reset.
	// The data-retrieval stage will not ask for this anymore.
	source eth.L1BlockRef
	sysCfg eth.SystemConfig

	// The next L1 origin after source. Not yet consumed.
	// This is moved into source once the next block is requested.
	// Or, if zero, the pipeline will temporarily stall, until the next L1 origin is made available.
	next eth.L1BlockRef
}

var _ derive.NextBlockProvider = (*L1Traversal)(nil)
var _ derive.ResettableStage = (*L1Traversal)(nil)

func NewL1Traversal(logger log.Logger, cfg *rollup.Config, l1Fetcher derive.L1ReceiptsFetcher) *L1Traversal {
	return &L1Traversal{logger: logger, cfg: cfg, l1Fetcher: l1Fetcher}
}

func (l *L1Traversal) Reset(ctx context.Context, base eth.L1BlockRef, baseCfg eth.SystemConfig) error {
	l.source = base
	l.sysCfg = baseCfg
	l.next = eth.L1BlockRef{}
	return nil
}

func (l *L1Traversal) NextL1Block(ctx context.Context) (eth.L1BlockRef, error) {
	if l.next == (eth.L1BlockRef{}) {
		// Next L1 block not found yet
		return eth.L1BlockRef{}, io.EOF
	}
	// Parse L1 receipts of the given block and update the L1 system configuration
	_, receipts, err := l.l1Fetcher.FetchReceipts(ctx, l.next.Hash)
	if err != nil {
		return eth.L1BlockRef{}, derive.NewTemporaryError(fmt.Errorf(
			"failed to fetch receipts of L1 block %s (parent: %s) for L1 sysCfg update: %w",
			l.next, l.next.ParentID(), err))
	}
	sysCfg := l.sysCfg
	if err := derive.UpdateSystemConfigWithL1Receipts(&sysCfg, receipts, l.cfg, l.next.Time); err != nil {
		// the sysCfg changes should always be formatted correctly.
		return eth.L1BlockRef{}, derive.NewCriticalError(fmt.Errorf(
			"failed to update L1 sysCfg with receipts from block %s: %w", l.next, err))
	}
	l.source = l.next
	l.next = eth.L1BlockRef{}
	return l.source, nil
}

func (l *L1Traversal) Origin() eth.L1BlockRef {
	return l.source
}

func (l *L1Traversal) SystemConfig() eth.SystemConfig {
	return l.sysCfg
}

func (l *L1Traversal) ProvideNext(l1Ref eth.L1BlockRef) {
	if l.next != (eth.L1BlockRef{}) {
		l.logger.Warn("Already have the next L1 origin", "have", l.next, "provided", l1Ref)
		return
	}
	// Check parent hash.
	// If it does not match, then the next derivation attempt can request the right thing to get instead.
	if l.source.Hash != l.next.ParentHash {
		l.logger.Warn("Unexpected next L1 block",
			"current", l.source, "provided", l1Ref, "expected", l1Ref.ParentID())
		return
	}
	l.next = l1Ref
}

// TraversalState identifies if the current and next block (could be zero if not prepared yet).
// The caller should check if a reorg is needed to be able to provide a canonical next block.
func (l *L1Traversal) TraversalState() (current, next eth.L1BlockRef) {
	return l.source, l.next
}
