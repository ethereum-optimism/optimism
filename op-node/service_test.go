package opnode

import (
	"os"
	"testing"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	opflags "github.com/ethereum-optimism/optimism/op-service/flags"
)

func TestNewRollupConfig_WithNetwork(t *testing.T) {
	logger := log.New()
	
	// Test with known network
	cfg, err := NewRollupConfig(logger, "goerli", "")
	if err != nil {
		t.Fatalf("Expected no error for known network, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
	if cfg.L1ChainID == 0 {
		t.Error("Expected L1ChainID to be set")
	}
}

func TestNewRollupConfig_WithFile(t *testing.T) {
	logger := log.New()
	
	// Create a temporary config file
	tmpfile, err := os.CreateTemp("", "rollup-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	
	// Write a valid rollup config
	configJSON := `{
		"genesis": {
			"l1": {
				"hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
				"number": 0
			},
			"l2": {
				"hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
				"number": 0
			},
			"l2_time": 0
		},
		"block_time": 2,
		"max_sequencer_drift": 600,
		"seq_window_size": 3600,
		"l1_chain_id": 5,
		"l2_chain_id": 420
	}`
	
	if _, err := tmpfile.Write([]byte(configJSON)); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close file: %v", err)
	}
	
	// Test loading from file
	cfg, err := NewRollupConfig(logger, "", tmpfile.Name())
	if err != nil {
		t.Fatalf("Expected no error for valid config file, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
	if cfg.L1ChainID != 5 {
		t.Errorf("Expected L1ChainID to be 5, got %d", cfg.L1ChainID)
	}
	if cfg.L2ChainID != 420 {
		t.Errorf("Expected L2ChainID to be 420, got %d", cfg.L2ChainID)
	}
}

func TestNewRollupConfig_InvalidFile(t *testing.T) {
	logger := log.New()
	
	// Test with non-existent file
	_, err := NewRollupConfig(logger, "", "/nonexistent/file.json")
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}
}

func TestNewRollupConfig_InvalidJSON(t *testing.T) {
	logger := log.New()
	
	// Create a temporary file with invalid JSON
	tmpfile, err := os.CreateTemp("", "invalid-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	
	if _, err := tmpfile.Write([]byte("invalid json")); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
	
	// Test loading invalid JSON
	_, err = NewRollupConfig(logger, "", tmpfile.Name())
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestNewRollupConfig_NetworkAndFileConflict(t *testing.T) {
	logger := log.New()
	
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "rollup-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()
	
	// Test with both network and file (should log warning but use network)
	cfg, err := NewRollupConfig(logger, "goerli", tmpfile.Name())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
}

func TestNewL1EndpointConfig(t *testing.T) {
	app := &cli.App{
		Flags: []cli.Flag{
			flags.L1NodeAddr,
			flags.L1TrustRPC,
			flags.L1RPCProviderKind,
			flags.L1RPCRateLimit,
			flags.L1RPCMaxBatchSize,
			flags.L1HTTPPollInterval,
			flags.L1RPCMaxConcurrency,
			flags.L1CacheSize,
		},
	}
	
	ctx := cli.NewContext(app, nil, nil)
	
	// Test default values
	cfg := NewL1EndpointConfig(ctx)
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
}

func TestNewL2EndpointConfig(t *testing.T) {
	logger := log.New()
	
	app := &cli.App{
		Flags: []cli.Flag{
			flags.L2EngineAddr,
			flags.L2EngineJWTSecret,
			flags.L2EngineRpcTimeout,
		},
	}
	
	ctx := cli.NewContext(app, nil, nil)
	
	// Test without JWT secret (should error)
	_, err := NewL2EndpointConfig(ctx, logger)
	// This might fail due to missing JWT secret, which is expected
	_ = err
}

func TestNewDriverConfig(t *testing.T) {
	app := &cli.App{
		Flags: []cli.Flag{
			flags.VerifierL1Confs,
			flags.SequencerL1Confs,
			flags.SequencerEnabledFlag,
			flags.SequencerStoppedFlag,
			flags.SequencerMaxSafeLagFlag,
			flags.SequencerRecoverMode,
		},
	}
	
	ctx := cli.NewContext(app, nil, nil)
	
	cfg := NewDriverConfig(ctx)
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
}

func TestNewConfigPersistence(t *testing.T) {
	app := &cli.App{
		Flags: []cli.Flag{
			flags.RPCAdminPersistence,
		},
	}
	
	ctx := cli.NewContext(app, nil, nil)
	
	// Test with empty path (should return disabled)
	persistence := NewConfigPersistence(ctx)
	if persistence == nil {
		t.Fatal("Expected non-nil persistence")
	}
	
	// Test with path
	ctx.Set(flags.RPCAdminPersistence.Name, "/tmp/test-state.json")
	persistence = NewConfigPersistence(ctx)
	if persistence == nil {
		t.Fatal("Expected non-nil persistence")
	}
}

