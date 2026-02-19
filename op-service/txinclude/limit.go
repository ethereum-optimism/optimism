package txinclude

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
)

type Limit struct {
	inner Includer
	sema  chan struct{}
}

func NewLimit(inner Includer, limit int) *Limit {
	return &Limit{
		inner: inner,
		sema:  make(chan struct{}, limit),
	}
}

func (l *Limit) Include(ctx context.Context, tx types.TxData) (*IncludedTx, error) {
	select {
	case l.sema <- struct{}{}:
		defer func() {
			<-l.sema
		}()
		return l.inner.Include(ctx, tx)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

var _ Includer = (*Limit)(nil)
