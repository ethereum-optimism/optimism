package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/tools"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/urfave/cli/v2"
)

var (
	InvalidProposerTraceTypeFlag = &cli.StringFlag{
		Name:    "trace-type",
		Usage:   "Trace type to create.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "TRACE_TYPE"),
		Value:   config.TraceTypeCannon.String(),
	}
	InvalidProposerProposalIntervalFlag = &cli.DurationFlag{
		Name:    "proposal-interval",
		Usage:   "Interval between creating invalid proposals",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "PROPOSAL_INTERVAL"),
		Value:   24 * time.Hour,
	}
)

func InvalidProposer(ctx *cli.Context, _ context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	logger, err := setupLogging(ctx)
	if err != nil {
		return nil, err
	}
	traceType := ctx.Uint64(TraceTypeFlag.Name)
	interval := ctx.Duration(InvalidProposerProposalIntervalFlag.Name)
	rollupRpc := ctx.String(flags.RollupRpcFlag.Name)

	if rollupRpc == "" {
		return nil, fmt.Errorf("missing %v", flags.RollupRpcFlag.Name)
	}

	contract, txMgr, err := NewContractWithTxMgr[*contracts.DisputeGameFactoryContract](ctx, flags.FactoryAddressFlag.Name, contracts.NewDisputeGameFactoryContract)
	if err != nil {
		return nil, fmt.Errorf("failed to create dispute game factory bindings: %w", err)
	}

	rollupClient, err := dial.DialRollupClientWithTimeout(ctx.Context, dial.DefaultDialTimeout, logger, rollupRpc)
	if err != nil {
		return nil, err
	}
	creator := tools.NewGameCreator(contract, txMgr)
	proposer := tools.NewInvalidProposer(logger, creator, rollupClient, traceType, interval, txMgr)

	return proposer, nil
}

func invalidProposerFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		flags.RollupRpcFlag,
		flags.FactoryAddressFlag,
		InvalidProposerTraceTypeFlag,
		InvalidProposerProposalIntervalFlag,
	}
	cliFlags = append(cliFlags, txmgr.CLIFlagsWithDefaults(flags.EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	cliFlags = append(cliFlags, oplog.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var InvalidProposerCommand = &cli.Command{
	Name:   "invalid-proposer",
	Usage:  "Periodically creates a dispute game with an invalid output root proposal",
	Action: cliapp.LifecycleCmd(InvalidProposer),
	Flags:  invalidProposerFlags(),
}
