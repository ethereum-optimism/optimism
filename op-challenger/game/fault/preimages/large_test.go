package preimages

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var mockAddLeavesError = errors.New("mock add leaves error")

func TestLargePreimageUploader_UploadPreimage(t *testing.T) {
	t.Run("InitFails", func(t *testing.T) {
		oracle, _, contract := newTestLargePreimageUploader(t)
		contract.initFails = true
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{})
		require.ErrorIs(t, err, mockInitLPPError)
		require.Equal(t, 1, contract.initCalls)
	})

	t.Run("AddLeavesFails", func(t *testing.T) {
		oracle, _, contract := newTestLargePreimageUploader(t)
		contract.addFails = true
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{})
		require.ErrorIs(t, err, mockAddLeavesError)
		require.Equal(t, 1, contract.addCalls)
	})

	t.Run("Success", func(t *testing.T) {
		fullLeaf := make([]byte, matrix.LeafSize)
		for i := 0; i < matrix.LeafSize; i++ {
			fullLeaf[i] = byte(i)
		}
		oracle, _, contract := newTestLargePreimageUploader(t)
		data := types.PreimageOracleData{
			OracleData: append(fullLeaf, fullLeaf...),
		}
		err := oracle.UploadPreimage(context.Background(), 0, &data)
		require.Equal(t, 1, contract.initCalls)
		require.Equal(t, 1, contract.addCalls)
		require.Equal(t, data.OracleData, contract.addData)
		// TODO(proofs#467): fix this to not error. See LargePreimageUploader.UploadPreimage.
		require.ErrorIs(t, err, errNotSupported)
	})
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
	initCalls int
	initFails bool
	addCalls  int
	addFails  bool
	addData   []byte
}

func (s *mockPreimageOracleContract) InitLargePreimage(_ *big.Int, _ uint32, _ uint32) (txmgr.TxCandidate, error) {
	s.initCalls++
	if s.initFails {
		return txmgr.TxCandidate{}, mockInitLPPError
	}
	return txmgr.TxCandidate{}, nil
}
func (s *mockPreimageOracleContract) AddLeaves(_ *big.Int, input []byte, _ [][32]byte, _ bool) (txmgr.TxCandidate, error) {
	s.addCalls++
	s.addData = append(s.addData, input...)
	if s.addFails {
		return txmgr.TxCandidate{}, mockAddLeavesError
	}
	return txmgr.TxCandidate{}, nil
}
func (s *mockPreimageOracleContract) Squeeze(_ common.Address, _ *big.Int, _ *matrix.StateMatrix, _ contracts.Leaf, _ contracts.MerkleProof, _ contracts.Leaf, _ contracts.MerkleProof) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, nil
}
