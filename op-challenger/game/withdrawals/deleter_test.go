package withdrawals

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
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
	proofTopic    = common.Hash{0x99}
	gameL1Head    = uint64(100)
)

func TestDeleter_DeletesProofsAgainstChallengerWinGame(t *testing.T) {
	deleter, portal, l1, sender, m := setupDeleterTest(t)
	proofs := l1.proofs(3)
	portal.record(proofs[0], gameAddr, 1)
	portal.record(proofs[1], gameAddr, 2)
	portal.record(proofs[2], gameAddr, 3)

	done, err := deleter.DeleteInvalidatedWithdrawals(context.Background(), gameAddr, gameL1Head, 200)
	require.NoError(t, err)
	require.True(t, done)

	require.Equal(t, proofs, sender.sent)
	require.Equal(t, 3, m.deleted)
	require.Equal(t, []rpcblock.Block{rpcblock.Latest}, portal.reads,
		"should read at latest to match the game status the caller acted on")
}

func TestDeleter_IgnoresProofsAgainstOtherGames(t *testing.T) {
	deleter, portal, l1, sender, _ := setupDeleterTest(t)
	proofs := l1.proofs(2)
	portal.record(proofs[0], otherGameAddr, 1)
	portal.record(proofs[1], gameAddr, 2)

	done, err := deleter.DeleteInvalidatedWithdrawals(context.Background(), gameAddr, gameL1Head, 200)
	require.NoError(t, err)
	require.True(t, done)

	require.Equal(t, proofs[1:], sender.sent)
}

func TestDeleter_SkipsAlreadyDeletedProofs(t *testing.T) {
	deleter, portal, l1, sender, m := setupDeleterTest(t)
	proofs := l1.proofs(2)
	portal.record(proofs[0], gameAddr, 0)
	portal.record(proofs[1], gameAddr, 0)

	done, err := deleter.DeleteInvalidatedWithdrawals(context.Background(), gameAddr, gameL1Head, 200)
	require.NoError(t, err)
	require.True(t, done)

	require.Empty(t, sender.sent)
	require.Zero(t, m.deleted)
}

func TestDeleter_TruncatesAtCapAndRescansUntilComplete(t *testing.T) {
	deleter, portal, l1, sender, _ := setupDeleterTest(t)
	logger, logs := testlog.CaptureLogger(t, log.LevelWarn)
	deleter.log = logger
	proofs := l1.proofs(maxDeletesPerGame + 5)
	for i, proof := range proofs {
		portal.record(proof, gameAddr, uint64(i+1))
	}

	done, err := deleter.DeleteInvalidatedWithdrawals(context.Background(), gameAddr, gameL1Head, 200)
	require.NoError(t, err)
	require.False(t, done, "should report the remaining proofs as outstanding")
	require.Equal(t, proofs[:maxDeletesPerGame], sender.sent)
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(log.LevelWarn),
		testlog.NewMessageFilter("Deferring some withdrawal proof deletions to a later run")))

	// The remainder is picked up when the same range is scanned again.
	for _, proof := range proofs[:maxDeletesPerGame] {
		portal.record(proof, gameAddr, 0)
	}
	sender.sent = nil
	done, err = deleter.DeleteInvalidatedWithdrawals(context.Background(), gameAddr, gameL1Head, 200)
	require.NoError(t, err)
	require.True(t, done)
	require.Equal(t, proofs[maxDeletesPerGame:], sender.sent)
	require.Equal(t, []uint64{gameL1Head, gameL1Head}, l1.queries)
}

func TestDeleter_ScansOnlyTheRequestedRange(t *testing.T) {
	deleter, _, l1, _, _ := setupDeleterTest(t)

	_, err := deleter.DeleteInvalidatedWithdrawals(context.Background(), gameAddr, 201, 300)
	require.NoError(t, err)

	require.Equal(t, []uint64{201}, l1.queries)
	require.Equal(t, []uint64{300}, l1.toQueries)
}

func TestDeleter_RescansAfterL1Reorg(t *testing.T) {
	deleter, _, l1, _, _ := setupDeleterTest(t)

	_, err := deleter.DeleteInvalidatedWithdrawals(context.Background(), gameAddr, 201, 150)
	require.NoError(t, err)

	require.Equal(t, []uint64{150}, l1.queries, "should rescan from the new head")
}

func TestDeleter_ReportsSendFailures(t *testing.T) {
	deleter, portal, l1, sender, m := setupDeleterTest(t)
	proofs := l1.proofs(1)
	portal.record(proofs[0], gameAddr, 1)
	sender.err = errors.New("boom")

	done, err := deleter.DeleteInvalidatedWithdrawals(context.Background(), gameAddr, gameL1Head, 200)
	require.ErrorIs(t, err, sender.err)
	require.False(t, done)
	require.Zero(t, m.deleted)
	require.Equal(t, 1, m.failed)
}

func setupDeleterTest(t *testing.T) (*Deleter, *stubPortal, *stubL1, *stubTxSender, *stubMetrics) {
	portal := &stubPortal{records: make(map[contracts.WithdrawalProof]contracts.ProvenWithdrawal)}
	l1 := &stubL1{}
	sender := &stubTxSender{}
	m := &stubMetrics{}
	return NewDeleter(testlog.Logger(t, log.LevelDebug), m, portal, l1, sender), portal, l1, sender, m
}

type stubPortal struct {
	records map[contracts.WithdrawalProof]contracts.ProvenWithdrawal
	reads   []rpcblock.Block
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

func (s *stubPortal) GetProvenWithdrawals(_ context.Context, block rpcblock.Block, proofs []contracts.WithdrawalProof) ([]contracts.ProvenWithdrawal, error) {
	s.reads = append(s.reads, block)
	records := make([]contracts.ProvenWithdrawal, len(proofs))
	for i, proof := range proofs {
		records[i] = s.records[proof]
	}
	return records, nil
}

func (s *stubPortal) DeleteProvenWithdrawalTx(proof contracts.WithdrawalProof) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{TxData: append(proof.WithdrawalHash.Bytes(), proof.ProofSubmitter.Bytes()...)}, nil
}

type stubL1 struct {
	logs      []ethTypes.Log
	queries   []uint64
	toQueries []uint64
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

func (s *stubL1) FilterLogs(_ context.Context, query ethereum.FilterQuery) ([]ethTypes.Log, error) {
	s.queries = append(s.queries, bigs.Uint64Strict(query.FromBlock))
	s.toQueries = append(s.toQueries, bigs.Uint64Strict(query.ToBlock))
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
	failed  int
}

func (s *stubMetrics) RecordWithdrawalDeleted() {
	s.deleted++
}

func (s *stubMetrics) RecordWithdrawalDeletionFailed() {
	s.failed++
}
