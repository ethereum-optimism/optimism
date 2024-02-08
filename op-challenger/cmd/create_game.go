package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var (
	TraceTypeFlag = &cli.StringFlag{
		Name:    "trace-type",
		Usage:   "Trace types to support.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "TRACE_TYPE"),
		Value:   config.TraceTypeCannon.String(),
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

func CreateGame(ctx *cli.Context) error {
	outputRoot := common.HexToHash(ctx.String(OutputRootFlag.Name))
	traceType := ctx.Uint64(TraceTypeFlag.Name)
	l2BlockNum := ctx.Uint64(L2BlockNumFlag.Name)

	contract, txMgr, err := NewContractWithTxMgr[*contracts.DisputeGameFactoryContract](ctx, flags.FactoryAddressFlag.Name, contracts.NewDisputeGameFactoryContract)
	if err != nil {
		return fmt.Errorf("failed to create dispute game factory bindings: %w", err)
	}

	txCandidate, err := contract.CreateTx(uint32(traceType), outputRoot, l2BlockNum)
	if err != nil {
		return fmt.Errorf("failed to create tx: %w", err)
	}

	rct, err := txMgr.Send(context.Background(), txCandidate)
	if err != nil {
		return fmt.Errorf("failed to send tx: %w", err)
	}
	fmt.Printf("Sent create transaction with status %v, tx_hash: %s\n", rct.Status, rct.TxHash.String())

	fetchedGameAddr, err := contract.GetGameFromParameters(context.Background(), uint32(traceType), outputRoot, l2BlockNum)
	if err != nil {
		return fmt.Errorf("failed to call games: %w", err)
	}
	fmt.Printf("Fetched Game Address: %s\n", fetchedGameAddr.String())

	return nil
}

func createGameFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		flags.FactoryAddressFlag,
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
	Action:      CreateGame,
	Flags:       createGameFlags(),
	Hidden:      true,
}
