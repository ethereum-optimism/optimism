package driver

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

var errTooManyEvents = errors.New("way too many events queued up, something is wrong")

type EndCondition interface {
	Closing() bool
	Result() (eth.L2BlockRef, error)
}

type Driver struct {
	logger log.Logger

	events []event.Event

	end     EndCondition
	deriver event.Deriver
}

func NewDriver(logger log.Logger, cfg *rollup.Config, depSet derive.DependencySet, l1Source derive.L1Fetcher,
	l1BlobsSource derive.L1BlobsFetcher, l2Source engine.Engine, targetBlockNum uint64) *Driver {

	d := &Driver{
		logger: logger,
	}

	pipeline := derive.NewDerivationPipeline(logger, cfg, depSet, l1Source, l1BlobsSource, altda.Disabled, l2Source, metrics.NoopMetrics, false)
	pipelineDeriver := derive.NewPipelineDeriver(context.Background(), pipeline)
	pipelineDeriver.AttachEmitter(d)

	ec := engine.NewEngineController(context.Background(), l2Source, logger, metrics.NoopMetrics, cfg, &sync.Config{SyncMode: sync.CLSync}, d)
	syncCfg := &sync.Config{SyncMode: sync.CLSync}
	engResetDeriv := engine.NewEngineResetDeriver(context.Background(), logger, cfg, l1Source, l2Source, syncCfg)
	engResetDeriv.AttachEmitter(d)

	prog := &ProgramDeriver{
		logger:         logger,
		Emitter:        d,
		closing:        false,
		result:         eth.L2BlockRef{},
		targetBlockNum: targetBlockNum,
	}

	d.deriver = &event.DeriverMux{
		prog,
		ec,
		pipelineDeriver,
		engResetDeriv,
	}
	d.end = prog

	return d
}

func (d *Driver) Emit(ctx context.Context, ev event.Event) {
	if d.end.Closing() {
		return
	}
	d.events = append(d.events, ev)
}

func (d *Driver) RunComplete() (eth.L2BlockRef, error) {
	// Initial reset
	d.Emit(context.Background(), engine.ResetEngineRequestEvent{})

	for !d.end.Closing() {
		if len(d.events) == 0 {
			d.logger.Info("Derivation complete: no further data to process")
			return d.end.Result()
		}
		if len(d.events) > 10000 { // sanity check, in case of bugs. Better than going OOM.
			return eth.L2BlockRef{}, errTooManyEvents
		}
		ev := d.events[0]
		d.events = d.events[1:]
		d.deriver.OnEvent(context.Background(), ev)
	}
	return d.end.Result()
}
