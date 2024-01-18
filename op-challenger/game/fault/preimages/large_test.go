package preimages

import (
	"context"
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

func TestLargePreimageUploader_UploadPreimage(t *testing.T) {
	t.Run("InitFails", func(t *testing.T) {
		oracle, _, contract := newTestLargePreimageUploader(t)
		contract.initFails = true
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{})
		require.ErrorIs(t, err, mockInitLPPError)
		require.Equal(t, 1, contract.initCalls)
	})

	t.Run("Success", func(t *testing.T) {
		oracle, _, contract := newTestLargePreimageUploader(t)
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{})
		require.Equal(t, 1, contract.initCalls)
		// TODO(proofs#467): fix this to not error. See LargePreimageUploader.UploadPreimage.
		require.ErrorIs(t, err, errNotSupported)
	})
}

func newTestLargePreimageUploader(t *testing.T) (*LargePreimageUploader, *mockTxMgr, *mockPreimageOracleContract) {
	logger := testlog.Logger(t, log.LvlError)
	txMgr := &mockTxMgr{}
	contract := &mockPreimageOracleContract{}
	return NewLargePreimageUploader(logger, txMgr, contract), txMgr, contract
}

type mockPreimageOracleContract struct {
	initCalls int
	initFails bool
}

func (s *mockPreimageOracleContract) InitLargePreimage(_ *big.Int, _ uint32, _ uint32) (txmgr.TxCandidate, error) {
	s.initCalls++
	if s.initFails {
		return txmgr.TxCandidate{}, mockInitLPPError
	}
	return txmgr.TxCandidate{}, nil
}
func (s *mockPreimageOracleContract) AddLeaves(_ *big.Int, _ []contracts.Leaf, _ bool) ([]txmgr.TxCandidate, error) {
	return []txmgr.TxCandidate{}, nil
}
func (s *mockPreimageOracleContract) Squeeze(_ common.Address, _ *big.Int, _ *matrix.StateMatrix, _ contracts.Leaf, _ contracts.MerkleProof, _ contracts.Leaf, _ contracts.MerkleProof) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, nil
}
