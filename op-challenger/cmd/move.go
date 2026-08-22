package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var (
	AttackFlag = &cli.BoolFlag{
		Name:    "attack",
		Usage:   "An attack move or ZK game challenge. If true, the defend flag must not be set.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "ATTACK"),
	}
	DefendFlag = &cli.BoolFlag{
		Name:    "defend",
		Usage:   "A defending move. If true, the attack flag must not be set.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "DEFEND"),
	}
	ParentIndexFlag = &cli.StringFlag{
		Name:    "parent-index",
		Usage:   "The index of the claim to move on (fault dispute games only).",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "PARENT_INDEX"),
	}
	ClaimFlag = &cli.StringFlag{
		Name:    "claim",
		Usage:   "The claim hash (fault dispute games only).",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "CLAIM"),
	}
)

func Move(ctx *cli.Context) error {
	attack := ctx.Bool(AttackFlag.Name)
	defend := ctx.Bool(DefendFlag.Name)

	if attack && defend {
		return fmt.Errorf("both attack and defense flags cannot be set")
	}
	if !attack && !defend {
		return fmt.Errorf("either attack or defense flag must be set")
	}

	caller, txMgr, err := newClientsFromCLI(ctx)
	if err != nil {
		return err
	}
	gameAddr, err := AddrFromFlag(GameAddressFlag.Name)(ctx)
	if err != nil {
		return fmt.Errorf("failed to parse game address: %w", err)
	}

	tx, err := createMoveTx(
		ctx.Context,
		caller,
		gameAddr,
		attack,
		ctx.Uint64(ParentIndexFlag.Name),
		common.HexToHash(ctx.String(ClaimFlag.Name)),
	)
	if err != nil {
		return err
	}

	rct, err := txMgr.Send(ctx.Context, tx)
	if err != nil {
		return fmt.Errorf("failed to send tx: %w", err)
	}
	fmt.Printf("Sent tx with status: %v, hash: %s\n", rct.Status, rct.TxHash.String())

	return nil
}

func createMoveTx(
	ctx context.Context,
	caller *batching.MultiCaller,
	gameAddr common.Address,
	attack bool,
	parentIndex uint64,
	claim common.Hash,
) (txmgr.TxCandidate, error) {
	gameType, err := contracts.DetectGameType(ctx, gameAddr, caller)
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to detect dispute game type: %w", err)
	}
	contract, err := contracts.NewDisputeGameContract(
		ctx,
		contractMetrics.NoopContractMetrics,
		caller,
		gameType,
		gameAddr,
	)
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to create dispute game bindings: %w", err)
	}

	switch contract := contract.(type) {
	case contracts.ZKDisputeGameContract:
		if !attack {
			return txmgr.TxCandidate{}, fmt.Errorf("zk dispute games do not support defense moves")
		}
		tx, err := contract.ChallengeTx(ctx)
		if err != nil {
			return txmgr.TxCandidate{}, fmt.Errorf("failed to create challenge tx: %w", err)
		}
		return tx, nil
	case contracts.FaultDisputeGameContract:
		parentClaim, err := contract.GetClaim(ctx, parentIndex)
		if err != nil {
			return txmgr.TxCandidate{}, fmt.Errorf("failed to get parent claim: %w", err)
		}
		if attack {
			tx, err := contract.AttackTx(ctx, parentClaim, claim)
			if err != nil {
				return txmgr.TxCandidate{}, fmt.Errorf("failed to create attack tx: %w", err)
			}
			return tx, nil
		}
		tx, err := contract.DefendTx(ctx, parentClaim, claim)
		if err != nil {
			return txmgr.TxCandidate{}, fmt.Errorf("failed to create defense tx: %w", err)
		}
		return tx, nil
	default:
		return txmgr.TxCandidate{}, fmt.Errorf("game type %v does not support moves", gameType)
	}
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
	Usage:       "Creates and sends an attack, defense, or ZK challenge transaction",
	Description: "Creates and sends an attack, defense, or ZK challenge transaction",
	Action:      Interruptible(Move),
	Flags:       moveFlags(),
}
