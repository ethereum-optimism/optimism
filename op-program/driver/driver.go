package driver

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-program/config"
	"github.com/ethereum/go-ethereum/log"
)

type Derivation interface {
	Step(ctx context.Context) error
	SafeL2Head() eth.L2BlockRef
}

type Driver struct {
	logger   log.Logger
	pipeline Derivation
}

func NewDriver(logger log.Logger, cfg *config.Config, l1Source derive.L1Fetcher, l2Source derive.Engine) *Driver {
	pipeline := derive.NewDerivationPipeline(logger, cfg.Rollup, l1Source, l2Source, metrics.NoopMetrics)
	pipeline.Reset()
	return &Driver{
		logger:   logger,
		pipeline: pipeline,
	}
}

// Step runs the next step of the derivation pipeline.
// Returns nil if there are further steps to be performed
// Returns io.EOF if the derivation completed successfully
// Returns a non-EOF error if the derivation failed
func (d *Driver) Step(ctx context.Context) error {
	if err := d.pipeline.Step(ctx); errors.Is(err, io.EOF) {
		return io.EOF
	} else if errors.Is(err, derive.NotEnoughData) {
		d.logger.Debug("Data is lacking")
		return nil
	} else if err != nil {
		return fmt.Errorf("pipeline err: %w", err)
	}
	return nil
}

func (d *Driver) SafeHead() eth.L2BlockRef {
	return d.pipeline.SafeL2Head()
}
