package main

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var (
	AttackFlag = &cli.BoolFlag{
		Name:    "attack",
		Usage:   "An attack move. If true, the defend flag must not be set.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "ATTACK"),
	}
	DefendFlag = &cli.BoolFlag{
		Name:    "defend",
		Usage:   "A defending move. If true, the attack flag must not be set.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "DEFEND"),
	}
	ParentIndexFlag = &cli.StringFlag{
		Name:    "parent-index",
		Usage:   "The index of the claim to move on.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "PARENT_INDEX"),
	}
	ClaimFlag = &cli.StringFlag{
		Name:    "claim",
		Usage:   "The claim hash.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "CLAIM"),
	}
)

func Move(ctx *cli.Context) error {
	attack := ctx.Bool(AttackFlag.Name)
	defend := ctx.Bool(DefendFlag.Name)
	parentIndex := ctx.Uint64(ParentIndexFlag.Name)
	claim := common.HexToHash(ctx.String(ClaimFlag.Name))

	if attack && defend {
		return fmt.Errorf("both attack and defense flags cannot be set")
	}

	contract, txMgr, err := NewContractWithTxMgr[contracts.FaultDisputeGameContract](ctx, AddrFromFlag(GameAddressFlag.Name), contracts.NewFaultDisputeGameContract)
	if err != nil {
		return fmt.Errorf("failed to create dispute game bindings: %w", err)
	}

	parentClaim, err := contract.GetClaim(ctx.Context, parentIndex)
	if err != nil {
		return fmt.Errorf("failed to get parent claim: %w", err)
	}
	var tx txmgr.TxCandidate
	if attack {
		tx, err = contract.AttackTx(ctx.Context, parentClaim, claim)
		if err != nil {
			return fmt.Errorf("failed to create attack tx: %w", err)
		}
	} else if defend {
		tx, err = contract.DefendTx(ctx.Context, parentClaim, claim)
		if err != nil {
			return fmt.Errorf("failed to create defense tx: %w", err)
		}
	} else {
		return fmt.Errorf("either attack or defense flag must be set")
	}

	rct, err := txMgr.Send(ctx.Context, tx)
	if err != nil {
		return fmt.Errorf("failed to send tx: %w", err)
	}
	fmt.Printf("Sent tx with status: %v, hash: %s\n", rct.Status, rct.TxHash.String())

	return nil
}

func moveFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		GameAddressFlag,
		AttackFlag,
		DefendFlag,
		ParentIndexFlag,
		ClaimFlag,
	}
	cliFlags = append(cliFlags, txmgr.CLIFlagsWithDefaults(flags.EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	cliFlags = append(cliFlags, oplog.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var MoveCommand = &cli.Command{
	Name:        "move",
	Usage:       "Creates and sends a move transaction to the dispute game",
	Description: "Creates and sends a move transaction to the dispute game",
	Action:      Interruptible(Move),
	Flags:       moveFlags(),
}
