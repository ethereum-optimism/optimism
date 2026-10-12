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
		return fmt.Errorf("failed to create dispute game bindings for game type %v: %w", gameType, err)
	}
	return listCredits(ctx.Context, contract)
}

// listCredits prints the DelayedWETH credits of any supported dispute game type.
// ZK games have no claim tree, so their recipient set is the game creator, the
// challenger and the prover rather than the claim participants.
func listCredits(ctx context.Context, game contracts.DisputeGameContract) error {
	switch game := game.(type) {
	case contracts.ZKDisputeGameContract:
		recipients, err := zkCreditRecipients(ctx, game)
		if err != nil {
			return err
		}
		return printCredits(ctx, game, recipients)
	case contracts.FaultDisputeGameContract:
		recipients, err := faultCreditRecipients(ctx, game)
		if err != nil {
			return err
		}
		return printCredits(ctx, game, recipients)
	default:
		return fmt.Errorf("%w: cannot list credits for game %v", contracts.ErrUnsupportedGameType, game.Addr())
	}
}

// faultCreditRecipients returns every address a fault dispute game can credit: the
// participants in the claim tree plus the L2 block number challenger.
func faultCreditRecipients(ctx context.Context, faultGame contracts.FaultDisputeGameContract) ([]common.Address, error) {
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
// ZKDisputeGame.sol, which assigns credit to the game creator, to
// claimData.challenger and to claimData.prover, and never to any other address.
//
// The zero challenger is deliberately kept. When the parent resolved
// CHALLENGER_WINS and this game was never challenged, resolve() credits
// normalModeCredit[claimData.challenger], which is address(0) (see
// ZKDisputeGame.sol). That bond is recoverable by the DelayedWETH owner, so it is
// a real credit that op-dispute-mon reports and list-credits should show.
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
	recipients[metadata.Challenger] = true
	if metadata.Prover != (common.Address{}) {
		recipients[metadata.Prover] = true
	}
	return slices.Collect(maps.Keys(recipients)), nil
}

// creditReader is the part of a dispute game contract that printCredits needs.
type creditReader interface {
	GetBalanceAndDelay(ctx context.Context, block rpcblock.Block) (*big.Int, time.Duration, common.Address, error)
	GetWithdrawals(ctx context.Context, block rpcblock.Block, recipients ...common.Address) ([]*contracts.WithdrawalRequest, error)
}

func printCredits(ctx context.Context, game creditReader, recipients []common.Address) error {
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
