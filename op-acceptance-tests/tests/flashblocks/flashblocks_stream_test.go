//go:build !ci

// use a tag prefixed with "!". Such tag ensures that the default behaviour of this test would be to be built/run even when the go toolchain (go test) doesn't specify any tag filter.
package flashblocks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	flashblocksStreamRate  = os.Getenv("FLASHBLOCKS_STREAM_RATE_MS")
	maxExpectedFlashblocks = 20
)

// TestFlashblocksStream checks we can connect to the flashblocks stream
func TestFlashblocksStream(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleFlashblocks(t)
	logger := testlog.Logger(t, log.LevelInfo).With("Test", "TestFlashblocksStream")
	tracer := t.Tracer()
	ctx := t.Ctx()
	logger.Info("Started Flashblocks Stream test")

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
	for l2Chain, flashblocksBuilderSet := range sys.FlashblocksBuilderSets {
		_, span = tracer.Start(ctx, "test chain")
		defer span.End()

		networkName := l2Chain.String()
		t.Run(fmt.Sprintf("L2_Chain_%s", networkName), func(tt devtest.T) {
			if len(flashblocksBuilderSet) == 0 {
				tt.Skip("no flashblocks builders for chain", l2Chain.String())
			}

			expectedChainID := l2Chain.ChainID().ToBig()
			for _, flashblocksBuilderNode := range flashblocksBuilderSet {
				require.Equal(t, flashblocksBuilderNode.Escape().ChainID().ToBig(), expectedChainID, "flashblocks builder node chain id should match expected chain id")

				mode := FlashblocksStreamMode_Follower
				if dsl.NewConductor(flashblocksBuilderNode.Escape().Conductor()).IsLeader() {
					mode = FlashblocksStreamMode_Leader
				}

				testFlashblocksStreamRbuilder(tt, logger, flashblocksBuilderNode, mode, flashblocksStreamRateMs)
			}

			for _, flashblocksWebsocketProxy := range sys.FlashblocksWebsocketProxies[l2Chain] {
				testFlashblocksStreamFbWsProxy(tt, logger, flashblocksWebsocketProxy, flashblocksStreamRateMs)
			}
		})
	}
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

// testFlashblocksStreamRbuilder tests the presence / absence of a flashblocks stream operating at a 250ms (configurable via env var FLASHBLOCKS_STREAM_RATE) rate from an rbuilder node
func testFlashblocksStreamRbuilder(t devtest.T, logger log.Logger, flashblocksBuilderNode *dsl.FlashblocksBuilderNode, mode FlashblocksStreamMode, expectedFlashblocksStreamRateMs int) {
	t.Run(fmt.Sprintf("Flashblocks_Stream_Rbuilder_%s_%s", flashblocksBuilderNode.Escape().ID(), mode), func(t devtest.T) {
		testDuration := time.Duration(int64(expectedFlashblocksStreamRateMs*maxExpectedFlashblocks)) * time.Millisecond
		failureTolerance := int(0.15 * float64(maxExpectedFlashblocks))

		logger.Debug("Test duration", "duration", testDuration, "failure tolerance (of flashblocks)", failureTolerance)

		require.Contains(t, []FlashblocksStreamMode{FlashblocksStreamMode_Leader, FlashblocksStreamMode_Follower}, mode, "mode should be either leader or follower")
		require.NotNil(t, flashblocksBuilderNode, "flashblocksBuilderNode should not be nil")

		output := make(chan []byte, maxExpectedFlashblocks)
		doneListening := make(chan struct{})
		streamedMessages := make([]string, 0)
		go flashblocksBuilderNode.ListenFor(logger, testDuration, output, doneListening) //nolint:errcheck

		for {
			select {
			case <-doneListening:
				goto done
			case msg := <-output:
				streamedMessages = append(streamedMessages, string(msg))
			}
		}
	done:

		defer close(output)

		logger.Info("Completed WebSocket stream reading", "message_count", len(streamedMessages))
		if mode == FlashblocksStreamMode_Follower {
			require.Equal(t, len(streamedMessages), 0, "follower should not receive any messages")
			return
		}

		totalFlashblocksProduced := evaluateFlashblocksStream(t, logger, streamedMessages, failureTolerance)

		minExpectedFlashblocks := maxExpectedFlashblocks - failureTolerance
		require.Greater(t,
			totalFlashblocksProduced, minExpectedFlashblocks,
			fmt.Sprintf("total flashblocks produced should be greater than %d (%d over %s with a %dms rate with a failure tolerance of %d flashblocks)",
				minExpectedFlashblocks,
				maxExpectedFlashblocks,
				testDuration,
				expectedFlashblocksStreamRateMs,
				failureTolerance,
			),
		)

		logger.Info("Flashblocks stream validation completed", "total_flashblocks_produced", totalFlashblocksProduced)
	})
}

// testFlashblocksStreamFbWsProxy tests the presence / absence of a flashblocks stream operating at a 250ms (configurable via env var FLASHBLOCKS_STREAM_RATE) rate from a flashblocks-websocket-proxy node
func testFlashblocksStreamFbWsProxy(t devtest.T, logger log.Logger, flashblocksWebsocketProxy *dsl.FlashblocksWebsocketProxy, expectedFlashblocksStreamRateMs int) {
	t.Run(fmt.Sprintf("Flashblocks_Stream_FbWsProxy_%s", flashblocksWebsocketProxy.Escape().ID()), func(t devtest.T) {
		testDuration := time.Duration(int64(expectedFlashblocksStreamRateMs*maxExpectedFlashblocks)) * time.Millisecond
		failureTolerance := int(0.15 * float64(maxExpectedFlashblocks))

		logger.Debug("Test duration", "duration", testDuration, "failure tolerance (of flashblocks)", failureTolerance)

		require.NotNil(t, flashblocksWebsocketProxy, "flashblocksWebsocketProxy should not be nil")

		output := make(chan []byte, maxExpectedFlashblocks)
		doneListening := make(chan struct{})
		streamedMessages := make([]string, 0)
		go flashblocksWebsocketProxy.ListenFor(logger, testDuration, output, doneListening) //nolint:errcheck

		for {
			select {
			case <-doneListening:
				goto done
			case msg := <-output:
				streamedMessages = append(streamedMessages, string(msg))
			}
		}
	done:

		defer close(output)

		logger.Info("Completed WebSocket stream reading", "message_count", len(streamedMessages))

		totalFlashblocksProduced := evaluateFlashblocksStream(t, logger, streamedMessages, failureTolerance)

		minExpectedFlashblocks := maxExpectedFlashblocks - failureTolerance
		require.Greater(t,
			totalFlashblocksProduced, minExpectedFlashblocks,
			fmt.Sprintf("total flashblocks produced should be greater than %d (%d over %s with a %dms rate with a failure tolerance of %d flashblocks)",
				minExpectedFlashblocks,
				maxExpectedFlashblocks,
				testDuration,
				expectedFlashblocksStreamRateMs,
				failureTolerance,
			),
		)

		logger.Info("Flashblocks stream validation completed", "total_flashblocks_produced", totalFlashblocksProduced)
	})
}
