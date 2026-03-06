package devnet

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// SepoliaRPCEndpoint returns an RPC endpoint for Sepolia, using a recording
// file if available, or falling back to SEPOLIA_RPC_URL with recording.
//
// When a recording file exists at recordingPath, requests are served from the
// recording without any network access. When no recording exists and
// SEPOLIA_RPC_URL is set, the proxy records all exchanges for future replay.
//
// Returns the endpoint URL. Call t.Cleanup to stop the proxy.
func SepoliaRPCEndpoint(t *testing.T, lgr log.Logger, recordingPath string) string {
	t.Helper()

	if _, err := os.Stat(recordingPath); err == nil {
		// Replay mode
		recording, err := LoadRecording(recordingPath)
		require.NoError(t, err, "failed to load RPC recording from %s", recordingPath)

		replayer := NewRPCReplayer(lgr, recording)
		require.NoError(t, replayer.Start(), "failed to start RPC replayer")
		t.Cleanup(func() {
			require.NoError(t, replayer.Stop())
		})

		lgr.Info("using recorded Sepolia RPC replay", "path", recordingPath, "entries", len(recording.Entries))
		return replayer.Endpoint()
	}

	// Record mode: need live RPC
	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	require.NotEmpty(t, rpcURL, "recording not found at %s and SEPOLIA_RPC_URL not set", recordingPath)

	recorder := NewRPCRecorder(lgr, rpcURL)
	require.NoError(t, recorder.Start(), "failed to start RPC recorder")
	t.Cleanup(func() {
		recording := recorder.Recording()
		require.NoError(t, recorder.Stop())

		// Save the recording for future replays
		if err := SaveRecording(recording, recordingPath); err != nil {
			t.Logf("WARNING: failed to save RPC recording to %s: %v", recordingPath, err)
		} else {
			t.Logf("RPC recording saved to %s (%d entries)", recordingPath, len(recording.Entries))
		}
	})

	lgr.Info("recording Sepolia RPC exchanges", "path", recordingPath)
	return recorder.Endpoint()
}
