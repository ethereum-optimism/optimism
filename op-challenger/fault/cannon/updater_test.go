package cannon

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testlog"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/stretchr/testify/require"
)

var (
	mockFdgAddress            = common.HexToAddress("0x1234")
	mockPreimageOracleAddress = common.HexToAddress("0x12345")
	mockSendError             = errors.New("mock send error")
)

type mockTxManager struct {
	from      common.Address
	sends     int
	calls     int
	sendFails bool
}

func (m *mockTxManager) Send(ctx context.Context, candidate txmgr.TxCandidate) (*ethtypes.Receipt, error) {
	if m.sendFails {
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
	if m.sendFails {
		return nil, mockSendError
	}
	m.calls++
	return []byte{}, nil
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
		_, _ = newTestCannonUpdater(t, false)
		// require.Nil(t, updater.UpdateOracle(context.Background(), types.PreimageOracleData{}))
		// require.Equal(t, 1, mockTxMgr.calls)
	})

	t.Run("send fails", func(t *testing.T) {
		_, _ = newTestCannonUpdater(t, true)
		// require.Error(t, updater.UpdateOracle(context.Background(), types.PreimageOracleData{}))
		// require.Equal(t, 1, mockTxMgr.calls)
	})
}
