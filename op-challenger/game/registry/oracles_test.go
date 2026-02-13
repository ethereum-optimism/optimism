package registry

import (
	"context"
	"math/big"
	"testing"

	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDeduplicateOracles(t *testing.T) {
	registry := NewOracleRegistry()
	oracleA := stubPreimageOracle{0xaa}
	oracleB := stubPreimageOracle{0xbb}
	registry.RegisterOracle(oracleA)
	registry.RegisterOracle(oracleB)
	registry.RegisterOracle(oracleB)
	oracles := registry.Oracles()
	require.Len(t, oracles, 2)
	require.Contains(t, oracles, oracleA)
	require.Contains(t, oracles, oracleB)
}

type stubPreimageOracle common.Address

func (s stubPreimageOracle) ChallengePeriod(_ context.Context) (uint64, error) {
	panic("not supported")
}

func (s stubPreimageOracle) GetProposalTreeRoot(_ context.Context, _ rpcblock.Block, _ keccakTypes.LargePreimageIdent) (common.Hash, error) {
	panic("not supported")
}

func (s stubPreimageOracle) ChallengeTx(_ keccakTypes.LargePreimageIdent, _ keccakTypes.Challenge) (txmgr.TxCandidate, error) {
	panic("not supported")
}

func (s stubPreimageOracle) GetInputDataBlocks(_ context.Context, _ rpcblock.Block, _ keccakTypes.LargePreimageIdent) ([]uint64, error) {
	panic("not supported")
}

func (s stubPreimageOracle) DecodeInputData(_ []byte) (*big.Int, keccakTypes.InputData, error) {
	panic("not supported")
}

func (s stubPreimageOracle) Addr() common.Address {
	return common.Address(s)
}

func (s stubPreimageOracle) GetActivePreimages(_ context.Context, _ common.Hash) ([]keccakTypes.LargePreimageMetaData, error) {
	return nil, nil
}
