package contracts

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

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

	methodGameCreator = "gameCreator"
	methodRootClaim   = "rootClaim"
	methodClaim       = "claimData"
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
	networkTimeout time.Duration
}

func NewDisputeGameFactory(addr common.Address, caller *batching.MultiCaller, networkTimeout time.Duration) *DisputeGameFactory {
	factoryABI := snapshots.LoadDisputeGameFactoryABI()
	// Note: Games might have different ABIs (eg SuperFaultDisputeGame) but since only a very small part of the ABI
	// is actually needed, proposer always uses the latest FaultDisputeGameABI. Compatibility with other ABIs is tested
	// in disputegamefactory_test.go
	gameABI := snapshots.LoadFaultDisputeGameABI()
	return &DisputeGameFactory{
		caller:         caller,
		contract:       batching.NewBoundContract(factoryABI, addr),
		gameABI:        gameABI,
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
