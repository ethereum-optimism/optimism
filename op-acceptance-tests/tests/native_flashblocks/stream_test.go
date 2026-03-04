package native_flashblocks

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
	"github.com/ethereum-optimism/optimism/op-service/logmods"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/stretchr/testify/require"
)

// Flashblock matches the wire format produced by native op-reth flashblocks (snake_case JSON).
type Flashblock struct {
	PayloadID string `json:"payload_id"`
	Index     int    `json:"index"`
	Diff      struct {
		StateRoot    string `json:"state_root"`
		ReceiptsRoot string `json:"receipts_root"`
		LogsBloom    string `json:"logs_bloom"`
		GasUsed      int    `json:"gas_used"`
		BlockHash    string `json:"block_hash"`
		Transactions []any  `json:"transactions"`
	} `json:"diff"`
	Metadata json.RawMessage `json:"metadata"`
}

// TestNativeFlashblocksStream verifies that op-reth's native flashblocks WS endpoint
// produces a valid stream of flashblocks without any external builder or rollup-boost.
func TestNativeFlashblocksStream(gt *testing.T) {
	t := devtest.SerialT(gt)
	logger := t.Logger()
	sys := presets.NewSingleChainWithNativeFlashblocks(t)
	filterHandler, ok := logmods.FindHandler[logfilter.FilterHandler](logger.Handler())
	if ok {
		filterHandler.Set(logfilter.DefaultMute(
			logfilter.Level(slog.LevelError).Show(),
			logfilter.Select("kind", "L2CLNode").Show(),
		))
	}
	ctx := t.Ctx()

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	require.NotEmpty(t, sys.FlashblocksWSURL, "native flashblocks WS URL must be set")
	logger.Info("Native flashblocks WS endpoint", "url", sys.FlashblocksWSURL)

	// Drive a few blocks via test sequencer to generate payload builds.
	driveBlocks(t, sys, 3)

	// Connect to the native flashblocks WS endpoint.
	wsClient := sys.FlashblocksClient(t)

	// Listen for flashblocks for a window.
	testDuration := 5 * time.Second
	output := make(chan []byte, 50)
	defer close(output)
	done := make(chan struct{})

	go func() {
		err := wsClient.ReadAll(ctx, logger.With("stream_source", "native-reth"), testDuration, output, done)
		require.NoError(t, err)
	}()

	messages := make([]string, 0)
	listening := true
	for listening {
		select {
		case <-done:
			listening = false
		case msg := <-output:
			messages = append(messages, string(msg))
		}
	}

	logger.Info("Native flashblocks stream results", "msg_count", len(messages))
	require.Greater(t, len(messages), 0, "should have received at least one flashblock from native WS")

	// Validate each message deserializes as a flashblock.
	for i, msg := range messages {
		var fb Flashblock
		err := json.Unmarshal([]byte(msg), &fb)
		require.NoError(t, err, "message %d must be valid flashblock JSON", i)
		require.NotEmpty(t, fb.PayloadID, "message %d must have payload_id", i)
		require.GreaterOrEqual(t, fb.Index, 0, "message %d index must be >= 0", i)
		logger.Debug("Flashblock received", "index", fb.Index, "payload_id", fb.PayloadID, "gas_used", fb.Diff.GasUsed)
	}

	logger.Info("Native flashblocks stream validation completed", "total_messages", len(messages))
}

// driveBlocks explicitly builds a few blocks to ensure the sequencer has payloads to produce
// flashblocks from before we start listening.
func driveBlocks(t devtest.T, sys *presets.SingleChainWithNativeFlashblocks, count int) {
	t.Helper()
	ts := sys.TestSequencer.Escape().ControlAPI(sys.L2Chain.ChainID())
	ctx := t.Ctx()

	head := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	for i := 0; i < count; i++ {
		require.NoError(t, ts.New(ctx, seqtypes.BuildOpts{Parent: head.Hash}))
		require.NoError(t, ts.Next(ctx))
		head = sys.L2EL.BlockRefByLabel(eth.Unsafe)
	}
	sys.L2EL.WaitForBlockNumber(1)

	head = sys.L2EL.BlockRefByLabel(eth.Unsafe)
	sys.Log.Info("Pre-listen unsafe head", "unsafe", head)
}
