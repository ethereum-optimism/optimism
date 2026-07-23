package contracts

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	methodGameCount   = "gameCount"
	methodGameAtIndex = "gameAtIndex"
	methodInitBonds   = "initBonds"
	methodCreateGame  = "create"
	methodVersion     = "version"
	methodGameArgs    = "gameArgs"

	methodGameCreator = "gameCreator"
	methodRootClaim   = "rootClaim"
	methodClaim       = "claimData"

	// ZKDisputeGame methods used for parent selection.
	methodStatus           = "status"
	methodL2SequenceNumber = "l2SequenceNumber"

	// AnchorStateRegistry methods used for parent selection.
	methodGetAnchorRoot     = "getAnchorRoot"
	methodIsGameBlacklisted = "isGameBlacklisted"
	methodIsGameRetired     = "isGameRetired"

	// gameStatusChallengerWins is the CHALLENGER_WINS variant of the GameStatus
	// enum in packages/contracts-bedrock/src/dispute/lib/Types.sol
	// (0 = IN_PROGRESS, 1 = CHALLENGER_WINS, 2 = DEFENDER_WINS).
	gameStatusChallengerWins = uint8(1)
)

type gameInfo struct {
	GameType  uint32
	Timestamp time.Time
	Address   common.Address
}

type proposalMetadata struct {
	Proposer common.Address
	Claim    common.Hash
}

type DisputeGameFactory struct {
	caller         *batching.MultiCaller
	contract       *batching.BoundContract
	gameABI        *abi.ABI
	zkGameABI      *abi.ABI
	asrABI         *abi.ABI
	networkTimeout time.Duration
}

func NewDisputeGameFactory(addr common.Address, caller *batching.MultiCaller, networkTimeout time.Duration) *DisputeGameFactory {
	factoryABI := snapshots.LoadDisputeGameFactoryABI()
	// Note: Games might have different ABIs (eg SuperFaultDisputeGame) but since only a very small part of the ABI
	// is actually needed, proposer always uses the latest FaultDisputeGameABI. Compatibility with other ABIs is tested
	// in disputegamefactory_test.go
	gameABI := snapshots.LoadFaultDisputeGameABI()
	// The ZK game and anchor state registry ABIs are only used when proposing
	// ZK dispute games (parent selection); they are loaded unconditionally to
	// match the gameABI pattern above.
	zkGameABI := snapshots.LoadZKDisputeGameABI()
	asrABI := snapshots.LoadAnchorStateRegistryABI()
	return &DisputeGameFactory{
		caller:         caller,
		contract:       batching.NewBoundContract(factoryABI, addr),
		gameABI:        gameABI,
		zkGameABI:      zkGameABI,
		asrABI:         asrABI,
		networkTimeout: networkTimeout,
	}
}

func (f *DisputeGameFactory) Version(ctx context.Context) (string, error) {
	cCtx, cancel := context.WithTimeout(ctx, f.networkTimeout)
	defer cancel()
	result, err := f.caller.SingleCall(cCtx, rpcblock.Latest, f.contract.Call(methodVersion))
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}
	return result.GetString(0), nil
}

// HasProposedSince attempts to find a game with the specified game type created by the specified proposer after the
// given cut off time. If one is found, returns true and the time the game was created at.
// If no matching proposal is found, returns false, time.Time{}, nil
func (f *DisputeGameFactory) HasProposedSince(ctx context.Context, proposer common.Address, cutoff time.Time, gameType uint32) (bool, time.Time, common.Hash, error) {
	gameCount, err := f.gameCount(ctx)
	if err != nil {
		return false, time.Time{}, common.Hash{}, fmt.Errorf("failed to get dispute game count: %w", err)
	}
	if gameCount == 0 {
		return false, time.Time{}, common.Hash{}, nil
	}
	for idx := gameCount - 1; ; idx-- {
		game, err := f.gameAtIndex(ctx, idx)
		if err != nil {
			return false, time.Time{}, common.Hash{}, fmt.Errorf("failed to get dispute game %d: %w", idx, err)
		}
		if game.Timestamp.Before(cutoff) {
			// Reached a game that is before the expected cutoff, so we haven't found a suitable proposal
			return false, time.Time{}, common.Hash{}, nil
		}
		if game.GameType == gameType {
			metadata, err := f.loadGameMetadata(ctx, game.Address)
			if err != nil {
				return false, time.Time{}, common.Hash{}, fmt.Errorf("failed to get metadata for dispute game %d: %w", idx, err)
			}
			if metadata.Proposer == proposer {
				// Found a matching proposal
				return true, game.Timestamp, metadata.Claim, nil
			}
		}
		if idx == 0 { // Need to check here rather than in the for condition to avoid underflow
			// Checked every game and didn't find a match
			return false, time.Time{}, common.Hash{}, nil
		}
	}
}

func (f *DisputeGameFactory) ProposalTx(ctx context.Context, gameType uint32, outputRoot common.Hash, extraData []byte) (txmgr.TxCandidate, error) {
	cCtx, cancel := context.WithTimeout(ctx, f.networkTimeout)
	defer cancel()
	result, err := f.caller.SingleCall(cCtx, rpcblock.Latest, f.contract.Call(methodInitBonds, gameType))
	if err != nil {
		return txmgr.TxCandidate{}, fmt.Errorf("failed to fetch init bond: %w", err)
	}
	initBond := result.GetBigInt(0)
	call := f.contract.Call(methodCreateGame, gameType, outputRoot, extraData)
	candidate, err := call.ToTxCandidate()
	if err != nil {
		return txmgr.TxCandidate{}, err
	}
	candidate.Value = initBond
	return candidate, err
}

func (f *DisputeGameFactory) gameCount(ctx context.Context) (uint64, error) {
	cCtx, cancel := context.WithTimeout(ctx, f.networkTimeout)
	defer cancel()
	result, err := f.caller.SingleCall(cCtx, rpcblock.Latest, f.contract.Call(methodGameCount))
	if err != nil {
		return 0, fmt.Errorf("failed to load game count: %w", err)
	}
	return bigs.Uint64Strict(result.GetBigInt(0)), nil
}

func (f *DisputeGameFactory) gameAtIndex(ctx context.Context, idx uint64) (gameInfo, error) {
	cCtx, cancel := context.WithTimeout(ctx, f.networkTimeout)
	defer cancel()
	result, err := f.caller.SingleCall(cCtx, rpcblock.Latest, f.contract.Call(methodGameAtIndex, new(big.Int).SetUint64(idx)))
	if err != nil {
		return gameInfo{}, fmt.Errorf("failed to load game %v: %w", idx, err)
	}
	return gameInfo{
		GameType:  result.GetUint32(0),
		Timestamp: time.Unix(int64(result.GetUint64(1)), 0),
		Address:   result.GetAddress(2),
	}, nil
}

func (f *DisputeGameFactory) loadGameMetadata(ctx context.Context, address common.Address) (proposalMetadata, error) {
	gameContract := batching.NewBoundContract(f.gameABI, address)
	cCtx, cancel := context.WithTimeout(ctx, f.networkTimeout)
	defer cancel()
	results, metadataErr := f.caller.Call(cCtx, rpcblock.Latest,
		gameContract.Call(methodGameCreator),
		gameContract.Call(methodRootClaim),
	)
	var claimant common.Address
	var claim common.Hash
	if metadataErr == nil {
		claimant = results[0].GetAddress(0)
		claim = results[1].GetHash(0)
	} else {
		cCtx, cancel = context.WithTimeout(ctx, f.networkTimeout)
		defer cancel()
		result, legacyErr := f.caller.SingleCall(cCtx, rpcblock.Latest, gameContract.Call(methodClaim, big.NewInt(0)))
		if legacyErr != nil {
			return proposalMetadata{}, errors.Join(
				fmt.Errorf("common getters failed: %w", metadataErr),
				fmt.Errorf("legacy claimData(0) failed: %w", legacyErr),
			)
		}
		claimant = result.GetAddress(2)
		claim = result.GetHash(4)
	}

	return proposalMetadata{
		Proposer: claimant,
		Claim:    claim,
	}, nil
}

// singleCall performs one contract read with its own network timeout.
func (f *DisputeGameFactory) singleCall(ctx context.Context, call batching.Call) (*batching.CallResult, error) {
	cCtx, cancel := context.WithTimeout(ctx, f.networkTimeout)
	defer cancel()
	return f.caller.SingleCall(cCtx, rpcblock.Latest, call)
}

// batchCall performs one batched contract read with its own network timeout.
func (f *DisputeGameFactory) batchCall(ctx context.Context, calls ...batching.Call) ([]*batching.CallResult, error) {
	cCtx, cancel := context.WithTimeout(ctx, f.networkTimeout)
	defer cancel()
	return f.caller.Call(cCtx, rpcblock.Latest, calls...)
}

// SelectZKParent picks the parent game index for a new ZK dispute game
// proposal. It walks factory games newest-to-oldest and returns the first
// candidate that ZKDisputeGame.initialize() would accept as a parent
// (see ZKDisputeGame.sol): same game type, status != CHALLENGER_WINS, not
// blacklisted, not retired, and an L2 sequence number strictly above the
// anchor. If no candidate qualifies it returns parentIndex = math.MaxUint32,
// anchoring the new game at the anchor state registry's anchor root.
// The returned starting sequence number is the value the new proposal's
// sequence number must strictly exceed: the chosen parent's, or the anchor's
// when there is no parent.
func (f *DisputeGameFactory) SelectZKParent(ctx context.Context, gameType uint32) (uint32, uint64, error) {
	asr, err := f.zkAnchorStateRegistry(ctx, gameType)
	if err != nil {
		return 0, 0, err
	}
	anchorSeq, err := f.anchorSequenceNumber(ctx, asr)
	if err != nil {
		return 0, 0, err
	}
	gameCount, err := f.gameCount(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get dispute game count: %w", err)
	}

	// Walk games newest-to-oldest, paging gameAtIndex reads through the
	// multicaller. Unlike HasProposedSince this walk has no time cutoff, so
	// one-call-per-index would stall the proposer for minutes on a mature
	// factory with no ZK games yet (the bootstrap case).
	pageSize := uint64(f.caller.BatchSize())
	for end := gameCount; end > 0; {
		start := uint64(0)
		if end > pageSize {
			start = end - pageSize
		}
		calls := make([]batching.Call, 0, end-start)
		for idx := end; idx > start; idx-- {
			calls = append(calls, f.contract.Call(methodGameAtIndex, new(big.Int).SetUint64(idx-1)))
		}
		results, err := f.batchCall(ctx, calls...)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to load games %v-%v: %w", start, end-1, err)
		}
		for i, result := range results {
			idx := end - 1 - uint64(i)
			if result.GetUint32(0) != gameType {
				continue
			}
			gameAddr := result.GetAddress(2)
			ok, seqNum, err := f.isValidZKParent(ctx, asr, gameAddr, gameType, anchorSeq)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to check ZK parent candidate %v: %w", idx, err)
			}
			if !ok {
				continue
			}
			if idx > math.MaxUint32 {
				return 0, 0, fmt.Errorf("valid parent game index %v does not fit in uint32", idx)
			}
			return uint32(idx), seqNum, nil
		}
		end = start
	}
	// No acceptable parent: propose a root game anchored at the anchor root.
	return math.MaxUint32, anchorSeq, nil
}

// isValidZKParent replicates the parent checks in ZKDisputeGame.initialize():
// status != CHALLENGER_WINS, not blacklisted, not retired, and an L2 sequence
// number strictly above the anchor. The game type is checked by the caller.
func (f *DisputeGameFactory) isValidZKParent(ctx context.Context, asr *batching.BoundContract, gameAddr common.Address, gameType uint32, anchorSeq uint64) (bool, uint64, error) {
	game := batching.NewBoundContract(f.zkGameABI, gameAddr)
	results, err := f.batchCall(ctx,
		game.Call(methodStatus),
		game.Call(methodL2SequenceNumber),
		asr.Call(methodIsGameBlacklisted, gameAddr),
		asr.Call(methodIsGameRetired, gameAddr),
	)
	if err != nil {
		return false, 0, fmt.Errorf("failed to load ZK parent candidate state: %w", err)
	}
	status := results[0].GetUint8(0)
	seqNum := bigs.Uint64Strict(results[1].GetBigInt(0))
	blacklisted := results[2].GetBool(0)
	retired := results[3].GetBool(0)
	valid := status != gameStatusChallengerWins && !blacklisted && !retired && seqNum > anchorSeq
	return valid, seqNum, nil
}

// zkAnchorStateRegistry resolves the anchor state registry for the given game
// type from the factory's registered game args.
func (f *DisputeGameFactory) zkAnchorStateRegistry(ctx context.Context, gameType uint32) (*batching.BoundContract, error) {
	result, err := f.singleCall(ctx, f.contract.Call(methodGameArgs, gameType))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game args for game type %v: %w", gameType, err)
	}
	args, err := gameargs.ParseZK(result.GetBytes(0))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ZK game args for game type %v: %w", gameType, err)
	}
	return batching.NewBoundContract(f.asrABI, args.AnchorStateRegistry), nil
}

// anchorSequenceNumber reads the anchor root and returns its L2 sequence
// number, rejecting an unset anchor (game creation would revert with
// AnchorRootNotFound).
func (f *DisputeGameFactory) anchorSequenceNumber(ctx context.Context, asr *batching.BoundContract) (uint64, error) {
	result, err := f.singleCall(ctx, asr.Call(methodGetAnchorRoot))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch anchor root: %w", err)
	}
	if result.GetHash(0) == (common.Hash{}) {
		return 0, errors.New("anchor state registry has no anchor root")
	}
	return bigs.Uint64Strict(result.GetBigInt(1)), nil
}
