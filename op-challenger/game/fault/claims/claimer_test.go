package claims

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	mockTxMgrSendError  = errors.New("mock tx mgr send error")
	mockCloseCheckError = errors.New("mock close check error")
	mockCreditReadError = errors.New("mock credit read error")
)

func TestClaimer_ClaimBonds(t *testing.T) {
	t.Run("MultipleBondClaimsSucceed", func(t *testing.T) {
		m := &mockClaimMetrics{}
		txSender := newMockTxSender()
		games := []types.GameMetadata{{Proxy: common.Address{0x01}}, {Proxy: common.Address{0x02}}, {Proxy: common.Address{0x03}}}
		for _, game := range games {
			txSender.games[game.Proxy] = newStubBondContract(game.Proxy)
			txSender.games[game.Proxy].credit[txSender.From()] = 1
		}
		contractCreator := func(game types.GameMetadata) (BondContract, error) {
			return txSender.games[game.Proxy], nil
		}
		c := NewBondClaimer(testlog.Logger(t, log.LevelDebug), m, contractCreator, txSender, false, txSender.From())

		err := c.ClaimBonds(context.Background(), games)

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

	t.Run("TwoStepClaimRecordsBondOnceOnPayout", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t)
		contract.twoPhase = true
		contract.credit[txSender.From()] = 5
		contract.isClosedResults = []bool{true}
		games := []types.GameMetadata{{Proxy: gameAddr}}

		require.NoError(t, c.ClaimBonds(context.Background(), games))
		require.Equal(t, 1, txSender.sends, "unlock should be sent")
		require.Zero(t, m.RecordBondClaimedCalls)

		contract.claimSimulationFails = true
		require.NoError(t, c.ClaimBonds(context.Background(), games))
		require.Equal(t, 1, txSender.sends, "payout should wait for the withdrawal delay")
		require.Zero(t, m.RecordBondClaimedCalls)

		contract.claimSimulationFails = false
		require.NoError(t, c.ClaimBonds(context.Background(), games))
		require.Equal(t, 2, txSender.sends, "payout should be sent once the withdrawal matures")
		require.Equal(t, 1, m.RecordBondClaimedCalls)
		require.Equal(t, int64(5), m.claimedAmount)

		require.NoError(t, c.ClaimBonds(context.Background(), games))
		require.Equal(t, 2, txSender.sends, "nothing left to claim")
		require.Equal(t, 1, m.RecordBondClaimedCalls)
		require.Zero(t, contract.closeGameTxCalls)
	})

	t.Run("FailedCreditReadAfterClaimSkipsMetric", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		c, m, contract, txSender := newTestClaimer(t)
		contract.credit[txSender.From()] = 1
		contract.creditReadFailsAfterClaim = true
		err := c.ClaimBonds(context.Background(), []types.GameMetadata{{Proxy: gameAddr}})
		require.NoError(t, err)
		require.Equal(t, 1, txSender.sends)
		require.Zero(t, m.RecordBondClaimedCalls)
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
	logger := testlog.Logger(t, log.LevelDebug)
	m := &mockClaimMetrics{}
	bondContract := newStubBondContract(common.HexToAddress("0x1234"))
	txSender := newMockTxSender(bondContract)
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
	claimedAmount          int64
}

func (m *mockClaimMetrics) RecordBondClaimed(amount *big.Int) {
	m.RecordBondClaimedCalls++
	m.claimedAmount += amount.Int64()
}

type mockTxSender struct {
	sends      int
	sendFails  bool
	statusFail bool
	games      map[common.Address]*stubBondContract
}

func newMockTxSender(games ...*stubBondContract) *mockTxSender {
	s := &mockTxSender{games: make(map[common.Address]*stubBondContract)}
	for _, game := range games {
		s.games[game.addr] = game
	}
	return s
}

func (s *mockTxSender) From() common.Address {
	return common.HexToAddress("0x33333")
}

func (s *mockTxSender) SendAndWaitSimple(_ string, txs ...txmgr.TxCandidate) error {
	s.sends++
	if s.sendFails {
		return mockTxMgrSendError
	}
	if s.statusFail {
		return errors.New("transaction reverted")
	}
	for _, tx := range txs {
		s.games[*tx.To].execute(tx)
	}
	return nil
}

// stubBondContract models a game's claimCredit: a claim pays credit out, or in a two-phase game first
// unlocks it into a pending DelayedWETH withdrawal that the next claim pays out.
type stubBondContract struct {
	addr                      common.Address
	credit                    map[common.Address]int64
	pending                   map[common.Address]int64
	twoPhase                  bool
	creditReadFailsAfterClaim bool
	status                    types.GameStatus
	isClosedResults           []bool
	isClosedErrors            []error
	claimSimulationFails      bool
	closeGameSimulationFails  bool
	closeGameNotSupported     bool
	claimCreditTxCalls        int
	isClosedCalls             int
	closeGameTxCalls          int
}

func newStubBondContract(addr common.Address) *stubBondContract {
	return &stubBondContract{
		addr:    addr,
		credit:  make(map[common.Address]int64),
		pending: make(map[common.Address]int64),
		status:  types.GameStatusChallengerWon,
	}
}

// execute applies a sent transaction. A claim's calldata is the claimant; closeGame changes no credit.
func (s *stubBondContract) execute(tx txmgr.TxCandidate) {
	if len(tx.TxData) == 0 {
		return
	}
	claimant := common.BytesToAddress(tx.TxData)
	if s.twoPhase && s.credit[claimant] > 0 {
		s.pending[claimant] = s.credit[claimant]
	} else {
		s.pending[claimant] = 0
	}
	s.credit[claimant] = 0
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
	if s.creditReadFailsAfterClaim && s.claimCreditTxCalls > 0 {
		return nil, s.status, mockCreditReadError
	}
	return big.NewInt(s.credit[addr] + s.pending[addr]), s.status, nil
}

func (s *stubBondContract) ClaimCreditTx(_ context.Context, addr common.Address) (txmgr.TxCandidate, error) {
	s.claimCreditTxCalls++
	if s.claimSimulationFails {
		return txmgr.TxCandidate{}, fmt.Errorf("failed: %w", contracts.ErrSimulationFailed)
	}
	return txmgr.TxCandidate{To: &s.addr, TxData: addr.Bytes()}, nil
}

func (s *stubBondContract) CloseGameTx(_ context.Context) (txmgr.TxCandidate, error) {
	s.closeGameTxCalls++
	if s.closeGameNotSupported {
		return txmgr.TxCandidate{}, contracts.ErrCloseGameNotSupported
	}
	if s.closeGameSimulationFails {
		return txmgr.TxCandidate{}, fmt.Errorf("failed: %w", contracts.ErrSimulationFailed)
	}
	return txmgr.TxCandidate{To: &s.addr}, nil
}
