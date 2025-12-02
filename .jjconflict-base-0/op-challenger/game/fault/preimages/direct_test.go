package preimages

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockUpdateOracleTxError = errors.New("mock update oracle tx error")
	mockTxMgrSendError      = errors.New("mock tx mgr send error")
	mockInitLPPError        = errors.New("mock init LPP error")
)

func TestDirectPreimageUploader_UploadPreimage(t *testing.T) {
	t.Run("UpdateOracleTxFails", func(t *testing.T) {
		oracle, txMgr, contract := newTestDirectPreimageUploader(t)
		contract.updateFails = true
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{})
		require.ErrorIs(t, err, mockUpdateOracleTxError)
		require.Equal(t, 1, contract.updateCalls)
		require.Equal(t, 0, txMgr.sends) // verify that the tx was not sent
	})

	t.Run("SendFails", func(t *testing.T) {
		oracle, txMgr, contract := newTestDirectPreimageUploader(t)
		txMgr.sendFails = true
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{})
		require.ErrorIs(t, err, mockTxMgrSendError)
		require.Equal(t, 1, contract.updateCalls)
		require.Equal(t, 1, txMgr.sends)
	})

	t.Run("NilPreimageData", func(t *testing.T) {
		oracle, _, _ := newTestDirectPreimageUploader(t)
		err := oracle.UploadPreimage(context.Background(), 0, nil)
		require.ErrorIs(t, err, ErrNilPreimageData)
	})

	t.Run("Success", func(t *testing.T) {
		oracle, _, contract := newTestDirectPreimageUploader(t)
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{})
		require.NoError(t, err)
		require.Equal(t, 1, contract.updateCalls)
	})
}

func newTestDirectPreimageUploader(t *testing.T) (*DirectPreimageUploader, *mockTxSender, *mockPreimageGameContract) {
	logger := testlog.Logger(t, log.LevelError)
	txMgr := &mockTxSender{}
	contract := &mockPreimageGameContract{}
	return NewDirectPreimageUploader(logger, txMgr, contract), txMgr, contract
}

type mockPreimageGameContract struct {
	updateCalls int
	updateFails bool
}

func (s *mockPreimageGameContract) UpdateOracleTx(_ context.Context, _ uint64, _ *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	s.updateCalls++
	if s.updateFails {
		return txmgr.TxCandidate{}, mockUpdateOracleTxError
	}
	return txmgr.TxCandidate{}, nil
}

type mockTxSender struct {
	sends      int
	sendFails  bool
	statusFail bool
}

func (s *mockTxSender) From() common.Address {
	return common.Address{}
}

func (s *mockTxSender) SendAndWaitSimple(_ string, _ ...txmgr.TxCandidate) error {
	s.sends++
	if s.sendFails {
		return mockTxMgrSendError
	}
	if s.statusFail {
		return errors.New("transaction reverted")
	}
	return nil
}
