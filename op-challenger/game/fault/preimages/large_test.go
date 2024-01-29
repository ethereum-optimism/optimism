package preimages

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/merkle"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
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
			data:         &types.PreimageOracleData{},
			expectedUUID: new(big.Int).SetBytes(common.Hex2Bytes("827b659bbda2a0bdecce2c91b8b68462545758f3eba2dbefef18e0daf84f5ccd")),
		},
		{
			name: "OracleDataAndOffset_Control",
			data: &types.PreimageOracleData{
				OracleData:   []byte{1, 2, 3},
				OracleOffset: 0x010203,
			},
			expectedUUID: new(big.Int).SetBytes(common.Hex2Bytes("641e230bcf3ade8c71b7e591d210184cdb190e853f61ba59a1411c3b7aca9890")),
		},
		{
			name: "OracleDataAndOffset_DifferentOffset",
			data: &types.PreimageOracleData{
				OracleData:   []byte{1, 2, 3},
				OracleOffset: 0x010204,
			},
			expectedUUID: new(big.Int).SetBytes(common.Hex2Bytes("aec56de44401325420e5793f72b777e3e547778de7d8344004b31be086a3136d")),
		},
		{
			name: "OracleDataAndOffset_DifferentData",
			data: &types.PreimageOracleData{
				OracleData:   []byte{1, 2, 3, 4},
				OracleOffset: 0x010203,
			},
			expectedUUID: new(big.Int).SetBytes(common.Hex2Bytes("ca38aa17d56805cf26376a050c2c7b15b6be4e709bc422a1c679fe21aa6aa8c7")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oracle, _, _, _ := newTestLargePreimageUploader(t)
			uuid := oracle.newUUID(test.data)
			require.Equal(t, test.expectedUUID, uuid)
		})
	}
}

func TestLargePreimageUploader_UploadPreimage_EdgeCases(t *testing.T) {
	t.Run("InitFails", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		contract.initFails = true
		data := mockPreimageOracleData()
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.ErrorIs(t, err, mockInitLPPError)
		require.Equal(t, 1, contract.initCalls)
	})

	t.Run("AddLeavesFails", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		contract.addFails = true
		data := mockPreimageOracleData()
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.ErrorIs(t, err, mockAddLeavesError)
		require.Equal(t, 1, contract.addCalls)
	})

	t.Run("NoBytesProcessed", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.claimedSize = uint32(len(data.OracleData))
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.NoError(t, err)
		require.Equal(t, 1, contract.initCalls)
		require.Equal(t, 6, contract.addCalls)
		require.Equal(t, data.OracleData, contract.addData)
	})

	t.Run("AlreadyInitialized", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.initialized = true
		contract.claimedSize = uint32(len(data.OracleData))
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.NoError(t, err)
		require.Equal(t, 0, contract.initCalls)
		require.Equal(t, 6, contract.addCalls)
	})

	t.Run("ChallengePeriodNotElapsed", func(t *testing.T) {
		oracle, cl, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.bytesProcessed = 5*MaxChunkSize + 1
		contract.claimedSize = uint32(len(data.OracleData))
		contract.timestamp = uint64(cl.Now().Unix())
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.ErrorIs(t, err, ErrChallengePeriodNotOver)
		require.Equal(t, 0, contract.squeezeCalls)
		// Squeeze should be called once the challenge period has elapsed.
		cl.AdvanceTime(time.Duration(mockChallengePeriod) * time.Second)
		err = oracle.UploadPreimage(context.Background(), 0, &data)
		require.NoError(t, err)
		require.Equal(t, 1, contract.squeezeCalls)
	})

	t.Run("SqueezeCallFails", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.bytesProcessed = 5*MaxChunkSize + 1
		contract.timestamp = 123
		contract.claimedSize = uint32(len(data.OracleData))
		contract.squeezeCallFails = true
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.ErrorIs(t, err, mockSqueezeCallError)
		require.Equal(t, 0, contract.squeezeCalls)
	})

	t.Run("SqueezeFails", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.bytesProcessed = 5*MaxChunkSize + 1
		contract.timestamp = 123
		contract.claimedSize = uint32(len(data.OracleData))
		contract.squeezeFails = true
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.ErrorIs(t, err, mockSqueezeError)
		require.Equal(t, 1, contract.squeezeCalls)
	})

	t.Run("AllBytesProcessed", func(t *testing.T) {
		oracle, _, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.bytesProcessed = 5*MaxChunkSize + 1
		contract.timestamp = 123
		contract.claimedSize = uint32(len(data.OracleData))
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.NoError(t, err)
		require.Equal(t, 0, contract.initCalls)
		require.Equal(t, 0, contract.addCalls)
		require.Empty(t, contract.addData)
	})
}

func mockPreimageOracleData() types.PreimageOracleData {
	fullLeaf := make([]byte, keccakTypes.BlockSize)
	for i := 0; i < keccakTypes.BlockSize; i++ {
		fullLeaf[i] = byte(i)
	}
	oracleData := make([]byte, 5*MaxBlocksPerChunk)
	for i := 0; i < 5*MaxBlocksPerChunk; i++ {
		oracleData = append(oracleData, fullLeaf...)
	}
	// Add a single byte to the end to make sure the last leaf is not processed.
	oracleData = append(oracleData, byte(1))
	return types.PreimageOracleData{
		OracleData: oracleData,
	}
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
			prestateLeaf: keccakTypes.Leaf{
				Input:           *fullLeaf,
				Index:           0,
				StateCommitment: common.HexToHash("9788a3b3bc36c482525b5890767be37130c997917bceca6e91a6c93359a4d1c6"),
			},
			poststateLeaf: keccakTypes.Leaf{
				Input:           [keccakTypes.BlockSize]byte{},
				Index:           1,
				StateCommitment: common.HexToHash("78358b902b7774b314bcffdf0948746f18d6044086e76e3924d585dca3486c7d"),
			},
		},
		{
			name:     "MultipleLeaves",
			input:    append(fullLeaf[:], append(fullLeaf[:], fullLeaf[:]...)...),
			addCalls: 1,
			prestateLeaf: keccakTypes.Leaf{
				Input:           *fullLeaf,
				Index:           2,
				StateCommitment: common.HexToHash("e3deed8ab6f8bbcf3d4fe825d74f703b3f2fc2f5b0afaa2574926fcfd0d4c895"),
			},
			poststateLeaf: keccakTypes.Leaf{
				Input:           [keccakTypes.BlockSize]byte{},
				Index:           3,
				StateCommitment: common.HexToHash("79115eeab1ff2eccf5baf3ea2dda13bc79c548ce906bdd16433a23089c679df2"),
			},
		},
		{
			name:     "MultipleLeavesUnaligned",
			input:    append(fullLeaf[:], append(fullLeaf[:], byte(9))...),
			addCalls: 1,
			prestateLeaf: keccakTypes.Leaf{
				Input:           *fullLeaf,
				Index:           1,
				StateCommitment: common.HexToHash("b5ea400e375b2c1ce348f3cc4ad5b6ad28e1b36759ddd2aba155f0b1d476b015"),
			},
			poststateLeaf: keccakTypes.Leaf{
				Input:           [keccakTypes.BlockSize]byte{byte(9)},
				Index:           2,
				StateCommitment: common.HexToHash("fa87e115dc4786e699bf80cc75d13ac1e2db0708c1418fc8cbc9800d17b5811a"),
			},
		},
		{
			name:     "MultipleChunks",
			input:    append(chunk, append(fullLeaf[:], fullLeaf[:]...)...),
			addCalls: 2,
			prestateLeaf: keccakTypes.Leaf{
				Input:           *fullLeaf,
				Index:           301,
				StateCommitment: common.HexToHash("4e9c55542478939feca4ff55ee98fbc632bb65a784a55b94536644bc87298ca4"),
			},
			poststateLeaf: keccakTypes.Leaf{
				Input:           [keccakTypes.BlockSize]byte{},
				Index:           302,
				StateCommitment: common.HexToHash("775020bfcaa93700263d040a4eeec3c8c3cf09e178457d04044594beaaf5e20b"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oracle, _, _, contract := newTestLargePreimageUploader(t)
			data := types.PreimageOracleData{
				OracleData: test.input,
			}
			err := oracle.UploadPreimage(context.Background(), 0, &data)
			require.NoError(t, err)
			require.Equal(t, test.addCalls, contract.addCalls)
			// There must always be at least one init and squeeze call
			// for successful large preimage upload calls.
			require.Equal(t, 1, contract.initCalls)
			require.Equal(t, 1, contract.squeezeCalls)
			require.Equal(t, test.prestateLeaf, contract.squeezePrestate)
			require.Equal(t, test.poststateLeaf, contract.squeezePoststate)
		})
	}

}

func newTestLargePreimageUploader(t *testing.T) (*LargePreimageUploader, *clock.AdvancingClock, *mockTxSender, *mockPreimageOracleContract) {
	logger := testlog.Logger(t, log.LvlError)
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

func (s *mockPreimageOracleContract) Squeeze(_ common.Address, _ *big.Int, _ *matrix.StateMatrix, prestate keccakTypes.Leaf, _ merkle.Proof, poststate keccakTypes.Leaf, _ merkle.Proof) (txmgr.TxCandidate, error) {
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

func (s *mockPreimageOracleContract) GetProposalMetadata(_ context.Context, _ batching.Block, idents ...keccakTypes.LargePreimageIdent) ([]keccakTypes.LargePreimageMetaData, error) {
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
func (s *mockPreimageOracleContract) CallSqueeze(_ context.Context, _ common.Address, _ *big.Int, _ *matrix.StateMatrix, _ keccakTypes.Leaf, _ merkle.Proof, _ keccakTypes.Leaf, _ merkle.Proof) error {
	if s.squeezeCallFails {
		return mockSqueezeCallError
	}
	return nil
}
