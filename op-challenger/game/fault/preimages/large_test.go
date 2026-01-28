package preimages

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/merkle"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockChallengePeriod  = uint64(10000000)
	mockAddLeavesError   = errors.New("mock add leaves error")
	mockSqueezeError     = errors.New("mock squeeze error")
	mockSqueezeCallError = errors.New("mock squeeze call error")
)

func TestLargePreimageUploader_NewUUID(t *testing.T) {
	tests := []struct {
		name         string
		data         *types.PreimageOracleData
		expectedUUID *big.Int
	}{
		{
			name:         "EmptyOracleData",
			data:         makePreimageData([]byte{}, 0),
			expectedUUID: new(big.Int).SetBytes(common.FromHex("827b659bbda2a0bdecce2c91b8b68462545758f3eba2dbefef18e0daf84f5ccd")),
		},
		{
			name:         "OracleDataAndOffset_Control",
			data:         makePreimageData([]byte{1, 2, 3}, 0x010203),
			expectedUUID: new(big.Int).SetBytes(common.FromHex("641e230bcf3ade8c71b7e591d210184cdb190e853f61ba59a1411c3b7aca9890")),
		},
		{
			name:         "OracleDataAndOffset_DifferentOffset",
			data:         makePreimageData([]byte{1, 2, 3}, 0x010204),
			expectedUUID: new(big.Int).SetBytes(common.FromHex("aec56de44401325420e5793f72b777e3e547778de7d8344004b31be086a3136d")),
		},
		{
			name:         "OracleDataAndOffset_DifferentData",
			data:         makePreimageData([]byte{1, 2, 3, 4}, 0x010203),
			expectedUUID: new(big.Int).SetBytes(common.FromHex("ca38aa17d56805cf26376a050c2c7b15b6be4e709bc422a1c679fe21aa6aa8c7")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oracle, _, _, _ := newTestLargePreimageUploader(t)
			uuid := NewUUID(oracle.txSender.From(), test.data)
			require.Equal(t, test.expectedUUID, uuid)
		})
	}
}

func TestLargePreimageUploader_UploadPreimage_EdgeCases(t *testing.T) {
	t.Run("InitFails", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		contract.initFails = true
		data := mockPreimageOracleData()
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, mockInitLPPError)
		require.Equal(t, 1, contract.initCalls)
	})

	t.Run("AddLeavesFails", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		contract.addFails = true
		data := mockPreimageOracleData()
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, mockAddLeavesError)
		require.Equal(t, 1, contract.addCalls)
	})

	t.Run("NoBytesProcessed", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.claimedSize = uint32(len(data.GetPreimageWithoutSize()))
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, ErrChallengePeriodNotOver, "Data uploaded but can't squeeze yet")
		require.Equal(t, 1, contract.initCalls)
		require.Equal(t, 6, contract.addCalls)
		require.Equal(t, data.GetPreimageWithoutSize(), contract.addData)
	})

	t.Run("AlreadyInitialized", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.initialized = true
		_, err := contract.InitLargePreimage(NewUUID(oracle.txSender.From(), data), 0, uint32(len(data.GetPreimageWithoutSize())))
		require.NoError(t, err)
		contract.initCalls = 0 // Reset
		contract.claimedSize = uint32(len(data.GetPreimageWithoutSize()))
		err = oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, ErrChallengePeriodNotOver, "Data uploaded but can't squeeze yet")
		require.Equal(t, 0, contract.initCalls)
		require.Equal(t, 6, contract.addCalls)
	})

	t.Run("PartiallyUploaded", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		uuid := NewUUID(oracle.txSender.From(), data)
		_, err := contract.InitLargePreimage(uuid, 0, uint32(len(data.GetPreimageWithoutSize())))
		require.NoError(t, err)
		// Add some but not all leaves
		s := matrix.NewStateMatrix()
		call, err := s.AbsorbUpTo(bytes.NewReader(data.GetPreimageWithoutSize()), keccakTypes.BlockSize*3)
		require.NoError(t, err)
		_, err = contract.AddLeaves(uuid, big.NewInt(0), call.Input, call.Commitments, call.Finalize)
		require.NoError(t, err)
		// Reset call counts
		contract.initCalls = 0
		contract.addCalls = 0

		// Then start the upload process again
		err = oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, ErrChallengePeriodNotOver, "Data uploaded but can't squeeze yet")
		require.Equal(t, 0, contract.initCalls)
		require.Equal(t, data.GetPreimageWithoutSize(), contract.addData)
		//require.Equal(t, 5, contract.addCalls)
	})

	t.Run("ChallengePeriodNotElapsed", func(t *testing.T) {
		oracle, cl, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		// Upload preimage successfully
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, ErrChallengePeriodNotOver, "Data uploaded but can't squeeze yet")

		// Can't squeeze before challenge period expires
		err = oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, ErrChallengePeriodNotOver)
		require.Equal(t, 0, contract.squeezeCalls)

		// Squeeze should be called once the challenge period has elapsed.
		cl.AdvanceTime(time.Duration(mockChallengePeriod+1) * time.Second)
		err = oracle.UploadPreimage(context.Background(), 0, data)
		require.NoError(t, err)
		require.Equal(t, 1, contract.squeezeCalls)
	})

	t.Run("SqueezeCallFails", func(t *testing.T) {
		oracle, cl, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()

		// Upload preimage successfully
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, ErrChallengePeriodNotOver, "Data uploaded but can't squeeze yet")

		// Advance time so squeeze is allowed
		cl.AdvanceTime(time.Duration(mockChallengePeriod+1) * time.Second)

		// But squeeze call fails
		contract.squeezeCallFails = true
		err = oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, mockSqueezeCallError)
		require.Equal(t, 0, contract.squeezeCalls)
	})

	t.Run("SqueezeFails", func(t *testing.T) {
		oracle, cl, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()

		// Upload preimage successfully
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, ErrChallengePeriodNotOver, "Data uploaded but can't squeeze yet")

		// Advance time so squeeze is allowed
		cl.AdvanceTime(time.Duration(mockChallengePeriod+1) * time.Second)

		// But squeeze call creation fails.
		contract.squeezeFails = true
		err = oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, mockSqueezeError)
		require.Equal(t, 1, contract.squeezeCalls)
	})

	t.Run("AllBytesProcessed", func(t *testing.T) {
		oracle, cl, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()

		// Upload preimage successfully
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, ErrChallengePeriodNotOver, "Data uploaded but can't squeeze yet")
		// Reset call counts
		contract.initCalls = 0
		contract.addCalls = 0
		contract.addData = nil

		// Advance time so squeeze is allowed
		cl.AdvanceTime(time.Duration(mockChallengePeriod+1) * time.Second)

		err = oracle.UploadPreimage(context.Background(), 0, data)
		require.NoError(t, err)
		require.Equal(t, 0, contract.initCalls)
		require.Equal(t, 0, contract.addCalls)
		require.Empty(t, contract.addData)
	})

	t.Run("StoragePartMustNotSpanChunk", func(t *testing.T) {
		for offsetFromMaxChunkSize := -41; offsetFromMaxChunkSize < 10; offsetFromMaxChunkSize++ {
			t.Run(fmt.Sprintf("offsetFromMaxChunkSize=%d", offsetFromMaxChunkSize), func(t *testing.T) {
				oracle, _, _, contract := newTestLargePreimageUploader(t)
				data := mockPreimageOracleData()
				data.OracleOffset = uint32(MaxChunkSize + offsetFromMaxChunkSize)

				// Upload preimage successfully
				err := oracle.UploadPreimage(context.Background(), 0, data)
				require.ErrorIs(t, err, ErrChallengePeriodNotOver, "Data uploaded but can't squeeze yet")
				require.Equal(t, data.GetPreimageWithoutSize(), contract.addData, "Did not upload correct data")
			})
		}
	})

	t.Run("StoragePartExtendsPastEndOfData", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		// Input data is just shorter than the max chunk size
		input := make([]byte, 2*MaxChunkSize-10)
		for i := range input {
			input[i] = byte(i)
		}
		// Offset is set so that the 32 bytes to store would extend past the end of the chunk
		// But we won't need to change the chunk size because it will reach the end of the data first.
		offset := uint32(2*MaxChunkSize - 20)
		data := makePreimageData(input, offset)

		// Upload preimage successfully
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, ErrChallengePeriodNotOver, "Data uploaded but can't squeeze yet")
		require.Equal(t, data.GetPreimageWithoutSize(), contract.addData, "Did not upload correct data")
		require.Equal(t, 2, contract.addCalls, "Should not need additional transactions")
	})
}

func mockPreimageOracleData() *types.PreimageOracleData {
	fullLeaf := make([]byte, keccakTypes.BlockSize)
	for i := 0; i < keccakTypes.BlockSize; i++ {
		fullLeaf[i] = byte(i)
	}
	oracleData := make([]byte, 0, 5*MaxBlocksPerChunk)
	for i := 0; i < 5*MaxBlocksPerChunk; i++ {
		oracleData = append(oracleData, fullLeaf...)
	}
	// Add a single byte to the end to make sure the last leaf is not processed.
	oracleData = append(oracleData, byte(1))
	return makePreimageData(oracleData, 0)
}

func makePreimageData(pre []byte, offset uint32) *types.PreimageOracleData {
	key := preimage.Keccak256Key(crypto.Keccak256Hash(pre)).PreimageKey()
	// add the length prefix
	preimage := make([]byte, 0, 8+len(pre))
	preimage = binary.BigEndian.AppendUint64(preimage, uint64(len(pre)))
	preimage = append(preimage, pre...)
	return types.NewPreimageOracleData(key[:], preimage, offset)
}

func TestLargePreimageUploader_UploadPreimage_Succeeds(t *testing.T) {
	fullLeaf := new([keccakTypes.BlockSize]byte)
	for i := 0; i < keccakTypes.BlockSize; i++ {
		fullLeaf[i] = byte(i)
	}
	chunk := make([]byte, 0, MaxChunkSize)
	for i := 0; i < MaxBlocksPerChunk; i++ {
		chunk = append(chunk, fullLeaf[:]...)
	}
	tests := []struct {
		name          string
		input         []byte
		addCalls      int
		prestateLeaf  keccakTypes.Leaf
		poststateLeaf keccakTypes.Leaf
	}{
		{
			name:     "FullLeaf",
			input:    fullLeaf[:],
			addCalls: 1,
		},
		{
			name:     "MultipleLeaves",
			input:    append(fullLeaf[:], append(fullLeaf[:], fullLeaf[:]...)...),
			addCalls: 1,
		},
		{
			name:     "MultipleLeavesUnaligned",
			input:    append(fullLeaf[:], append(fullLeaf[:], byte(9))...),
			addCalls: 1,
		},
		{
			name:     "MultipleChunks",
			input:    append(chunk, append(fullLeaf[:], fullLeaf[:]...)...),
			addCalls: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oracle, cl, _, contract := newTestLargePreimageUploader(t)
			data := makePreimageData(test.input, 0)
			err := oracle.UploadPreimage(context.Background(), 0, data)
			require.ErrorIs(t, err, ErrChallengePeriodNotOver) // Not able to squeeze yet

			// Advance time so we can squeeze
			cl.AdvanceTime(time.Duration(mockChallengePeriod+1) * time.Second)

			// Reattempt upload to trigger squeeze.
			err = oracle.UploadPreimage(context.Background(), 0, data)
			require.NoError(t, err)

			require.Equal(t, test.addCalls, contract.addCalls)
			// There must always be at least one init and squeeze call
			// for successful large preimage upload calls.
			require.Equal(t, 1, contract.initCalls)
			require.Equal(t, 1, contract.squeezeCalls)

			// Use the StateMatrix to determine the expected leaves so it includes padding correctly.
			// We rely on the unit tests for StateMatrix to confirm that it does the right thing.
			s := matrix.NewStateMatrix()
			_, err = s.AbsorbUpTo(bytes.NewReader(test.input), keccakTypes.BlockSize*10000)
			require.ErrorIs(t, err, io.EOF)
			prestate, _ := s.PrestateWithProof()
			poststate, _ := s.PoststateWithProof()
			require.Equal(t, prestate, contract.squeezePrestate)
			require.Equal(t, poststate, contract.squeezePoststate)
			require.Equal(t, data.GetPreimageWithoutSize(), contract.addData, "Did not upload correct data")
		})
	}
}

func newTestLargePreimageUploader(t *testing.T) (*LargePreimageUploader, *clock.AdvancingClock, *mockTxSender, *mockPreimageOracleContract) {
	logger := testlog.Logger(t, log.LevelError)
	cl := clock.NewAdvancingClock(time.Second)
	cl.Start()
	txSender := &mockTxSender{}
	contract := &mockPreimageOracleContract{
		t:       t,
		cl:      cl,
		addData: make([]byte, 0),
	}
	return NewLargePreimageUploader(logger, cl, txSender, contract), cl, txSender, contract
}

type mockPreimageOracleContract struct {
	t        *testing.T
	metadata map[string]keccakTypes.LargePreimageMetaData
	cl       clock.Clock

	initCalls        int
	initFails        bool
	initialized      bool
	claimedSize      uint32
	addCalls         int
	addFails         bool
	addData          []byte
	squeezeCalls     int
	squeezeFails     bool
	squeezeCallFails bool
	squeezePrestate  keccakTypes.Leaf
	squeezePoststate keccakTypes.Leaf
}

func (s *mockPreimageOracleContract) InitLargePreimage(uuid *big.Int, partOffset uint32, claimedSize uint32) (txmgr.TxCandidate, error) {
	if s.metadata == nil {
		s.metadata = make(map[string]keccakTypes.LargePreimageMetaData)
	}
	_, ok := s.metadata[uuid.String()]
	require.False(s.t, ok, "init called twice for same uuid")
	s.metadata[uuid.String()] = keccakTypes.LargePreimageMetaData{
		LargePreimageIdent: keccakTypes.LargePreimageIdent{
			UUID: uuid,
		},
		Timestamp:       0,
		PartOffset:      partOffset,
		ClaimedSize:     claimedSize,
		BlocksProcessed: 0,
		BytesProcessed:  0,
		Countered:       false,
	}
	s.initCalls++
	if s.initFails {
		return txmgr.TxCandidate{}, mockInitLPPError
	}
	return txmgr.TxCandidate{}, nil
}

func (s *mockPreimageOracleContract) AddLeaves(uuid *big.Int, startingBlockIndex *big.Int, input []byte, commitments []common.Hash, finalize bool) (txmgr.TxCandidate, error) {
	metadata, ok := s.metadata[uuid.String()]
	require.True(s.t, ok, "adding leaves without init")
	require.True(s.t, startingBlockIndex.IsUint64())
	require.EqualValues(s.t, metadata.BlocksProcessed, bigs.Uint64Strict(startingBlockIndex))

	expectedCommitments := len(input) / keccakTypes.BlockSize
	if finalize {
		metadata.Timestamp = uint64(s.cl.Now().Unix())
		// Finalizing will pad out the final block to full size.
		// Either this will round up the length (previously rounded down) or it will add another block.
		expectedCommitments++
	} else {
		require.Zero(s.t, len(input)%keccakTypes.BlockSize, "Input length must be a multiple of BlockSize")
	}
	// Round input leng next block size - when finalizing the last block is automatically padded.
	require.Equal(s.t, expectedCommitments, len(commitments), "Input length / BlockSize must be equal to the number of state commitments")

	s.addCalls++
	if s.addFails {
		return txmgr.TxCandidate{}, mockAddLeavesError
	}

	// Enforce: if the preimage part intersects this chunk, the full 32-byte part must be present in `input`,
	// unless we're finalizing (tail-end partial data allowed). This mirrors `_extractPreimagePart` in PreimageOracle.sol.
	// The metadata.PartOffset is calculated including the 8 byte length prefix.
	// The uploaded data and metadata.BytesProcessed excludes the 8 byte length prefix so we need to adjust for it.
	offset := uint64(metadata.PartOffset)
	currentSize := uint64(metadata.BytesProcessed) + 8 // Add implicit 8 byte length prefix
	if offset >= currentSize && offset < currentSize+uint64(len(input)) {
		// The start of the preimage is in this piece, check that the end is as well or that we're finalizing.
		// We always store 32 bytes of the preimage.
		if offset+32 >= currentSize+uint64(len(input)) && !finalize {
			require.Fail(s.t, "PartOffsetOOB: full preimage part not available in input")
		}
	}

	metadata.BytesProcessed += uint32(len(input))
	metadata.BlocksProcessed += uint32(len(commitments))
	s.metadata[uuid.String()] = metadata
	s.addData = append(s.addData, input...)
	return txmgr.TxCandidate{}, nil
}

func (s *mockPreimageOracleContract) Squeeze(_ common.Address, uuid *big.Int, _ keccakTypes.StateSnapshot, prestate keccakTypes.Leaf, _ merkle.Proof, poststate keccakTypes.Leaf, _ merkle.Proof) (txmgr.TxCandidate, error) {
	meta, ok := s.metadata[uuid.String()]
	require.True(s.t, ok, "squeeze called without init")
	require.NotZero(s.t, meta.Timestamp, "ActiveProposal")
	require.False(s.t, meta.Countered, "Countered")
	require.Greater(s.t, uint64(s.cl.Now().Unix())-meta.Timestamp, mockChallengePeriod, "ChallengePeriodNotElapsed")
	s.squeezeCalls++
	s.squeezePrestate = prestate
	s.squeezePoststate = poststate
	if s.squeezeFails {
		return txmgr.TxCandidate{}, mockSqueezeError
	}
	return txmgr.TxCandidate{}, nil
}

func (s *mockPreimageOracleContract) ChallengePeriod(_ context.Context) (uint64, error) {
	return mockChallengePeriod, nil
}

func (s *mockPreimageOracleContract) GetProposalMetadata(_ context.Context, _ rpcblock.Block, idents ...keccakTypes.LargePreimageIdent) ([]keccakTypes.LargePreimageMetaData, error) {
	metadata := make([]keccakTypes.LargePreimageMetaData, 0)
	for _, ident := range idents {
		meta := s.metadata[ident.UUID.String()]
		metadata = append(metadata, meta)
	}
	return metadata, nil
}

func (s *mockPreimageOracleContract) GetMinBondLPP(_ context.Context) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (s *mockPreimageOracleContract) CallSqueeze(_ context.Context, _ common.Address, _ *big.Int, _ keccakTypes.StateSnapshot, _ keccakTypes.Leaf, _ merkle.Proof, _ keccakTypes.Leaf, _ merkle.Proof) error {
	if s.squeezeCallFails {
		return mockSqueezeCallError
	}
	return nil
}
