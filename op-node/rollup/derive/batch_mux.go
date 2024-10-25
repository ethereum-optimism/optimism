package derive

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// BatchMux multiplexes between different batch stages.
// Stages are swapped on demand during Reset calls, or explicitly with Transform.
// It currently chooses the BatchQueue pre-Holocene and the BatchStage post-Holocene.
type BatchMux struct {
	log     log.Logger
	cfg     *rollup.Config
	initial *BatchQueue

	// embedded active stage
	SingularBatchProvider
}

var _ SingularBatchProvider = (*BatchMux)(nil)

// NewBatchMux returns a BatchMux with the BatchQueue as activated stage. Reset has to be called before
// calling other methods, to activate the right stage for a given L1 origin.
func NewBatchMux(lgr log.Logger, cfg *rollup.Config, prev NextBatchProvider, l2 SafeBlockFetcher) *BatchMux {
	bq := NewBatchQueue(lgr, cfg, prev, l2)
	return &BatchMux{log: lgr, cfg: cfg, initial: bq, SingularBatchProvider: bq}
}

func (b *BatchMux) Reset(ctx context.Context, base eth.L1BlockRef, sysCfg eth.SystemConfig) error {
	isHolocene := b.cfg.IsHolocene(base.Time)
	switch bp := b.SingularBatchProvider.(type) {
	case *BatchQueue:
		if isHolocene { // this case can happen at startup
			b.log.Info("BatchMux: transforming to Holocene stage during reset", "origin", base)
			b.SingularBatchProvider = bp.TransformHolocene()
		}
	case *BatchStage:
		if !isHolocene {
			b.log.Info("BatchMux: reverting to pre-Holocene stage during reset", "origin", base)
			b.SingularBatchProvider = b.initial
		}
	default:
		panic(fmt.Sprintf("unknown batch stage type: %T", bp))
	}
	return b.SingularBatchProvider.Reset(ctx, base, sysCfg)
}

func (b *BatchMux) Transform(f rollup.ForkName) {
	switch f {
	case rollup.Holocene:
		b.TransformHolocene()
	}
}

func (b *BatchMux) TransformHolocene() {
	b.log.Info("BatchMux: transforming to Holocene")
	switch bp := b.SingularBatchProvider.(type) {
	case *BatchQueue:
		b.SingularBatchProvider = bp.TransformHolocene()
		// TODO(12490): do we want to reset the the BatchQueue at this point, to garbage collect
		// left-over state? Note that we still need to retain the *BatchQueue itself in case of a future
		// reset to a pre-Holocene L1 block
	case *BatchStage:
		// Even if the pipeline is Reset to the activation block, the previous origin will be the
		// same, so transfromStages isn't called.
		panic(fmt.Sprintf("Holocene BatchStage already active, old origin: %v", bp.Origin()))
	default:
		panic(fmt.Sprintf("unknown batch stage type: %T", bp))
	}
}
