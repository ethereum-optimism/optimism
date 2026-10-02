package helpers

import (
	"testing"

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
