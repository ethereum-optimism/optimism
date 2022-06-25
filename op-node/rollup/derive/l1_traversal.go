package derive

import (
	"context"
	"errors"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
)

type L1BlockRefByNumberFetcher interface {
	L1BlockRefByNumber(context.Context, uint64) (eth.L1BlockRef, error)
}

type L1Traversal struct {
	log      log.Logger
	l1Blocks L1BlockRefByNumberFetcher
	next     OriginStage
}

var _ Stage = (*L1Traversal)(nil)

func NewL1Traversal(log log.Logger, l1Blocks L1BlockRefByNumberFetcher, next OriginStage) *L1Traversal {
	return &L1Traversal{
		log:      log,
		l1Blocks: l1Blocks,
		next:     next,
	}
}

func (l1s *L1Traversal) CurrentOrigin() eth.L1BlockRef {
	return l1s.next.CurrentOrigin()
}

func (l1s *L1Traversal) Step(ctx context.Context) error {
	// close previous data if we need to (when this stage is hit it always means we
	if l1s.next.IsOriginOpen() {
		l1s.next.CloseOrigin()
		l1s.log.Warn("closing next origin")
		return nil
	}

	origin := l1s.CurrentOrigin()
	nextL1Origin, err := l1s.l1Blocks.L1BlockRefByNumber(ctx, origin.Number+1)
	if errors.Is(err, ethereum.NotFound) {
		l1s.log.Warn("can't find next L1 block info (yet)", "number", origin.Number+1, "origin", origin)
		return io.EOF
	} else if err != nil {
		l1s.log.Warn("failed to find L1 block info by number", "number", origin.Number+1, "origin", origin, "err", err)
		return err
	}

	return l1s.next.OpenOrigin(nextL1Origin)
}

func (l1s *L1Traversal) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	l1s.log.Info("completed reset of derivation pipeline", "origin", l1s.next.CurrentOrigin())
	return io.EOF
}
