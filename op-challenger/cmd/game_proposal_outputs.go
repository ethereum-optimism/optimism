package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

// proposalOutputsConcurrency bounds how many games are queried against the rollup node at once.
const proposalOutputsConcurrency = 10

// GameProposalOutputs asks OUR op-node what it derives for each dispute game so we can tell
// whether a challenger pointed at that node would agree with the proposal:
//   - optimism_outputAtBlock(<game l2BlockNumber>) -> the output root our node derives
//   - optimism_safeHeadAtL1Block(<game l1Head>)    -> our node's L2 safe head at the L1 view
//     the game is anchored to
//
// A safe head BELOW the proposed block is the fingerprint of an incomplete safe-head DB /
// lagging node: the challenger clamps to the safe head and disputes valid proposals.
func GameProposalOutputs(ctx *cli.Context) error {
	logger, err := setupLogging(ctx)
	if err != nil {
		return err
	}
	rpcUrl := ctx.String(flags.L1EthRpcFlag.Name)
	if rpcUrl == "" {
		return fmt.Errorf("missing %v", flags.L1EthRpcFlag.Name)
	}
	rollupUrl := ctx.String(flags.RollupRpcFlag.Name)
	if rollupUrl == "" {
		return fmt.Errorf("missing %v", flags.RollupRpcFlag.Name)
	}
	format := ctx.String(FormatFlag.Name)
	switch format {
	case formatText, formatJSON:
	default:
		return fmt.Errorf("invalid %v %q: must be one of %v, %v", FormatFlag.Name, format, formatText, formatJSON)
	}
	gameWindow := ctx.Duration(flags.GameWindowFlag.Name)

	l1Client, err := dial.DialEthClientWithTimeout(ctx.Context, dial.DefaultDialTimeout, logger, rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to dial L1: %w", err)
	}
	defer l1Client.Close()

	rollupClient, err := dial.DialRollupClientWithTimeout(ctx.Context, logger, rollupUrl)
	if err != nil {
		return fmt.Errorf("failed to dial rollup node: %w", err)
	}
	defer rollupClient.Close()

	caller := batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize)

	games, err := resolveGameRefs(ctx, caller, l1Client, gameWindow)
	if err != nil {
		return err
	}

	records, err := gameProposalOutputs(ctx.Context, caller, l1Client, rollupClient, games)
	if err != nil {
		return err
	}

	switch format {
	case formatJSON:
		return renderProposalOutputsJSON(os.Stdout, records)
	default:
		return renderProposalOutputsText(os.Stdout, records)
	}
}

type gameRef struct {
	addr  common.Address
	index *uint64
}

// resolveGameRefs returns the games to inspect: explicit positional game addresses when given,
// otherwise every game in the factory within the game window.
func resolveGameRefs(ctx *cli.Context, caller *batching.MultiCaller, l1Client *ethclient.Client, gameWindow time.Duration) ([]gameRef, error) {
	if ctx.Args().Len() > 0 {
		games := make([]gameRef, 0, ctx.Args().Len())
		for _, arg := range ctx.Args().Slice() {
			addr, err := opservice.ParseAddress(arg)
			if err != nil {
				return nil, fmt.Errorf("invalid game address %q: %w", arg, err)
			}
			games = append(games, gameRef{addr: addr})
		}
		return games, nil
	}

	factoryAddr, err := flags.FactoryAddress(ctx)
	if err != nil {
		return nil, err
	}
	factory, err := contracts.NewDisputeGameFactoryContract(ctx.Context, metrics.NoopContractMetrics, factoryAddr, caller)
	if err != nil {
		return nil, fmt.Errorf("failed to create dispute game factory contract: %w", err)
	}
	head, err := l1Client.HeaderByNumber(ctx.Context, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve current head block: %w", err)
	}
	earliestTimestamp := clock.MinCheckedTimestamp(clock.SystemClock, gameWindow)
	metas, err := factory.GetGamesAtOrAfter(ctx.Context, head.Hash(), earliestTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve games: %w", err)
	}
	games := make([]gameRef, 0, len(metas))
	for _, meta := range metas {
		index := meta.Index
		games = append(games, gameRef{addr: meta.Proxy, index: &index})
	}
	return games, nil
}

func gameProposalOutputs(ctx context.Context, caller *batching.MultiCaller, l1 *ethclient.Client, rollup *sources.RollupClient, games []gameRef) ([]proposalOutputRecord, error) {
	records := make([]proposalOutputRecord, len(games))
	errs := make([]error, len(games))
	sem := make(chan struct{}, proposalOutputsConcurrency)
	var wg sync.WaitGroup
	for i, g := range games {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, g gameRef) {
			defer wg.Done()
			defer func() { <-sem }()
			records[i], errs[i] = queryProposalOutput(ctx, caller, l1, rollup, g)
		}(i, g)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return records, nil
}

func queryProposalOutput(ctx context.Context, caller *batching.MultiCaller, l1 *ethclient.Client, rollup *sources.RollupClient, g gameRef) (proposalOutputRecord, error) {
	game, err := contracts.NewFaultDisputeGameContract(ctx, metrics.NoopContractMetrics, g.addr, caller)
	if err != nil {
		return proposalOutputRecord{}, fmt.Errorf("failed to create dispute game contract %v: %w", g.addr, err)
	}
	meta, err := game.GetMetadata(ctx, rpcblock.Latest)
	if err != nil {
		return proposalOutputRecord{}, fmt.Errorf("failed to retrieve metadata for game %v: %w", g.addr, err)
	}
	l1Header, err := l1.HeaderByHash(ctx, meta.L1Head)
	if err != nil {
		return proposalOutputRecord{}, fmt.Errorf("failed to retrieve l1Head %v for game %v: %w", meta.L1Head, g.addr, err)
	}
	l1Num := bigs.Uint64Strict(l1Header.Number)
	output, err := rollup.OutputAtBlock(ctx, meta.L2SequenceNum)
	if err != nil {
		return proposalOutputRecord{}, fmt.Errorf("failed to retrieve output at block %v for game %v: %w", meta.L2SequenceNum, g.addr, err)
	}
	safeHead, err := rollup.SafeHeadAtL1Block(ctx, l1Num)
	if err != nil {
		return proposalOutputRecord{}, fmt.Errorf("failed to retrieve safe head at L1 block %v for game %v: %w", l1Num, g.addr, err)
	}
	ourRoot := common.Hash(output.OutputRoot)
	return proposalOutputRecord{
		Index:                  g.index,
		Game:                   g.addr.Hex(),
		Status:                 meta.Status.String(),
		L2BlockNumber:          meta.L2SequenceNum,
		ProposedRoot:           meta.ProposedRoot.Hex(),
		OutputRoot:             ourRoot.Hex(),
		RootMatch:              ourRoot == meta.ProposedRoot,
		L1Head:                 meta.L1Head.Hex(),
		L1HeadNumber:           l1Num,
		SafeHead:               safeHead.SafeHead.Number,
		SafeHeadAtOrAboveBlock: safeHead.SafeHead.Number >= meta.L2SequenceNum,
	}, nil
}

// proposalOutputRecord is the structured, machine-readable view of one game's proposal vs our node.
type proposalOutputRecord struct {
	Index                  *uint64 `json:"index,omitempty"` // factory index, omitted for explicit game args
	Game                   string  `json:"game"`
	Status                 string  `json:"status"`
	L2BlockNumber          uint64  `json:"l2BlockNumber"`
	ProposedRoot           string  `json:"proposedRoot"`           // root claim on-chain
	OutputRoot             string  `json:"outputRoot"`             // what our node derives at L2BlockNumber
	RootMatch              bool    `json:"rootMatch"`              // OutputRoot == ProposedRoot
	L1Head                 string  `json:"l1Head"`                 // game's L1 head anchor (hash)
	L1HeadNumber           uint64  `json:"l1HeadNumber"`           // L1 head block number
	SafeHead               uint64  `json:"safeHead"`               // our node's safe head at L1Head
	SafeHeadAtOrAboveBlock bool    `json:"safeHeadAtOrAboveBlock"` // SafeHead >= L2BlockNumber
}

func renderProposalOutputsText(out io.Writer, records []proposalOutputRecord) error {
	lineFormat := "%-42v %-21v %14v %-66v %5v %12v %12v %5v\n"
	if _, err := fmt.Fprintf(out, lineFormat,
		"Game", "Status", "L2 Block", "Output Root (ours)", "Match", "L1 Head", "Safe Head", "Ok"); err != nil {
		return err
	}
	for _, r := range records {
		if _, err := fmt.Fprintf(out, lineFormat,
			r.Game, r.Status, r.L2BlockNumber, r.OutputRoot, r.RootMatch,
			r.L1HeadNumber, r.SafeHead, r.SafeHeadAtOrAboveBlock); err != nil {
			return err
		}
	}
	return nil
}

func renderProposalOutputsJSON(out io.Writer, records []proposalOutputRecord) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{"games": records})
}

func gameProposalOutputsFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		flags.RollupRpcFlag,
		flags.NetworkFlag,
		flags.FactoryAddressFlag,
		flags.GameWindowFlag,
		FormatFlag,
	}
	cliFlags = append(cliFlags, oplog.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var GameProposalOutputsCommand = &cli.Command{
	Name:      "game-proposal-outputs",
	Usage:     "Compare dispute game proposals against what our op-node derives",
	ArgsUsage: "[game-addr ...]",
	Description: "For each dispute game (enumerated from the factory, or the explicitly provided game " +
		"addresses), reports the output root our op-node derives at the proposed L2 block and our node's " +
		"safe head at the game's L1 head anchor. A root mismatch, or a safe head below the proposed block, " +
		"indicates a challenger pointed at this node would dispute the proposal.",
	Action: Interruptible(GameProposalOutputs),
	Flags:  gameProposalOutputsFlags(),
}
