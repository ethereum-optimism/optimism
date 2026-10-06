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
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/super"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

// proposalOutputsConcurrency bounds how many games are queried against our nodes at once.
const proposalOutputsConcurrency = 10

// GameProposalOutputs asks OUR nodes what they derive for each dispute game so we can tell
// whether a challenger pointed at them would agree with the proposal.
//
// Output-root games (--rollup-rpc):
//   - optimism_outputAtBlock(<game l2BlockNumber>) -> the output root our node derives
//   - optimism_safeHeadAtL1Block(<game l1Head>)    -> our node's L2 safe head at the L1 view
//     the game is anchored to
//
// A safe head BELOW the proposed block is the fingerprint of an incomplete safe-head DB /
// lagging node: the challenger clamps to the safe head and disputes valid proposals.
//
// Super-root games: super-cannon-kona, super-permissioned and zk (--superroot-rpc):
//   - superroot_atTimestamp(<game l2SequenceNumber>) -> the super root our node derives at that
//     timestamp, and the L1 block it has processed up to
//
// A node that has not processed past the game's l1Head cannot judge the proposal yet, and a
// missing super root means the timestamp is not safe on our node, so the proposal is invalid.
// Challengers dispute invalid super-cannon-kona and zk proposals. Super-permissioned games
// resolve at creation, so an invalid one needs a guardian blacklist instead.
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
	superRootUrl := ctx.String(flags.SuperRootRpcFlag.Name)
	if rollupUrl == "" && superRootUrl == "" {
		return fmt.Errorf("missing %v or %v", flags.RollupRpcFlag.Name, flags.SuperRootRpcFlag.Name)
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

	// Declared as interfaces so an undialled client stays a nil interface rather than a
	// non-nil interface holding a nil pointer.
	var rollup outputs.OutputRollupClient
	if rollupUrl != "" {
		cl, err := dial.DialRollupClientWithTimeout(ctx.Context, logger, rollupUrl)
		if err != nil {
			return fmt.Errorf("failed to dial rollup node: %w", err)
		}
		defer cl.Close()
		rollup = cl
	}
	var superRoots super.SuperNodeRootProvider
	if superRootUrl != "" {
		cl, err := dial.DialSuperNodeClientWithTimeout(ctx.Context, logger, superRootUrl)
		if err != nil {
			return fmt.Errorf("failed to dial super root RPC: %w", err)
		}
		defer cl.Close()
		superRoots = cl
	}

	caller := batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize)

	games, err := resolveGameRefs(ctx, caller, l1Client, gameWindow)
	if err != nil {
		return err
	}

	records, err := gameProposalOutputs(ctx.Context, caller, l1Client, rollup, superRoots, games)
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
	addr     common.Address
	gameType gameTypes.GameType
	index    *uint64
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
			gameType, err := contracts.DetectGameType(ctx.Context, addr, caller)
			if err != nil {
				return nil, fmt.Errorf("game %v: %w", addr, err)
			}
			games = append(games, gameRef{addr: addr, gameType: gameType})
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
		games = append(games, gameRef{addr: meta.Proxy, gameType: gameTypes.GameType(meta.GameType), index: &index})
	}
	return games, nil
}

// l1HeaderSource is the subset of the L1 client the per-game query needs.
type l1HeaderSource interface {
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
}

func gameProposalOutputs(ctx context.Context, caller *batching.MultiCaller, l1 l1HeaderSource, rollup outputs.OutputRollupClient, superRoots super.SuperNodeRootProvider, games []gameRef) ([]proposalOutputRecord, error) {
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
			records[i], errs[i] = queryProposalOutput(ctx, caller, l1, rollup, superRoots, g)
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

func queryProposalOutput(ctx context.Context, caller *batching.MultiCaller, l1 l1HeaderSource, rollup outputs.OutputRollupClient, superRoots super.SuperNodeRootProvider, g gameRef) (proposalOutputRecord, error) {
	switch g.gameType {
	case gameTypes.CannonGameType,
		gameTypes.PermissionedGameType,
		gameTypes.CannonKonaGameType,
		gameTypes.AlphabetGameType,
		gameTypes.FastGameType:
		if rollup == nil {
			return proposalOutputRecord{}, fmt.Errorf("missing %v for game %v of type %v", flags.RollupRpcFlag.Name, g.addr, g.gameType)
		}
		return queryOutputRootProposal(ctx, caller, l1, rollup, g)
	case gameTypes.SuperCannonKonaGameType, gameTypes.SuperPermissionedGameType, gameTypes.ZKDisputeGameType:
		if superRoots == nil {
			return proposalOutputRecord{}, fmt.Errorf("missing %v for game %v of type %v", flags.SuperRootRpcFlag.Name, g.addr, g.gameType)
		}
		return querySuperRootProposal(ctx, caller, l1, superRoots, g)
	default:
		return proposalOutputRecord{}, fmt.Errorf("unsupported game type %v for game %v", g.gameType, g.addr)
	}
}

// loadProposal reads the game's on-chain proposal into the fields shared by every game kind.
func loadProposal(ctx context.Context, caller *batching.MultiCaller, l1 l1HeaderSource, g gameRef) (proposalOutputRecord, contracts.GenericGameMetadata, error) {
	game, err := contracts.NewDisputeGameContract(ctx, metrics.NoopContractMetrics, caller, g.gameType, g.addr)
	if err != nil {
		return proposalOutputRecord{}, contracts.GenericGameMetadata{}, fmt.Errorf("failed to create dispute game contract %v: %w", g.addr, err)
	}
	meta, err := game.GetMetadata(ctx, rpcblock.Latest)
	if err != nil {
		return proposalOutputRecord{}, contracts.GenericGameMetadata{}, fmt.Errorf("failed to retrieve metadata for game %v: %w", g.addr, err)
	}
	l1Header, err := l1.HeaderByHash(ctx, meta.L1Head)
	if err != nil {
		return proposalOutputRecord{}, contracts.GenericGameMetadata{}, fmt.Errorf("failed to retrieve l1Head %v for game %v: %w", meta.L1Head, g.addr, err)
	}
	return proposalOutputRecord{
		Index:        g.index,
		Game:         g.addr.Hex(),
		Status:       meta.Status.String(),
		ProposedRoot: meta.ProposedRoot.Hex(),
		L1Head:       meta.L1Head.Hex(),
		L1HeadNumber: bigs.Uint64Strict(l1Header.Number),
	}, meta, nil
}

func queryOutputRootProposal(ctx context.Context, caller *batching.MultiCaller, l1 l1HeaderSource, rollup outputs.OutputRollupClient, g gameRef) (proposalOutputRecord, error) {
	record, meta, err := loadProposal(ctx, caller, l1, g)
	if err != nil {
		return proposalOutputRecord{}, err
	}
	output, err := rollup.OutputAtBlock(ctx, meta.L2SequenceNum)
	if err != nil {
		return proposalOutputRecord{}, fmt.Errorf("failed to retrieve output at block %v for game %v: %w", meta.L2SequenceNum, g.addr, err)
	}
	safeHead, err := rollup.SafeHeadAtL1Block(ctx, record.L1HeadNumber)
	if err != nil {
		return proposalOutputRecord{}, fmt.Errorf("failed to retrieve safe head at L1 block %v for game %v: %w", record.L1HeadNumber, g.addr, err)
	}
	ourRoot := common.Hash(output.OutputRoot)
	record.L2BlockNumber = ptr.New(meta.L2SequenceNum)
	record.OutputRoot = ourRoot.Hex()
	record.RootMatch = ourRoot == meta.ProposedRoot
	record.SafeHead = ptr.New(safeHead.SafeHead.Number)
	record.SafeHeadAtOrAboveBlock = ptr.New(safeHead.SafeHead.Number >= meta.L2SequenceNum)
	return record, nil
}

func querySuperRootProposal(ctx context.Context, caller *batching.MultiCaller, l1 l1HeaderSource, superRoots super.SuperNodeRootProvider, g gameRef) (proposalOutputRecord, error) {
	record, meta, err := loadProposal(ctx, caller, l1, g)
	if err != nil {
		return proposalOutputRecord{}, err
	}
	resp, err := superRoots.SuperRootAtTimestamp(ctx, meta.L2SequenceNum)
	if err != nil {
		return proposalOutputRecord{}, fmt.Errorf("failed to retrieve super root at timestamp %v for game %v: %w", meta.L2SequenceNum, g.addr, err)
	}
	record.Timestamp = ptr.New(meta.L2SequenceNum)
	// Only L1 blocks strictly below CurrentL1 are fully processed, the same gate the challenger applies.
	record.NodeSynced = ptr.New(resp.CurrentL1.Number > record.L1HeadNumber)
	if resp.Data != nil {
		ourRoot := common.Hash(resp.Data.SuperRoot)
		// Super fault games treat a super root that was not derivable from L1 data up to the game's
		// l1Head as the invalid transition (trace/super/provider.go). ZK games compare the root as is.
		if g.gameType != gameTypes.ZKDisputeGameType && resp.Data.VerifiedRequiredL1.Number > record.L1HeadNumber {
			ourRoot = eth.InvalidTransitionHash
		}
		record.SuperRoot = ourRoot.Hex()
		record.RootMatch = ourRoot == meta.ProposedRoot
	}
	return record, nil
}

// proposalOutputRecord is the structured, machine-readable view of one game's proposal vs our node.
// Output-root games set L2BlockNumber, OutputRoot and the safe-head fields; super-root games set
// Timestamp, SuperRoot and NodeSynced. The other kind's fields are omitted from JSON.
type proposalOutputRecord struct {
	Index                  *uint64 `json:"index,omitempty"` // factory index, omitted for explicit game args
	Game                   string  `json:"game"`
	Status                 string  `json:"status"`
	L2BlockNumber          *uint64 `json:"l2BlockNumber,omitempty"`
	ProposedRoot           string  `json:"proposedRoot"`                     // root claim on-chain
	OutputRoot             string  `json:"outputRoot,omitempty"`             // what our node derives at L2BlockNumber
	Timestamp              *uint64 `json:"timestamp,omitempty"`              // super-root timestamp the game proposes
	SuperRoot              string  `json:"superRoot,omitempty"`              // what our node derives at Timestamp; omitted when it has none
	RootMatch              bool    `json:"rootMatch"`                        // our root == ProposedRoot
	L1Head                 string  `json:"l1Head"`                           // game's L1 head anchor (hash)
	L1HeadNumber           uint64  `json:"l1HeadNumber"`                     // L1 head block number
	SafeHead               *uint64 `json:"safeHead,omitempty"`               // our node's safe head at L1Head
	SafeHeadAtOrAboveBlock *bool   `json:"safeHeadAtOrAboveBlock,omitempty"` // SafeHead >= L2BlockNumber
	NodeSynced             *bool   `json:"nodeSynced,omitempty"`             // our node has processed past L1Head
}

func renderProposalOutputsText(out io.Writer, records []proposalOutputRecord) error {
	var outputRoots, superRoots []proposalOutputRecord
	for _, r := range records {
		if r.Timestamp != nil {
			superRoots = append(superRoots, r)
		} else {
			outputRoots = append(outputRoots, r)
		}
	}
	if len(outputRoots) > 0 || len(superRoots) == 0 {
		lineFormat := "%-42v %-21v %14v %-66v %5v %12v %12v %5v\n"
		if _, err := fmt.Fprintf(out, lineFormat,
			"Game", "Status", "L2 Block", "Output Root (ours)", "Match", "L1 Head", "Safe Head", "Ok"); err != nil {
			return err
		}
		for _, r := range outputRoots {
			if _, err := fmt.Fprintf(out, lineFormat,
				r.Game, r.Status, *r.L2BlockNumber, r.OutputRoot, r.RootMatch,
				r.L1HeadNumber, *r.SafeHead, *r.SafeHeadAtOrAboveBlock); err != nil {
				return err
			}
		}
	}
	if len(superRoots) == 0 {
		return nil
	}
	if len(outputRoots) > 0 {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
	}
	lineFormat := "%-42v %-21v %14v %-66v %5v %12v %6v\n"
	if _, err := fmt.Fprintf(out, lineFormat,
		"Game", "Status", "Timestamp", "Super Root (ours)", "Match", "L1 Head", "Synced"); err != nil {
		return err
	}
	for _, r := range superRoots {
		superRoot := r.SuperRoot
		if superRoot == "" {
			superRoot = "-"
		}
		if _, err := fmt.Fprintf(out, lineFormat,
			r.Game, r.Status, *r.Timestamp, superRoot, r.RootMatch, r.L1HeadNumber, *r.NodeSynced); err != nil {
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
		flags.SuperRootRpcFlag,
		flags.NetworkFlag,
		flags.FactoryAddressFlag,
		flags.GameWindowFlag,
		FormatFlag,
	}
	cliFlags = append(cliFlags, logcli.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var GameProposalOutputsCommand = &cli.Command{
	Name:      "game-proposal-outputs",
	Usage:     "Compare dispute game proposals against what our nodes derive",
	ArgsUsage: "[game-addr ...]",
	Description: "For each dispute game (enumerated from the factory, or the explicitly provided game " +
		"addresses), reports what our nodes derive for the proposal. Output-root games use --rollup-rpc: " +
		"the output root at the proposed L2 block and the safe head at the game's L1 head anchor. " +
		"Super-root games (super-cannon-kona, super-permissioned, zk) use --superroot-rpc: the super root " +
		"at the proposed timestamp and whether the node has processed past the game's L1 head. " +
		"A root mismatch, a safe head below the proposed block, or a missing super root means the " +
		"proposal is invalid by these nodes. At least one of the two RPC flags " +
		"is required; the command fails if any game needs a flag that is not set.",
	Action: Interruptible(GameProposalOutputs),
	Flags:  gameProposalOutputsFlags(),
}
