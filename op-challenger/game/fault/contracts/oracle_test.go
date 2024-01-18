package contracts

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPreimageOracleContract_LoadKeccak256(t *testing.T) {
	stubRpc, oracle := setupPreimageOracleTest(t)

	data := &types.PreimageOracleData{
		OracleKey:    common.Hash{0xcc}.Bytes(),
		OracleData:   make([]byte, 20),
		OracleOffset: 545,
	}
	stubRpc.SetResponse(oracleAddr, methodLoadKeccak256PreimagePart, batching.BlockLatest, []interface{}{
		new(big.Int).SetUint64(uint64(data.OracleOffset)),
		data.GetPreimageWithoutSize(),
	}, nil)

	tx, err := oracle.AddGlobalDataTx(data)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestPreimageOracleContract_InitLargePreimage(t *testing.T) {
	stubRpc, oracle := setupPreimageOracleTest(t)

	uuid := big.NewInt(123)
	partOffset := uint32(1)
	claimedSize := uint32(2)
	stubRpc.SetResponse(oracleAddr, methodInitLPP, batching.BlockLatest, []interface{}{
		uuid,
		partOffset,
		claimedSize,
	}, nil)

	tx, err := oracle.InitLargePreimage(uuid, partOffset, claimedSize)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestPreimageOracleContract_AddLeaves(t *testing.T) {
	stubRpc, oracle := setupPreimageOracleTest(t)

	uuid := big.NewInt(123)
	leaves := []Leaf{{
		Input:           [136]byte{0x12},
		Index:           big.NewInt(123),
		StateCommitment: common.Hash{0x34},
	}}
	finalize := true
	stubRpc.SetResponse(oracleAddr, methodAddLeavesLPP, batching.BlockLatest, []interface{}{
		uuid,
		leaves[0].Input[:],
		[][32]byte{([32]byte)(leaves[0].StateCommitment.Bytes())},
		finalize,
	}, nil)

	txs, err := oracle.AddLeaves(uuid, leaves, finalize)
	require.NoError(t, err)
	require.Len(t, txs, 1)
	stubRpc.VerifyTxCandidate(txs[0])
}

func TestPreimageOracleContract_Squeeze(t *testing.T) {
	stubRpc, oracle := setupPreimageOracleTest(t)

	claimant := common.Address{0x12}
	uuid := big.NewInt(123)
	stateMatrix := matrix.NewStateMatrix()
	preState := Leaf{
		Input:           [136]byte{0x12},
		Index:           big.NewInt(123),
		StateCommitment: common.Hash{0x34},
	}
	preStateProof := MerkleProof{{0x34}}
	postState := Leaf{
		Input:           [136]byte{0x34},
		Index:           big.NewInt(456),
		StateCommitment: common.Hash{0x56},
	}
	postStateProof := MerkleProof{{0x56}}
	stubRpc.SetResponse(oracleAddr, methodSqueezeLPP, batching.BlockLatest, []interface{}{
		claimant,
		uuid,
		abiEncodeStateMatrix(stateMatrix),
		preState.toPreimageOracleLeaf(),
		preStateProof.toSized(),
		postState.toPreimageOracleLeaf(),
		postStateProof.toSized(),
	}, nil)

	tx, err := oracle.Squeeze(claimant, uuid, stateMatrix, preState, preStateProof, postState, postStateProof)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestGetActivePreimages(t *testing.T) {
	stubRpc, oracle := setupPreimageOracleTest(t)
	blockHash := common.Hash{0xaa}
	stubRpc.SetResponse(
		oracleAddr,
		methodProposalCount,
		batching.BlockByHash(blockHash),
		[]interface{}{},
		[]interface{}{big.NewInt(3)})

	preimage1 := gameTypes.LargePreimageMetaData{
		Claimant: common.Address{0xaa},
		UUID:     big.NewInt(1111),
	}
	preimage2 := gameTypes.LargePreimageMetaData{
		Claimant: common.Address{0xbb},
		UUID:     big.NewInt(2222),
	}
	preimage3 := gameTypes.LargePreimageMetaData{
		Claimant: common.Address{0xcc},
		UUID:     big.NewInt(3333),
	}
	expectGetProposals(stubRpc, batching.BlockByHash(blockHash), preimage1, preimage2, preimage3)
	preimages, err := oracle.GetActivePreimages(context.Background(), blockHash)
	require.NoError(t, err)
	require.Equal(t, []gameTypes.LargePreimageMetaData{preimage1, preimage2, preimage3}, preimages)
}

func expectGetProposals(stubRpc *batchingTest.AbiBasedRpc, block batching.Block, proposals ...gameTypes.LargePreimageMetaData) {
	for i, proposal := range proposals {
		expectGetProposal(stubRpc, block, int64(i), proposal)
	}
}

func expectGetProposal(stubRpc *batchingTest.AbiBasedRpc, block batching.Block, idx int64, proposal gameTypes.LargePreimageMetaData) {
	stubRpc.SetResponse(
		oracleAddr,
		methodProposals,
		block,
		[]interface{}{big.NewInt(idx)},
		[]interface{}{
			proposal.Claimant,
			proposal.UUID,
		})
}

func setupPreimageOracleTest(t *testing.T) (*batchingTest.AbiBasedRpc, *PreimageOracleContract) {
	oracleAbi, err := bindings.PreimageOracleMetaData.GetAbi()
	require.NoError(t, err)

	stubRpc := batchingTest.NewAbiBasedRpc(t, oracleAddr, oracleAbi)
	oracleContract, err := NewPreimageOracleContract(oracleAddr, batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize))
	require.NoError(t, err)

	return stubRpc, oracleContract
}
