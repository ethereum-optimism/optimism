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
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/event"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

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

func NewDriver(logger log.Logger, cfg *rollup.Config, l1Source derive.L1Fetcher,
	l1BlobsSource derive.L1BlobsFetcher, l2Source engine.Engine, targetBlockNum uint64) *Driver {

	d := &Driver{
		logger: logger,
	}

	var altdaImpl driver.AltDAIface

	altdaConfig, err := cfg.GetOPAltDAConfig()
	if err != nil {
		panic("cfg.GetOPAltDAConfig() ")
	}

	// using using altda then always try to proxy
	if altdaConfig.CommitmentType == altda.GenericCommitmentType {
		// using eigenda proxy, this is a hack, clearly it cannot compile given it requires rpc access
		addr := "http://127.0.0.1:3100"

		daClient := altda.NewDAClient(addr, false, false)
		daMgr := altda.NewAltDAWithStorage(logger, altdaConfig, daClient, &altda.NoopMetrics{})

		altdaImpl = daMgr

		log.Info("NewL2FaultProofEnv", "dp.AllocType", "EigenDAFaker")
	} else {
		altdaImpl = &altda.AltDADisabled{}
	}

	pipeline := derive.NewDerivationPipeline(logger, cfg, l1Source, l1BlobsSource, altdaImpl, l2Source, metrics.NoopMetrics, false)
	pipelineDeriver := derive.NewPipelineDeriver(context.Background(), pipeline)
	pipelineDeriver.AttachEmitter(d)

	ec := engine.NewEngineController(l2Source, logger, metrics.NoopMetrics, cfg, &sync.Config{SyncMode: sync.CLSync}, d)
	engineDeriv := engine.NewEngDeriver(logger, context.Background(), cfg, metrics.NoopMetrics, ec)
	engineDeriv.AttachEmitter(d)
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
		engineDeriv,
		pipelineDeriver,
		engResetDeriv,
	}
	d.end = prog

	return d
}

func (d *Driver) Emit(ev event.Event) {
	if d.end.Closing() {
		return
	}
	d.events = append(d.events, ev)
}

func (d *Driver) RunComplete() (eth.L2BlockRef, error) {
	// Initial reset
	d.Emit(engine.ResetEngineRequestEvent{})

	for !d.end.Closing() {
		if len(d.events) == 0 {
			d.logger.Info("Derivation complete: no further data to process")
			return d.end.Result()
		}
		if len(d.events) > 10000 { // sanity check, in case of bugs. Better than going OOM.
			return eth.L2BlockRef{}, errors.New("way too many events queued up, something is wrong")
		}
		ev := d.events[0]
		d.events = d.events[1:]
		d.deriver.OnEvent(ev)
	}
	return d.end.Result()
}
