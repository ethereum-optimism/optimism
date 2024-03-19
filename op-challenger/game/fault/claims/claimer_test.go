package claims

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockTxMgrSendError = errors.New("mock tx mgr send error")
)

func TestClaimer_ClaimBonds(t *testing.T) {
	t.Run("MultipleBondClaimsSucceed", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t, gameAddr)
		contract.credit = 1
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}, {Proxy: gameAddr}, {Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 3, txSender.sends)
		require.Equal(t, 3, m.RecordBondClaimedCalls)
	})

	t.Run("BondClaimSucceeds", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t, gameAddr)
		contract.credit = 1
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 1, txSender.sends)
		require.Equal(t, 1, m.RecordBondClaimedCalls)
	})

	t.Run("BondClaimFails", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t, gameAddr)
		txSender.sendFails = true
		contract.credit = 1
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.ErrorIs(t, err, mockTxMgrSendError)
		require.Equal(t, 1, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})

	t.Run("ZeroCreditReturnsNil", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t, gameAddr)
		contract.credit = 0
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 0, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})

	t.Run("MultipleBondClaimFails", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t, gameAddr)
		contract.credit = 1
		txSender.sendFails = true
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}, {Proxy: gameAddr}, {Proxy: gameAddr}})
		require.ErrorIs(t, err, mockTxMgrSendError)
		require.Equal(t, 3, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})
}

func newTestClaimer(t *testing.T, gameAddr common.Address) (*Claimer, *mockClaimMetrics, *stubBondContract, *mockTxSender) {
	logger := testlog.Logger(t, log.LvlDebug)
	m := &mockClaimMetrics{}
	txSender := &mockTxSender{}
	bondContract := &stubBondContract{}
	contractCreator := func(game types.GameMetadata) (BondContract, error) {
		return bondContract, nil
	}
	c := NewBondClaimer(logger, m, contractCreator, txSender)
	return c, m, bondContract, txSender
}

type mockClaimMetrics struct {
	RecordBondClaimedCalls int
}

func (m *mockClaimMetrics) RecordBondClaimed(amount uint64) {
	m.RecordBondClaimedCalls++
}

type mockTxSender struct {
	sends      int
	sendFails  bool
	statusFail bool
}

func (s *mockTxSender) From() common.Address {
	return common.HexToAddress("0x33333")
}

func (s *mockTxSender) SendAndWait(_ string, _ ...txmgr.TxCandidate) ([]*ethtypes.Receipt, error) {
	s.sends++
	if s.sendFails {
		return nil, mockTxMgrSendError
	}
	if s.statusFail {
		return []*ethtypes.Receipt{{Status: ethtypes.ReceiptStatusFailed}}, nil
	}
	return []*ethtypes.Receipt{{Status: ethtypes.ReceiptStatusSuccessful}}, nil
}

type stubBondContract struct {
	credit int64
}

func (s *stubBondContract) GetCredit(_ context.Context, _ common.Address) (*big.Int, error) {
	return big.NewInt(s.credit), nil
}

func (s *stubBondContract) ClaimCredit(_ common.Address) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, nil
}
