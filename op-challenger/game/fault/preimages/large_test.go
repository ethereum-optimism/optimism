package preimages

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var mockAddLeavesError = errors.New("mock add leaves error")

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
			oracle, _, _ := newTestLargePreimageUploader(t)
			uuid := oracle.newUUID(test.data)
			require.Equal(t, test.expectedUUID, uuid)
		})
	}
}

func TestLargePreimageUploader_UploadPreimage(t *testing.T) {
	t.Run("InitFails", func(t *testing.T) {
		oracle, _, contract := newTestLargePreimageUploader(t)
		contract.initFails = true
		data := mockPreimageOracleData()
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.ErrorIs(t, err, mockInitLPPError)
		require.Equal(t, 1, contract.initCalls)
	})

	t.Run("AddLeavesFails", func(t *testing.T) {
		oracle, _, contract := newTestLargePreimageUploader(t)
		contract.addFails = true
		data := mockPreimageOracleData()
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.ErrorIs(t, err, mockAddLeavesError)
		require.Equal(t, 1, contract.addCalls)
	})

	t.Run("AlreadyInitialized", func(t *testing.T) {
		oracle, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.initialized = true
		contract.claimedSize = uint32(len(data.OracleData))
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.Equal(t, 0, contract.initCalls)
		require.Equal(t, 6, contract.addCalls)
		// TODO(client-pod#467): fix this to not error. See LargePreimageUploader.UploadPreimage.
		require.ErrorIs(t, err, errNotSupported)
	})

	t.Run("NoBytesProcessed", func(t *testing.T) {
		oracle, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.Equal(t, 1, contract.initCalls)
		require.Equal(t, 6, contract.addCalls)
		require.Equal(t, data.OracleData, contract.addData)
		// TODO(client-pod#467): fix this to not error. See LargePreimageUploader.UploadPreimage.
		require.ErrorIs(t, err, errNotSupported)
	})

	t.Run("PartialBytesProcessed", func(t *testing.T) {
		oracle, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.bytesProcessed = 3 * MaxChunkSize
		contract.claimedSize = uint32(len(data.OracleData))
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.Equal(t, 0, contract.initCalls)
		require.Equal(t, 3, contract.addCalls)
		require.Equal(t, data.OracleData[contract.bytesProcessed:], contract.addData)
		// TODO(client-pod#467): fix this to not error. See LargePreimageUploader.UploadPreimage.
		require.ErrorIs(t, err, errNotSupported)
	})

	t.Run("LastLeafNotProcessed", func(t *testing.T) {
		oracle, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.bytesProcessed = 5 * MaxChunkSize
		contract.claimedSize = uint32(len(data.OracleData))
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.Equal(t, 0, contract.initCalls)
		require.Equal(t, 1, contract.addCalls)
		require.Equal(t, data.OracleData[contract.bytesProcessed:], contract.addData)
		// TODO(client-pod#467): fix this to not error. See LargePreimageUploader.UploadPreimage.
		require.ErrorIs(t, err, errNotSupported)
	})

	t.Run("AllBytesProcessed", func(t *testing.T) {
		oracle, _, contract := newTestLargePreimageUploader(t)
		data := mockPreimageOracleData()
		contract.bytesProcessed = 5*MaxChunkSize + 1
		contract.timestamp = 123
		contract.claimedSize = uint32(len(data.OracleData))
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.Equal(t, 0, contract.initCalls)
		require.Equal(t, 0, contract.addCalls)
		require.Empty(t, contract.addData)
		// TODO(client-pod#467): fix this to not error. See LargePreimageUploader.UploadPreimage.
		require.ErrorIs(t, err, errNotSupported)
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

func newTestLargePreimageUploader(t *testing.T) (*LargePreimageUploader, *mockTxMgr, *mockPreimageOracleContract) {
	logger := testlog.Logger(t, log.LvlError)
	txMgr := &mockTxMgr{}
	contract := &mockPreimageOracleContract{
		addData: make([]byte, 0),
	}
	return NewLargePreimageUploader(logger, txMgr, contract), txMgr, contract
}

type mockPreimageOracleContract struct {
	initCalls      int
	initFails      bool
	initialized    bool
	claimedSize    uint32
	bytesProcessed int
	timestamp      uint64
	addCalls       int
	addFails       bool
	addData        []byte
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

func (s *mockPreimageOracleContract) Squeeze(_ common.Address, _ *big.Int, _ *matrix.StateMatrix, _ keccakTypes.Leaf, _ contracts.MerkleProof, _ keccakTypes.Leaf, _ contracts.MerkleProof) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, nil
}
func (s *mockPreimageOracleContract) GetProposalMetadata(_ context.Context, _ batching.Block, idents ...keccakTypes.LargePreimageIdent) ([]keccakTypes.LargePreimageMetaData, error) {
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
	return []keccakTypes.LargePreimageMetaData{{LargePreimageIdent: idents[0]}}, nil
}
