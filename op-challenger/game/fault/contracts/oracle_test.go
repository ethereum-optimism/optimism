package contracts

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
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
	input := []byte{0x12}
	commitments := []common.Hash{{0x34}}
	finalize := true
	stubRpc.SetResponse(oracleAddr, methodAddLeavesLPP, batching.BlockLatest, []interface{}{
		uuid,
		input,
		commitments,
		finalize,
	}, nil)

	tx, err := oracle.AddLeaves(uuid, input, commitments, finalize)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestPreimageOracleContract_Squeeze(t *testing.T) {
	stubRpc, oracle := setupPreimageOracleTest(t)

	claimant := common.Address{0x12}
	uuid := big.NewInt(123)
	stateMatrix := matrix.NewStateMatrix()
	preState := keccakTypes.Leaf{
		Input:           [136]byte{0x12},
		Index:           big.NewInt(123),
		StateCommitment: common.Hash{0x34},
	}
	preStateProof := MerkleProof{{0x34}}
	postState := keccakTypes.Leaf{
		Input:           [136]byte{0x34},
		Index:           big.NewInt(456),
		StateCommitment: common.Hash{0x56},
	}
	postStateProof := MerkleProof{{0x56}}
	stubRpc.SetResponse(oracleAddr, methodSqueezeLPP, batching.BlockLatest, []interface{}{
		claimant,
		uuid,
		abiEncodeStateMatrix(stateMatrix),
		toPreimageOracleLeaf(preState),
		preStateProof.toSized(),
		toPreimageOracleLeaf(postState),
		postStateProof.toSized(),
	}, nil)

	tx, err := oracle.Squeeze(claimant, uuid, stateMatrix, preState, preStateProof, postState, postStateProof)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestGetActivePreimages(t *testing.T) {
	blockHash := common.Hash{0xaa}
	_, oracle, proposals := setupPreimageOracleTestWithProposals(t, batching.BlockByHash(blockHash))
	preimages, err := oracle.GetActivePreimages(context.Background(), blockHash)
	require.NoError(t, err)
	require.Equal(t, proposals, preimages)
}

func TestGetProposalMetadata(t *testing.T) {
	blockHash := common.Hash{0xaa}
	block := batching.BlockByHash(blockHash)
	stubRpc, oracle, proposals := setupPreimageOracleTestWithProposals(t, block)
	preimages, err := oracle.GetProposalMetadata(
		context.Background(),
		block,
		proposals[0].LargePreimageIdent,
		proposals[1].LargePreimageIdent,
		proposals[2].LargePreimageIdent,
	)
	require.NoError(t, err)
	require.Equal(t, proposals, preimages)

	// Fetching a proposal that doesn't exist should return an empty metadata object.
	ident := keccakTypes.LargePreimageIdent{Claimant: common.Address{0x12}, UUID: big.NewInt(123)}
	meta := new(metadata)
	stubRpc.SetResponse(
		oracleAddr,
		methodProposalMetadata,
		block,
		[]interface{}{ident.Claimant, ident.UUID},
		[]interface{}{meta})
	preimages, err = oracle.GetProposalMetadata(context.Background(), batching.BlockByHash(blockHash), ident)
	require.NoError(t, err)
	require.Equal(t, []keccakTypes.LargePreimageMetaData{{LargePreimageIdent: ident}}, preimages)
}

func setupPreimageOracleTestWithProposals(t *testing.T, block batching.Block) (*batchingTest.AbiBasedRpc, *PreimageOracleContract, []keccakTypes.LargePreimageMetaData) {
	stubRpc, oracle := setupPreimageOracleTest(t)
	stubRpc.SetResponse(
		oracleAddr,
		methodProposalCount,
		block,
		[]interface{}{},
		[]interface{}{big.NewInt(3)})

	preimage1 := keccakTypes.LargePreimageMetaData{
		LargePreimageIdent: keccakTypes.LargePreimageIdent{
			Claimant: common.Address{0xaa},
			UUID:     big.NewInt(1111),
		},
		Timestamp:       1234,
		PartOffset:      1,
		ClaimedSize:     100,
		BlocksProcessed: 10,
		BytesProcessed:  100,
		Countered:       false,
	}
	preimage2 := keccakTypes.LargePreimageMetaData{
		LargePreimageIdent: keccakTypes.LargePreimageIdent{
			Claimant: common.Address{0xbb},
			UUID:     big.NewInt(2222),
		},
		Timestamp:       2345,
		PartOffset:      2,
		ClaimedSize:     200,
		BlocksProcessed: 20,
		BytesProcessed:  200,
		Countered:       true,
	}
	preimage3 := keccakTypes.LargePreimageMetaData{
		LargePreimageIdent: keccakTypes.LargePreimageIdent{
			Claimant: common.Address{0xcc},
			UUID:     big.NewInt(3333),
		},
		Timestamp:       0,
		PartOffset:      3,
		ClaimedSize:     300,
		BlocksProcessed: 30,
		BytesProcessed:  233,
		Countered:       false,
	}

	proposals := []keccakTypes.LargePreimageMetaData{preimage1, preimage2, preimage3}

	for i, proposal := range proposals {
		stubRpc.SetResponse(
			oracleAddr,
			methodProposals,
			block,
			[]interface{}{big.NewInt(int64(i))},
			[]interface{}{
				proposal.Claimant,
				proposal.UUID,
			})
		meta := new(metadata)
		meta.setTimestamp(proposal.Timestamp)
		meta.setPartOffset(proposal.PartOffset)
		meta.setClaimedSize(proposal.ClaimedSize)
		meta.setBlocksProcessed(proposal.BlocksProcessed)
		meta.setBytesProcessed(proposal.BytesProcessed)
		meta.setCountered(proposal.Countered)
		stubRpc.SetResponse(
			oracleAddr,
			methodProposalMetadata,
			block,
			[]interface{}{proposal.Claimant, proposal.UUID},
			[]interface{}{meta})
	}

	return stubRpc, oracle, proposals

}

func setupPreimageOracleTest(t *testing.T) (*batchingTest.AbiBasedRpc, *PreimageOracleContract) {
	oracleAbi, err := bindings.PreimageOracleMetaData.GetAbi()
	require.NoError(t, err)

	stubRpc := batchingTest.NewAbiBasedRpc(t, oracleAddr, oracleAbi)
	oracleContract, err := NewPreimageOracleContract(oracleAddr, batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize))
	require.NoError(t, err)

	return stubRpc, oracleContract
}

func TestMetadata(t *testing.T) {
	uint32Values := []uint32{0, 1, 2, 3252354, math.MaxUint32}
	tests := []struct {
		name   string
		setter func(meta *metadata, val uint32)
		getter func(meta *metadata) uint32
	}{
		{
			name:   "partOffset",
			setter: (*metadata).setPartOffset,
			getter: (*metadata).partOffset,
		},
		{
			name:   "claimedSize",
			setter: (*metadata).setClaimedSize,
			getter: (*metadata).claimedSize,
		},
		{
			name:   "blocksProcessed",
			setter: (*metadata).setBlocksProcessed,
			getter: (*metadata).blocksProcessed,
		},
		{
			name:   "bytesProcessed",
			setter: (*metadata).setBytesProcessed,
			getter: (*metadata).bytesProcessed,
		},
	}
	for _, test := range tests {
		test := test
		for _, value := range uint32Values {
			value := value
			t.Run(fmt.Sprintf("%v-%v", test.name, value), func(t *testing.T) {
				meta := new(metadata)
				require.Zero(t, test.getter(meta))
				test.setter(meta, value)
				require.Equal(t, value, test.getter(meta))
			})
		}
	}
}

func TestMetadata_Timestamp(t *testing.T) {
	values := []uint64{0, 1, 2, 3252354, math.MaxUint32, math.MaxUint32 + 1, math.MaxUint64}
	var meta metadata
	require.Zero(t, meta.timestamp())
	for _, value := range values {
		meta.setTimestamp(value)
		require.Equal(t, value, meta.timestamp())
	}
}

func TestMetadata_Countered(t *testing.T) {
	var meta metadata
	require.False(t, meta.countered())
	meta.setCountered(true)
	require.True(t, meta.countered())
	meta.setCountered(false)
	require.False(t, meta.countered())
}
