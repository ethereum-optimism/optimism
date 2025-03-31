package presets

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/sysgo"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/syskt"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// lockedOrchestrator is the global variable that stores
// the global orchestrator that tests may use.
// Presets are expected to use the global orchestrator,
// unless explicitly told otherwise using a WithOrchestrator option.
var lockedOrchestrator locks.RWValue[stack.Orchestrator]

// DoMain runs the pre- and post-processing of tests,
// to setup the default global orchestrator and global logger.
func DoMain(m *testing.M) {
	defer func() {
		if x := recover(); x != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Panic during test Main: %v\n", x)
			os.Exit(1)
		}
	}()

	// This may be tuned with env or CLI flags in the future, to customize test output
	logger := oplog.NewLogger(os.Stdout, oplog.CLIConfig{
		Level:  log.LevelInfo,
		Color:  true,
		Format: oplog.FormatTerminal,
		Pid:    false,
	})

	t := stack.NewToolingT("Main", logger)

	// For the global geth logs,
	// capture them in the global test logger.
	// No other tool / test should change the global logger.
	// TODO(#15139): set log-level filter, reduce noise
	//log.SetDefault(t.Log.New("logger", "global"))

	initOrchestrator(t, t.Log)
	code := m.Run()
	t.RunCleanup()
	os.Exit(code)
}

func initOrchestrator(t stack.T, logger log.Logger) {
	lockedOrchestrator.Lock()
	defer lockedOrchestrator.Unlock()
	if lockedOrchestrator.Value != nil {
		return
	}
	kind, ok := os.LookupEnv("DEVSTACK_ORCHESTRATOR")
	if !ok {
		logger.Warn("Selecting sysgo as default devstack orchestrator")
		kind = "sysgo"
	}
	switch kind {
	case "sysgo":
		lockedOrchestrator.Value = sysgo.NewOrchestrator(t, logger)
	case "syskt":
		lockedOrchestrator.Value = syskt.NewOrchestrator(t, logger)
	default:
		logger.Crit("Unknown devstack backend", "kind", kind)
	}
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

// WithGlobalOrchestrator attaches the main global Orchestrator() to the setup.
func WithGlobalOrchestrator() stack.Option {
	return func(setup *stack.Setup) {
		setup.Require.Nil(setup.Orchestrator, "cannot change existing orchestrator of setup")
		setup.Orchestrator = Orchestrator()
	}
}

// WithTestLogger attaches a test-logger
func WithTestLogger() stack.Option {
	return func(setup *stack.Setup) {
		setup.Require.Nil(setup.Log, "must not already have a logger")
		setup.Log = testlog.Logger(setup.T, log.LevelInfo)
	}
}

// WithEmptySystem attaches an empty system, for other options to add components to
func WithEmptySystem() stack.Option {
	return func(setup *stack.Setup) {
		setup.Require.Nil(setup.System, "must not already have a system")
		setup.Require.NotNil(setup.Log, "need logger")
		setup.System = shim.NewSystem(shim.SystemConfig{
			CommonConfig: shim.CommonConfig{
				Log: setup.Log,
				T:   setup.T,
			},
		})
	}
}

// NewSetup creates a new empty Setup with nil system and nil orchestrator.
// The orchestrator can be configured with an option.
func NewSetup(t stack.T, opts ...stack.Option) *stack.Setup {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Create a test-setup, unique to the test
	setup := &stack.Setup{
		Ctx:          ctx,
		Log:          nil,
		T:            t,
		Require:      require.New(t),
		System:       nil,
		Orchestrator: nil,
	}
	// apply any initial options to the system
	for _, opt := range opts {
		opt(setup)
	}
	return setup
}
