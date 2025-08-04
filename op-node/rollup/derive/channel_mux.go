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
	log  log.Logger
	spec *rollup.ChainSpec
	prev NextFrameProvider
	m    Metrics

	// embedded active stage
	RawChannelProvider
}

var _ RawChannelProvider = (*ChannelMux)(nil)

// NewChannelMux returns a ChannelMux with the ChannelBank as activated stage. Reset has to be called before
// calling other methods, to activate the right stage for a given L1 origin.
func NewChannelMux(log log.Logger, spec *rollup.ChainSpec, prev NextFrameProvider, m Metrics) *ChannelMux {
	return &ChannelMux{
		log:  log,
		spec: spec,
		prev: prev,
		m:    m,
	}
}

func (c *ChannelMux) Reset(ctx context.Context, base eth.L1BlockRef, sysCfg eth.SystemConfig) error {
	// TODO(12490): change to a switch over c.cfg.ActiveFork(base.Time)
	switch {
	case c.spec.IsJovian(base.Time):
		if _, ok := c.RawChannelProvider.(*ChannelAssembler); !ok {
			c.log.Info("ChannelMux: activating Jovian stage during reset", "origin", base)
			c.RawChannelProvider = NewChannelAssembler(c.log, c.spec, c.prev, c.m)
		}
	case c.spec.IsHolocene(base.Time):
		if _, ok := c.RawChannelProvider.(*ChannelAssembler); !ok {
			c.log.Info("ChannelMux: activating Holocene stage during reset", "origin", base)
			c.RawChannelProvider = NewChannelAssembler(c.log, c.spec, c.prev, c.m)
		}
	default:
		if _, ok := c.RawChannelProvider.(*ChannelBank); !ok {
			c.log.Info("ChannelMux: activating pre-Holocene stage during reset", "origin", base)
			c.RawChannelProvider = NewChannelBank(c.log, c.spec, c.prev, c.m)
		}
	}
	return c.RawChannelProvider.Reset(ctx, base, sysCfg)
}

func (c *ChannelMux) Transform(f rollup.ForkName) {
	switch f {
	case rollup.Holocene:
		c.TransformHolocene()
	case rollup.Jovian:
		c.TransformJovian()
	}
}

func (c *ChannelMux) TransformHolocene() {
	switch cp := c.RawChannelProvider.(type) {
	case *ChannelBank:
		c.log.Info("ChannelMux: transforming to Holocene stage")
		c.RawChannelProvider = NewChannelAssembler(c.log, c.spec, c.prev, c.m)
	case *ChannelAssembler:
		// Even if the pipeline is Reset to the activation block, the previous origin will be the
		// same, so transformStages isn't called.
		panic(fmt.Sprintf("Holocene ChannelAssembler already active, old origin: %v", cp.Origin()))
	default:
		panic(fmt.Sprintf("unknown channel stage type: %T", cp))
	}
}

func (c *ChannelMux) TransformJovian() {
	switch cp := c.RawChannelProvider.(type) {
	case *ChannelBank:
		c.log.Info("ChannelMux: transforming to Jovian stage")
		c.RawChannelProvider = NewChannelAssembler(c.log, c.spec, c.prev, c.m)
	case *ChannelAssembler:
		// Jovian ChannelAssembler already active - this can happen during reorgs or multiple pipeline steps
		c.log.Debug("ChannelMux: Jovian ChannelAssembler already active", "origin", cp.Origin())
	default:
		panic(fmt.Sprintf("unknown channel stage type: %T", cp))
	}
}
