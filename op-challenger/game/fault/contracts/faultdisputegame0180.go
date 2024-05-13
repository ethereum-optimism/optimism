package contracts

import (
	"context"
	_ "embed"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
)

//go:embed abis/FaultDisputeGame-0.18.1.json
var faultDisputeGameAbi0180 []byte

type FaultDisputeGameContract0180 struct {
	FaultDisputeGameContractLatest
}

func (f *FaultDisputeGameContract0180) IsL2BlockNumberChallenged(_ context.Context, _ rpcblock.Block) (bool, error) {
	return false, nil
}

func (f *FaultDisputeGameContract0180) ChallengeL2BlockNumberTx(_ *types.InvalidL2BlockNumberChallenge) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, ErrChallengeL2BlockNotSupported
}

func (f *FaultDisputeGameContract0180) AttackTx(ctx context.Context, parent types.Claim, pivot common.Hash) (txmgr.TxCandidate, error) {
	call := f.contract.Call(methodAttack, big.NewInt(int64(parent.ContractIndex)), pivot)
	return f.txWithBond(ctx, parent.Position.Attack(), call)
}

func (f *FaultDisputeGameContract0180) DefendTx(ctx context.Context, parent types.Claim, pivot common.Hash) (txmgr.TxCandidate, error) {
	call := f.contract.Call(methodDefend, big.NewInt(int64(parent.ContractIndex)), pivot)
	return f.txWithBond(ctx, parent.Position.Defend(), call)
}
