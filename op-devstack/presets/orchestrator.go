package presets

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"go.opentelemetry.io/otel"

	"github.com/ethereum-optimism/optimism/devnet-sdk/telemetry"
	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysext"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/flags"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
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

		cfg := flags.ReadTestConfig()
		logHandler := oplog.NewLogHandler(os.Stdout, cfg.LogConfig)
		logHandler = logfilter.WrapFilterHandler(logHandler)
		logHandler.(logfilter.FilterHandler).Set(logfilter.DefaultMute(logfilter.Level(log.LevelInfo).Show()))
		logHandler = logfilter.WrapContextHandler(logHandler)
		// The default can be changed using the WithLogFilters option which replaces this default
		logger := log.NewLogger(logHandler)
		oplog.SetGlobalLogHandler(logHandler)

		ctx, otelShutdown, err := telemetry.SetupOpenTelemetry(context.Background())
		if err != nil {
			logger.Warn("Failed to setup OpenTelemetry", "error", err)
		} else {
			defer otelShutdown()
		}

		ctx, run := otel.Tracer("run").Start(ctx, "test suite")
		defer run.End()

		// All tests will inherit this package-level context
		devtest.RootContext = ctx

		// Make the package-level logger use this context
		logger.SetContext(ctx)

		onFail := func(now bool) {
			logger.Error("Main failed")
			debug.PrintStack()
			failed.Store(true)
			if now {
				panic("critical Main fail")
			}
		}

		onSkipNow := func() {
			logger.Info("Main skipped")
			os.Exit(0)
		}
		p := devtest.NewP(ctx, logger, onFail, onSkipNow)
		defer p.Close()

		p.Require().NotEmpty(opts, "Expecting orchestrator options")

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

	p.Logger().InfoContext(ctx, "initializing orchestrator", "backend", backend)
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

// WithCompatibleTypes is a common option that can be used to ensure that the orchestrator is compatible with the preset.
// If the orchestrator is not compatible, the test will either:
// - fail with a non-zero exit code (42) if DEVNET_EXPECT_PRECONDITIONS_MET is non-empty
// - skip the whole test otherwise
// This is useful to ensure that the preset is only used with the correct orchestrator type.
// Do yourself a favor, if you use this option, add a good comment (or a TODO) justifying it!
func WithCompatibleTypes(t ...compat.Type) stack.CommonOption {
	return stack.FnOption[stack.Orchestrator]{
		BeforeDeployFn: func(orch stack.Orchestrator) {
			if !slices.Contains(t, orch.Type()) {
				p := orch.P()

				if os.Getenv(devtest.ExpectPreconditionsMet) != "" {
					p.Errorf("Orchestrator type %s is incompatible with this preset", orch.Type())
					os.Exit(compat.CompatErrorCode)
				} else {
					p.SkipNow()
				}
			}
		},
	}
}
