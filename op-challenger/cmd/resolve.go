package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/urfave/cli/v2"
)

func Resolve(ctx *cli.Context) error {
	contract, txMgr, err := NewContractWithTxMgr[contracts.FaultDisputeGameContract](ctx, AddrFromFlag(GameAddressFlag.Name), contracts.NewFaultDisputeGameContract)
	if err != nil {
		return fmt.Errorf("failed to create dispute game bindings: %w", err)
	}

	tx, err := contract.ResolveTx()
	if err != nil {
		return fmt.Errorf("failed to create resolve tx: %w", err)
	}

	rct, err := txMgr.Send(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("failed to send tx: %w", err)
	}

	fmt.Printf("Sent resolve tx with status: %v, hash: %s\n", rct.Status, rct.TxHash.String())

	return nil
}

func resolveFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		GameAddressFlag,
	}
	cliFlags = append(cliFlags, txmgr.CLIFlagsWithDefaults(flags.EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	cliFlags = append(cliFlags, oplog.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var ResolveCommand = &cli.Command{
	Name:        "resolve",
	Usage:       "Resolves the specified dispute game if possible",
	Description: "Resolves the specified dispute game if possible",
	Action:      Interruptible(Resolve),
	Flags:       resolveFlags(),
}
