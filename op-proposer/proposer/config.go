package proposer

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-proposer/flags"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	ErrMissingRollupRpc    = errors.New("missing rollup rpc")
	ErrMissingSuperRootRpc = errors.New("missing super root rpc")
	ErrMissingSource       = errors.New("missing proposal source rpc (rollup or super root)")
	ErrConflictingSource   = errors.New("must specify exactly one of rollup rpc or super root rpc")
	ErrUnsupportedGameType = errors.New("unsupported game type")

	// outputRootGameTypes are the game types op-proposer proposes output roots for, read from a rollup RPC.
	// PERMISSIONED_CANNON (1) is the guardian fallback of every CANNON_KONA (8) deployment, and FAST (254)
	// is the e2e withdrawal game.
	outputRootGameTypes = []uint32{1, 8, 254}

	// superRootGameTypes are the game types op-proposer proposes super roots for, read from super root RPCs.
	superRootGameTypes = []uint32{5, 9}
)

// CLIConfig is a well typed config that is parsed from the CLI params.
// This also contains config options for auxiliary services.
// It is transformed into a `Config` before the L2 output submitter is started.
type CLIConfig struct {
	/* Required Params */

	// L1EthRpc is the HTTP provider URL for L1.
	L1EthRpc string

	// RollupRpc is the HTTP provider URL for the rollup node. A comma-separated list enables the active rollup provider.
	RollupRpc string

	// SuperRootRpcs is the list of HTTP provider URLs for super root RPC sources.
	// Mutually exclusive with RollupRpc.
	SuperRootRpcs []string

	// PollInterval is the delay between periodic checks on whether it is time to load an output root and propose it.
	PollInterval time.Duration

	// AllowNonFinalized can be set to true to propose outputs
	// for L2 blocks derived from non-finalized L1 data.
	AllowNonFinalized bool

	TxMgrConfig txmgr.CLIConfig

	RPCConfig oprpc.CLIConfig

	LogConfig logcli.CLIConfig

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
	usesSuperRoots := slices.Contains(superRootGameTypes, c.DisputeGameType)
	if !usesSuperRoots && !slices.Contains(outputRootGameTypes, c.DisputeGameType) {
		supported := slices.Sorted(slices.Values(slices.Concat(outputRootGameTypes, superRootGameTypes)))
		return fmt.Errorf("%w %d: set --game-type (OP_PROPOSER_GAME_TYPE) to the chain's respected game type, one of %v", ErrUnsupportedGameType, c.DisputeGameType, supported)
	}
	if c.RollupRpc != "" && len(c.SuperRootRpcs) != 0 {
		return ErrConflictingSource
	}
	if usesSuperRoots && len(c.SuperRootRpcs) == 0 {
		return ErrMissingSuperRootRpc
	}
	if !usesSuperRoots && c.RollupRpc == "" {
		return ErrMissingRollupRpc
	}

	return nil
}

// NewConfig parses the Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context) *CLIConfig {
	return &CLIConfig{
		L1EthRpc:                     ctx.String(flags.L1EthRpcFlag.Name),
		RollupRpc:                    ctx.String(flags.RollupRpcFlag.Name),
		SuperRootRpcs:                ctx.StringSlice(flags.SuperRootRpcsFlag.Name),
		PollInterval:                 ctx.Duration(flags.PollIntervalFlag.Name),
		TxMgrConfig:                  txmgr.ReadCLIConfig(ctx),
		AllowNonFinalized:            ctx.Bool(flags.AllowNonFinalizedFlag.Name),
		RPCConfig:                    oprpc.ReadCLIConfig(ctx),
		LogConfig:                    logcli.ReadCLIConfig(ctx),
		MetricsConfig:                opmetrics.ReadCLIConfig(ctx),
		PprofConfig:                  oppprof.ReadCLIConfig(ctx),
		DGFAddress:                   ctx.String(flags.DisputeGameFactoryAddressFlag.Name),
		ProposalInterval:             ctx.Duration(flags.ProposalIntervalFlag.Name),
		DisputeGameType:              uint32(ctx.Uint(flags.DisputeGameTypeFlag.Name)),
		ActiveSequencerCheckDuration: ctx.Duration(flags.ActiveSequencerCheckDurationFlag.Name),
		WaitNodeSync:                 ctx.Bool(flags.WaitNodeSyncFlag.Name),
	}
}
