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

func TestCallResolveClaim(t *testing.T) {
	t.Run("SendFails", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		mockTxMgr.callFails = true
		err := responder.CallResolveClaim(context.Background(), 0)
		require.ErrorIs(t, err, mockCallError)
		require.Equal(t, 0, mockTxMgr.calls)
	})

	t.Run("Success", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		err := responder.CallResolveClaim(context.Background(), 0)
		require.NoError(t, err)
		require.Equal(t, 1, mockTxMgr.calls)
	})
}

func TestResolveClaim(t *testing.T) {
	t.Run("SendFails", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		mockTxMgr.sendFails = true
		err := responder.ResolveClaim(context.Background(), 0)
		require.ErrorIs(t, err, mockSendError)
		require.Equal(t, 0, mockTxMgr.sends)
	})

	t.Run("Success", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		err := responder.ResolveClaim(context.Background(), 0)
		require.NoError(t, err)
		require.Equal(t, 1, mockTxMgr.sends)
	})
}

// TestRespond tests the [Responder.Respond] method.
func TestPerformAction(t *testing.T) {
	t.Run("send fails", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		mockTxMgr.sendFails = true
		err := responder.PerformAction(context.Background(), types.Action{
			Type:      types.ActionTypeMove,
			ParentIdx: 123,
			IsAttack:  true,
			Value:     common.Hash{0xaa},
		})
		require.ErrorIs(t, err, mockSendError)
		require.Equal(t, 0, mockTxMgr.sends)
	})

	t.Run("sends response", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		err := responder.PerformAction(context.Background(), types.Action{
			Type:      types.ActionTypeMove,
			ParentIdx: 123,
			IsAttack:  true,
			Value:     common.Hash{0xaa},
		})
		require.NoError(t, err)
		require.Equal(t, 1, mockTxMgr.sends)
	})

	t.Run("attack", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		action := types.Action{
			Type:      types.ActionTypeMove,
			ParentIdx: 123,
			IsAttack:  true,
			Value:     common.Hash{0xaa},
		}
		err := responder.PerformAction(context.Background(), action)
		require.NoError(t, err)

		// Pack the tx data manually.
		fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
		require.NoError(t, err)
		expected, err := fdgAbi.Pack("attack", big.NewInt(int64(action.ParentIdx)), action.Value)
		require.NoError(t, err)

		require.Len(t, mockTxMgr.sent, 1)
		require.Equal(t, expected, mockTxMgr.sent[0].TxData)
	})

	t.Run("defend", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		action := types.Action{
			Type:      types.ActionTypeMove,
			ParentIdx: 123,
			IsAttack:  false,
			Value:     common.Hash{0xaa},
		}
		err := responder.PerformAction(context.Background(), action)
		require.NoError(t, err)

		// Pack the tx data manually.
		fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
		require.NoError(t, err)
		expected, err := fdgAbi.Pack("defend", big.NewInt(int64(action.ParentIdx)), action.Value)
		require.NoError(t, err)

		require.Len(t, mockTxMgr.sent, 1)
		require.Equal(t, expected, mockTxMgr.sent[0].TxData)
	})

	t.Run("step", func(t *testing.T) {
		responder, mockTxMgr := newTestFaultResponder(t)
		action := types.Action{
			Type:      types.ActionTypeStep,
			ParentIdx: 123,
			IsAttack:  true,
			PreState:  []byte{1, 2, 3},
			ProofData: []byte{4, 5, 6},
		}
		err := responder.PerformAction(context.Background(), action)
		require.NoError(t, err)

		// Pack the tx data manually.
		fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
		require.NoError(t, err)
		expected, err := fdgAbi.Pack("step", big.NewInt(int64(action.ParentIdx)), true, action.PreState, action.ProofData)
		require.NoError(t, err)

		require.Len(t, mockTxMgr.sent, 1)
		require.Equal(t, expected, mockTxMgr.sent[0].TxData)
	})
}

func newTestFaultResponder(t *testing.T) (*FaultResponder, *mockTxManager) {
	log := testlog.Logger(t, log.LvlError)
	mockTxMgr := &mockTxManager{}
	responder, err := NewFaultResponder(log, mockTxMgr, mockFdgAddress)
	require.NoError(t, err)
	return responder, mockTxMgr
}

type mockTxManager struct {
	from      common.Address
	sends     int
	sent      []txmgr.TxCandidate
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
	m.sent = append(m.sent, candidate)
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
