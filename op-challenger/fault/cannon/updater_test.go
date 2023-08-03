package cannon

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum-optimism/optimism/op-node/testlog"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/stretchr/testify/require"
)

var (
	mockFdgAddress            = common.HexToAddress("0x1234")
	mockPreimageOracleAddress = common.HexToAddress("0x12345")
	mockSendError             = errors.New("mock send error")
)

type mockTxManager struct {
	from        common.Address
	sends       int
	failedSends int
	sendFails   bool
}

func (m *mockTxManager) Send(ctx context.Context, candidate txmgr.TxCandidate) (*ethtypes.Receipt, error) {
	if m.sendFails {
		m.failedSends++
		return nil, mockSendError
	}
	m.sends++
	return ethtypes.NewReceipt(
		[]byte{},
		false,
		0,
	), nil
}

func (m *mockTxManager) Call(_ context.Context, _ ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	panic("not implemented")
}

func (m *mockTxManager) BlockNumber(ctx context.Context) (uint64, error) {
	panic("not implemented")
}

func (m *mockTxManager) From() common.Address {
	return m.from
}

func newTestCannonUpdater(t *testing.T, sendFails bool) (*cannonUpdater, *mockTxManager) {
	logger := testlog.Logger(t, log.LvlInfo)
	txMgr := &mockTxManager{
		from:      mockFdgAddress,
		sendFails: sendFails,
	}
	updater, err := NewOracleUpdater(logger, txMgr, mockFdgAddress, mockPreimageOracleAddress)
	require.NoError(t, err)
	return updater, txMgr
}

// TestCannonUpdater_UpdateOracle tests the [cannonUpdater]
// UpdateOracle function.
func TestCannonUpdater_UpdateOracle(t *testing.T) {
	t.Run("succeeds", func(t *testing.T) {
		updater, mockTxMgr := newTestCannonUpdater(t, false)
		require.Nil(t, updater.UpdateOracle(context.Background(), types.PreimageOracleData{
			OracleData: common.Hex2Bytes("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
		}))
		require.Equal(t, 1, mockTxMgr.sends)
	})

	t.Run("send fails", func(t *testing.T) {
		updater, mockTxMgr := newTestCannonUpdater(t, true)
		require.Error(t, updater.UpdateOracle(context.Background(), types.PreimageOracleData{
			OracleData: common.Hex2Bytes("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
		}))
		require.Equal(t, 1, mockTxMgr.failedSends)
	})
}

// TestCannonUpdater_BuildLocalOracleData tests the [cannonUpdater]
// builds a valid tx candidate for a local oracle update.
func TestCannonUpdater_BuildLocalOracleData(t *testing.T) {
	updater, _ := newTestCannonUpdater(t, false)
	oracleData := types.PreimageOracleData{
		OracleKey:    common.Hex2Bytes("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		OracleData:   common.Hex2Bytes("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
		OracleOffset: 7,
	}

	txData, err := updater.BuildLocalOracleData(oracleData)
	require.NoError(t, err)

	var addLocalDataBytes4 = crypto.Keccak256([]byte("addLocalData(uint256,uint256)"))[:4]

	// Pack the tx data manually.
	var expected []byte
	expected = append(expected, addLocalDataBytes4...)
	expected = append(expected, common.Hex2Bytes("00aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")...)
	expected = append(expected, common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000007")...)

	require.Equal(t, expected, txData)
}

// TestCannonUpdater_BuildGlobalOracleData tests the [cannonUpdater]
// builds a valid tx candidate for a global oracle update.
func TestCannonUpdater_BuildGlobalOracleData(t *testing.T) {
	updater, _ := newTestCannonUpdater(t, false)
	oracleData := types.PreimageOracleData{
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
