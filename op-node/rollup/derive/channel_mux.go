package derive

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// ChannelMux multiplexes between different channel stages.
// Stages are swapped on demand during Reset calls, or explicitly with Transform.
// It currently chooses the ChannelBank pre-Holocene and the ChannelAssembler post-Holocene.
type ChannelMux struct {
	log     log.Logger
	spec    *rollup.ChainSpec
	initial *ChannelBank

	// embedded active stage
	RawChannelProvider
}

var _ RawChannelProvider = (*ChannelMux)(nil)

// NewChannelMux returns a ChannelMux with the ChannelBank as activated stage. Reset has to be called before
// calling other methods, to activate the right stage for a given L1 origin.
func NewChannelMux(log log.Logger, spec *rollup.ChainSpec, prev NextFrameProvider, m Metrics) *ChannelMux {
	initial := NewChannelBank(log, spec, prev, m)
	return &ChannelMux{
		log:                log,
		spec:               spec,
		initial:            initial,
		RawChannelProvider: initial,
	}
}

func (c *ChannelMux) Reset(ctx context.Context, base eth.L1BlockRef, sysCfg eth.SystemConfig) error {
	isHolocene := c.spec.IsHolocene(base.Time)
	switch cp := c.RawChannelProvider.(type) {
	case *ChannelBank:
		if isHolocene { // this case can happen at startup
			c.log.Info("ChannelMux: transforming to Holocene stage during reset", "origin", base)
			c.RawChannelProvider = cp.TransformHolocene()
		}
	case *ChannelAssembler:
		if !isHolocene {
			c.log.Info("ChannelMux: reverting to pre-Holocene stage during reset", "origin", base)
			c.RawChannelProvider = c.initial
		}
	default:
		panic(fmt.Sprintf("unknown channel stage type: %T", cp))
	}
	return c.RawChannelProvider.Reset(ctx, base, sysCfg)
}

func (c *ChannelMux) Transform(f rollup.ForkName) {
	switch f {
	case rollup.Holocene:
		c.TransformHolocene()
	}
}

func (c *ChannelMux) TransformHolocene() {
	switch cp := c.RawChannelProvider.(type) {
	case *ChannelBank:
		c.log.Info("ChannelMux: transforming to Holocene stage")
		c.RawChannelProvider = cp.TransformHolocene()
		// TODO(12490): do we want to reset the the ChannelBank at this point, to garbage collect
		// left-over state? Note that we still need to retain the *ChannelBank itself in case of a future
		// reset to a pre-Holocene L1 block
	case *ChannelAssembler:
		// Even if the pipeline is Reset to the activation block, the previous origin will be the
		// same, so transfromStages isn't called.
		panic(fmt.Sprintf("Holocene ChannelAssembler already active, old origin: %v", cp.Origin()))
	default:
		panic(fmt.Sprintf("unknown channel stage type: %T", cp))
	}
}
