package main

import (
	"context"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
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
	gameType, err := contracts.DetectGameType(ctx.Context, gameAddr, caller)
	if err != nil {
		return fmt.Errorf("failed to detect dispute game type: %w", err)
	}
	contract, err := contracts.NewDisputeGameContract(ctx.Context, metrics.NoopContractMetrics, caller, gameType, gameAddr)
	if err != nil {
		return err
	}
	creditGameContract, ok := contract.(creditGame)
	if !ok {
		return fmt.Errorf("%w: cannot list credits for game type %s", contracts.ErrUnsupportedGameType, gameType)
	}
	return listCredits(ctx.Context, creditGameContract)
}

// listCredits prints the DelayedWETH credits of any supported dispute game type.
// ZK games have no claim tree, so their recipient set is the game creator, the
// challenger and the prover rather than the claim participants.
func listCredits(ctx context.Context, game creditGame) error {
	recipients, err := creditRecipients(ctx, game)
	if err != nil {
		return err
	}
	return printCredits(ctx, game, recipients)
}

func creditRecipients(ctx context.Context, game creditGame) ([]common.Address, error) {
	if zkGame, ok := game.(contracts.ZKDisputeGameContract); ok {
		return zkCreditRecipients(ctx, zkGame)
	}
	faultGame, ok := game.(contracts.FaultDisputeGameContract)
	if !ok {
		return nil, fmt.Errorf("%w: cannot list credits for game type %T", contracts.ErrUnsupportedGameType, game)
	}
	claims, err := faultGame.GetAllClaims(ctx, rpcblock.Latest)
	if err != nil {
		return nil, fmt.Errorf("failed to load claims: %w", err)
	}
	metadata, err := faultGame.GetExtendedMetadata(ctx, rpcblock.Latest)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
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
	return slices.Collect(maps.Keys(recipients)), nil
}

// zkCreditRecipients returns the only addresses a ZK game can credit. This mirrors
// ZKDisputeGame.sol, which assigns credit to the game creator in every case and to
// the challenger or prover for the proposal, and never to any other address.
func zkCreditRecipients(ctx context.Context, game contracts.ZKDisputeGameContract) ([]common.Address, error) {
	bonds, err := game.GetBondMetadata(ctx, rpcblock.Latest)
	if err != nil {
		return nil, fmt.Errorf("failed to load bond metadata: %w", err)
	}
	recipients := make(map[common.Address]bool)
	if bonds.GameCreator != (common.Address{}) {
		recipients[bonds.GameCreator] = true
	}
	metadata, err := game.GetChallengerMetadata(ctx, rpcblock.Latest)
	if err != nil {
		return nil, fmt.Errorf("failed to load challenger metadata: %w", err)
	}
	if metadata.Challenger != (common.Address{}) {
		recipients[metadata.Challenger] = true
	}
	if metadata.Prover != (common.Address{}) {
		recipients[metadata.Prover] = true
	}
	return slices.Collect(maps.Keys(recipients)), nil
}

// creditGame is the part of a dispute game contract that list-credits needs once the
// recipient set is known. Both FaultDisputeGameContract and ZKDisputeGameContract
// satisfy it, so the ZK path reuses the existing output formatting unchanged.
type creditGame interface {
	contracts.DisputeGameContract
	GetBalanceAndDelay(ctx context.Context, block rpcblock.Block) (*big.Int, time.Duration, common.Address, error)
	GetWithdrawals(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*contracts.WithdrawalRequest, error)
}

func printCredits(ctx context.Context, game creditGame, recipients []common.Address) error {
	balance, withdrawalDelay, wethAddress, err := game.GetBalanceAndDelay(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to get DelayedWETH info: %w", err)
	}
	withdrawals, err := game.GetWithdrawals(ctx, rpcblock.Latest, recipients...)
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
		info += fmt.Sprintf(lineFormat, recipients[i], amount, unlockTime)
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
	cliFlags = append(cliFlags, logcli.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var ListCreditsCommand = &cli.Command{
	Name:        "list-credits",
	Usage:       "List the credits in a dispute game",
	Description: "Lists the credits in a dispute game",
	Action:      Interruptible(ListCredits),
	Flags:       listCreditsFlags(),
}
