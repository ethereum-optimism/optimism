package driver

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
)

type EndCondition interface {
	Closing() bool
	Result() error
}

type Driver struct {
	logger log.Logger

	events []rollup.Event

	end     EndCondition
	deriver rollup.Deriver
}

func NewDriver(logger log.Logger, cfg *rollup.Config, l1Source derive.L1Fetcher,
	l1BlobsSource derive.L1BlobsFetcher, l2Source engine.Engine, targetBlockNum uint64) *Driver {

	d := &Driver{
		logger: logger,
	}

	pipeline := derive.NewDerivationPipeline(logger, cfg, l1Source, l1BlobsSource, plasma.Disabled, l2Source, metrics.NoopMetrics)
	pipelineDeriver := derive.NewPipelineDeriver(context.Background(), pipeline, d)

	ec := engine.NewEngineController(l2Source, logger, metrics.NoopMetrics, cfg, sync.CLSync, d)
	engineDeriv := engine.NewEngDeriver(logger, context.Background(), cfg, ec, d)
	syncCfg := &sync.Config{SyncMode: sync.CLSync}
	engResetDeriv := engine.NewEngineResetDeriver(context.Background(), logger, cfg, l1Source, l2Source, syncCfg, d)

	prog := &ProgramDeriver{
		logger:         logger,
		Emitter:        d,
		closing:        false,
		result:         nil,
		targetBlockNum: targetBlockNum,
	}

	d.deriver = &rollup.SynchronousDerivers{
		prog,
		engineDeriv,
		pipelineDeriver,
		engResetDeriv,
	}
	d.end = prog

	return d
}

func (d *Driver) Emit(ev rollup.Event) {
	if d.end.Closing() {
		return
	}
	d.events = append(d.events, ev)
}

var ExhaustErr = errors.New("exhausted events before completing program")

func (d *Driver) RunComplete() error {
	// Initial reset
	d.Emit(engine.ResetEngineRequestEvent{})

	for !d.end.Closing() {
		if len(d.events) == 0 {
			return ExhaustErr
		}
		if len(d.events) > 10000 { // sanity check, in case of bugs. Better than going OOM.
			return errors.New("way too many events queued up, something is wrong")
		}
		ev := d.events[0]
		d.events = d.events[1:]
		d.deriver.OnEvent(ev)
	}
	return d.end.Result()
}
