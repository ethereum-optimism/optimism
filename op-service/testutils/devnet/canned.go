package devnet

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
)

type CleanupFunc func() error

func NewForked(lgr log.Logger, rpcURL string, anvilOpts ...AnvilOption) (*Anvil, CleanupFunc, error) {
	retryProxy := NewRetryProxy(lgr, rpcURL)
	if err := retryProxy.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start retry proxy: %w", err)
	}

	anvil, err := NewAnvil(lgr, append([]AnvilOption{WithForkURL(retryProxy.Endpoint()), WithBlockTime(3)}, anvilOpts...)...)
	if err != nil {
		_ = retryProxy.Stop()
		return nil, nil, fmt.Errorf("failed to create Anvil: %w", err)
	}

	if err := anvil.Start(); err != nil {
		_ = retryProxy.Stop()
		return nil, nil, fmt.Errorf("failed to start Anvil: %w", err)
	}

	cleanup := func() error {
		if err := anvil.Stop(); err != nil {
			return fmt.Errorf("failed to stop Anvil: %w", err)
		}
		if err := retryProxy.Stop(); err != nil {
			return fmt.Errorf("failed to stop retry proxy: %w", err)
		}
		return nil
	}

	return anvil, cleanup, nil
}

func NewForkedSepolia(lgr log.Logger) (*Anvil, CleanupFunc, error) {
	url := os.Getenv("SEPOLIA_RPC_URL")
	if url == "" {
		return nil, nil, fmt.Errorf("SEPOLIA_RPC_URL not set")
	}
	return NewForked(lgr, url)
}

func NewForkedSepoliaFromBlock(lgr log.Logger, block uint64) (*Anvil, CleanupFunc, error) {
	url := os.Getenv("SEPOLIA_RPC_URL")
	if url == "" {
		return nil, nil, fmt.Errorf("SEPOLIA_RPC_URL not set")
	}
	return NewForked(lgr, url, WithForkBlockNumber(block))
}

const SepoliaChainID = 11155111

// NewForkedWithRecording forks a chain via RPC while recording all RPC exchanges
// to a file. The recording can later be replayed via NewForkedWithReplay.
func NewForkedWithRecording(lgr log.Logger, rpcURL string, recordingPath string, anvilOpts ...AnvilOption) (*Anvil, CleanupFunc, error) {
	recorder := NewRPCRecorder(lgr, rpcURL)
	if err := recorder.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start recorder: %w", err)
	}

	anvil, err := NewAnvil(lgr, append([]AnvilOption{WithForkURL(recorder.Endpoint()), WithBlockTime(3)}, anvilOpts...)...)
	if err != nil {
		_ = recorder.Stop()
		return nil, nil, fmt.Errorf("failed to create Anvil: %w", err)
	}

	if err := anvil.Start(); err != nil {
		_ = recorder.Stop()
		return nil, nil, fmt.Errorf("failed to start Anvil: %w", err)
	}

	cleanup := func() error {
		anvilErr := anvil.Stop()
		recording := recorder.Recording()
		recorderErr := recorder.Stop()

		// Save recording regardless of errors
		if saveErr := SaveRecording(recording, recordingPath); saveErr != nil {
			lgr.Error("failed to save recording", "err", saveErr)
		} else {
			lgr.Info("RPC recording saved", "path", recordingPath, "entries", len(recording.Entries))
		}

		if anvilErr != nil {
			return anvilErr
		}
		return recorderErr
	}

	return anvil, cleanup, nil
}

// NewForkedWithReplay starts an anvil instance that forks from a recorded RPC
// replay instead of a live RPC endpoint. No network access is needed.
func NewForkedWithReplay(lgr log.Logger, recordingPath string, anvilOpts ...AnvilOption) (*Anvil, CleanupFunc, error) {
	recording, err := LoadRecording(recordingPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load recording: %w", err)
	}

	replayer := NewRPCReplayer(lgr, recording)
	if err := replayer.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start replayer: %w", err)
	}

	anvil, err := NewAnvil(lgr, append([]AnvilOption{WithForkURL(replayer.Endpoint()), WithBlockTime(3)}, anvilOpts...)...)
	if err != nil {
		_ = replayer.Stop()
		return nil, nil, fmt.Errorf("failed to create Anvil: %w", err)
	}

	if err := anvil.Start(); err != nil {
		_ = replayer.Stop()
		return nil, nil, fmt.Errorf("failed to start Anvil: %w", err)
	}

	cleanup := func() error {
		anvilErr := anvil.Stop()
		replayerErr := replayer.Stop()
		if anvilErr != nil {
			return anvilErr
		}
		return replayerErr
	}

	return anvil, cleanup, nil
}

// NewForkedSepoliaFromBlockWithReplay tries to use a recorded RPC replay first.
// If the recording file does not exist and SEPOLIA_RPC_URL is set, it falls back
// to a live fork while recording for future use. If neither is available, it errors.
func NewForkedSepoliaFromBlockWithReplay(lgr log.Logger, block uint64, recordingPath string) (*Anvil, CleanupFunc, error) {
	if _, err := os.Stat(recordingPath); err == nil {
		lgr.Info("using recorded RPC replay", "path", recordingPath, "block", block)
		return NewForkedWithReplay(lgr, recordingPath, WithForkBlockNumber(block))
	}

	url := os.Getenv("SEPOLIA_RPC_URL")
	if url == "" {
		return nil, nil, fmt.Errorf("recording not found at %s and SEPOLIA_RPC_URL not set", recordingPath)
	}

	lgr.Info("recording not found, forking live Sepolia with recording", "path", recordingPath, "block", block)
	return NewForkedWithRecording(lgr, url, recordingPath, WithForkBlockNumber(block))
}

// NewForkedSepoliaWithReplay tries to use a recorded RPC replay first (for latest-block forks).
// Falls back to live fork with recording if no replay file exists.
func NewForkedSepoliaWithReplay(lgr log.Logger, recordingPath string) (*Anvil, CleanupFunc, error) {
	if _, err := os.Stat(recordingPath); err == nil {
		lgr.Info("using recorded RPC replay", "path", recordingPath)
		return NewForkedWithReplay(lgr, recordingPath)
	}

	url := os.Getenv("SEPOLIA_RPC_URL")
	if url == "" {
		return nil, nil, fmt.Errorf("recording not found at %s and SEPOLIA_RPC_URL not set", recordingPath)
	}

	lgr.Info("recording not found, forking live Sepolia with recording", "path", recordingPath)
	return NewForkedWithRecording(lgr, url, recordingPath)
}
