package main

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/tools"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/urfave/cli/v2"
)

var (
	InvalidProposalTraceTypeFlag = &cli.StringFlag{
		Name:    "trace-type",
		Usage:   "Trace type to create.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "TRACE_TYPE"),
		Value:   config.TraceTypeCannon.String(),
	}
)

func InvalidProposal(ctx *cli.Context) error {
	logger, err := setupLogging(ctx)
	if err != nil {
		return err
	}
	traceType := ctx.Uint64(TraceTypeFlag.Name)
	rollupRpc := ctx.String(flags.RollupRpcFlag.Name)

	if rollupRpc == "" {
		return fmt.Errorf("missing %v", flags.RollupRpcFlag.Name)
	}

	contract, txMgr, err := NewContractWithTxMgr[*contracts.DisputeGameFactoryContract](ctx, flags.FactoryAddressFlag.Name, contracts.NewDisputeGameFactoryContract)
	if err != nil {
		return fmt.Errorf("failed to create dispute game factory bindings: %w", err)
	}

	rollupClient, err := dial.DialRollupClientWithTimeout(ctx.Context, dial.DefaultDialTimeout, logger, rollupRpc)
	if err != nil {
		return err
	}
	creator := tools.NewGameCreator(contract, txMgr)
	proposer := tools.NewInvalidProposer(logger, creator, rollupClient, traceType)
	return proposer.Propose(ctx.Context)
}

func invalidProposalFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		flags.RollupRpcFlag,
		flags.FactoryAddressFlag,
		InvalidProposalTraceTypeFlag,
	}
	cliFlags = append(cliFlags, txmgr.CLIFlagsWithDefaults(flags.EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	cliFlags = append(cliFlags, oplog.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var InvalidProposalCommand = &cli.Command{
	Name:   "invalid-proposal",
	Usage:  "Creates a dispute game with an invalid output root proposal",
	Action: InvalidProposal,
	Flags:  invalidProposalFlags(),
}
