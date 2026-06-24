package filter

import (
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
)

// DefaultMessageExpiryWindow is 7 days, matching the interop message expiry default.
const DefaultMessageExpiryWindow = 7 * 24 * time.Hour

type Config struct {
	L2RPCs                      []string
	RollupConfigs               map[eth.ChainID]*rollup.Config // Rollup configs keyed by chain ID
	DataDir                     string
	BackfillDuration            time.Duration
	MessageExpiryWindow         uint64 // Message expiry window in seconds (default: 7 days)
	MessageExpiryWindowExplicit bool   // True if explicitly set via flag
	JWTSecretPath               string
	RPCAddr                     string // Address for public RPC server
	RPCPort                     int    // Port for public RPC server (default: 8545)
	AdminRPCAddr                string // Address for admin RPC server (empty = disabled)
	AdminRPCPort                int    // Port for admin RPC server (default: 8546)
	Version                     string
	PollInterval                time.Duration // Interval for polling new blocks (default: 2s)
	ValidationInterval          time.Duration // Interval for cross-chain validation (default: 500ms)
	FailsafeLogInterval         time.Duration // Interval for re-logging the active failsafe reason (default: 1m)
	ReorgRecoveryEnabled        bool          // If true, automatically rewinds reorg-triggered failsafe to finalized
	Passthrough                 bool          // If true, all transactions pass through without filtering
	LegacyCheckAccessListFormat bool          // If true, allows access list requests that omit executing chainID
	RPCConcurrency              int           // Max concurrent RPC requests per chain (default: 100)
	FetchConcurrency            int           // Number of blocks to fetch concurrently (default: 64)

	LogConfig     oplog.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
}

func (c *Config) Check() error {
	var result error
	if len(c.L2RPCs) == 0 {
		result = errors.Join(result, errors.New("at least one L2 RPC is required"))
	}
	if len(c.RollupConfigs) == 0 {
		result = errors.Join(result, errors.New("at least one rollup config is required (use --networks or --rollup-configs)"))
	}
	// Admin RPC requires JWT secret for authentication.
	if c.AdminRPCAddr != "" && c.JWTSecretPath == "" {
		result = errors.Join(result, errors.New("admin.rpc.addr requires admin.jwt-secret for authentication"))
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
	// FailsafeLogInterval is intentionally not required: a zero value means
	// "use the default" and is defaulted to defaultFailsafeLogInterval by the
	// backend. The CLI flag defaults to 1m, and callers that build Config
	// directly (e.g. tests) may leave it unset.
	if c.RPCConcurrency <= 0 {
		result = errors.Join(result, errors.New("rpc-concurrency must be positive"))
	}
	if c.FetchConcurrency <= 0 {
		result = errors.Join(result, errors.New("fetch-concurrency must be positive"))
	}
	if c.FetchConcurrency > c.RPCConcurrency {
		result = errors.Join(result, errors.New("fetch-concurrency must be less than or equal to rpc-concurrency"))
	}
	result = errors.Join(result, c.MetricsConfig.Check())
	result = errors.Join(result, c.PprofConfig.Check())
	return result
}

func NewConfig(ctx *cli.Context, version string) (*Config, error) {
	backfillDuration := ctx.Duration(flags.BackfillDurationFlag.Name)
	if backfillDuration <= 0 {
		return nil, fmt.Errorf("backfill-duration must be positive, got %s", backfillDuration)
	}
	if uint64(backfillDuration.Seconds()) > uint64(time.Now().Unix()) {
		return nil, fmt.Errorf("backfill-duration (%s) exceeds current timestamp", backfillDuration)
	}

	messageExpiryWindow := ctx.Duration(flags.MessageExpiryWindowFlag.Name)
	if messageExpiryWindow <= 0 {
		return nil, fmt.Errorf("message-expiry-window must be positive, got %s", messageExpiryWindow)
	}

	pollInterval := ctx.Duration(flags.PollIntervalFlag.Name)
	if pollInterval <= 0 {
		return nil, fmt.Errorf("poll-interval must be positive, got %s", pollInterval)
	}

	validationInterval := ctx.Duration(flags.ValidationIntervalFlag.Name)
	if validationInterval <= 0 {
		return nil, fmt.Errorf("validation-interval must be positive, got %s", validationInterval)
	}
	failsafeLogInterval := ctx.Duration(flags.FailsafeLogIntervalFlag.Name)
	if failsafeLogInterval <= 0 {
		return nil, fmt.Errorf("failsafe-log-interval must be positive, got %s", failsafeLogInterval)
	}
	rpcConcurrency := ctx.Int(flags.RPCConcurrencyFlag.Name)
	if rpcConcurrency <= 0 {
		return nil, fmt.Errorf("rpc-concurrency must be positive, got %d", rpcConcurrency)
	}
	fetchConcurrency := ctx.Int(flags.FetchConcurrencyFlag.Name)
	if fetchConcurrency <= 0 {
		return nil, fmt.Errorf("fetch-concurrency must be positive, got %d", fetchConcurrency)
	}
	if fetchConcurrency > rpcConcurrency {
		return nil, fmt.Errorf("fetch-concurrency (%d) must be less than or equal to rpc-concurrency (%d)", fetchConcurrency, rpcConcurrency)
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
		RPCAddr:                     ctx.String(flags.RPCAddrFlag.Name),
		RPCPort:                     ctx.Int(flags.RPCPortFlag.Name),
		AdminRPCAddr:                ctx.String(flags.AdminRPCAddrFlag.Name),
		AdminRPCPort:                ctx.Int(flags.AdminRPCPortFlag.Name),
		Version:                     version,
		PollInterval:                pollInterval,
		ValidationInterval:          validationInterval,
		FailsafeLogInterval:         failsafeLogInterval,
		ReorgRecoveryEnabled:        ctx.Bool(flags.ReorgRecoveryEnabledFlag.Name),
		Passthrough:                 ctx.Bool(flags.DangerouslyEnablePassthroughFlag.Name),
		LegacyCheckAccessListFormat: ctx.Bool(flags.SupportLegacyCheckAccessListFormatFlag.Name),
		RPCConcurrency:              rpcConcurrency,
		FetchConcurrency:            fetchConcurrency,
		LogConfig:                   oplog.ReadCLIConfig(ctx),
		MetricsConfig:               opmetrics.ReadCLIConfig(ctx),
		PprofConfig:                 oppprof.ReadCLIConfig(ctx),
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

	var rollupConfig rollup.Config
	return &rollupConfig, rollupConfig.ParseRollupConfig(file)
}
