package filter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-interop-filter/flags"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

// DefaultMessageExpiryWindow is 7 days, matching op-supervisor's default
const DefaultMessageExpiryWindow = 7 * 24 * time.Hour

type Config struct {
	L2RPCs                      []string
	RollupConfigs               map[eth.ChainID]*rollup.Config // Rollup configs keyed by chain ID
	DataDir                     string
	BackfillDuration            time.Duration
	MessageExpiryWindow         uint64 // Message expiry window in seconds (default: 7 days)
	MessageExpiryWindowExplicit bool   // True if explicitly set via flag
	JWTSecretPath               string
	Version                     string
	PollInterval                time.Duration // Interval for polling new blocks (default: 2s)
	ValidationInterval          time.Duration // Interval for cross-chain validation (default: 500ms)

	LogConfig     oplog.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
	RPC           oprpc.CLIConfig
}

func (c *Config) Check() error {
	var result error
	if len(c.L2RPCs) == 0 {
		result = errors.Join(result, errors.New("at least one L2 RPC is required"))
	}
	if len(c.RollupConfigs) == 0 {
		result = errors.Join(result, errors.New("at least one rollup config is required (use --networks or --rollup-configs)"))
	}
	// Admin API must be JWT protected.
	if c.RPC.EnableAdmin && c.JWTSecretPath == "" {
		result = errors.Join(result, errors.New("rpc.enable-admin requires admin.jwt-secret for authentication"))
	}
	// Durations must be positive
	if c.BackfillDuration <= 0 {
		result = errors.Join(result, errors.New("backfill-duration must be positive"))
	}
	if c.MessageExpiryWindow == 0 {
		result = errors.Join(result, errors.New("message-expiry-window must be positive"))
	}
	if c.PollInterval <= 0 {
		result = errors.Join(result, errors.New("poll-interval must be positive"))
	}
	if c.ValidationInterval <= 0 {
		result = errors.Join(result, errors.New("validation-interval must be positive"))
	}
	result = errors.Join(result, c.MetricsConfig.Check())
	result = errors.Join(result, c.PprofConfig.Check())
	result = errors.Join(result, c.RPC.Check())
	return result
}

func NewConfig(ctx *cli.Context, version string) (*Config, error) {
	backfillDuration, err := time.ParseDuration(ctx.String(flags.BackfillDurationFlag.Name))
	if err != nil {
		return nil, fmt.Errorf("invalid backfill-duration: %w", err)
	}
	if backfillDuration <= 0 {
		return nil, fmt.Errorf("backfill-duration must be positive, got %s", backfillDuration)
	}
	if uint64(backfillDuration.Seconds()) > uint64(time.Now().Unix()) {
		return nil, fmt.Errorf("backfill-duration (%s) exceeds current timestamp", backfillDuration)
	}

	messageExpiryWindow, err := time.ParseDuration(ctx.String(flags.MessageExpiryWindowFlag.Name))
	if err != nil {
		return nil, fmt.Errorf("invalid message-expiry-window: %w", err)
	}
	if messageExpiryWindow <= 0 {
		return nil, fmt.Errorf("message-expiry-window must be positive, got %s", messageExpiryWindow)
	}

	pollInterval, err := time.ParseDuration(ctx.String(flags.PollIntervalFlag.Name))
	if err != nil {
		return nil, fmt.Errorf("invalid poll-interval: %w", err)
	}
	if pollInterval <= 0 {
		return nil, fmt.Errorf("poll-interval must be positive, got %s", pollInterval)
	}

	validationInterval, err := time.ParseDuration(ctx.String(flags.ValidationIntervalFlag.Name))
	if err != nil {
		return nil, fmt.Errorf("invalid validation-interval: %w", err)
	}
	if validationInterval <= 0 {
		return nil, fmt.Errorf("validation-interval must be positive, got %s", validationInterval)
	}

	// Load rollup configs from --networks and --rollup-configs
	rollupConfigs, err := loadRollupConfigs(
		ctx.StringSlice(flags.NetworksFlag.Name),
		ctx.StringSlice(flags.RollupConfigsFlag.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load rollup configs: %w", err)
	}

	return &Config{
		L2RPCs:                      ctx.StringSlice(flags.L2RPCsFlag.Name),
		RollupConfigs:               rollupConfigs,
		DataDir:                     ctx.String(flags.DataDirFlag.Name),
		BackfillDuration:            backfillDuration,
		MessageExpiryWindow:         uint64(messageExpiryWindow.Seconds()),
		MessageExpiryWindowExplicit: ctx.IsSet(flags.MessageExpiryWindowFlag.Name),
		JWTSecretPath:               ctx.String(flags.JWTSecretFlag.Name),
		Version:                     version,
		PollInterval:                pollInterval,
		ValidationInterval:          validationInterval,
		LogConfig:                   oplog.ReadCLIConfig(ctx),
		MetricsConfig:               opmetrics.ReadCLIConfig(ctx),
		PprofConfig:                 oppprof.ReadCLIConfig(ctx),
		RPC:                         oprpc.ReadCLIConfig(ctx),
	}, nil
}

// loadRollupConfigs loads rollup configs from networks (superchain registry) and custom JSON files.
func loadRollupConfigs(networks []string, configPaths []string) (map[eth.ChainID]*rollup.Config, error) {
	configs := make(map[eth.ChainID]*rollup.Config)

	// Load from superchain registry by network name
	for _, network := range networks {
		cfg, err := chaincfg.GetRollupConfig(network)
		if err != nil {
			return nil, fmt.Errorf("failed to load rollup config for network %q: %w", network, err)
		}
		chainID := eth.ChainIDFromBig(cfg.L2ChainID)
		if _, exists := configs[chainID]; exists {
			return nil, fmt.Errorf("duplicate chain ID %s: network %q conflicts with another config", chainID, network)
		}
		configs[chainID] = cfg
	}

	// Load from custom JSON files
	for _, path := range configPaths {
		cfg, err := loadRollupConfigFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load rollup config from %q: %w", path, err)
		}
		chainID := eth.ChainIDFromBig(cfg.L2ChainID)
		if _, exists := configs[chainID]; exists {
			return nil, fmt.Errorf("duplicate chain ID %s: file %q conflicts with another config", chainID, path)
		}
		configs[chainID] = cfg
	}

	return configs, nil
}

// loadRollupConfigFromFile loads a rollup config from a JSON file.
func loadRollupConfigFromFile(path string) (*rollup.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var cfg rollup.Config
	dec := json.NewDecoder(file)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}
	return &cfg, nil
}
