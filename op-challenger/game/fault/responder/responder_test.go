package responder

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/stretchr/testify/require"
)

var (
	mockSendError = errors.New("mock send error")
	mockCallError = errors.New("mock call error")
)

// TestCallResolve tests the [Responder.CallResolve].
func TestCallResolve(t *testing.T) {
	t.Run("SendFails", func(t *testing.T) {
		responder, _, contract := newTestFaultResponder(t)
		contract.callFails = true
		status, err := responder.CallResolve(context.Background())
		require.ErrorIs(t, err, mockCallError)
		require.Equal(t, gameTypes.GameStatusInProgress, status)
		require.Equal(t, 0, contract.calls)
	})

	t.Run("Success", func(t *testing.T) {
		responder, _, contract := newTestFaultResponder(t)
		status, err := responder.CallResolve(context.Background())
		require.NoError(t, err)
		require.Equal(t, gameTypes.GameStatusInProgress, status)
		require.Equal(t, 1, contract.calls)
	})
}

// TestResolve tests the [Responder.Resolve] method.
func TestResolve(t *testing.T) {
	t.Run("SendFails", func(t *testing.T) {
		responder, mockTxMgr, _ := newTestFaultResponder(t)
		mockTxMgr.sendFails = true
		err := responder.Resolve(context.Background())
		require.ErrorIs(t, err, mockSendError)
		require.Equal(t, 0, mockTxMgr.sends)
	})

	t.Run("Success", func(t *testing.T) {
		responder, mockTxMgr, _ := newTestFaultResponder(t)
		err := responder.Resolve(context.Background())
		require.NoError(t, err)
		require.Equal(t, 1, mockTxMgr.sends)
	})
}

func TestCallResolveClaim(t *testing.T) {
	t.Run("SendFails", func(t *testing.T) {
		responder, _, contract := newTestFaultResponder(t)
		contract.callFails = true
		err := responder.CallResolveClaim(context.Background(), 0)
		require.ErrorIs(t, err, mockCallError)
		require.Equal(t, 0, contract.calls)
	})

	t.Run("Success", func(t *testing.T) {
		responder, _, contract := newTestFaultResponder(t)
		err := responder.CallResolveClaim(context.Background(), 0)
		require.NoError(t, err)
		require.Equal(t, 1, contract.calls)
	})
}

func TestResolveClaim(t *testing.T) {
	t.Run("SendFails", func(t *testing.T) {
		responder, mockTxMgr, _ := newTestFaultResponder(t)
		mockTxMgr.sendFails = true
		err := responder.ResolveClaim(context.Background(), 0)
		require.ErrorIs(t, err, mockSendError)
		require.Equal(t, 0, mockTxMgr.sends)
	})

	t.Run("Success", func(t *testing.T) {
		responder, mockTxMgr, _ := newTestFaultResponder(t)
		err := responder.ResolveClaim(context.Background(), 0)
		require.NoError(t, err)
		require.Equal(t, 1, mockTxMgr.sends)
	})
}

// TestRespond tests the [Responder.Respond] method.
func TestPerformAction(t *testing.T) {
	t.Run("send fails", func(t *testing.T) {
		responder, mockTxMgr, _ := newTestFaultResponder(t)
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
		responder, mockTxMgr, _ := newTestFaultResponder(t)
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
		responder, mockTxMgr, contract := newTestFaultResponder(t)
		action := types.Action{
			Type:      types.ActionTypeMove,
			ParentIdx: 123,
			IsAttack:  true,
			Value:     common.Hash{0xaa},
		}
		err := responder.PerformAction(context.Background(), action)
		require.NoError(t, err)

		require.Len(t, mockTxMgr.sent, 1)
		require.EqualValues(t, []interface{}{uint64(action.ParentIdx), action.Value}, contract.attackArgs)
		require.Equal(t, ([]byte)("attack"), mockTxMgr.sent[0].TxData)
	})

	t.Run("defend", func(t *testing.T) {
		responder, mockTxMgr, contract := newTestFaultResponder(t)
		action := types.Action{
			Type:      types.ActionTypeMove,
			ParentIdx: 123,
			IsAttack:  false,
			Value:     common.Hash{0xaa},
		}
		err := responder.PerformAction(context.Background(), action)
		require.NoError(t, err)

		require.Len(t, mockTxMgr.sent, 1)
		require.EqualValues(t, []interface{}{uint64(action.ParentIdx), action.Value}, contract.defendArgs)
		require.Equal(t, ([]byte)("defend"), mockTxMgr.sent[0].TxData)
	})

	t.Run("step", func(t *testing.T) {
		responder, mockTxMgr, contract := newTestFaultResponder(t)
		action := types.Action{
			Type:      types.ActionTypeStep,
			ParentIdx: 123,
			IsAttack:  true,
			PreState:  []byte{1, 2, 3},
			ProofData: []byte{4, 5, 6},
		}
		err := responder.PerformAction(context.Background(), action)
		require.NoError(t, err)

		require.Len(t, mockTxMgr.sent, 1)
		require.EqualValues(t, []interface{}{uint64(action.ParentIdx), action.IsAttack, action.PreState, action.ProofData}, contract.stepArgs)
		require.Equal(t, ([]byte)("step"), mockTxMgr.sent[0].TxData)
	})

	t.Run("stepWithOracleData", func(t *testing.T) {
		responder, mockTxMgr, contract := newTestFaultResponder(t)
		action := types.Action{
			Type:      types.ActionTypeStep,
			ParentIdx: 123,
			IsAttack:  true,
			PreState:  []byte{1, 2, 3},
			ProofData: []byte{4, 5, 6},
			OracleData: &types.PreimageOracleData{
				IsLocal: true,
			},
		}
		err := responder.PerformAction(context.Background(), action)
		require.NoError(t, err)

		require.Len(t, mockTxMgr.sent, 2)
		require.EqualValues(t, action.OracleData, contract.updateOracleArgs)
		require.EqualValues(t, action.ParentIdx, contract.updateOracleClaimIdx)
		require.EqualValues(t, []interface{}{uint64(action.ParentIdx), action.IsAttack, action.PreState, action.ProofData}, contract.stepArgs)
		// Important that the oracle is updated first
		require.Equal(t, ([]byte)("updateOracle"), mockTxMgr.sent[0].TxData)
		require.Equal(t, ([]byte)("step"), mockTxMgr.sent[1].TxData)
	})
}

func newTestFaultResponder(t *testing.T) (*FaultResponder, *mockTxManager, *mockContract) {
	log := testlog.Logger(t, log.LvlError)
	mockTxMgr := &mockTxManager{}
	contract := &mockContract{}
	responder, err := NewFaultResponder(log, mockTxMgr, contract)
	require.NoError(t, err)
	return responder, mockTxMgr, contract
}

type mockTxManager struct {
	from      common.Address
	sends     int
	sent      []txmgr.TxCandidate
	sendFails bool
}

func (m *mockTxManager) Send(_ context.Context, candidate txmgr.TxCandidate) (*ethtypes.Receipt, error) {
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

func (m *mockTxManager) BlockNumber(_ context.Context) (uint64, error) {
	panic("not implemented")
}

func (m *mockTxManager) From() common.Address {
	return m.from
}

func (m *mockTxManager) Close() {
}

type mockContract struct {
	calls                int
	callFails            bool
	attackArgs           []interface{}
	defendArgs           []interface{}
	stepArgs             []interface{}
	updateOracleClaimIdx uint64
	updateOracleArgs     *types.PreimageOracleData
}

func (m *mockContract) CallResolve(_ context.Context) (gameTypes.GameStatus, error) {
	if m.callFails {
		return gameTypes.GameStatusInProgress, mockCallError
	}
	m.calls++
	return gameTypes.GameStatusInProgress, nil
}

func (m *mockContract) ResolveTx() (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, nil
}

func (m *mockContract) CallResolveClaim(_ context.Context, _ uint64) error {
	if m.callFails {
		return mockCallError
	}
	m.calls++
	return nil
}

func (m *mockContract) ResolveClaimTx(_ uint64) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, nil
}

func (m *mockContract) AttackTx(parentClaimId uint64, claim common.Hash) (txmgr.TxCandidate, error) {
	m.attackArgs = []interface{}{parentClaimId, claim}
	return txmgr.TxCandidate{TxData: ([]byte)("attack")}, nil
}

func (m *mockContract) DefendTx(parentClaimId uint64, claim common.Hash) (txmgr.TxCandidate, error) {
	m.defendArgs = []interface{}{parentClaimId, claim}
	return txmgr.TxCandidate{TxData: ([]byte)("defend")}, nil
}

func (m *mockContract) StepTx(claimIdx uint64, isAttack bool, stateData []byte, proofData []byte) (txmgr.TxCandidate, error) {
	m.stepArgs = []interface{}{claimIdx, isAttack, stateData, proofData}
	return txmgr.TxCandidate{TxData: ([]byte)("step")}, nil
}

func (m *mockContract) UpdateOracleTx(_ context.Context, claimIdx uint64, data *types.PreimageOracleData) (txmgr.TxCandidate, error) {
	m.updateOracleClaimIdx = claimIdx
	m.updateOracleArgs = data
	return txmgr.TxCandidate{TxData: ([]byte)("updateOracle")}, nil
}
