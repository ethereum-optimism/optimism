package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/urfave/cli/v2"
)

var (
	ClaimIdxFlag = &cli.Uint64Flag{
		Name:    "claim",
		Usage:   "Index of the claim to resolve.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "CLAIM"),
	}
)

func ResolveClaim(ctx *cli.Context) error {
	if !ctx.IsSet(ClaimIdxFlag.Name) {
		return fmt.Errorf("must specify %v flag", ClaimIdxFlag.Name)
	}
	idx := ctx.Uint64(ClaimIdxFlag.Name)

	contract, txMgr, err := NewContractWithTxMgr[contracts.FaultDisputeGameContract](ctx, AddrFromFlag(GameAddressFlag.Name), contracts.NewFaultDisputeGameContract)
	if err != nil {
		return fmt.Errorf("failed to create dispute game bindings: %w", err)
	}

	err = contract.CallResolveClaim(ctx.Context, idx)
	if err != nil {
		return fmt.Errorf("claim is not resolvable: %w", err)
	}

	tx, err := contract.ResolveClaimTx(idx)
	if err != nil {
		return fmt.Errorf("failed to create resolve claim tx: %w", err)
	}

	rct, err := txMgr.Send(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("failed to send tx: %w", err)
	}

	fmt.Printf("Sent resolve claim tx with status: %v, hash: %s\n", rct.Status, rct.TxHash.String())

	return nil
}

func resolveClaimFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		GameAddressFlag,
		ClaimIdxFlag,
	}
	cliFlags = append(cliFlags, txmgr.CLIFlagsWithDefaults(flags.EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	cliFlags = append(cliFlags, oplog.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var ResolveClaimCommand = &cli.Command{
	Name:        "resolve-claim",
	Usage:       "Resolves the specified claim if possible",
	Description: "Resolves the specified claim if possible",
	Action:      Interruptible(ResolveClaim),
	Flags:       resolveClaimFlags(),
}
