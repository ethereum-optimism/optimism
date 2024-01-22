package contracts

import (
	"bytes"
	"context"
	"fmt"
	"math"
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
	input := []byte{0x12}
	commitments := [][32]byte{{0x34}}
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
	preState := gameTypes.Leaf{
		Input:           bytes.Repeat([]byte{0x12}, gameTypes.LeafSize),
		Index:           big.NewInt(123),
		StateCommitment: common.Hash{0x34},
	}
	preStateProof := MerkleProof{{0x34}}
	postState := gameTypes.Leaf{
		Input:           bytes.Repeat([]byte{0x14}, gameTypes.LeafSize),
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
	stubRpc, oracle := setupPreimageOracleTest(t)
	blockHash := common.Hash{0xaa}
	stubRpc.SetResponse(
		oracleAddr,
		methodProposalCount,
		batching.BlockByHash(blockHash),
		[]interface{}{},
		[]interface{}{big.NewInt(3)})

	preimage1 := gameTypes.LargePreimageMetaData{
		LargePreimageIdent: gameTypes.LargePreimageIdent{
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
	preimage2 := gameTypes.LargePreimageMetaData{
		LargePreimageIdent: gameTypes.LargePreimageIdent{
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
	preimage3 := gameTypes.LargePreimageMetaData{
		LargePreimageIdent: gameTypes.LargePreimageIdent{
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

func TestGetLeafBlocks(t *testing.T) {
	stubRpc, oracle := setupPreimageOracleTest(t)
	block := batching.BlockByHash(common.Hash{0xaa})

	preimage := gameTypes.LargePreimageIdent{
		Claimant: common.Address{0xbb},
		UUID:     big.NewInt(2222),
	}

	stubRpc.SetResponse(
		oracleAddr,
		methodProposalBlocksLen,
		block,
		[]interface{}{preimage.Claimant, preimage.UUID},
		[]interface{}{big.NewInt(3)})

	blockNums := []uint64{10, 35, 67}

	for i, blockNum := range blockNums {
		stubRpc.SetResponse(
			oracleAddr,
			methodProposalBlocks,
			block,
			[]interface{}{preimage.Claimant, preimage.UUID, big.NewInt(int64(i))},
			[]interface{}{blockNum})
	}

	actual, err := oracle.GetLeafBlocks(context.Background(), block, preimage)
	require.NoError(t, err)
	require.Len(t, actual, 3)
	require.Equal(t, blockNums, actual)
}

type InputType uint8

const (
	InputTypeFull InputType = iota
	InputTypeFullyTrimmed
	InputTypePartiallyTrimmed
	InputTypeTruncateFirstLeaf
)

func TestDecodeLeafData(t *testing.T) {
	ident := gameTypes.LargePreimageIdent{
		Claimant: common.Address{0xaa},
		UUID:     big.NewInt(1111),
	}
	_, oracle := setupPreimageOracleTest(t)
	leaf1 := gameTypes.Leaf{
		Input:           bytes.Repeat([]byte{1}, gameTypes.LeafSize),
		StateCommitment: common.Hash{0xaa, 0xbb, 0xcc},
	}
	fullLeaf := gameTypes.Leaf{
		Input:           bytes.Repeat([]byte{2}, gameTypes.LeafSize),
		StateCommitment: common.Hash{0xdd, 0xee, 0xff},
	}
	finalLeaf := gameTypes.Leaf{
		Input:           []byte{22, 23, 24, 25},
		StateCommitment: common.Hash{0xdd, 0xee, 0xff},
	}
	trailingZerosLeaf := gameTypes.Leaf{
		Input:           []byte{22, 23, 24, 25, 0, 0},
		StateCommitment: common.Hash{0xdd, 0xee, 0xff, 0x00, 0x00},
	}
	leafUntrimmable := gameTypes.Leaf{
		Input:           bytes.Repeat([]byte{0}, gameTypes.LeafSize),
		StateCommitment: common.Hash{0xdd, 0xee, 0xff},
	}
	leafUntrimmable.Input[0] = 1
	leafUntrimmable.Input[gameTypes.LeafSize-1] = 1

	tests := []struct {
		name           string
		input          []byte
		expectedTxData string
		expectedLeafs  []gameTypes.Leaf
		expectedErr    error
	}{
		{
			name:           "UnknownMethod",
			input:          []byte{0xaa, 0xbb, 0xcc, 0xdd},
			expectedTxData: "aabbccdd",
			expectedErr:    ErrInvalidAddLeavesCall,
		},
		{
			name:           "SingleLeaf",
			input:          toAddLeavesTxData(t, oracle, ident.UUID, false, InputTypeFull, leaf1),
			expectedTxData: "9f99ef8200000000000000000000000000000000000000000000000000000000000004570000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000088010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001aabbcc0000000000000000000000000000000000000000000000000000000000",
			expectedLeafs:  []gameTypes.Leaf{leaf1},
			expectedErr:    nil,
		},
		{
			name:           "MultipleLeafs",
			input:          toAddLeavesTxData(t, oracle, ident.UUID, false, InputTypeFull, leaf1, fullLeaf),
			expectedTxData: "9f99ef820000000000000000000000000000000000000000000000000000000000000457000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000001c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001100101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010102020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002aabbcc0000000000000000000000000000000000000000000000000000000000ddeeff0000000000000000000000000000000000000000000000000000000000",
			expectedLeafs:  []gameTypes.Leaf{leaf1, fullLeaf},
			expectedErr:    nil,
		},
		{
			name:           "MultipleLeafs-InputTooShort",
			input:          toAddLeavesTxData(t, oracle, ident.UUID, false, InputTypePartiallyTrimmed, leaf1, fullLeaf),
			expectedTxData: "9f99ef820000000000000000000000000000000000000000000000000000000000000457000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000001c00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010c0101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010102020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020202020200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002aabbcc0000000000000000000000000000000000000000000000000000000000ddeeff0000000000000000000000000000000000000000000000000000000000",
			expectedErr:    ErrInvalidAddLeavesCall,
		},
		{
			name:           "MultipleLeafs-FinalizePadding",
			input:          toAddLeavesTxData(t, oracle, ident.UUID, true, InputTypeFullyTrimmed, leaf1, finalLeaf),
			expectedTxData: "9f99ef820000000000000000000000000000000000000000000000000000000000000457000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000008c010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101011617181900000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002aabbcc0000000000000000000000000000000000000000000000000000000000ddeeff0000000000000000000000000000000000000000000000000000000000",
			expectedLeafs:  []gameTypes.Leaf{leaf1, finalLeaf},
			expectedErr:    nil,
		},
		{
			name:           "MultipleLeafs-FinalizePadding-NotTrimmed",
			input:          toAddLeavesTxData(t, oracle, ident.UUID, true, InputTypeFull, leaf1, leafUntrimmable),
			expectedTxData: "9f99ef820000000000000000000000000000000000000000000000000000000000000457000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000001c0000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001100101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002aabbcc0000000000000000000000000000000000000000000000000000000000ddeeff0000000000000000000000000000000000000000000000000000000000",
			expectedLeafs:  []gameTypes.Leaf{leaf1, leafUntrimmable},
			expectedErr:    nil,
		},
		{
			name:           "MultipleLeafs-FinalizePadding-TrailingZeros",
			input:          toAddLeavesTxData(t, oracle, ident.UUID, true, InputTypeFull, leaf1, trailingZerosLeaf),
			expectedTxData: "9f99ef820000000000000000000000000000000000000000000000000000000000000457000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000008e010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101011617181900000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002aabbcc0000000000000000000000000000000000000000000000000000000000ddeeff0000000000000000000000000000000000000000000000000000000000",
			expectedLeafs:  []gameTypes.Leaf{leaf1, trailingZerosLeaf},
			expectedErr:    nil,
		},
		{
			name:           "MultipleLeafs-FinalizePadding-FirstLeafTooShort",
			input:          toAddLeavesTxData(t, oracle, ident.UUID, true, InputTypeTruncateFirstLeaf, leaf1, fullLeaf),
			expectedTxData: "9f99ef8200000000000000000000000000000000000000000000000000000000000004570000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000082010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002aabbcc0000000000000000000000000000000000000000000000000000000000ddeeff0000000000000000000000000000000000000000000000000000000000",
			expectedErr:    ErrInvalidAddLeavesCall,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expectedTxData, common.Bytes2Hex(test.input),
				"ABI has been changed. Add tests to ensure historic transactions can be parsed before updating expectedTxData")
			uuid, leaves, err := oracle.DecodeLeafData(test.input)
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, ident.UUID, uuid)
			}
			require.Equal(t, test.expectedLeafs, leaves)
		})
	}
}

func toAddLeavesTxData(t *testing.T, oracle *PreimageOracleContract, uuid *big.Int, finalize bool, mod InputType, leaves ...gameTypes.Leaf) []byte {
	var input []byte
	var commitments [][32]byte
	for _, leaf := range leaves {
		input = append(input, leaf.Input...)
		commitments = append(commitments, leaf.StateCommitment)
	}
	if mod == InputTypeFullyTrimmed {
		// Trim trailing zeros from the final leaf data if we are finalizing - the contract will pad automatically
		i := len(input)
		for i >= 0 && input[i-1] == 0 {
			i--
		}
		input = input[0:i]
	}
	if mod == InputTypePartiallyTrimmed {
		// Assumes the last 4 bytes are 0s
		input = input[0 : len(input)-4]
	}
	if mod == InputTypeTruncateFirstLeaf {
		input = input[0 : gameTypes.LeafSize-6]
	}
	tx, err := oracle.AddLeaves(uuid, input, commitments, finalize)
	require.NoError(t, err)
	return tx.TxData
}
