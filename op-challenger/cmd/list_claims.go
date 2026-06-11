package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
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

const (
	formatText = "text"
	formatJSON = "json"
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
		Usage:   "Verbose output (full claim values and bonds in text format)",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "VERBOSE"),
	}
	FormatFlag = &cli.StringFlag{
		Name:    "format",
		Usage:   fmt.Sprintf("Output format. One of: %v, %v. json emits full values and is intended for scripting.", formatText, formatJSON),
		Value:   formatText,
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "FORMAT"),
	}
)

// claimRecord is the structured, machine-readable view of a single claim.
type claimRecord struct {
	Index            int    `json:"index"`
	Move             string `json:"move"` // "Attack" or "Defend"
	ParentIndex      int    `json:"parentIndex"`
	Depth            uint64 `json:"depth"`
	TraceIndex       string `json:"traceIndex"`
	Value            string `json:"value"` // full 0x-prefixed hash
	Claimant         string `json:"claimant"`
	BondWei          string `json:"bondWei"`
	BondEth          string `json:"bondEth"`
	Timestamp        int64  `json:"timestamp"` // unix seconds
	Time             string `json:"time"`      // RFC3339
	ClockUsedSeconds int64  `json:"clockUsedSeconds"`
	CounteredBy      string `json:"counteredBy,omitempty"`
	Resolved         bool   `json:"resolved"`
	ResolvableAt     string `json:"resolvableAt,omitempty"` // RFC3339, when not yet resolved

	// valueTerminal and resolution are only used by the text renderer.
	valueTerminal string
	resolution    string
}

// claimsReport is the structured, machine-readable view of a whole game.
type claimsReport struct {
	Status                  string        `json:"status"`
	ResolutionTime          string        `json:"resolutionTime,omitempty"` // RFC3339, when resolved
	L2StartBlock            uint64        `json:"l2StartBlock"`
	L2BlockNumber           uint64        `json:"l2BlockNumber"`
	L2BlockNumberChallenged bool          `json:"l2BlockNumberChallenged"`
	L2BlockNumberChallenger string        `json:"l2BlockNumberChallenger,omitempty"`
	SplitDepth              uint64        `json:"splitDepth"`
	MaxDepth                uint64        `json:"maxDepth"`
	ClaimCount              int           `json:"claimCount"`
	Claims                  []claimRecord `json:"claims"`
}

func ListClaims(ctx *cli.Context) error {
	logger, err := setupLogging(ctx)
	if err != nil {
		return err
	}
	format := ctx.String(FormatFlag.Name)
	switch format {
	case formatText, formatJSON:
	default:
		return fmt.Errorf("invalid %v %q: must be one of %v, %v", FormatFlag.Name, format, formatText, formatJSON)
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
	report, err := buildClaimsReport(ctx.Context, contract)
	if err != nil {
		return err
	}
	switch format {
	case formatJSON:
		return renderJSON(os.Stdout, report)
	default:
		return renderText(os.Stdout, report, ctx.Bool(VerboseFlag.Name))
	}
}

func buildClaimsReport(ctx context.Context, game contracts.FaultDisputeGameContract) (claimsReport, error) {
	metadata, err := game.GetExtendedMetadata(ctx, rpcblock.Latest)
	if err != nil {
		return claimsReport{}, fmt.Errorf("failed to retrieve metadata: %w", err)
	}
	maxDepth, err := game.GetMaxGameDepth(ctx)
	if err != nil {
		return claimsReport{}, fmt.Errorf("failed to retrieve max depth: %w", err)
	}
	maxClockDuration, err := game.GetMaxClockDuration(ctx)
	if err != nil {
		return claimsReport{}, fmt.Errorf("failed to retrieve max clock duration: %w", err)
	}
	splitDepth, err := game.GetSplitDepth(ctx)
	if err != nil {
		return claimsReport{}, fmt.Errorf("failed to retrieve split depth: %w", err)
	}
	status := metadata.Status
	l2StartBlockNum, l2BlockNum, err := game.GetGameRange(ctx)
	if err != nil {
		return claimsReport{}, fmt.Errorf("failed to retrieve status: %w", err)
	}

	claims, err := game.GetAllClaims(ctx, rpcblock.Latest)
	if err != nil {
		return claimsReport{}, fmt.Errorf("failed to retrieve claims: %w", err)
	}

	report := claimsReport{
		Status:                  status.String(),
		L2StartBlock:            l2StartBlockNum,
		L2BlockNumber:           l2BlockNum,
		L2BlockNumberChallenged: metadata.L2BlockNumberChallenged,
		SplitDepth:              uint64(splitDepth),
		MaxDepth:                uint64(maxDepth),
		ClaimCount:              len(claims),
		Claims:                  make([]claimRecord, 0, len(claims)),
	}
	if metadata.L2BlockNumberChallenged {
		report.L2BlockNumberChallenger = metadata.L2BlockNumberChallenger.Hex()
	}
	if status != gameTypes.GameStatusInProgress {
		resolutionTime, err := game.GetResolvedAt(ctx, rpcblock.Latest)
		if err != nil {
			return claimsReport{}, fmt.Errorf("failed to retrieve resolved at: %w", err)
		}
		report.ResolutionTime = resolutionTime.Format(time.RFC3339)
	}

	// The top game runs from depth 0 to split depth *inclusive*.
	// The - 1 here accounts for the fact that the split depth is included in the top game.
	bottomDepth := maxDepth - splitDepth - 1

	resolved, err := game.IsResolved(ctx, rpcblock.Latest, claims...)
	if err != nil {
		return claimsReport{}, fmt.Errorf("failed to retrieve claim resolution: %w", err)
	}

	gameState := types.NewGameState(claims, maxDepth)
	now := time.Now()
	for i, claim := range claims {
		parentIdx := claim.ParentContractIndex
		var elapsed time.Duration // Root claim does not accumulate any time on its team's chess clock
		if claim.IsRoot() {
			parentIdx = -1
		} else {
			parentClaim, err := gameState.GetParent(claim)
			if err != nil {
				return claimsReport{}, fmt.Errorf("failed to retrieve parent claim: %w", err)
			}
			// Total chess clock time accumulated by the team that posted this claim, at the time of the claim.
			elapsed = gameState.ChessClock(claim.Clock.Timestamp, parentClaim)
		}

		rec := claimRecord{
			Index:            i,
			Move:             "Attack",
			ParentIndex:      parentIdx,
			Depth:            uint64(claim.Depth()),
			Value:            claim.Value.Hex(),
			Claimant:         claim.Claimant.Hex(),
			BondWei:          claim.Bond.String(),
			BondEth:          fmt.Sprintf("%f", eth.WeiToEther(claim.Bond)),
			Timestamp:        claim.Clock.Timestamp.Unix(),
			Time:             claim.Clock.Timestamp.Format(time.RFC3339),
			ClockUsedSeconds: int64(elapsed.Seconds()),
			valueTerminal:    claim.Value.TerminalString(),
		}
		if gameState.DefendsParent(claim) {
			rec.Move = "Defend"
		}

		var traceIdx *big.Int
		if claim.Depth() <= splitDepth {
			traceIdx = claim.TraceIndex(splitDepth)
		} else {
			relativePos, err := claim.Position.RelativeToAncestorAtDepth(splitDepth + 1)
			if err != nil {
				return claimsReport{}, fmt.Errorf("failed calculating relative position for claim %v: %w", claim.ContractIndex, err)
			}
			traceIdx = relativePos.TraceIndex(bottomDepth)
		}
		rec.TraceIndex = traceIdx.String()

		// A claim countered by a winning step has counteredBy set the moment the step lands,
		// but its subgame is only resolved later by a separate resolveClaim call once the clock
		// expires. Report the on-chain resolution status, independent of the displayed outcome.
		rec.Resolved = resolved[i]

		if claim.CounteredBy != (common.Address{}) {
			rec.CounteredBy = claim.CounteredBy.Hex()
			rec.resolution = "❌ " + claim.CounteredBy.Hex()
		} else if !resolved[i] {
			clock := gameState.ChessClock(now, claim)
			resolvableAt := now.Add(maxClockDuration - clock)
			rec.ResolvableAt = resolvableAt.Format(time.RFC3339)
			rec.resolution = fmt.Sprintf("⏱️  %v", resolvableAt.Format(time.DateTime))
		} else if claim.IsRoot() && metadata.L2BlockNumberChallenged {
			rec.CounteredBy = metadata.L2BlockNumberChallenger.Hex()
			rec.resolution = "❌ " + metadata.L2BlockNumberChallenger.Hex()
		} else {
			rec.resolution = "✅"
		}

		report.Claims = append(report.Claims, rec)
	}
	return report, nil
}

func weiToEther(weiStr string) float64 {
	wei, ok := new(big.Int).SetString(weiStr, 10)
	if !ok {
		return 0
	}
	return eth.WeiToEther(wei)
}

func renderText(out io.Writer, report claimsReport, verbose bool) error {
	valueFormat := "%-14v"
	if verbose {
		valueFormat = "%-66v"
	}
	lineFormat := "%3v %-7v %6v %5v %14v " + valueFormat + " %-42v %12v %-19v %10v %v\n"
	info := fmt.Sprintf(lineFormat, "Idx", "Move", "Parent", "Depth", "Trace", "Value", "Claimant", "Bond (ETH)", "Time", "Clock Used", "Resolution")
	for _, c := range report.Claims {
		parent := strconv.Itoa(c.ParentIndex)
		if c.ParentIndex < 0 {
			parent = "-"
		}
		value := c.valueTerminal
		bond := fmt.Sprintf("%12.8f", weiToEther(c.BondWei))
		if verbose {
			value = c.Value
			bond = c.BondEth
		}
		info += fmt.Sprintf(lineFormat,
			c.Index, c.Move, parent, c.Depth, c.TraceIndex, value, c.Claimant, bond,
			time.Unix(c.Timestamp, 0).Format(time.DateTime), time.Duration(c.ClockUsedSeconds)*time.Second, c.resolution)
	}
	blockNumChallenger := "Unchallenged"
	if report.L2BlockNumberChallenged {
		blockNumChallenger = "❌ " + report.L2BlockNumberChallenger
	}
	statusStr := report.Status
	if report.ResolutionTime != "" {
		statusStr = fmt.Sprintf("%v • Resolution Time: %v", statusStr, report.ResolutionTime)
	}
	_, err := fmt.Fprintf(out, "Status: %v • L2 Blocks: %v to %v (%v) • Split Depth: %v • Max Depth: %v • Claim Count: %v\n%v\n",
		statusStr, report.L2StartBlock, report.L2BlockNumber, blockNumChallenger, report.SplitDepth, report.MaxDepth, report.ClaimCount, info)
	return err
}

func renderJSON(out io.Writer, report claimsReport) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func listClaimsFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		GameAddressFlag,
		VerboseFlag,
		FormatFlag,
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
