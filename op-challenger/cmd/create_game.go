package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/tools"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var (
	TraceTypeFlag = &cli.StringFlag{
		Name:    "trace-type",
		Usage:   "Trace types to support.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "TRACE_TYPE"),
		Value:   types.TraceTypeCannon.String(),
	}
	OutputRootFlag = &cli.StringFlag{
		Name:    "output-root",
		Usage:   "The output root for the fault dispute game.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "OUTPUT_ROOT"),
	}
	L2BlockNumFlag = &cli.StringFlag{
		Name:    "l2-block-num",
		Usage:   "The l2 block number for the game.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "L2_BLOCK_NUM"),
	}
)

// CreateGame is a CLI command handler function that creates a new game in the dispute game factory contract.
// The game creation requires an output root hash, trace type, and L2 block number as inputs.
// It interacts with an on-chain smart contract using the provided transaction manager to create the game.
//
// Parameters:
// - ctx: The CLI context (cli.Context), which provides access to the command-line flags and arguments.
//
// Command-line flags used:
// - OutputRootFlag: A flag that provides the output root as a hexadecimal string, converted to a hash.
// - TraceTypeFlag: A flag that provides the trace type as a uint64 value.
// - L2BlockNumFlag: A flag that provides the L2 block number as a uint64 value.
//
// Returns:
// - error: An error if the game creation fails, or nil if the operation is successful.
func CreateGame(ctx *cli.Context) error {
	outputRoot := common.HexToHash(ctx.String(OutputRootFlag.Name))
	traceType := ctx.Uint64(TraceTypeFlag.Name)
	l2BlockNum := ctx.Uint64(L2BlockNumFlag.Name)

	contract, txMgr, err := NewContractWithTxMgr[*contracts.DisputeGameFactoryContract](ctx, flags.FactoryAddress,
		func(ctx context.Context, metricer contractMetrics.ContractMetricer, address common.Address, caller *batching.MultiCaller) (*contracts.DisputeGameFactoryContract, error) {
			return contracts.NewDisputeGameFactoryContract(metricer, address, caller), nil
		})
	if err != nil {
		return fmt.Errorf("failed to create dispute game factory bindings: %w", err)
	}

	creator := tools.NewGameCreator(contract, txMgr)
	gameAddr, err := creator.CreateGame(ctx.Context, outputRoot, traceType, l2BlockNum)
	if err != nil {
		return fmt.Errorf("failed to create game: %w", err)
	}
	fmt.Printf("Fetched Game Address: %s\n", gameAddr.String())
	return nil
}

func createGameFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		flags.NetworkFlag,
		flags.FactoryAddressFlag,
		TraceTypeFlag,
		OutputRootFlag,
		L2BlockNumFlag,
	}
	cliFlags = append(cliFlags, txmgr.CLIFlagsWithDefaults(flags.EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	cliFlags = append(cliFlags, oplog.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var CreateGameCommand = &cli.Command{
	Name:        "create-game",
	Usage:       "Creates a dispute game via the factory",
	Description: "Creates a dispute game via the factory",
	Action:      Interruptible(CreateGame),
	Flags:       createGameFlags(),
}
