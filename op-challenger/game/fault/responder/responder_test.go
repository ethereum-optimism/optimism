package responder

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/stretchr/testify/require"
)

var (
	mockFdgAddress = common.HexToAddress("0x1234")
	mockSendError  = errors.New("mock send error")
	mockCallError  = errors.New("mock call error")
)

// TestCallResolve tests the [Responder.CallResolve].
func TestCallResolve(t *testing.T) {
	t.Run("SendFails", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		mockTxMgr.callFails = true
		status, err := responder.CallResolve(context.Background())
		require.ErrorIs(t, err, mockCallError)
		require.Equal(t, gameTypes.GameStatusInProgress, status)
		require.Equal(t, 0, mockTxMgr.calls)
	})

	t.Run("UnpackFails", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		mockTxMgr.callBytes = []byte{0x00, 0x01}
		status, err := responder.CallResolve(context.Background())
		require.Error(t, err)
		require.Equal(t, gameTypes.GameStatusInProgress, status)
		require.Equal(t, 1, mockTxMgr.calls)
	})

	t.Run("Success", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		status, err := responder.CallResolve(context.Background())
		require.NoError(t, err)
		require.Equal(t, gameTypes.GameStatusInProgress, status)
		require.Equal(t, 1, mockTxMgr.calls)
	})
}

// TestResolve tests the [Responder.Resolve] method.
func TestResolve(t *testing.T) {
	t.Run("SendFails", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		mockTxMgr.sendFails = true
		err := responder.Resolve(context.Background())
		require.ErrorIs(t, err, mockSendError)
		require.Equal(t, 0, mockTxMgr.sends)
	})

	t.Run("Success", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		err := responder.Resolve(context.Background())
		require.NoError(t, err)
		require.Equal(t, 1, mockTxMgr.sends)
	})
}

// TestRespond tests the [Responder.Respond] method.
func TestRespond(t *testing.T) {
	t.Run("send fails", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		mockTxMgr.sendFails = true
		err := responder.Respond(context.Background(), generateMockResponseClaim())
		require.ErrorIs(t, err, mockSendError)
		require.Equal(t, 0, mockTxMgr.sends)
	})

	t.Run("sends response", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		err := responder.Respond(context.Background(), generateMockResponseClaim())
		require.NoError(t, err)
		require.Equal(t, 1, mockTxMgr.sends)
	})
}

// TestBuildTx tests the [Responder.BuildTx] method.
func TestBuildTx(t *testing.T) {
	t.Run("attack", func(t *testing.T) {
		responder, _ := newTestFaultResponder(t)
		responseClaim := generateMockResponseClaim()
		responseClaim.ParentContractIndex = 7
		tx, err := responder.BuildTx(context.Background(), responseClaim)
		require.NoError(t, err)

		// Pack the tx data manually.
		fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
		require.NoError(t, err)
		parent := big.NewInt(int64(7))
		claim := responseClaim.ValueBytes()
		expected, err := fdgAbi.Pack("attack", parent, claim)

		require.NoError(t, err)
		require.Equal(t, expected, tx)
	})

	t.Run("defend", func(t *testing.T) {
		responder, _ := newTestFaultResponder(t)
		responseClaim := types.Claim{
			ClaimData: types.ClaimData{
				Value:    common.Hash{0x01},
				Position: types.NewPositionFromGIndex(3),
			},
			Parent: types.ClaimData{
				Value:    common.Hash{0x02},
				Position: types.NewPositionFromGIndex(6),
			},
			ContractIndex:       0,
			ParentContractIndex: 7,
		}
		tx, err := responder.BuildTx(context.Background(), responseClaim)
		require.NoError(t, err)

		// Pack the tx data manually.
		fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
		require.NoError(t, err)
		parent := big.NewInt(int64(7))
		claim := responseClaim.ValueBytes()
		expected, err := fdgAbi.Pack("defend", parent, claim)

		require.NoError(t, err)
		require.Equal(t, expected, tx)
	})
}

func newTestFaultResponder(t *testing.T) (*faultResponder, *mockTxManager) {
	log := testlog.Logger(t, log.LvlError)
	mockTxMgr := &mockTxManager{}
	responder, err := NewFaultResponder(log, mockTxMgr, mockFdgAddress)
	require.NoError(t, err)
	return responder, mockTxMgr
}

type mockTxManager struct {
	from      common.Address
	sends     int
	calls     int
	sendFails bool
	callFails bool
	callBytes []byte
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
	if m.callFails {
		return nil, mockCallError
	}
	m.calls++
	if m.callBytes != nil {
		return m.callBytes, nil
	}
	return common.Hex2Bytes(
		"0000000000000000000000000000000000000000000000000000000000000000",
	), nil
}

func (m *mockTxManager) BlockNumber(ctx context.Context) (uint64, error) {
	panic("not implemented")
}

func (m *mockTxManager) From() common.Address {
	return m.from
}

func generateMockResponseClaim() types.Claim {
	return types.Claim{
		ClaimData: types.ClaimData{
			Value:    common.Hash{0x01},
			Position: types.NewPositionFromGIndex(2),
		},
		Parent: types.ClaimData{
			Value:    common.Hash{0x02},
			Position: types.NewPositionFromGIndex(1),
		},
		ContractIndex:       0,
		ParentContractIndex: 0,
	}
}
