package cannon

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/stretchr/testify/require"
)

var (
	mockPreimageOracleAddress = common.HexToAddress("0x12345")
	mockSendError             = errors.New("mock send error")
)

type mockTxManager struct {
	from        common.Address
	sent        []txmgr.TxCandidate
	failedSends int
	sendFails   bool
}

func (m *mockTxManager) Send(ctx context.Context, candidate txmgr.TxCandidate) (*ethtypes.Receipt, error) {
	m.sent = append(m.sent, candidate)
	if m.sendFails {
		m.failedSends++
		return nil, mockSendError
	}
	return &ethtypes.Receipt{
		Type:              ethtypes.LegacyTxType,
		PostState:         []byte{},
		CumulativeGasUsed: 0,
		Status:            ethtypes.ReceiptStatusSuccessful,
	}, nil
}

func (m *mockTxManager) BlockNumber(ctx context.Context) (uint64, error) {
	panic("not implemented")
}

func (m *mockTxManager) From() common.Address {
	return m.from
}

func newTestCannonUpdater(t *testing.T, sendFails bool) (*cannonUpdater, *mockTxManager, *mockGameContract) {
	logger := testlog.Logger(t, log.LvlInfo)
	txMgr := &mockTxManager{
		from:      common.HexToAddress("0x1234"),
		sendFails: sendFails,
	}
	gameContract := &mockGameContract{}
	updater, err := NewOracleUpdaterWithOracle(logger, txMgr, gameContract, mockPreimageOracleAddress)
	require.NoError(t, err)
	return updater, txMgr, gameContract
}

// TestCannonUpdater_UpdateOracle tests the [cannonUpdater]
// UpdateOracle function.
func TestCannonUpdater_UpdateOracle(t *testing.T) {
	t.Run("local_succeeds", func(t *testing.T) {
		updater, mockTxMgr, gameContract := newTestCannonUpdater(t, false)
		gameContract.tx = txmgr.TxCandidate{
			TxData: []byte{5, 6, 7, 8},
		}
		require.NoError(t, updater.UpdateOracle(context.Background(), &types.PreimageOracleData{
			IsLocal:      true,
			LocalContext: 3,
			OracleKey:    common.Hash{0xaa}.Bytes(),
			OracleData:   common.Hex2Bytes("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
		}))
		require.Len(t, mockTxMgr.sent, 1)
		require.Equal(t, gameContract.tx, mockTxMgr.sent[0])
	})

	t.Run("local_fails", func(t *testing.T) {
		updater, mockTxMgr, gameContract := newTestCannonUpdater(t, true)
		gameContract.tx = txmgr.TxCandidate{
			TxData: []byte{5, 6, 7, 8},
		}
		require.Error(t, updater.UpdateOracle(context.Background(), &types.PreimageOracleData{
			IsLocal:      true,
			LocalContext: 3,
			OracleKey:    common.Hash{0xaa}.Bytes(),
			OracleData:   common.Hex2Bytes("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
		}))
		require.Len(t, mockTxMgr.sent, 1)
		require.Equal(t, gameContract.tx, mockTxMgr.sent[0])
		require.Equal(t, 1, mockTxMgr.failedSends)
	})

	t.Run("global_succeeds", func(t *testing.T) {
		updater, mockTxMgr, _ := newTestCannonUpdater(t, false)
		require.NoError(t, updater.UpdateOracle(context.Background(), &types.PreimageOracleData{
			IsLocal:    false,
			OracleKey:  common.Hash{0xaa}.Bytes(),
			OracleData: common.Hex2Bytes("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
		}))
		require.Len(t, mockTxMgr.sent, 1)
		require.Equal(t, mockPreimageOracleAddress, *mockTxMgr.sent[0].To)
	})

	t.Run("local_fails", func(t *testing.T) {
		updater, mockTxMgr, _ := newTestCannonUpdater(t, true)
		require.Error(t, updater.UpdateOracle(context.Background(), &types.PreimageOracleData{
			IsLocal:    false,
			OracleKey:  common.Hash{0xaa}.Bytes(),
			OracleData: common.Hex2Bytes("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
		}))
		require.Len(t, mockTxMgr.sent, 1)
		require.Equal(t, mockPreimageOracleAddress, *mockTxMgr.sent[0].To)
		require.Equal(t, 1, mockTxMgr.failedSends)
	})
}

// TestCannonUpdater_BuildGlobalOracleData tests the [cannonUpdater]
// builds a valid tx candidate for a global oracle update.
func TestCannonUpdater_BuildGlobalOracleData(t *testing.T) {
	updater, _, _ := newTestCannonUpdater(t, false)
	oracleData := &types.PreimageOracleData{
		OracleKey:    common.Hex2Bytes("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		OracleData:   common.Hex2Bytes("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
		OracleOffset: 7,
	}

	txData, err := updater.BuildGlobalOracleData(oracleData)
	require.NoError(t, err)

	var loadKeccak256PreimagePartBytes4 = crypto.Keccak256([]byte("loadKeccak256PreimagePart(uint256,bytes)"))[:4]

	// Pack the tx data manually.
	var expected []byte
	expected = append(expected, loadKeccak256PreimagePartBytes4...)
	expected = append(expected, common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000007")...)
	expected = append(expected, common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000040")...)
	expected = append(expected, common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000018")...)
	expected = append(expected, common.Hex2Bytes("cccccccccccccccccccccccccccccccccccccccccccccccc0000000000000000")...)

	require.Equal(t, expected, txData)
}

type mockGameContract struct {
	tx  txmgr.TxCandidate
	err error
}

func (m *mockGameContract) VMAddr(_ context.Context) (common.Address, error) {
	return common.Address{0xcc}, nil
}

func (m *mockGameContract) AddLocalDataTx(_ *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	return m.tx, m.err
}
