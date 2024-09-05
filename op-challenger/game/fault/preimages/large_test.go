package preimages

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/merkle"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
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
		require.NoError(t, err)
		require.Equal(t, 1, contract.initCalls)
		require.Equal(t, 6, contract.addCalls)
		require.Equal(t, data.GetPreimageWithoutSize(), contract.addData)
	})

	t.Run("AlreadyInitialized", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.initialized = true
		contract.claimedSize = uint32(len(data.GetPreimageWithoutSize()))
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.NoError(t, err)
		require.Equal(t, 0, contract.initCalls)
		require.Equal(t, 6, contract.addCalls)
	})

	t.Run("ChallengePeriodNotElapsed", func(t *testing.T) {
		oracle, cl, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.bytesProcessed = 5*MaxChunkSize + 1
		contract.claimedSize = uint32(len(data.GetPreimageWithoutSize()))
		contract.timestamp = uint64(cl.Now().Unix())
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, ErrChallengePeriodNotOver)
		require.Equal(t, 0, contract.squeezeCalls)
		// Squeeze should be called once the challenge period has elapsed.
		cl.AdvanceTime(time.Duration(mockChallengePeriod) * time.Second)
		err = oracle.UploadPreimage(context.Background(), 0, data)
		require.NoError(t, err)
		require.Equal(t, 1, contract.squeezeCalls)
	})

	t.Run("SqueezeCallFails", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.bytesProcessed = 5*MaxChunkSize + 1
		contract.timestamp = 123
		contract.claimedSize = uint32(len(data.GetPreimageWithoutSize()))
		contract.squeezeCallFails = true
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, mockSqueezeCallError)
		require.Equal(t, 0, contract.squeezeCalls)
	})

	t.Run("SqueezeFails", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.bytesProcessed = 5*MaxChunkSize + 1
		contract.timestamp = 123
		contract.claimedSize = uint32(len(data.GetPreimageWithoutSize()))
		contract.squeezeFails = true
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.ErrorIs(t, err, mockSqueezeError)
		require.Equal(t, 1, contract.squeezeCalls)
	})

	t.Run("AllBytesProcessed", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.bytesProcessed = 5*MaxChunkSize + 1
		contract.timestamp = 123
		contract.claimedSize = uint32(len(data.GetPreimageWithoutSize()))
		err := oracle.UploadPreimage(context.Background(), 0, data)
		require.NoError(t, err)
		require.Equal(t, 0, contract.initCalls)
		require.Equal(t, 0, contract.addCalls)
		require.Empty(t, contract.addData)
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
			oracle, _, _, contract := newTestLargePreimageUploader(t)
			data := makePreimageData(test.input, 0)
			err := oracle.UploadPreimage(context.Background(), 0, data)
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
		})
	}
}

func newTestLargePreimageUploader(t *testing.T) (*LargePreimageUploader, *clock.AdvancingClock, *mockTxSender, *mockPreimageOracleContract) {
	logger := testlog.Logger(t, log.LevelError)
	cl := clock.NewAdvancingClock(time.Second)
	cl.Start()
	txSender := &mockTxSender{}
	contract := &mockPreimageOracleContract{
		addData: make([]byte, 0),
	}
	return NewLargePreimageUploader(logger, cl, txSender, contract), cl, txSender, contract
}

type mockPreimageOracleContract struct {
	initCalls            int
	initFails            bool
	initialized          bool
	claimedSize          uint32
	bytesProcessed       int
	timestamp            uint64
	addCalls             int
	addFails             bool
	addData              []byte
	squeezeCalls         int
	squeezeFails         bool
	squeezeCallFails     bool
	squeezeCallClaimSize uint32
	squeezePrestate      keccakTypes.Leaf
	squeezePoststate     keccakTypes.Leaf
}

func (s *mockPreimageOracleContract) InitLargePreimage(_ *big.Int, _ uint32, _ uint32) (txmgr.TxCandidate, error) {
	s.initCalls++
	if s.initFails {
		return txmgr.TxCandidate{}, mockInitLPPError
	}
	return txmgr.TxCandidate{}, nil
}

func (s *mockPreimageOracleContract) AddLeaves(_ *big.Int, _ *big.Int, input []byte, _ []common.Hash, _ bool) (txmgr.TxCandidate, error) {
	s.addCalls++
	s.addData = append(s.addData, input...)
	if s.addFails {
		return txmgr.TxCandidate{}, mockAddLeavesError
	}
	return txmgr.TxCandidate{}, nil
}

func (s *mockPreimageOracleContract) Squeeze(_ common.Address, _ *big.Int, _ keccakTypes.StateSnapshot, prestate keccakTypes.Leaf, _ merkle.Proof, poststate keccakTypes.Leaf, _ merkle.Proof) (txmgr.TxCandidate, error) {
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
	if s.squeezeCallClaimSize > 0 {
		metadata := make([]keccakTypes.LargePreimageMetaData, 0)
		for _, ident := range idents {
			metadata = append(metadata, keccakTypes.LargePreimageMetaData{
				LargePreimageIdent: ident,
				ClaimedSize:        s.squeezeCallClaimSize,
				BytesProcessed:     uint32(s.bytesProcessed),
				Timestamp:          s.timestamp,
			})
		}
		return metadata, nil
	}
	if s.initialized || s.bytesProcessed > 0 {
		metadata := make([]keccakTypes.LargePreimageMetaData, 0)
		for _, ident := range idents {
			metadata = append(metadata, keccakTypes.LargePreimageMetaData{
				LargePreimageIdent: ident,
				ClaimedSize:        s.claimedSize,
				BytesProcessed:     uint32(s.bytesProcessed),
				Timestamp:          s.timestamp,
			})
		}
		return metadata, nil
	}
	s.squeezeCallClaimSize = 1
	return []keccakTypes.LargePreimageMetaData{{LargePreimageIdent: idents[0]}}, nil
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
