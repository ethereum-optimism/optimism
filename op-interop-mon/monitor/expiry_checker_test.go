package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type fakeExpiryMetrics struct {
	checks       map[string]int
	violations   map[string]int
	lastVerified map[string]uint64
	inFlight     float64
}

func newFakeExpiryMetrics() *fakeExpiryMetrics {
	return &fakeExpiryMetrics{checks: map[string]int{}, violations: map[string]int{}, lastVerified: map[string]uint64{}}
}

func (m *fakeExpiryMetrics) RecordExpiryCheck(chainID string, check string, valid bool) {
	m.checks[fmt.Sprintf("%s/%s/%v", chainID, check, valid)]++
}

func (m *fakeExpiryMetrics) RecordExpiryViolation(chainID string, check string, reason string) {
	m.violations[fmt.Sprintf("%s/%s/%s", chainID, check, reason)]++
}

func (m *fakeExpiryMetrics) RecordExpiryLastVerifiedBlock(chainID string, blockNumber uint64) {
	m.lastVerified[chainID] = blockNumber
}

func (m *fakeExpiryMetrics) RecordETHInFlight(ether float64, _ uint64) { m.inFlight = ether }

func newScenarioChecker(t *testing.T, s *expiryScenario) (*ExpiryChecker, *fakeExpiryMetrics) {
	m := newFakeExpiryMetrics()
	c := NewExpiryChecker(map[eth.ChainID]ExpiryClient{s.a.id: s.a, s.b.id: s.b}, eth.Finalized, scenarioWindow, nil, m,
		testlog.Logger(t, slog.LevelInfo))
	return c, m
}

// observeAll hands the checker every block of both chains, as the Finders would.
func observeAll(c *ExpiryChecker, s *expiryScenario) {
	for _, chain := range []*fakeChain{s.a, s.b} {
		for n := range chain.headers {
			receipts := chain.receipts[uint64(n)]
			if len(receipts) == 0 {
				// Every L2 block has at least its L1 attributes deposit.
				receipts = []*types.Receipt{{BlockNumber: big.NewInt(int64(n)), BlockHash: chain.hashOf(uint64(n))}}
			}
			c.Observe(chain.id, receipts)
			obs := <-c.inbox
			c.observed[obs.chainID] = max(c.observed[obs.chainID], obs.block)
			c.pending = append(c.pending, obs.events...)
		}
	}
}

func TestExpiryCheckerVerifiesFlow(t *testing.T) {
	s := newExpiryScenario()
	c, m := newScenarioChecker(t, s)
	observeAll(c, s)
	require.Len(t, c.pending, 2, "the expiry and the refund are queued")

	c.processEvents(context.Background())
	require.Empty(t, c.pending)
	require.Equal(t, map[string]int{"901/expiry/true": 1, "901/refund/true": 1}, m.checks)
	require.Empty(t, m.violations)
	require.Equal(t, uint64(len(s.a.headers)-1), m.lastVerified["901"])
	require.Equal(t, uint64(len(s.b.headers)-1), m.lastVerified["902"])

	c.checkLiquidity(context.Background())
	require.Equal(t, 1, m.checks[clusterChainID+"/conservation/true"])
	require.Equal(t, 1, m.checks["901/conservation/true"])
	require.Equal(t, 1, m.checks["902/conservation/true"])
	require.Zero(t, m.inFlight)
}

func TestExpiryCheckerRecordsViolations(t *testing.T) {
	s := newExpiryScenario()
	s.a.storage[[2]common.Hash{common.BytesToHash(predeploys.L2toL2CrossDomainMessengerAddr.Bytes()),
		mappingSlot(s.hash, expiredMessagesSlot)}] = nil
	s.b.removeLogs(undeliveredExpTopic)
	c, m := newScenarioChecker(t, s)
	observeAll(c, s)

	c.processEvents(context.Background())
	require.Equal(t, map[string]int{"901/expiry/false": 1, "901/refund/false": 1}, m.checks)
	require.Equal(t, map[string]int{
		"901/expiry/" + reasonExpiredFlagUnset:    1,
		"901/expiry/" + reasonNoExport:            1,
		"901/refund/" + reasonRefundWithoutExpiry: 1,
	}, m.violations)
}

func TestExpiryCheckerWaitsForSafeBlocks(t *testing.T) {
	s := newExpiryScenario()
	c, m := newScenarioChecker(t, s)
	observeAll(c, s)
	s.a.safe = s.refundBlock - 1

	c.processEvents(context.Background())
	require.Equal(t, map[string]int{"901/expiry/true": 1}, m.checks, "the refund is not safe yet")
	require.Len(t, c.pending, 1)
	require.Equal(t, s.refundBlock-1, m.lastVerified["901"])

	s.a.safe = uint64(len(s.a.headers) - 1)
	c.processEvents(context.Background())
	require.Equal(t, 1, m.checks["901/refund/true"])
	require.Empty(t, c.pending)
}

func TestExpiryCheckerDropsReorgedEvents(t *testing.T) {
	s := newExpiryScenario()
	c, m := newScenarioChecker(t, s)
	observeAll(c, s)
	// Replace A's refund block, as a reorg would.
	s.a.headers[s.refundBlock].Extra = []byte("reorged")

	c.processEvents(context.Background())
	require.Empty(t, c.pending)
	require.Equal(t, map[string]int{"901/expiry/true": 1}, m.checks)
}

func TestExpiryCheckerChecksRefundsFirst(t *testing.T) {
	s := newExpiryScenario()
	c, _ := newScenarioChecker(t, s)
	observeAll(c, s)
	// Hold every check back, so the queue order is visible.
	s.a.safe = s.expireBlock - 1
	c.processEvents(context.Background())
	require.Len(t, c.pending, 2)
	require.Equal(t, checkRefund, c.pending[0].check)
	require.Equal(t, checkExpiry, c.pending[len(c.pending)-1].check)
}
