package withdrawals

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	gameAddr      = common.Address{0xaa}
	otherGameAddr = common.Address{0xbb}
	gameL1Head    = common.Hash{0x11}
	proofTopic    = common.Hash{0x99}
	l1HeadNumber  = uint64(100)
)

func TestDeleter_DeletesProofsAgainstChallengerWinGame(t *testing.T) {
	deleter, portal, games, l1, sender, m := setupDeleterTest(t)
	games.state(gameAddr, types.GameStatusChallengerWon)
	proofs := l1.proofs(3)
	portal.record(proofs[0], gameAddr, 1)
	portal.record(proofs[1], gameAddr, 2)
	portal.record(proofs[2], gameAddr, 3)

	require.NoError(t, deleter.DeleteInvalidatedWithdrawals(context.Background(), 200, gameList(gameAddr)))

	require.Equal(t, proofs, sender.sent)
	require.Equal(t, 3, m.deleted)
}

func TestDeleter_IgnoresProofsAgainstOtherGames(t *testing.T) {
	deleter, portal, games, l1, sender, _ := setupDeleterTest(t)
	games.state(gameAddr, types.GameStatusChallengerWon)
	proofs := l1.proofs(2)
	portal.record(proofs[0], otherGameAddr, 1)
	portal.record(proofs[1], gameAddr, 2)

	require.NoError(t, deleter.DeleteInvalidatedWithdrawals(context.Background(), 200, gameList(gameAddr)))

	require.Equal(t, proofs[1:], sender.sent)
}

func TestDeleter_SkipsAlreadyDeletedProofs(t *testing.T) {
	deleter, portal, games, l1, sender, m := setupDeleterTest(t)
	games.state(gameAddr, types.GameStatusChallengerWon)
	proofs := l1.proofs(2)
	portal.record(proofs[0], gameAddr, 0)
	portal.record(proofs[1], gameAddr, 0)

	require.NoError(t, deleter.DeleteInvalidatedWithdrawals(context.Background(), 200, gameList(gameAddr)))

	require.Empty(t, sender.sent)
	require.Zero(t, m.deleted)
}

func TestDeleter_IgnoresUnresolvedAndDefenderWinGames(t *testing.T) {
	for _, status := range []types.GameStatus{types.GameStatusInProgress, types.GameStatusDefenderWon} {
		t.Run(status.String(), func(t *testing.T) {
			deleter, portal, games, l1, sender, _ := setupDeleterTest(t)
			games.state(gameAddr, status)
			proofs := l1.proofs(2)
			portal.record(proofs[0], gameAddr, 1)

			require.NoError(t, deleter.DeleteInvalidatedWithdrawals(context.Background(), 200, gameList(gameAddr)))

			require.Empty(t, sender.sent)
			require.Empty(t, l1.queries, "should not scan logs for a game the challenger did not win")
		})
	}
}

func TestDeleter_TruncatesAtCapAndRescansUntilComplete(t *testing.T) {
	deleter, portal, games, l1, sender, _ := setupDeleterTest(t)
	logger, logs := testlog.CaptureLogger(t, log.LevelWarn)
	deleter.log = logger
	games.state(gameAddr, types.GameStatusChallengerWon)
	proofs := l1.proofs(maxDeletesPerGame + 5)
	for i, proof := range proofs {
		portal.record(proof, gameAddr, uint64(i+1))
	}

	require.NoError(t, deleter.DeleteInvalidatedWithdrawals(context.Background(), 200, gameList(gameAddr)))
	require.Equal(t, proofs[:maxDeletesPerGame], sender.sent)
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(log.LevelWarn),
		testlog.NewMessageFilter("Deferring some withdrawal proof deletions to a later run")))

	// The cursor must not have advanced, so the next run picks up the remainder.
	for _, proof := range proofs[:maxDeletesPerGame] {
		portal.record(proof, gameAddr, 0)
	}
	sender.sent = nil
	require.NoError(t, deleter.DeleteInvalidatedWithdrawals(context.Background(), 200, gameList(gameAddr)))
	require.Equal(t, proofs[maxDeletesPerGame:], sender.sent)
	require.Equal(t, []uint64{l1HeadNumber, l1HeadNumber}, l1.queries)
}

func TestDeleter_CursorOnlyScansNewBlocks(t *testing.T) {
	deleter, portal, games, l1, sender, _ := setupDeleterTest(t)
	games.state(gameAddr, types.GameStatusChallengerWon)
	proofs := l1.proofs(1)
	portal.record(proofs[0], gameAddr, 1)

	require.NoError(t, deleter.DeleteInvalidatedWithdrawals(context.Background(), 200, gameList(gameAddr)))
	require.Equal(t, proofs, sender.sent)

	sender.sent = nil
	l1.logs = nil
	require.NoError(t, deleter.DeleteInvalidatedWithdrawals(context.Background(), 300, gameList(gameAddr)))
	require.Empty(t, sender.sent)
	require.Equal(t, []uint64{l1HeadNumber, 201}, l1.queries)
}

func TestDeleter_RescansAfterL1Reorg(t *testing.T) {
	deleter, portal, games, l1, _, _ := setupDeleterTest(t)
	games.state(gameAddr, types.GameStatusChallengerWon)
	portal.record(l1.proofs(1)[0], gameAddr, 1)

	require.NoError(t, deleter.DeleteInvalidatedWithdrawals(context.Background(), 200, gameList(gameAddr)))
	require.NoError(t, deleter.DeleteInvalidatedWithdrawals(context.Background(), 150, gameList(gameAddr)))

	require.Equal(t, []uint64{l1HeadNumber, 150}, l1.queries)
}

func TestDeleter_ReportsSendFailures(t *testing.T) {
	deleter, portal, games, l1, sender, m := setupDeleterTest(t)
	games.state(gameAddr, types.GameStatusChallengerWon)
	proofs := l1.proofs(1)
	portal.record(proofs[0], gameAddr, 1)
	sender.err = errors.New("boom")

	err := deleter.DeleteInvalidatedWithdrawals(context.Background(), 200, gameList(gameAddr))
	require.ErrorIs(t, err, sender.err)
	require.Zero(t, m.deleted)

	// The cursor must not have advanced so the failed delete is retried.
	sender.err = nil
	require.NoError(t, deleter.DeleteInvalidatedWithdrawals(context.Background(), 200, gameList(gameAddr)))
	require.Equal(t, []uint64{l1HeadNumber, l1HeadNumber}, l1.queries)
}

func gameList(addrs ...common.Address) []types.GameMetadata {
	games := make([]types.GameMetadata, len(addrs))
	for i, addr := range addrs {
		games[i] = types.GameMetadata{Proxy: addr}
	}
	return games
}

func setupDeleterTest(t *testing.T) (*Deleter, *stubPortal, *stubGames, *stubL1, *stubTxSender, *stubMetrics) {
	portal := &stubPortal{records: make(map[contracts.WithdrawalProof]contracts.ProvenWithdrawal)}
	games := &stubGames{states: make(map[common.Address]GameState)}
	l1 := &stubL1{}
	sender := &stubTxSender{}
	m := &stubMetrics{}
	return NewDeleter(testlog.Logger(t, log.LevelDebug), m, portal, games, l1, sender), portal, games, l1, sender, m
}

type stubPortal struct {
	records map[contracts.WithdrawalProof]contracts.ProvenWithdrawal
}

func (s *stubPortal) record(proof contracts.WithdrawalProof, game common.Address, timestamp uint64) {
	s.records[proof] = contracts.ProvenWithdrawal{DisputeGameProxy: game, Timestamp: timestamp}
}

func (s *stubPortal) Addr() common.Address {
	return common.Address{0xef}
}

func (s *stubPortal) WithdrawalProvenExtension1Topic() common.Hash {
	return proofTopic
}

func (s *stubPortal) DecodeWithdrawalProvenExtension1(l *ethTypes.Log) (contracts.WithdrawalProof, error) {
	if len(l.Topics) != 3 || l.Topics[0] != proofTopic {
		return contracts.WithdrawalProof{}, fmt.Errorf("unexpected log %v", l)
	}
	return contracts.WithdrawalProof{
		WithdrawalHash: l.Topics[1],
		ProofSubmitter: common.BytesToAddress(l.Topics[2].Bytes()),
	}, nil
}

func (s *stubPortal) GetProvenWithdrawals(_ context.Context, _ rpcblock.Block, proofs []contracts.WithdrawalProof) ([]contracts.ProvenWithdrawal, error) {
	records := make([]contracts.ProvenWithdrawal, len(proofs))
	for i, proof := range proofs {
		records[i] = s.records[proof]
	}
	return records, nil
}

func (s *stubPortal) DeleteProvenWithdrawalTx(proof contracts.WithdrawalProof) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{TxData: append(proof.WithdrawalHash.Bytes(), proof.ProofSubmitter.Bytes()...)}, nil
}

type stubGames struct {
	states map[common.Address]GameState
}

func (s *stubGames) state(game common.Address, status types.GameStatus) {
	s.states[game] = GameState{Status: status, L1Head: gameL1Head}
}

func (s *stubGames) GetGameStates(_ context.Context, _ rpcblock.Block, games []common.Address) ([]GameState, error) {
	states := make([]GameState, len(games))
	for i, game := range games {
		states[i] = s.states[game]
	}
	return states, nil
}

type stubL1 struct {
	logs    []ethTypes.Log
	queries []uint64
}

// proofs adds count withdrawal proof logs to the L1 chain and returns them in log order.
func (s *stubL1) proofs(count int) []contracts.WithdrawalProof {
	proofs := make([]contracts.WithdrawalProof, count)
	for i := range proofs {
		proofs[i] = contracts.WithdrawalProof{
			WithdrawalHash: common.BigToHash(big.NewInt(int64(i + 1))),
			ProofSubmitter: common.BigToAddress(big.NewInt(int64(i + 1))),
		}
		s.logs = append(s.logs, ethTypes.Log{Topics: []common.Hash{
			proofTopic,
			proofs[i].WithdrawalHash,
			common.BytesToHash(proofs[i].ProofSubmitter.Bytes()),
		}})
	}
	return proofs
}

func (s *stubL1) HeaderByHash(_ context.Context, hash common.Hash) (*ethTypes.Header, error) {
	if hash != gameL1Head {
		return nil, fmt.Errorf("unknown header %v", hash)
	}
	return &ethTypes.Header{Number: new(big.Int).SetUint64(l1HeadNumber)}, nil
}

func (s *stubL1) FilterLogs(_ context.Context, query ethereum.FilterQuery) ([]ethTypes.Log, error) {
	s.queries = append(s.queries, query.FromBlock.Uint64())
	return s.logs, nil
}

type stubTxSender struct {
	sent []contracts.WithdrawalProof
	err  error
}

func (s *stubTxSender) SendAndWaitDetailed(_ string, txs ...txmgr.TxCandidate) []error {
	errs := make([]error, len(txs))
	for i, tx := range txs {
		s.sent = append(s.sent, contracts.WithdrawalProof{
			WithdrawalHash: common.BytesToHash(tx.TxData[:32]),
			ProofSubmitter: common.BytesToAddress(tx.TxData[32:]),
		})
		errs[i] = s.err
	}
	return errs
}

type stubMetrics struct {
	deleted int
}

func (s *stubMetrics) RecordWithdrawalDeleted() {
	s.deleted++
}
