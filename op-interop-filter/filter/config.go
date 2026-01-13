package filter

import (
	"errors"
	"fmt"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-interop-filter/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

// DefaultMessageExpiryWindow is 7 days, matching op-supervisor's default
const DefaultMessageExpiryWindow = 7 * 24 * time.Hour

type Config struct {
	L2RPCs                      []string
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

	return &Config{
		L2RPCs:                      ctx.StringSlice(flags.L2RPCsFlag.Name),
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
