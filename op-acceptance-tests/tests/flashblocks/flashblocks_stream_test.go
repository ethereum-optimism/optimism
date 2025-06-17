//go:build !ci

// use a tag prefixed with "!". Such tag ensures that the default behaviour of this test would be to be built/run even when the go toolchain (go test) doesn't specify any tag filter.
package flashblocks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

var (
	flashblocksStreamRate  = os.Getenv("FLASHBLOCKS_STREAM_RATE_MS")
	maxExpectedFlashblocks = 20
)

// Define a struct to represent the flashblock data structure
type Flashblock struct {
	PayloadID string `json:"payload_id"`
	Index     int    `json:"index"`
	Diff      struct {
		StateRoot    string `json:"state_root"`
		ReceiptsRoot string `json:"receipts_root"`
		LogsBloom    string `json:"logs_bloom"`
		GasUsed      string `json:"gas_used"`
		BlockHash    string `json:"block_hash"`
		Transactions []any  `json:"transactions"`
		Withdrawals  []any  `json:"withdrawals"`
	} `json:"diff"`
	Metadata struct {
		BlockNumber        int                    `json:"block_number"`
		NewAccountBalances map[string]string      `json:"new_account_balances"`
		Receipts           map[string]interface{} `json:"receipts"`
	} `json:"metadata"`
}

type FlashblocksStreamMode string

const (
	FlashblocksStreamMode_Leader   FlashblocksStreamMode = "leader"
	FlashblocksStreamMode_Follower FlashblocksStreamMode = "follower"
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
			expectedChainID := l2Chain.ChainID().ToBig()
			for _, flashblocksBuilderNode := range flashblocksBuilderSet {
				flashblocksBuilderNodeStack := flashblocksBuilderNode.Escape()
				require.Equal(t, flashblocksBuilderNodeStack.ChainID().ToBig(), expectedChainID, "flashblocks builder node chain id should match expected chain id")

				var associatedConductor stack.Conductor
				for _, conductor := range sys.ConductorSets[l2Chain] {
					if flashblocksBuilderNodeStack.ConductorID() == conductor.Escape().ID() {
						associatedConductor = conductor.Escape()
					}
				}
				require.NotNil(t, associatedConductor, "there must be a conductor associated with the flashblocks builder node")

				mode := FlashblocksStreamMode_Follower
				if dsl.NewConductor(associatedConductor).IsLeader() {
					mode = FlashblocksStreamMode_Leader
				}

				testFlashblocksStream(tt, logger, dsl.NewFlashblocksBuilderNode(flashblocksBuilderNodeStack), mode, flashblocksStreamRateMs)
			}
		})
	}
}

// testFlashblocksStream tests the presence / absence of a flashblocks stream operating at a 250ms (configurable via env var FLASHBLOCKS_STREAM_RATE) rate
func testFlashblocksStream(t devtest.T, logger log.Logger, flashblocksBuilderNode *dsl.FlashblocksBuilderNode, mode FlashblocksStreamMode, expectedFlashblocksStreamRateMs int) {
	t.Run(fmt.Sprintf("Flashblocks_Stream_%s", mode), func(t devtest.T) {
		testDuration := time.Duration(int64(expectedFlashblocksStreamRateMs*maxExpectedFlashblocks)) * time.Millisecond
		failureTolerance := int(0.15 * float64(maxExpectedFlashblocks))

		logger.Debug("Test duration", "duration", testDuration, "failure tolerance (of flashblocks)", failureTolerance)

		require.Contains(t, []FlashblocksStreamMode{FlashblocksStreamMode_Leader, FlashblocksStreamMode_Follower}, mode, "mode should be either leader or follower")
		require.NotNil(t, flashblocksBuilderNode, "flashblocksBuilderNode should not be nil")

		wsURL := flashblocksBuilderNode.Escape().FlashblocksWsUrl()
		logger.Info("Testing WebSocket connection to", "url", wsURL)

		dialer := &websocket.Dialer{
			HandshakeTimeout: 6 * time.Second,
		}

		conn, _, err := dialer.Dial(wsURL, nil)
		require.NoError(t, err, "failed to connect to FLashblocks WebSocket endpoint", "url", wsURL)
		defer conn.Close()

		logger.Info("WebSocket connection established, reading stream for 5 seconds")

		streamedMessages := make([]string, 0)

		timeout := time.After(5 * time.Second)
		for {
			select {
			case <-timeout:
				goto done
			default:
				err = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				require.NoError(t, err, "failed to set read deadline")
				_, message, err := conn.ReadMessage()
				if err != nil && !strings.Contains(err.Error(), "timeout") {
					logger.Error("Error reading WebSocket message", "error", err)
					break
				}
				if err == nil {
					streamedMessages = append(streamedMessages, string(message))
				}
			}
		}
	done:

		logger.Info("Completed WebSocket stream reading", "message_count", len(streamedMessages))
		if mode == FlashblocksStreamMode_Follower {
			require.Equal(t, len(streamedMessages), 0, "follower should not receive any messages")
			return
		}

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
