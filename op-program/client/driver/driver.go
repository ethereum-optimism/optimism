package driver

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrClaimNotValid = errors.New("invalid claim")
)

type Derivation interface {
	Step(ctx context.Context) error
	SafeL2Head() eth.L2BlockRef
}

type L2Source interface {
	derive.Engine
	L2OutputRoot(uint64) (eth.Bytes32, error)
}

type Driver struct {
	logger         log.Logger
	pipeline       Derivation
	l2OutputRoot   func(uint64) (eth.Bytes32, error)
	targetBlockNum uint64
}

func NewDriver(logger log.Logger, cfg *rollup.Config, l1Source derive.L1Fetcher, l2Source L2Source, targetBlockNum uint64) *Driver {
	pipeline := derive.NewDerivationPipeline(logger, cfg, l1Source, l2Source, metrics.NoopMetrics, &sync.Config{})
	pipeline.Reset()
	return &Driver{
		logger:         logger,
		pipeline:       pipeline,
		l2OutputRoot:   l2Source.L2OutputRoot,
		targetBlockNum: targetBlockNum,
	}
}

// Step runs the next step of the derivation pipeline.
// Returns nil if there are further steps to be performed
// Returns io.EOF if the derivation completed successfully
// Returns a non-EOF error if the derivation failed
func (d *Driver) Step(ctx context.Context) error {
	if err := d.pipeline.Step(ctx); errors.Is(err, io.EOF) {
		d.logger.Info("Derivation complete: reached L1 head", "head", d.pipeline.SafeL2Head())
		return io.EOF
	} else if errors.Is(err, derive.NotEnoughData) {
		head := d.pipeline.SafeL2Head()
		if head.Number >= d.targetBlockNum {
			d.logger.Info("Derivation complete: reached L2 block", "head", head)
			return io.EOF
		}
		d.logger.Debug("Data is lacking")
		return nil
	} else if errors.Is(err, derive.ErrTemporary) {
		// While most temporary errors are due to requests for external data failing which can't happen,
		// they may also be returned due to other events like channels timing out so need to be handled
		d.logger.Warn("Temporary error in derivation", "err", err)
		return nil
	} else if err != nil {
		return fmt.Errorf("pipeline err: %w", err)
	}
	return nil
}

func (d *Driver) SafeHead() eth.L2BlockRef {
	return d.pipeline.SafeL2Head()
}

func (d *Driver) ValidateClaim(l2ClaimBlockNum uint64, claimedOutputRoot eth.Bytes32) error {
	outputRoot, err := d.l2OutputRoot(l2ClaimBlockNum)
	if err != nil {
		return fmt.Errorf("calculate L2 output root: %w", err)
	}
	d.logger.Info("Validating claim", "head", d.SafeHead(), "output", outputRoot, "claim", claimedOutputRoot)
	if claimedOutputRoot != outputRoot {
		return fmt.Errorf("%w: claim: %v actual: %v", ErrClaimNotValid, claimedOutputRoot, outputRoot)
	}
	return nil
}
