package main

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

func ListCredits(ctx *cli.Context) error {
	logger, err := setupLogging(ctx)
	if err != nil {
		return err
	}
	rpcUrl := ctx.String(flags.L1EthRpcFlag.Name)
	if rpcUrl == "" {
		return fmt.Errorf("missing %v", flags.L1EthRpcFlag.Name)
	}
	gameAddr, err := opservice.ParseAddress(ctx.String(GameAddressFlag.Name))
	if err != nil {
		return err
	}

	l1Client, err := dial.DialEthClientWithTimeout(ctx.Context, dial.DefaultDialTimeout, logger, rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to dial L1: %w", err)
	}
	defer l1Client.Close()

	caller := batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize)
	contract, err := contracts.NewFaultDisputeGameContract(ctx.Context, metrics.NoopContractMetrics, gameAddr, caller)
	if err != nil {
		return err
	}
	return listCredits(ctx.Context, contract)
}

func listCredits(ctx context.Context, game contracts.FaultDisputeGameContract) error {
	claims, err := game.GetAllClaims(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to load claims: %w", err)
	}
	metadata, err := game.GetGameMetadata(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}
	recipients := make(map[common.Address]bool)
	for _, claim := range claims {
		if claim.CounteredBy != (common.Address{}) {
			recipients[claim.CounteredBy] = true
		}
		recipients[claim.Claimant] = true
	}
	if metadata.L2BlockNumberChallenger != (common.Address{}) {
		recipients[metadata.L2BlockNumberChallenger] = true
	}

	balance, withdrawalDelay, wethAddress, err := game.GetBalanceAndDelay(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to get DelayedWETH info: %w", err)
	}
	claimants := maps.Keys(recipients)
	withdrawals, err := game.GetWithdrawals(ctx, rpcblock.Latest, claimants...)
	if err != nil {
		return fmt.Errorf("failed to get withdrawals: %w", err)
	}
	lineFormat := "%-42v %12v %-19v\n"
	info := fmt.Sprintf(lineFormat, "Claimant", "ETH", "Unlock Time")
	for i, withdrawal := range withdrawals {
		var amount string
		if withdrawal.Amount.Cmp(big.NewInt(0)) == 0 {
			amount = "-"
		} else {
			amount = fmt.Sprintf("%12.8f", eth.WeiToEther(withdrawal.Amount))
		}
		var unlockTime string
		if withdrawal.Timestamp.Cmp(big.NewInt(0)) == 0 {
			unlockTime = "-"
		} else {
			unlockTime = time.Unix(withdrawal.Timestamp.Int64(), 0).Add(withdrawalDelay).Format(time.DateTime)
		}
		info += fmt.Sprintf(lineFormat, claimants[i], amount, unlockTime)
	}
	fmt.Printf("DelayedWETH Contract: %v • Total Balance (ETH): %12.8f • Delay: %v\n%v\n",
		wethAddress, eth.WeiToEther(balance), withdrawalDelay, info)
	return nil
}

func listCreditsFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		GameAddressFlag,
	}
	cliFlags = append(cliFlags, oplog.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var ListCreditsCommand = &cli.Command{
	Name:        "list-credits",
	Usage:       "List the credits in a dispute game",
	Description: "Lists the credits in a dispute game",
	Action:      Interruptible(ListCredits),
	Flags:       listCreditsFlags(),
}
