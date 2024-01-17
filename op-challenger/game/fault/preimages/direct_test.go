package preimages

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockUpdateOracleTxError = errors.New("mock update oracle tx error")
	mockTxMgrSendError      = errors.New("mock tx mgr send error")
)

func TestDirectPreimageUploader_UploadPreimage(t *testing.T) {
	t.Run("UpdateOracleTxFails", func(t *testing.T) {
		oracle, txMgr, contract := newTestDirectPreimageUploader(t)
		contract.updateFails = true
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{})
		require.ErrorIs(t, err, mockUpdateOracleTxError)
		require.Equal(t, 1, contract.updates)
		require.Equal(t, 0, txMgr.sends) // verify that the tx was not sent
	})

	t.Run("SendFails", func(t *testing.T) {
		oracle, txMgr, contract := newTestDirectPreimageUploader(t)
		txMgr.sendFails = true
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{})
		require.ErrorIs(t, err, mockTxMgrSendError)
		require.Equal(t, 1, contract.updates)
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
		require.Equal(t, 1, contract.updates)
	})
}

func TestDirectPreimageUploader_SendTxAndWait(t *testing.T) {
	t.Run("SendFails", func(t *testing.T) {
		oracle, txMgr, _ := newTestDirectPreimageUploader(t)
		txMgr.sendFails = true
		err := oracle.sendTxAndWait(context.Background(), txmgr.TxCandidate{})
		require.ErrorIs(t, err, mockTxMgrSendError)
		require.Equal(t, 1, txMgr.sends)
	})

	t.Run("ReceiptStatusFailed", func(t *testing.T) {
		oracle, txMgr, _ := newTestDirectPreimageUploader(t)
		txMgr.statusFail = true
		err := oracle.sendTxAndWait(context.Background(), txmgr.TxCandidate{})
		require.NoError(t, err)
		require.Equal(t, 1, txMgr.sends)
	})

	t.Run("Success", func(t *testing.T) {
		oracle, txMgr, _ := newTestDirectPreimageUploader(t)
		err := oracle.sendTxAndWait(context.Background(), txmgr.TxCandidate{})
		require.NoError(t, err)
		require.Equal(t, 1, txMgr.sends)
	})
}

func newTestDirectPreimageUploader(t *testing.T) (*DirectPreimageUploader, *mockTxMgr, *mockPreimageGameContract) {
	logger := testlog.Logger(t, log.LvlError)
	txMgr := &mockTxMgr{}
	contract := &mockPreimageGameContract{}
	return NewDirectPreimageUploader(logger, txMgr, contract), txMgr, contract
}

type mockPreimageGameContract struct {
	updates     int
	updateFails bool
}

func (s *mockPreimageGameContract) UpdateOracleTx(_ context.Context, _ uint64, _ *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	s.updates++
	if s.updateFails {
		return txmgr.TxCandidate{}, mockUpdateOracleTxError
	}
	return txmgr.TxCandidate{}, nil
}

type mockTxMgr struct {
	sends      int
	sendFails  bool
	statusFail bool
}

func (s *mockTxMgr) Send(_ context.Context, _ txmgr.TxCandidate) (*ethtypes.Receipt, error) {
	s.sends++
	if s.sendFails {
		return nil, mockTxMgrSendError
	}
	if s.statusFail {
		return &ethtypes.Receipt{Status: ethtypes.ReceiptStatusFailed}, nil
	}
	return &ethtypes.Receipt{}, nil
}

func (s *mockTxMgr) BlockNumber(_ context.Context) (uint64, error) { return 0, nil }
func (s *mockTxMgr) From() common.Address                          { return common.Address{} }
func (s *mockTxMgr) Close()                                        {}
