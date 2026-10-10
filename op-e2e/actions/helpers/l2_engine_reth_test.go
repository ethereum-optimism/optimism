package helpers

import (
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestRethEngineStderrLoggedAtInfo checks that the engine subprocess's stderr reaches the test
// logger at Info, so engine panics and startup errors show at the usual test log levels.
func TestRethEngineStderrLoggedAtInfo(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, DefaultRollupTestParams())
	sd := e2eutils.Setup(t, dp, DefaultAlloc)
	logger, logs := testlog.CaptureLogger(t, log.LevelInfo)

	engine := newRethL2Engine(t, logger, sd.L2Cfg)
	require.NoError(t, engine.Close())

	require.NotNil(t, logs.FindLog(
		testlog.NewMessageFilter("op-reth-test-engine"),
		testlog.NewAttributesContainsFilter("line", "ready on"),
	), "engine stderr not logged at Info")
}

// abortingTesting stands in for Testing in an action run on its own goroutine: it records failures
// without failing the test, and ends the action on FailNow, Fatalf or InvalidAction, as the
// default Testing does.
type abortingTesting struct {
	Testing
	failed atomic.Bool
}

func (a *abortingTesting) Errorf(string, ...any) { a.failed.Store(true) }

func (a *abortingTesting) FailNow() {
	a.failed.Store(true)
	runtime.Goexit()
}

func (a *abortingTesting) Fatalf(string, ...any) { a.FailNow() }

func (a *abortingTesting) InvalidAction(string, ...any) { a.FailNow() }

// runAborting runs act with an abortingTesting on its own goroutine and reports whether it failed.
func runAborting(t Testing, act Action) bool {
	at := &abortingTesting{Testing: t}
	done := make(chan struct{})
	go func() {
		defer close(done)
		act(at)
	}()
	<-done
	return at.failed.Load()
}

// TestRethIncludeTxIgnoreForcedEmptyRestoresFlag checks that the block's force-empty flag is
// restored when the include fails and ends the action.
func TestRethIncludeTxIgnoreForcedEmptyRestoresFlag(gt *testing.T) {
	gt.Setenv(ELSelectorEnv, elRethTestEngine)
	t := SubTest(gt)
	dp := e2eutils.MakeDeployParams(t, DefaultRollupTestParams())
	sd := e2eutils.Setup(t, dp, DefaultAlloc)
	logger := testlog.Logger(t, log.LevelInfo)
	miner, engine, sequencer := SetupSequencerTest(t, sd, logger)
	miner.ActEmptyBlock(t)
	sequencer.ActL2PipelineFull(t)
	sequencer.ActL2StartBlock(t)
	engine.reth.setForceEmpty(t, true)

	// Nothing from this sender is parked, so there is nothing to include.
	failed := runAborting(t, engine.ActL2IncludeTxIgnoreForcedEmpty(common.Address{0xaa}))
	require.True(t, failed, "include should fail")
	require.True(t, engine.ForcedEmpty(t), "force-empty flag not restored")
}

// TestRethEngineRejectsUnsupportedOptions checks that the reth backend fails on an engine option it
// cannot honour instead of dropping it.
func TestRethEngineRejectsUnsupportedOptions(gt *testing.T) {
	gt.Setenv(ELSelectorEnv, elRethTestEngine)
	t := SubTest(gt)
	dp := e2eutils.MakeDeployParams(t, DefaultRollupTestParams())
	sd := e2eutils.Setup(t, dp, DefaultAlloc)
	logger := testlog.Logger(t, log.LevelInfo)
	jwtPath := e2eutils.WriteDefaultJWT(t)

	withDataDir := func(_ *ethconfig.Config, nodeCfg *node.Config) error {
		nodeCfg.DataDir = t.TempDir()
		return nil
	}
	require.True(t, runAborting(t, func(t Testing) {
		NewL2Engine(t, logger, sd.L2Cfg, jwtPath, withDataDir)
	}), "an unsupported option must fail")
}
