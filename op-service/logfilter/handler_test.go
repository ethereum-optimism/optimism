package logfilter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/logmods"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestWrapFilterHandler(t *testing.T) {
	// Create a logger, with capturing and filtering
	logger := testlog.LoggerWithHandlerMod(t, log.LevelInfo,
		testlog.WrapCaptureLogger, WrapFilterHandler)

	capturer, ok := logmods.FindHandler[testlog.Capturer](logger.Handler())
	require.True(t, ok)

	// The filter runs before capture (as outermost handler wrapper).
	// It will mute things before it reaches the capturer
	filterer := logger.Handler().(Handler)
	filterer.Add(Mute())
	filterer.Add(IfMatch(ctxTestKey, "alice", As(log.LevelInfo)))

	// Log some things
	logger.InfoContext(context.Background(), "unrecognized context 1")
	logger.InfoContext(context.WithValue(context.Background(), ctxTestKey, "alice"), "matched context")
	logger.InfoContext(context.Background(), "unrecognized context 2")

	// Now see if the filter worked
	rec := capturer.FindLog(testlog.NewMessageFilter("matched context"))
	require.Equal(t, "matched context", rec.Record.Message)
}
