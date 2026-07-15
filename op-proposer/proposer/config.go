package proposer

import (
	"errors"
	"slices"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-proposer/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

type proposalRootFormat uint8

const (
	unknownRootFormat proposalRootFormat = iota
	outputRootFormat
	superRootFormat
)

var (
	ErrMissingRollupRpc  = errors.New("missing rollup rpc")
	ErrMissingSource     = errors.New("missing proposal source rpc (rollup or supernode)")
	ErrConflictingSource = errors.New("must specify exactly one of rollup rpc or supernode rpc")

	outputRootGameTypes = []uint32{0, 1, 2, 3, 6, 8, 254, 255, 1337}
	superRootGameTypes  = []uint32{5, 9}
)

func proposalRootFormatForGameType(gameType uint32) proposalRootFormat {
	switch {
	case slices.Contains(outputRootGameTypes, gameType):
		return outputRootFormat
	case slices.Contains(superRootGameTypes, gameType):
		return superRootFormat
	default:
		return unknownRootFormat
	}
}

// CLIConfig is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is transformed into a `Config` before the L2 output submitter is started.
type CLIConfig struct {
	/* Required Params */

	// L1EthRpc is the HTTP provider URL for L1.
	L1EthRpc string

	// RollupRpc is the HTTP provider URL for the rollup node. A comma-separated list enables the active rollup provider.
	RollupRpc string

	// SuperNodeRpcs is the list of HTTP provider URLs for supernode instances.
	// Mutually exclusive with RollupRpc.
	SuperNodeRpcs []string

	// PollInterval is the delay between periodic checks on whether it is time to load an output root and propose it.
	PollInterval time.Duration

	// AllowNonFinalized can be set to true to propose outputs
	// for L2 blocks derived from non-finalized L1 data.
	AllowNonFinalized bool

	TxMgrConfig txmgr.CLIConfig

	RPCConfig oprpc.CLIConfig

	LogConfig oplog.CLIConfig

	MetricsConfig opmetrics.CLIConfig

	PprofConfig oppprof.CLIConfig

	// DGFAddress is the DisputeGameFactory contract address.
	DGFAddress string

	// ProposalInterval is the delay between submitting L2 output proposals when the DGFAddress is set.
	ProposalInterval time.Duration

	// DisputeGameType is the type of dispute game to create when submitting an output proposal.
	DisputeGameType uint32

	// ActiveSequencerCheckDuration is the duration between checks to determine the active sequencer endpoint.
	ActiveSequencerCheckDuration time.Duration

	// Whether to wait for the sequencer to sync to a recent block at startup.
	WaitNodeSync bool
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
	if err := c.TxMgrConfig.Check(); err != nil {
		return err
	}

	if c.DGFAddress == "" {
		return errors.New("`DisputeGameFactory` is required")
	}
	if c.DGFAddress != "" && c.ProposalInterval == 0 {
		return errors.New("the `DisputeGameFactory` address was provided but the `ProposalInterval` was not set")
	}
	if c.ProposalInterval != 0 && c.DGFAddress == "" {
		return errors.New("the `ProposalInterval` was provided but the `DisputeGameFactory` address was not set")
	}
	// Check for conflicting RPC sources - only one should be specified
	sourceCount := 0
	if c.RollupRpc != "" {
		sourceCount++
	}
	if len(c.SuperNodeRpcs) != 0 {
		sourceCount++
	}
	if sourceCount > 1 {
		return ErrConflictingSource
	}
	if proposalRootFormatForGameType(c.DisputeGameType) == outputRootFormat && c.RollupRpc == "" {
		return ErrMissingRollupRpc
	}
	// All game types require a proposal source.
	if sourceCount == 0 {
		return ErrMissingSource
	}

	return nil
}

// NewConfig parses the Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context) *CLIConfig {
	return &CLIConfig{
		L1EthRpc:                     ctx.String(flags.L1EthRpcFlag.Name),
		RollupRpc:                    ctx.String(flags.RollupRpcFlag.Name),
		SuperNodeRpcs:                ctx.StringSlice(flags.SuperNodeRpcsFlag.Name),
		PollInterval:                 ctx.Duration(flags.PollIntervalFlag.Name),
		TxMgrConfig:                  txmgr.ReadCLIConfig(ctx),
		AllowNonFinalized:            ctx.Bool(flags.AllowNonFinalizedFlag.Name),
		RPCConfig:                    oprpc.ReadCLIConfig(ctx),
		LogConfig:                    oplog.ReadCLIConfig(ctx),
		MetricsConfig:                opmetrics.ReadCLIConfig(ctx),
		PprofConfig:                  oppprof.ReadCLIConfig(ctx),
		DGFAddress:                   ctx.String(flags.DisputeGameFactoryAddressFlag.Name),
		ProposalInterval:             ctx.Duration(flags.ProposalIntervalFlag.Name),
		DisputeGameType:              uint32(ctx.Uint(flags.DisputeGameTypeFlag.Name)),
		ActiveSequencerCheckDuration: ctx.Duration(flags.ActiveSequencerCheckDurationFlag.Name),
		WaitNodeSync:                 ctx.Bool(flags.WaitNodeSyncFlag.Name),
	}
}
