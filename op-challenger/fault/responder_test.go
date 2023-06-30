package fault

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/stretchr/testify/require"
)

var (
	mockFdgAddress = common.HexToAddress("0x1234")
	mockSendError  = errors.New("mock send error")
)

type mockTxManager struct {
	from      common.Address
	sends     int
	sendFails bool
}

func (m *mockTxManager) Send(ctx context.Context, candidate txmgr.TxCandidate) (*types.Receipt, error) {
	if m.sendFails {
		return nil, mockSendError
	}
	m.sends++
	return types.NewReceipt(
		[]byte{},
		false,
		0,
	), nil
}

func (m *mockTxManager) From() common.Address {
	return m.from
}

func newTestFaultResponder(t *testing.T, sendFails bool) (*faultResponder, *mockTxManager) {
	log := testlog.Logger(t, log.LvlError)
	mockTxMgr := &mockTxManager{}
	mockTxMgr.sendFails = sendFails
	responder, err := NewFaultResponder(log, mockTxMgr, mockFdgAddress)
	require.NoError(t, err)
	return responder, mockTxMgr
}

// TestResponder_Respond_SendFails tests the [Responder.Respond] method
// bubbles up the error returned by the [txmgr.Send] method.
func TestResponder_Respond_SendFails(t *testing.T) {
	responder, mockTxMgr := newTestFaultResponder(t, true)
	err := responder.Respond(context.Background(), Claim{
		ClaimData: ClaimData{
			Value:    common.Hash{0x01},
			Position: NewPositionFromGIndex(2),
		},
		Parent: ClaimData{
			Value:    common.Hash{0x02},
			Position: NewPositionFromGIndex(1),
		},
		ContractIndex:       0,
		ParentContractIndex: 0,
	})
	require.ErrorIs(t, err, mockSendError)
	require.Equal(t, 0, mockTxMgr.sends)
}

// TestResponder_Respond_Success tests the [Responder.Respond] method
// succeeds when the tx candidate is successfully sent through the txmgr.
func TestResponder_Respond_Success(t *testing.T) {
	responder, mockTxMgr := newTestFaultResponder(t, false)
	err := responder.Respond(context.Background(), Claim{
		ClaimData: ClaimData{
			Value:    common.Hash{0x01},
			Position: NewPositionFromGIndex(2),
		},
		Parent: ClaimData{
			Value:    common.Hash{0x02},
			Position: NewPositionFromGIndex(1),
		},
		ContractIndex:       0,
		ParentContractIndex: 0,
	})
	require.NoError(t, err)
	require.Equal(t, 1, mockTxMgr.sends)
}

// TestResponder_BuildTx_Attack tests the [Responder.BuildTx] method
// returns a tx candidate with the correct data for an attack tx.
func TestResponder_BuildTx_Attack(t *testing.T) {
	responder, _ := newTestFaultResponder(t, false)
	responseClaim := Claim{
		ClaimData: ClaimData{
			Value:    common.Hash{0x01},
			Position: NewPositionFromGIndex(2),
		},
		Parent: ClaimData{
			Value:    common.Hash{0x02},
			Position: NewPositionFromGIndex(1),
		},
		ContractIndex:       0,
		ParentContractIndex: 7,
	}
	tx, err := responder.BuildTx(context.Background(), responseClaim)
	require.NoError(t, err)

	// Pack the tx data manually.
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	require.NoError(t, err)
	expected, err := fdgAbi.Pack(
		"attack",
		big.NewInt(int64(7)),
		responseClaim.ValueBytes(),
	)
	require.NoError(t, err)
	require.Equal(t, expected, tx)
}

// TestResponder_BuildTx_Defend tests the [Responder.BuildTx] method
// returns a tx candidate with the correct data for a defend tx.
func TestResponder_BuildTx_Defend(t *testing.T) {
	responder, _ := newTestFaultResponder(t, false)
	responseClaim := Claim{
		ClaimData: ClaimData{
			Value:    common.Hash{0x01},
			Position: NewPositionFromGIndex(3),
		},
		Parent: ClaimData{
			Value:    common.Hash{0x02},
			Position: NewPositionFromGIndex(6),
		},
		ContractIndex:       0,
		ParentContractIndex: 7,
	}
	tx, err := responder.BuildTx(context.Background(), responseClaim)
	require.NoError(t, err)

	// Pack the tx data manually.
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	require.NoError(t, err)
	expected, err := fdgAbi.Pack(
		"defend",
		big.NewInt(int64(7)),
		responseClaim.ValueBytes(),
	)
	require.NoError(t, err)
	require.Equal(t, expected, tx)
}
