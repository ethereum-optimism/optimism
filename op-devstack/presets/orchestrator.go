package presets

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"go.opentelemetry.io/otel"

	"github.com/ethereum-optimism/optimism/devnet-sdk/telemetry"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysext"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

// lockedOrchestrator is the global variable that stores
// the global orchestrator that tests may use.
// Presets are expected to use the global orchestrator,
// unless explicitly told otherwise using a WithOrchestrator option.
var lockedOrchestrator locks.RWValue[stack.Orchestrator]

type backendKind string

const (
	backendKindSysGo  backendKind = "sysgo"
	backendKindSysExt backendKind = "sysext"
)

// DoMain runs M with the pre- and post-processing of tests,
// to setup the default global orchestrator and global logger.
// This will os.Exit(code) and not return.
func DoMain(m *testing.M, opts ...stack.CommonOption) {
	// nest the function, so we can defer-recover and defer-cleanup, before os.Exit
	code := func() (errCode int) {
		failed := new(atomic.Bool)
		defer func() {
			if failed.Load() {
				errCode = 1
			}
		}()
		defer func() {
			if x := recover(); x != nil {
				debug.PrintStack()
				_, _ = fmt.Fprintf(os.Stderr, "Panic during test Main: %v\n", x)

				failed.Store(true)
			}
		}()

		// This may be tuned with env or CLI flags in the future, to customize test output
		logger := oplog.NewLogger(os.Stdout, oplog.CLIConfig{
			Level:  log.LevelInfo,
			Color:  true,
			Format: oplog.FormatTerminal,
			Pid:    false,
		})

		ctx, otelShutdown, err := telemetry.SetupOpenTelemetry(context.Background())
		if err != nil {
			logger.Warn("Failed to setup OpenTelemetry", "error", err)
		} else {
			defer otelShutdown()
		}

		ctx, run := otel.Tracer("run").Start(ctx, "test suite")
		defer run.End()

		devtest.RootContext = ctx
		p := devtest.NewP(ctx, logger, func() {
			debug.PrintStack()
			failed.Store(true)
			panic("setup fail")
		})
		defer p.Close()

		p.Require().NotEmpty(opts, "Expecting orchestrator options")

		// For the global geth logs,
		// capture them in the global test logger.
		// No other tool / test should change the global logger.
		// TODO(#15139): set log-level filter, reduce noise
		//log.SetDefault(t.Log.New("logger", "global"))

		initOrchestrator(ctx, p, stack.Combine(opts...))

		errCode = m.Run()
		return
	}()
	_, _ = fmt.Fprintf(os.Stderr, "\nExiting, code: %d\n", code)
	os.Exit(code)
}

func initOrchestrator(ctx context.Context, p devtest.P, opt stack.CommonOption) {
	ctx, span := p.Tracer().Start(ctx, "initializing orchestrator")
	defer span.End()

	lockedOrchestrator.Lock()
	defer lockedOrchestrator.Unlock()
	if lockedOrchestrator.Value != nil {
		return
	}
	backend := backendKindSysGo
	if override, ok := os.LookupEnv("DEVSTACK_ORCHESTRATOR"); ok {
		backend = backendKind(override)
	}
	switch backend {
	case backendKindSysGo:
		lockedOrchestrator.Value = sysgo.NewOrchestrator(p, stack.SystemHook(opt))
	case backendKindSysExt:
		lockedOrchestrator.Value = sysext.NewOrchestrator(p, stack.SystemHook(opt))
	default:
		panic(fmt.Sprintf("Unknown backend for initializing orchestrator: %s", backend))
	}

	p.Logger().WithContext(ctx).Info("initializing orchestrator", "backend", backend)
	stack.ApplyOptionLifecycle(opt, lockedOrchestrator.Value)
}

// Orchestrator returns the globally configured orchestrator.
//
// Add a TestMain to your test package init the orchestrator:
//
//	func TestMain(m *testing.M) {
//	    presets.DoMain(m)
//	}
func Orchestrator() stack.Orchestrator {
	out := lockedOrchestrator.Get()
	if out == nil {
		panic(`
Add a TestMain to your test package init the orchestrator:

	func TestMain(m *testing.M) {
		presets.DoMain(m)
	}
`)
	}
	return out
}
