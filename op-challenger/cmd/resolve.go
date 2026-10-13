package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

func Resolve(ctx *cli.Context) error {
	caller, txMgr, err := newClientsFromCLI(ctx)
	if err != nil {
		return err
	}
	gameAddr, err := AddrFromFlag(GameAddressFlag.Name)(ctx)
	if err != nil {
		return fmt.Errorf("failed to parse game address: %w", err)
	}

	tx, err := createResolveTx(ctx.Context, caller, gameAddr)
	if err != nil {
		return err
	}

	rct, err := txMgr.Send(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("failed to send tx: %w", err)
	}

	fmt.Printf("Sent resolve tx with status: %v, hash: %s\n", rct.Status, rct.TxHash.String())

	return nil
}

// createResolveTx builds the resolve transaction for any supported dispute game type.
// The game type is read from the contract itself so ZK games, which have no claim tree,
// resolve through the same command as permissioned and cannon games.
func createResolveTx(ctx context.Context, caller *batching.MultiCaller, gameAddr common.Address) (txmgr.TxCandidate, error) {
	gameType, err := contracts.DetectGameType(ctx, gameAddr, caller)
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to detect dispute game type: %w", err)
	}
	contract, err := contracts.NewDisputeGameContract(ctx, contractMetrics.NoopContractMetrics, caller, gameType, gameAddr)
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to create dispute game bindings: %w", err)
	}
	tx, err := contract.ResolveTx()
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to create resolve tx: %w", err)
	}
	return tx, nil
}

func resolveFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		GameAddressFlag,
	}
	cliFlags = append(cliFlags, txmgr.CLIFlagsWithDefaults(flags.EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	cliFlags = append(cliFlags, logcli.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var ResolveCommand = &cli.Command{
	Name:        "resolve",
	Usage:       "Resolves the specified dispute game if possible",
	Description: "Resolves the specified dispute game if possible",
	Action:      Interruptible(Resolve),
	Flags:       resolveFlags(),
}
