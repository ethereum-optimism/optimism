package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
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

	caller, txMgr, err := newClientsFromCLI(ctx)
	if err != nil {
		return err
	}
	gameAddr, err := AddrFromFlag(GameAddressFlag.Name)(ctx)
	if err != nil {
		return fmt.Errorf("failed to parse game address: %w", err)
	}

	tx, err := createResolveClaimTx(ctx.Context, caller, gameAddr, idx)
	if err != nil {
		return err
	}

	rct, err := txMgr.Send(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("failed to send tx: %w", err)
	}

	fmt.Printf("Sent resolve claim tx with status: %v, hash: %s\n", rct.Status, rct.TxHash.String())

	return nil
}

func createResolveClaimTx(ctx context.Context, caller *batching.MultiCaller, gameAddr common.Address, claimIdx uint64) (txmgr.TxCandidate, error) {
	gameType, err := contracts.DetectGameType(ctx, gameAddr, caller)
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to detect dispute game type: %w", err)
	}
	contract, err := contracts.NewDisputeGameContract(ctx, contractMetrics.NoopContractMetrics, caller, gameType, gameAddr)
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to create dispute game bindings: %w", err)
	}

	switch contract := contract.(type) {
	case contracts.ZKDisputeGameContract:
		return txmgr.TxCandidate{}, fmt.Errorf("zk dispute game %v has no claims to resolve, use the resolve command", gameAddr)
	case contracts.FaultDisputeGameContract:
		if err := contract.CallResolveClaim(ctx, claimIdx); err != nil {
			return txmgr.TxCandidate{}, fmt.Errorf("claim is not resolvable: %w", err)
		}
		tx, err := contract.ResolveClaimTx(claimIdx)
		if err != nil {
			return txmgr.TxCandidate{}, fmt.Errorf("failed to create resolve claim tx: %w", err)
		}
		return tx, nil
	default:
		return txmgr.TxCandidate{}, fmt.Errorf("game type %v does not support resolving claims", gameType)
	}
}

func resolveClaimFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		GameAddressFlag,
		ClaimIdxFlag,
	}
	cliFlags = append(cliFlags, txmgr.CLIFlagsWithDefaults(flags.EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	cliFlags = append(cliFlags, logcli.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var ResolveClaimCommand = &cli.Command{
	Name:        "resolve-claim",
	Usage:       "Resolves the specified claim if possible",
	Description: "Resolves the specified claim if possible",
	Action:      Interruptible(ResolveClaim),
	Flags:       resolveClaimFlags(),
}
