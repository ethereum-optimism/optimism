package claims

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	methodCredit       = "credit"
	mockTxMgrSendError = errors.New("mock tx mgr send error")
)

func TestClaimer_ClaimBonds(t *testing.T) {
	t.Run("MultipleBondClaimsSucceed", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, rpc, txSender := newTestClaimer(t, gameAddr)
		rpc.SetResponse(gameAddr, methodCredit, batching.BlockLatest, []interface{}{txSender.From()}, []interface{}{big.NewInt(1)})
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}, {Proxy: gameAddr}, {Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 3, txSender.sends)
		require.Equal(t, 3, m.RecordBondClaimedCalls)
	})

	t.Run("BondClaimSucceeds", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, rpc, txSender := newTestClaimer(t, gameAddr)
		rpc.SetResponse(gameAddr, methodCredit, batching.BlockLatest, []interface{}{txSender.From()}, []interface{}{big.NewInt(1)})
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 1, txSender.sends)
		require.Equal(t, 1, m.RecordBondClaimedCalls)
	})

	t.Run("BondClaimFails", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, rpc, txSender := newTestClaimer(t, gameAddr)
		txSender.sendFails = true
		rpc.SetResponse(gameAddr, methodCredit, batching.BlockLatest, []interface{}{txSender.From()}, []interface{}{big.NewInt(1)})
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.ErrorIs(t, err, mockTxMgrSendError)
		require.Equal(t, 1, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})

	t.Run("ZeroCreditReturnsNil", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, rpc, txSender := newTestClaimer(t, gameAddr)
		rpc.SetResponse(gameAddr, methodCredit, batching.BlockLatest, []interface{}{txSender.From()}, []interface{}{big.NewInt(0)})
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 0, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})

	t.Run("MultipleBondClaimFails", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, rpc, txSender := newTestClaimer(t, gameAddr)
		rpc.SetResponse(gameAddr, methodCredit, batching.BlockLatest, []interface{}{txSender.From()}, []interface{}{big.NewInt(1)})
		txSender.sendFails = true
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}, {Proxy: gameAddr}, {Proxy: gameAddr}})
		require.ErrorIs(t, err, mockTxMgrSendError)
		require.Equal(t, 3, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})
}

func newTestClaimer(t *testing.T, gameAddr common.Address) (*claimer, *mockClaimMetrics, *batchingTest.AbiBasedRpc, *mockTxSender) {
	logger := testlog.Logger(t, log.LvlDebug)
	m := &mockClaimMetrics{}
	txSender := &mockTxSender{}
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	require.NoError(t, err)
	stubRpc := batchingTest.NewAbiBasedRpc(t, gameAddr, fdgAbi)
	caller := batching.NewMultiCaller(stubRpc, 100)
	c := NewBondClaimer(logger, m, caller, txSender)
	return c, m, stubRpc, txSender
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
