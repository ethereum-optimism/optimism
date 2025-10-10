package opnode

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	nodeflags "github.com/ethereum-optimism/optimism/op-node/flags"
)

func TestNewL1ChainConfig_KnownChains(t *testing.T) {
	logger := log.New()
	app := cli.NewApp()
	ctx := cli.NewContext(app, nil, nil)

	t.Run("mainnet", func(t *testing.T) {
		cfg, err := NewL1ChainConfig(new(big.Int).Set(params.MainnetChainConfig.ChainID), ctx, logger)
		require.NoError(t, err)
		require.Equal(t, params.MainnetChainConfig, cfg)
	})

	t.Run("sepolia", func(t *testing.T) {
		cfg, err := NewL1ChainConfig(new(big.Int).Set(params.SepoliaChainConfig.ChainID), ctx, logger)
		require.NoError(t, err)
		require.Equal(t, params.SepoliaChainConfig, cfg)
	})
}

func TestNewL1ChainConfig_CustomDirectAndEmbeddedAndNil(t *testing.T) {
	logger := log.New()

	testChainID := big.NewInt(424242)

	// Build a minimal custom ChainConfig
	custom := &params.ChainConfig{
		ChainID:            testChainID,
		BlobScheduleConfig: &params.BlobScheduleConfig{},
	}

	customFaulty := &params.ChainConfig{
		ChainID:            testChainID,
		BlobScheduleConfig: nil,
	}

	// Prepare temp dir
	dir := t.TempDir()

	encode := func(path string, cfg any) {
		f, err := os.Create(path)
		require.NoError(t, err)
		enc := json.NewEncoder(f)
		err = enc.Encode(cfg)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}

	// Direct JSON file containing a ChainConfig
	directPath := filepath.Join(dir, "chainconfig.json")
	encode(directPath, custom)

	directFaultyPath := filepath.Join(dir, "chainconfig_faulty.json")
	encode(directFaultyPath, customFaulty)

	// Embedded JSON file that contains { "config": <ChainConfig> }
	embeddedPath := filepath.Join(dir, "genesis_like.json")
	type wrapper struct {
		Config *params.ChainConfig `json:"config"`
	}
	encode(embeddedPath, wrapper{Config: custom})

	// Helper to run the CLI with a given file path
	runWithPath := func(path string) (*params.ChainConfig, error) {
		app := cli.NewApp()
		app.Flags = []cli.Flag{nodeflags.L1ChainConfig}
		var out *params.ChainConfig
		app.Action = func(ctx *cli.Context) error {
			cfg, err := NewL1ChainConfig(testChainID, ctx, logger)
			out = cfg
			return err
		}
		// run with arg: --rollup.l1-chain-config <path>
		err := app.Run([]string{"op-node", "--" + nodeflags.L1ChainConfig.Name, path})
		return out, err
	}

	t.Run("custom-direct", func(t *testing.T) {
		cfg, err := runWithPath(directPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, custom.ChainID, cfg.ChainID)
	})

	t.Run("custom-embedded", func(t *testing.T) {
		cfg, err := runWithPath(embeddedPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, custom.ChainID, cfg.ChainID)
	})

	t.Run("nil-chainid-panics", func(t *testing.T) {
		app := cli.NewApp()
		ctx := cli.NewContext(app, nil, nil)
		require.Panics(t, func() {
			_, _ = NewL1ChainConfig(nil, ctx, logger)
		})
	})

	t.Run("nil-blob-schedule-config-returns-error", func(t *testing.T) {
		cfg, err := runWithPath(directFaultyPath)
		require.Nil(t, cfg)
		require.Error(t, err)
	})
}
