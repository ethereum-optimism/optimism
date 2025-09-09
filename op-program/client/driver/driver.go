package driver

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/attributes"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type EndCondition interface {
	Closing() bool
	Result() (eth.L2BlockRef, error)
}

type Driver struct {
	logger log.Logger

	// Event system components
	sys          event.System
	exec         *event.SingleThreadCooperativeExec
	driverEmiter event.Emitter

	end EndCondition
}

func NewDriver(logger log.Logger, cfg *rollup.Config, depSet derive.DependencySet, l1Source derive.L1Fetcher,
	l1BlobsSource derive.L1BlobsFetcher, l2Source engine.Engine, targetBlockNum uint64) *Driver {

	exec := event.NewSingleThreadCooperative(context.Background())
	sys := event.NewSystem(logger, exec)

	// Create derivation pipeline and register as deriver (emitter auto-attached)
	pipeline := derive.NewDerivationPipeline(logger, cfg, depSet, l1Source, l1BlobsSource, altda.Disabled, l2Source, metrics.NoopMetrics, false)
	pipelineDeriver := derive.NewPipelineDeriver(context.Background(), pipeline)
	sys.Register("pipeline", pipelineDeriver)

	// Engine controller needs an emitter at construction time
	ecEmitter := sys.Register("engine-controller", nil)
	ec := engine.NewEngineController(context.Background(), l2Source, logger, metrics.NoopMetrics, cfg, &sync.Config{SyncMode: sync.CLSync}, ecEmitter)
	// And also needs to be registered as a deriver to consume events
	sys.Register("engine", ec)

	// Attributes handler only used as a resetter in this client path
	attrHandler := attributes.NewAttributesHandler(logger, cfg, context.Background(), l2Source, ec)
	ec.SetAttributesResetter(attrHandler)
	ec.SetPipelineResetter(pipelineDeriver)

	// Register engine reset deriver
	syncCfg := &sync.Config{SyncMode: sync.CLSync}
	engResetDeriv := engine.NewEngineResetDeriver(context.Background(), logger, cfg, l1Source, l2Source, syncCfg)
	sys.Register("engine-reset", engResetDeriv)
	engResetDeriv.SetEngController(ec)

	// Program deriver coordinates high-level flow
	prog := &ProgramDeriver{
		logger:           logger,
		engineController: ec,
		closing:          false,
		result:           eth.L2BlockRef{},
		targetBlockNum:   targetBlockNum,
		// In the op-program trace-extension path there is no external sequencer
		// injecting more events after the pipeline idles; close on idle.
		closeOnIdle: true,
	}
	sys.Register("program", prog)

	d := &Driver{
		logger:       logger,
		sys:          sys,
		exec:         exec,
		driverEmiter: sys.Register("driver", nil),
		end:          prog,
	}
	return d
}

func (d *Driver) RunComplete() (eth.L2BlockRef, error) {
	// Initial reset
	ctx := event.WithSystem(context.Background(), d.sys)
	d.driverEmiter.Emit(ctx, engine.ResetEngineRequestEvent{})

	// Drive the single-thread executor inline until completion.
	// We avoid a separate Drive goroutine to ensure run-to-completion without await races.
	for !d.end.Closing() {
		// Drain any queued events synchronously
		_ = d.exec.Drain()
		if d.end.Closing() {
			break
		}
		// Await for more events to avoid busy spinning
		<-d.exec.Await()
	}
	return d.end.Result()
}
