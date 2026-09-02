package dsl

import (
	"encoding/binary"
	"math"
	"math/big"
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBridgeGameSequenceAndOutputRootZK(t *testing.T) {
	chainID := uint64(10)
	outputRoot := eth.Bytes32{0: 0xaa}
	superRoot := eth.NewSuperV1(1234, eth.ChainIDAndOutput{
		ChainID: eth.ChainIDFromUInt64(chainID),
		Output:  outputRoot,
	}).Marshal()
	extraData := make([]byte, 4+len(superRoot))
	binary.BigEndian.PutUint32(extraData, math.MaxUint32)
	copy(extraData[4:], superRoot)

	sequence, actualRoot, ok, err := bridgeGameSequenceAndOutputRoot(
		bindings.GameSearchResult{ExtraData: extraData},
		gameTypes.ZKDisputeGameType,
		new(big.Int).SetUint64(chainID),
	)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, uint64(1234), bigs.Uint64Strict(sequence))
	require.Equal(t, common.Hash(outputRoot), actualRoot)
}

func TestBridgeGameSequenceAndOutputRootZKRejectsInvalidExtraData(t *testing.T) {
	invalid := map[string][]byte{
		"empty":         nil,
		"one byte":      make([]byte, 1),
		"two bytes":     make([]byte, 2),
		"three bytes":   make([]byte, 3),
		"prefix only":   make([]byte, 4),
		"invalid proof": append(make([]byte, 4), 0xff),
	}

	for name, extraData := range invalid {
		t.Run(name, func(t *testing.T) {
			sequence, outputRoot, ok, err := bridgeGameSequenceAndOutputRoot(
				bindings.GameSearchResult{ExtraData: extraData},
				gameTypes.ZKDisputeGameType,
				big.NewInt(10),
			)
			require.Error(t, err)
			require.Nil(t, sequence)
			require.Equal(t, common.Hash{}, outputRoot)
			require.False(t, ok)
		})
	}
}

func TestBridgeGameSequenceAndOutputRootZKMissingChain(t *testing.T) {
	superRoot := eth.NewSuperV1(1234, eth.ChainIDAndOutput{
		ChainID: eth.ChainIDFromUInt64(10),
		Output:  eth.Bytes32{0: 0xaa},
	}).Marshal()
	extraData := append(make([]byte, 4), superRoot...)

	sequence, outputRoot, ok, err := bridgeGameSequenceAndOutputRoot(
		bindings.GameSearchResult{ExtraData: extraData},
		gameTypes.ZKDisputeGameType,
		big.NewInt(11),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1234), bigs.Uint64Strict(sequence))
	require.Equal(t, common.Hash{}, outputRoot)
	require.False(t, ok)
}

func TestWithdrawalProvenDisputeGameIndexReturnsCopy(t *testing.T) {
	dt := devtest.SerialT(t)
	stored := big.NewInt(7)
	withdrawal := &Withdrawal{
		commonImpl:   commonFromT(dt),
		proveParams:  ProvenWithdrawalParameters{DisputeGameIndex: stored},
		proveReceipt: &types.Receipt{},
	}

	actual := withdrawal.ProvenDisputeGameIndex()
	require.Equal(t, int64(7), actual.Int64())
	require.NotSame(t, stored, actual)
	actual.SetInt64(8)
	require.Equal(t, int64(7), withdrawal.ProvenDisputeGameIndex().Int64())
}
