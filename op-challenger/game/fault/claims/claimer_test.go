package claims

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockTxMgrSendError  = errors.New("mock tx mgr send error")
	mockCloseCheckError = errors.New("mock close check error")
)

func TestClaimer_ClaimBonds(t *testing.T) {
	t.Run("MultipleBondClaimsSucceed", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t)
		contract.credit[txSender.From()] = 1
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}, {Proxy: gameAddr}, {Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 3, txSender.sends)
		require.Equal(t, 3, m.RecordBondClaimedCalls)
	})

	t.Run("BondClaimSucceeds", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t)
		contract.credit[txSender.From()] = 1
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 1, txSender.sends)
		require.Equal(t, 1, m.RecordBondClaimedCalls)
	})

	t.Run("BondClaimSucceedsForMultipleAddresses", func(t *testing.T) {
		claimant1 := common.Address{0xaa}
		claimant2 := common.Address{0xbb}
		claimant3 := common.Address{0xcc}
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t, claimant1, claimant2, claimant3)
		contract.credit[claimant1] = 1
		contract.credit[claimant2] = 2
		contract.credit[claimant3] = 0
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 2, txSender.sends)
		require.Equal(t, 2, m.RecordBondClaimedCalls)
	})

	t.Run("BondClaimSkippedForInProgressGame", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t)
		contract.credit[txSender.From()] = 1
		contract.status = types.GameStatusInProgress
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 0, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})

	t.Run("BondClaimFails", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t)
		txSender.sendFails = true
		contract.credit[txSender.From()] = 1
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.ErrorIs(t, err, mockTxMgrSendError)
		require.Equal(t, 1, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})

	t.Run("BondStillLocked", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t)
		contract.claimSimulationFails = true
		contract.credit[txSender.From()] = 1
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 0, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})

	t.Run("ZeroCreditClosesGameWhenOpen", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t)
		contract.credit[txSender.From()] = 0
		contract.isClosedResults = []bool{false}
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 1, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})

	t.Run("ZeroCreditSkipsCloseWhenAlreadyClosed", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t)
		contract.credit[txSender.From()] = 0
		contract.isClosedResults = []bool{true}
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 0, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})

	t.Run("ZeroCreditSkipsCloseWhenCloseNotSupported", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t)
		contract.credit[txSender.From()] = 0
		contract.isClosedResults = []bool{false}
		contract.closeGameNotSupported = true
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 0, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})

	t.Run("ZeroCreditSkipsCloseWhenSimulationFails", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t)
		contract.credit[txSender.From()] = 0
		contract.isClosedResults = []bool{false}
		contract.closeGameSimulationFails = true
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 0, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})

	t.Run("ZeroCreditSkipsCloseWhenSelectiveMode", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimerWithSelective(t, true)
		contract.credit[txSender.From()] = 0
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 0, txSender.sends)
		require.Equal(t, 0, contract.isClosedCalls)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})

	t.Run("SelectiveBondCapableModeOnlyClaimsPositiveCredit", func(t *testing.T) {
		claimantWithCredit := common.Address{0xaa}
		claimantWithoutCredit := common.Address{0xbb}
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimerWithSelective(t, true, claimantWithCredit, claimantWithoutCredit)
		contract.credit[claimantWithCredit] = 1
		contract.credit[claimantWithoutCredit] = 0

		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})

		require.NoError(t, err)
		require.Equal(t, 1, txSender.sends)
		require.Equal(t, 2, contract.getCreditCalls)
		require.Equal(t, 1, contract.claimCreditTxCalls)
		require.Equal(t, 0, contract.isClosedCalls)
		require.Equal(t, 0, contract.closeGameTxCalls)
		require.Equal(t, 1, m.RecordBondClaimedCalls)
	})

	t.Run("FailedCloseSendIsBenignWhenGameClosedConcurrently", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, _, contract, txSender := newTestClaimer(t)
		contract.credit[txSender.From()] = 0
		contract.isClosedResults = []bool{false, true}
		txSender.sendFails = true

		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})

		require.NoError(t, err)
		require.Equal(t, 1, txSender.sends)
		require.Equal(t, 2, contract.isClosedCalls)
	})

	t.Run("FailedCloseSendIsReturnedWhenGameRemainsOpen", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, _, contract, txSender := newTestClaimer(t)
		contract.credit[txSender.From()] = 0
		contract.isClosedResults = []bool{false, false}
		txSender.sendFails = true

		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})

		require.ErrorIs(t, err, mockTxMgrSendError)
		require.Equal(t, 1, txSender.sends)
		require.Equal(t, 2, contract.isClosedCalls)
	})

	t.Run("FailedCloseCheckJoinsSendAndCheckErrors", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, _, contract, txSender := newTestClaimer(t)
		contract.credit[txSender.From()] = 0
		contract.isClosedResults = []bool{false}
		contract.isClosedErrors = []error{nil, mockCloseCheckError}
		txSender.sendFails = true

		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})

		require.ErrorIs(t, err, mockTxMgrSendError)
		require.ErrorIs(t, err, mockCloseCheckError)
		require.Equal(t, 1, txSender.sends)
		require.Equal(t, 2, contract.isClosedCalls)
	})

	t.Run("MultipleBondClaimFails", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t)
		contract.credit[txSender.From()] = 1
		txSender.sendFails = true
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}, {Proxy: gameAddr}, {Proxy: gameAddr}})
		require.ErrorIs(t, err, mockTxMgrSendError)
		require.Equal(t, 3, txSender.sends)
		require.Equal(t, 0, m.RecordBondClaimedCalls)
	})
}

func newTestClaimer(t *testing.T, claimants ...common.Address) (*Claimer, *mockClaimMetrics, *stubBondContract, *mockTxSender) {
	return newTestClaimerWithSelective(t, false, claimants...)
}

func newTestClaimerWithSelective(t *testing.T, selective bool, claimants ...common.Address) (*Claimer, *mockClaimMetrics, *stubBondContract, *mockTxSender) {
	logger := testlog.Logger(t, log.LvlDebug)
	m := &mockClaimMetrics{}
	txSender := &mockTxSender{}
	bondContract := &stubBondContract{
		status: types.GameStatusChallengerWon,
		credit: make(map[common.Address]int64),
	}
	contractCreator := func(game types.GameMetadata) (BondContract, error) {
		return bondContract, nil
	}
	if len(claimants) == 0 {
		claimants = []common.Address{txSender.From()}
	}
	c := NewBondClaimer(logger, m, contractCreator, txSender, selective, claimants...)
	return c, m, bondContract, txSender
}

type mockClaimMetrics struct {
	RecordBondClaimedCalls int
}

func (m *mockClaimMetrics) RecordBondClaimed(amount *big.Int) {
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

func (s *mockTxSender) SendAndWaitSimple(_ string, _ ...txmgr.TxCandidate) error {
	s.sends++
	if s.sendFails {
		return mockTxMgrSendError
	}
	if s.statusFail {
		return errors.New("transaction reverted")
	}
	return nil
}

type stubBondContract struct {
	credit                   map[common.Address]int64
	status                   types.GameStatus
	isClosedResults          []bool
	isClosedErrors           []error
	claimSimulationFails     bool
	closeGameSimulationFails bool
	closeGameNotSupported    bool
	getCreditCalls           int
	claimCreditTxCalls       int
	isClosedCalls            int
	closeGameTxCalls         int
}

func (s *stubBondContract) IsClosed(_ context.Context) (bool, error) {
	call := s.isClosedCalls
	s.isClosedCalls++
	var closed bool
	if call < len(s.isClosedResults) {
		closed = s.isClosedResults[call]
	}
	var err error
	if call < len(s.isClosedErrors) {
		err = s.isClosedErrors[call]
	}
	return closed, err
}

func (s *stubBondContract) GetCredit(_ context.Context, addr common.Address) (*big.Int, types.GameStatus, error) {
	s.getCreditCalls++
	return big.NewInt(s.credit[addr]), s.status, nil
}

func (s *stubBondContract) ClaimCreditTx(_ context.Context, _ common.Address) (txmgr.TxCandidate, error) {
	s.claimCreditTxCalls++
	if s.claimSimulationFails {
		return txmgr.TxCandidate{}, fmt.Errorf("failed: %w", contracts.ErrSimulationFailed)
	}
	return txmgr.TxCandidate{}, nil
}

func (s *stubBondContract) CloseGameTx(_ context.Context) (txmgr.TxCandidate, error) {
	s.closeGameTxCalls++
	if s.closeGameNotSupported {
		return txmgr.TxCandidate{}, contracts.ErrCloseGameNotSupported
	}
	if s.closeGameSimulationFails {
		return txmgr.TxCandidate{}, fmt.Errorf("failed: %w", contracts.ErrSimulationFailed)
	}
	return txmgr.TxCandidate{}, nil
}
