package contracts

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

//go:embed abis/FaultDisputeGame-0.18.1.json
var faultDisputeGameAbi0180 []byte

type FaultDisputeGameContract0180 struct {
	FaultDisputeGameContractLatest
}

// GetGameMetadata returns the game's L1 head, L2 block number, root claim, status, and max clock duration.
func (f *FaultDisputeGameContract0180) GetGameMetadata(ctx context.Context, block rpcblock.Block) (common.Hash, uint64, common.Hash, gameTypes.GameStatus, uint64, bool, error) {
	defer f.metrics.StartContractRequest("GetGameMetadata")()
	results, err := f.multiCaller.Call(ctx, block,
		f.contract.Call(methodL1Head),
		f.contract.Call(methodL2BlockNumber),
		f.contract.Call(methodRootClaim),
		f.contract.Call(methodStatus),
		f.contract.Call(methodMaxClockDuration),
	)
	if err != nil {
		return common.Hash{}, 0, common.Hash{}, 0, 0, false, fmt.Errorf("failed to retrieve game metadata: %w", err)
	}
	if len(results) != 5 {
		return common.Hash{}, 0, common.Hash{}, 0, 0, false, fmt.Errorf("expected 5 results but got %v", len(results))
	}
	l1Head := results[0].GetHash(0)
	l2BlockNumber := results[1].GetBigInt(0).Uint64()
	rootClaim := results[2].GetHash(0)
	status, err := gameTypes.GameStatusFromUint8(results[3].GetUint8(0))
	if err != nil {
		return common.Hash{}, 0, common.Hash{}, 0, 0, false, fmt.Errorf("failed to convert game status: %w", err)
	}
	duration := results[4].GetUint64(0)
	return l1Head, l2BlockNumber, rootClaim, status, duration, false, nil
}

func (f *FaultDisputeGameContract0180) IsL2BlockNumberChallenged(_ context.Context, _ rpcblock.Block) (bool, error) {
	return false, nil
}

func (f *FaultDisputeGameContract0180) ChallengeL2BlockNumberTx(_ *types.InvalidL2BlockNumberChallenge) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, ErrChallengeL2BlockNotSupported
}
