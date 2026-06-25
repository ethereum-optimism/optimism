package proposer

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-proposer/flags"
	opflags "github.com/ethereum-optimism/optimism/op-service/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	ErrMissingRollupRpc    = errors.New("missing rollup rpc")
	ErrMissingSuperNodeRpc = errors.New("missing supernode rpc")
	ErrMissingSource       = errors.New("missing proposal source rpc (rollup or supernode)")
	ErrConflictingSource   = errors.New("must specify exactly one of rollup rpc or supernode rpc")

	// preInteropGameTypes are game types that enforce having a rollup rpc.
	// It is ok if this list isn't complete, unknown game types will allow either rollup or supernode.
	// We just want to reduce foot-guns during the migration period.
	preInteropGameTypes = []uint32{0, 1, 2, 3, 6, 254, 255, 1337}

	// postInteropGameTypes are game types that enforce having a supernode rpc.
	// It is ok if this list isn't complete, unknown game types will allow either rollup or supernode.
	// We just want to reduce foot-guns during the migration period.
	postInteropGameTypes = []uint32{4, 5}
)

const DisputeGameTypeAuto = "auto"

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

	// SystemConfigAddress is the SystemConfig contract address.
	SystemConfigAddress string

	// Network is a predefined superchain registry network name.
	Network string

	// ProposalInterval is the delay between submitting L2 output proposals when the DGFAddress is set.
	ProposalInterval time.Duration

	// DisputeGameType is the type of dispute game to create when submitting an output proposal.
	DisputeGameType uint32

	// DisputeGameTypeAuto enables resolving the current respected game type from L1.
	DisputeGameTypeAuto bool

	// DisputeGameTypeRaw is the CLI value before parsing.
	DisputeGameTypeRaw string

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

	if c.Network != "" {
		chain := chaincfg.ChainByName(c.Network)
		if chain == nil {
			return fmt.Errorf("unknown network %q", c.Network)
		}
	}
	hasDGFSource := c.DGFAddress != "" || c.SystemConfigAddress != "" || c.Network != ""
	if !hasDGFSource {
		return errors.New("`DisputeGameFactory`, `SystemConfig`, or `network` is required")
	}
	if hasDGFSource && c.ProposalInterval == 0 {
		return errors.New("the `DisputeGameFactory`, `SystemConfig`, or `network` was provided but the `ProposalInterval` was not set")
	}
	if c.ProposalInterval != 0 && !hasDGFSource {
		return errors.New("the `ProposalInterval` was provided but none of `DisputeGameFactory`, `SystemConfig`, or `network` was set")
	}
	gameType, gameTypeAuto, err := c.ResolveDisputeGameType()
	if err != nil {
		return err
	}
	if gameTypeAuto && c.SystemConfigAddress == "" && c.Network == "" {
		return errors.New("`SystemConfig` or `network` is required when `game-type` is auto")
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
	// Require rollup RPC for pre interop game types
	if hasDGFSource && !gameTypeAuto && slices.Contains(preInteropGameTypes, gameType) && c.RollupRpc == "" {
		return ErrMissingRollupRpc
	}
	// Require supernode RPC for post interop game types
	if hasDGFSource && !gameTypeAuto && slices.Contains(postInteropGameTypes, gameType) && len(c.SuperNodeRpcs) == 0 {
		return ErrMissingSuperNodeRpc
	}
	// For unknown game types, allow any source, but require at least one.
	if sourceCount == 0 {
		return ErrMissingSource
	}

	return nil
}

func (c *CLIConfig) ResolveDisputeGameType() (uint32, bool, error) {
	value := strings.TrimSpace(c.DisputeGameTypeRaw)
	if value == "" {
		return c.DisputeGameType, c.DisputeGameTypeAuto, nil
	}
	if strings.EqualFold(value, DisputeGameTypeAuto) {
		return 0, true, nil
	}
	gameType, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, false, fmt.Errorf("invalid `game-type` %q: expected uint32 or %q", value, DisputeGameTypeAuto)
	}
	return uint32(gameType), false, nil
}

// NewConfig parses the Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context) *CLIConfig {
	gameTypeRaw := ctx.String(flags.DisputeGameTypeFlag.Name)
	gameType, gameTypeAuto, _ := (&CLIConfig{DisputeGameTypeRaw: gameTypeRaw}).ResolveDisputeGameType()
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
		SystemConfigAddress:          ctx.String(flags.SystemConfigAddressFlag.Name),
		Network:                      ctx.String(opflags.NetworkFlagName),
		ProposalInterval:             ctx.Duration(flags.ProposalIntervalFlag.Name),
		DisputeGameType:              gameType,
		DisputeGameTypeAuto:          gameTypeAuto,
		DisputeGameTypeRaw:           gameTypeRaw,
		ActiveSequencerCheckDuration: ctx.Duration(flags.ActiveSequencerCheckDurationFlag.Name),
		WaitNodeSync:                 ctx.Bool(flags.WaitNodeSyncFlag.Name),
	}
}
