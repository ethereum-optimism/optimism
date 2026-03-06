package flashblocks

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
	"github.com/ethereum-optimism/optimism/op-service/logmods"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	flashblocksStreamRate  = os.Getenv("FLASHBLOCKS_STREAM_RATE_MS")
	maxExpectedFlashblocks = 20
)

// TestFlashblocksStream checks we can connect to the flashblocks stream across multiple CL backends.
func TestFlashblocksStream(gt *testing.T) {
	t := devtest.SerialT(gt)
	logger := t.Logger()
	sys := presets.NewSingleChainWithFlashblocks(t)
	filterHandler, ok := logmods.FindHandler[logfilter.FilterHandler](logger.Handler())
	if ok {
		filterHandler.Set(logfilter.DefaultMute(
			logfilter.Level(slog.LevelError).Show(),
			logfilter.Select("kind", "L2CLNode").Show(),
		))
	}
	tracer := t.Tracer()
	ctx := t.Ctx()

	ctx, span := tracer.Start(ctx, "test chains")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if flashblocksStreamRate == "" {
		logger.Warn("FLASHBLOCKS_STREAM_RATE_MS is not set, using default of 250ms")
		flashblocksStreamRate = "250"
	}

	flashblocksStreamRateMs, err := strconv.Atoi(flashblocksStreamRate)
	require.NoError(t, err, "failed to parse FLASHBLOCKS_STREAM_RATE_MS: %s", err)

	logger.Info("Flashblocks stream rate", "rate", flashblocksStreamRateMs)

	// Test all L2 chains in the system
	oprbuilderNode := sys.L2OPRBuilder
	rollupBoostNode := sys.L2RollupBoost
	_, span = tracer.Start(ctx, "test chain")
	defer span.End()

	expectedChainID := sys.L2Chain.ChainID().ToBig()
	require.Equal(t, oprbuilderNode.Escape().ChainID().ToBig(), expectedChainID, "flashblocks builder node chain id should match expected chain id")

	DriveViaTestSequencer(t, sys, 3)

	// Test the presence / absence of a flashblocks stream operating at a 250ms rate from a flashblocks-websocket-proxy node.
	// Allow a generous window for first flashblocks to appear.
	testDuration := time.Duration(int64(flashblocksStreamRateMs*maxExpectedFlashblocks*2)) * time.Millisecond
	// Allow up to 15% of expected flashblocks to be missing due to timing variations
	failureTolerance := int(0.15 * float64(maxExpectedFlashblocks))

	logger.Debug("Test duration", "duration", testDuration, "failure tolerance (of flashblocks)", failureTolerance)

	// Instrument builder stream separately to confirm flashblocks emission upstream.
	builderOutput := make(chan []byte, maxExpectedFlashblocks)
	defer close(builderOutput)
	builderDone := make(chan struct{})
	go func() {
		err := oprbuilderNode.FlashblocksClient().ReadAll(ctx, logger.With("stream_source", "op-rbuilder"), testDuration, builderOutput, builderDone)
		require.NoError(t, err)
	}()
	builderMessages := make([]string, 0)

	output := make(chan []byte, maxExpectedFlashblocks)
	defer close(output)
	doneListening := make(chan struct{})
	streamedMessages := make([]string, 0)
	go func() {
		err := rollupBoostNode.FlashblocksClient().ReadAll(ctx, logger.With("stream_source", "rollup-boost"), testDuration, output, doneListening)
		require.NoError(t, err)
	}()

	listening := true
	for listening {
		select {
		case <-doneListening:
			doneListening = nil
		case <-builderDone:
			builderDone = nil
		case msg := <-output:
			streamedMessages = append(streamedMessages, string(msg))
		case msg := <-builderOutput:
			builderMessages = append(builderMessages, string(msg))
		}

		if doneListening == nil && builderDone == nil {
			listening = false
		}
	}

	logger.Info("Completed WebSocket stream reading", "msg_count", len(streamedMessages), "builder_msg_count", len(builderMessages))

	if len(builderMessages) > 0 {
		logger.Info("Sample builder message", "payload", builderMessages[0])
	}

	totalFlashblocksProduced := evaluateFlashblocksStream(t, logger, streamedMessages, failureTolerance)
	require.Greater(t, totalFlashblocksProduced, 0, "expected to receive flashblocks from rollup-boost stream")
	logger.Info("Flashblocks stream validation completed", "total_flashblocks_produced", totalFlashblocksProduced)
}

func evaluateFlashblocksStream(t devtest.T, logger log.Logger, streamedMessages []string, failureTolerance int) int {
	require.Greater(t, len(streamedMessages), 0, "should have received at least one message from WebSocket")
	flashblocks := make([]Flashblock, len(streamedMessages))

	failures := 0
	for i, msg := range streamedMessages {
		var flashblock Flashblock
		if err := json.Unmarshal([]byte(msg), &flashblock); err != nil {
			logger.Warn("Failed to unmarshal WebSocket message", "error", err)
			failures++
			if failures > failureTolerance {
				logger.Error("failed to unmarshal streamed messages into flashblocks beyond the failure tolerance of %d", failureTolerance)
				t.FailNow()
			}
			continue
		}

		flashblocks[i] = flashblock
	}

	totalFlashblocksProduced := 0

	lastIndex := -1
	lastBlockNumber := -1

	for _, flashblock := range flashblocks {
		currentIndex, currentBlockNumber := flashblock.Index, flashblock.Metadata.BlockNumber

		if lastBlockNumber == -1 {
			totalFlashblocksProduced += 1
			lastIndex = currentIndex
			lastBlockNumber = currentBlockNumber
			continue
		}

		require.Greater(t, lastIndex, -1, "some bug: last index should be greater than -1 by now")
		require.Greater(t, currentIndex, -1, "some bug: current index should be greater than -1 by now")

		// same block number, just the flashblock incremented
		if currentBlockNumber == lastBlockNumber {
			require.Greater(t, currentIndex, lastIndex, "some bug: current index should be greater than last index from the stream")

			totalFlashblocksProduced += (currentIndex - lastIndex)
		} else if currentBlockNumber > lastBlockNumber { // new block number
			totalFlashblocksProduced += (currentIndex + 1) // assuming it's a new block number whose flashblocks begin from 0th-index
		}

		lastIndex = currentIndex
		lastBlockNumber = currentBlockNumber
	}

	return totalFlashblocksProduced
}
