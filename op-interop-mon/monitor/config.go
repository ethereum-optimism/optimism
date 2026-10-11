package monitor

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-interop-mon/flags"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

type CLIConfig struct {
	L2Rpcs            []string
	DependencySetPath string
	PollInterval      time.Duration

	// InteropFilterEndpoint, when set, enables a read-only cross-check against the op-interop-filter.
	InteropFilterEndpoint  string
	InteropFilterMinSafety string

	// SupernodeEndpoints, when set, enable read-only observation of op-supernode liveness,
	// per-chain heads, and cross-safety violations.
	SupernodeEndpoints []string

	// LiquidityBaselineBlocks are <chain-id>=<block> pairs naming the block each chain's
	// ETHLiquidity balance is measured from. Chains not named are measured from genesis.
	LiquidityBaselineBlocks []string

	// ExpiryCheckHead is the head the expiry checks read up to: finalized (the default), or safe.
	ExpiryCheckHead string

	RPCConfig     oprpc.CLIConfig
	LogConfig     logcli.CLIConfig
	MetricsConfig opmetrics.CLIConfig
	PprofConfig   oppprof.CLIConfig
}

func (c *CLIConfig) Check() error {
	if err := c.RPCConfig.Check(); err != nil {
		return err
	}
	if err := c.MetricsConfig.Check(); err != nil {
		return err
	}
	if err := c.PprofConfig.Check(); err != nil {
		return err
	}

	if len(c.L2Rpcs) == 0 {
		return errors.New("l2 rpcs are required")
	}

	if c.DependencySetPath == "" {
		return errors.New("dependency-set is required")
	}

	if _, err := c.LiquidityBaselines(); err != nil {
		return err
	}

	if _, err := c.ExpiryHead(); err != nil {
		return err
	}

	return nil
}

func NewConfig(ctx *cli.Context) *CLIConfig {
	return &CLIConfig{
		// Required Flags
		L2Rpcs:            ctx.StringSlice(flags.L2RpcsFlag.Name),
		DependencySetPath: ctx.String(flags.DependencySetFlag.Name),

		// Optional Flags
		InteropFilterEndpoint:   ctx.String(flags.InteropFilterEndpointFlag.Name),
		InteropFilterMinSafety:  ctx.String(flags.InteropFilterMinSafetyFlag.Name),
		SupernodeEndpoints:      ctx.StringSlice(flags.SupernodeEndpointsFlag.Name),
		LiquidityBaselineBlocks: ctx.StringSlice(flags.LiquidityBaselineBlocksFlag.Name),
		ExpiryCheckHead:         ctx.String(flags.ExpiryCheckHeadFlag.Name),
		RPCConfig:               oprpc.ReadCLIConfig(ctx),
		LogConfig:               logcli.ReadCLIConfig(ctx),
		MetricsConfig:           opmetrics.ReadCLIConfig(ctx),
		PprofConfig:             oppprof.ReadCLIConfig(ctx),
	}
}

// LiquidityBaselines parses LiquidityBaselineBlocks.
func (c *CLIConfig) LiquidityBaselines() (map[eth.ChainID]uint64, error) {
	out := make(map[eth.ChainID]uint64, len(c.LiquidityBaselineBlocks))
	for _, entry := range c.LiquidityBaselineBlocks {
		chain, block, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("eth-liquidity-baseline-blocks entry %q is not <chain-id>=<block>", entry)
		}
		chainID, err := eth.ParseDecimalChainID(chain)
		if err != nil {
			return nil, fmt.Errorf("eth-liquidity-baseline-blocks entry %q: %w", entry, err)
		}
		number, err := strconv.ParseUint(block, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("eth-liquidity-baseline-blocks entry %q: %w", entry, err)
		}
		out[chainID] = number
	}
	return out, nil
}

// ExpiryHead parses ExpiryCheckHead.
func (c *CLIConfig) ExpiryHead() (eth.BlockLabel, error) {
	switch c.ExpiryCheckHead {
	case "", string(eth.Finalized):
		return eth.Finalized, nil
	case string(eth.Safe):
		return eth.Safe, nil
	default:
		return "", fmt.Errorf("expiry-check-head %q unsupported; use %q or %q", c.ExpiryCheckHead, eth.Finalized, eth.Safe)
	}
}
