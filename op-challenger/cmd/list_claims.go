package main

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var (
	GameAddressFlag = &cli.StringFlag{
		Name:    "game-address",
		Usage:   "Address of the fault game contract.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "GAME_ADDRESS"),
	}
	VerboseFlag = &cli.BoolFlag{
		Name:    "verbose",
		Aliases: []string{"v"},
		Usage:   "Verbose output",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "VERBOSE"),
	}
)

func ListClaims(ctx *cli.Context) error {
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
	return listClaims(ctx.Context, contract, ctx.Bool(VerboseFlag.Name))
}

func listClaims(ctx context.Context, game contracts.FaultDisputeGameContract, verbose bool) error {
	metadata, err := game.GetGameMetadata(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to retrieve metadata: %w", err)
	}
	maxDepth, err := game.GetMaxGameDepth(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve max depth: %w", err)
	}
	maxClockDuration, err := game.GetMaxClockDuration(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve max clock duration: %w", err)
	}
	splitDepth, err := game.GetSplitDepth(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve split depth: %w", err)
	}
	status := metadata.Status
	l2StartBlockNum, l2BlockNum, err := game.GetBlockRange(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve status: %w", err)
	}

	claims, err := game.GetAllClaims(ctx, rpcblock.Latest)
	if err != nil {
		return fmt.Errorf("failed to retrieve claims: %w", err)
	}

	var resolutionTime time.Time
	if status != gameTypes.GameStatusInProgress {
		resolutionTime, err = game.GetResolvedAt(ctx, rpcblock.Latest)
		if err != nil {
			return fmt.Errorf("failed to retrieve resolved at: %w", err)
		}
	}

	// The top game runs from depth 0 to split depth *inclusive*.
	// The - 1 here accounts for the fact that the split depth is included in the top game.
	bottomDepth := maxDepth - splitDepth - 1

	resolved, err := game.IsResolved(ctx, rpcblock.Latest, claims...)
	if err != nil {
		return fmt.Errorf("failed to retrieve claim resolution: %w", err)
	}

	gameState := types.NewGameState(claims, maxDepth)
	valueFormat := "%-14v"
	if verbose {
		valueFormat = "%-66v"
	}
	now := time.Now()
	lineFormat := "%3v %-7v %6v %5v %14v " + valueFormat + " %-42v %12v %-19v %10v %v\n"
	info := fmt.Sprintf(lineFormat, "Idx", "Move", "Parent", "Depth", "Trace", "Value", "Claimant", "Bond (ETH)", "Time", "Clock Used", "Resolution")
	for i, claim := range claims {
		pos := claim.Position
		parent := strconv.Itoa(claim.ParentContractIndex)
		var elapsed time.Duration // Root claim does not accumulate any time on its team's chess clock
		if claim.IsRoot() {
			parent = "-"
		} else {
			parentClaim, err := gameState.GetParent(claim)
			if err != nil {
				return fmt.Errorf("failed to retrieve parent claim: %w", err)
			}
			// Get the total chess clock time accumulated by the team that posted this claim at the time of the claim.
			elapsed = gameState.ChessClock(claim.Clock.Timestamp, parentClaim)
		}
		var countered string
		if !resolved[i] {
			clock := gameState.ChessClock(now, claim)
			resolvableAt := now.Add(maxClockDuration - clock).Format(time.DateTime)
			countered = fmt.Sprintf("⏱️  %v", resolvableAt)
		} else if claim.IsRoot() && metadata.L2BlockNumberChallenged {
			countered = "❌ " + metadata.L2BlockNumberChallenger.Hex()
		} else if claim.CounteredBy != (common.Address{}) {
			countered = "❌ " + claim.CounteredBy.Hex()
		} else {
			countered = "✅"
		}
		move := "Attack"
		if gameState.DefendsParent(claim) {
			move = "Defend"
		}
		var traceIdx *big.Int
		if claim.Depth() <= splitDepth {
			traceIdx = claim.TraceIndex(splitDepth)
		} else {
			relativePos, err := claim.Position.RelativeToAncestorAtDepth(splitDepth + 1)
			if err != nil {
				fmt.Printf("Error calculating relative position for claim %v: %v", claim.ContractIndex, err)
				traceIdx = big.NewInt(-1)
			} else {
				traceIdx = relativePos.TraceIndex(bottomDepth)
			}
		}
		value := claim.Value.TerminalString()
		if verbose {
			value = claim.Value.Hex()
		}
		timestamp := claim.Clock.Timestamp.Format(time.DateTime)
		bond := fmt.Sprintf("%12.8f", eth.WeiToEther(claim.Bond))
		if verbose {
			bond = fmt.Sprintf("%f", eth.WeiToEther(claim.Bond))
		}
		info = info + fmt.Sprintf(lineFormat,
			i, move, parent, pos.Depth(), traceIdx, value, claim.Claimant, bond, timestamp, elapsed, countered)
	}
	blockNumChallenger := "Unchallenged"
	if metadata.L2BlockNumberChallenged {
		blockNumChallenger = "❌ " + metadata.L2BlockNumberChallenger.Hex()
	}
	statusStr := status.String()
	if status != gameTypes.GameStatusInProgress {
		statusStr = fmt.Sprintf("%v • Resolution Time: %v", statusStr, resolutionTime.Format(time.DateTime))
	}
	fmt.Printf("Status: %v • L2 Blocks: %v to %v (%v) • Split Depth: %v • Max Depth: %v • Claim Count: %v\n%v\n",
		statusStr, l2StartBlockNum, l2BlockNum, blockNumChallenger, splitDepth, maxDepth, len(claims), info)
	return nil
}

func listClaimsFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		GameAddressFlag,
		VerboseFlag,
	}
	cliFlags = append(cliFlags, oplog.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var ListClaimsCommand = &cli.Command{
	Name:        "list-claims",
	Usage:       "List the claims in a dispute game",
	Description: "Lists the claims in a dispute game",
	Action:      Interruptible(ListClaims),
	Flags:       listClaimsFlags(),
}
