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

type Config struct {
	L2RPCs           []string
	DataDir          string
	BackfillDuration time.Duration
	JWTSecretPath    string
	Version          string

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
	// Admin API requires JWT authentication
	if c.RPC.EnableAdmin && c.JWTSecretPath == "" {
		result = errors.Join(result, errors.New("admin RPC requires JWT setup, but no JWT path was specified"))
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

	return &Config{
		L2RPCs:           ctx.StringSlice(flags.L2RPCsFlag.Name),
		DataDir:          ctx.String(flags.DataDirFlag.Name),
		BackfillDuration: backfillDuration,
		JWTSecretPath:    ctx.String(flags.JWTSecretFlag.Name),
		Version:          version,
		LogConfig:        oplog.ReadCLIConfig(ctx),
		MetricsConfig:    opmetrics.ReadCLIConfig(ctx),
		PprofConfig:      oppprof.ReadCLIConfig(ctx),
		RPC:              oprpc.ReadCLIConfig(ctx),
	}, nil
}
