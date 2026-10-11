package monitor

import (
	"cmp"
	"context"
	"errors"
	"math/big"
	"slices"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// ExpiryMetrics are the metrics the expiry checker records.
type ExpiryMetrics interface {
	RecordExpiryCheck(chainID string, check string, valid bool)
	RecordExpiryViolation(chainID string, check string, reason string)
	RecordExpiryLastVerifiedBlock(chainID string, blockNumber uint64)
	RecordETHInFlight(ether float64, timestamp uint64)
}

// clusterChainID labels the metrics of checks that span every chain.
const clusterChainID = "cluster"

// How often pending events are checked, and how often ETH conservation is.
var (
	expiryCheckInterval  = 1 * time.Second
	liquidityCheckPeriod = 12 * time.Second
)

// expiryEvent is a MessageExpired or RefundETH log found by a Finder, waiting to be checked.
type expiryEvent struct {
	chainID eth.ChainID
	check   string
	log     types.Log
}

// expiryObservation is what a Finder saw in one block.
type expiryObservation struct {
	chainID eth.ChainID
	block   uint64
	events  []expiryEvent
}

// ExpiryChecker verifies interop message expiries and SuperchainETHBridge refunds, and watches
// that the cluster never mints more ETH than it burned. It enforces the two safety rules of message
// expiry from chain data, without trusting the contracts' own checks: a destination never relays
// a message its source marked expired, and every ETH burn is matched by at most one mint.
//
// Finders hand it the MessageExpired and RefundETH events of each block they process. It checks
// an event once the event's block is at or below the configured head, refunds first, and records
// each verdict.
type ExpiryChecker struct {
	verifier *expiryVerifier
	tripwire *liquidityTripwire

	inbox  chan expiryObservation
	closed chan struct{}
	done   chan struct{}
	cancel context.CancelFunc

	// Owned by the run loop.
	pending  []expiryEvent
	observed map[eth.ChainID]uint64

	m   ExpiryMetrics
	log log.Logger
}

// NewExpiryChecker creates a checker that reads `clients` up to their `head` blocks, finalized or
// safe, so that a reorg cannot produce a false violation.
func NewExpiryChecker(clients map[eth.ChainID]ExpiryClient, head eth.BlockLabel, messageExpiryWindow uint64,
	liquidityBaselineBlocks map[eth.ChainID]uint64, m ExpiryMetrics, log log.Logger) *ExpiryChecker {
	return &ExpiryChecker{
		verifier: &expiryVerifier{clients: clients, messageExpiryWindow: messageExpiryWindow, head: head},
		tripwire: newLiquidityTripwire(clients, head, liquidityBaselineBlocks),
		inbox:    make(chan expiryObservation, inboxDepth),
		closed:   make(chan struct{}),
		done:     make(chan struct{}),
		observed: make(map[eth.ChainID]uint64),
		m:        m,
		log:      log.New("component", "expiry_checker"),
	}
}

// ObservingJobFilter wraps a Finder's JobFilter so the checker also sees every block's receipts.
func (c *ExpiryChecker) ObservingJobFilter(next JobFilter) JobFilter {
	return func(receipts []*types.Receipt, chainID eth.ChainID) []*Job {
		c.Observe(chainID, receipts)
		return next(receipts, chainID)
	}
}

// Observe queues the MessageExpired and RefundETH events of one block's receipts.
func (c *ExpiryChecker) Observe(chainID eth.ChainID, receipts []*types.Receipt) {
	if len(receipts) == 0 || receipts[0].BlockNumber == nil || c.Stopped() {
		return
	}
	obs := expiryObservation{chainID: chainID, block: bigs.Uint64Strict(receipts[0].BlockNumber)}
	for _, rcpt := range receipts {
		for _, l := range rcpt.Logs {
			if check := expiryCheckFor(l); check != "" {
				obs.events = append(obs.events, expiryEvent{chainID: chainID, check: check, log: *l})
			}
		}
	}
	select {
	case c.inbox <- obs:
	case <-c.closed:
	}
}

// expiryCheckFor returns the check a log needs, if any.
func expiryCheckFor(l *types.Log) string {
	if len(l.Topics) == 0 {
		return ""
	}
	switch {
	case l.Address == predeploys.L2toL2CrossDomainMessengerAddr && l.Topics[0] == messageExpiredTopic:
		return checkExpiry
	case l.Address == predeploys.SuperchainETHBridgeAddr && l.Topics[0] == refundETHTopic:
		return checkRefund
	}
	return ""
}

func (c *ExpiryChecker) Start(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)
	go c.Run(ctx)
	return nil
}

func (c *ExpiryChecker) Run(ctx context.Context) {
	defer close(c.done)
	checkTicker := time.NewTicker(expiryCheckInterval)
	defer checkTicker.Stop()
	liquidityTicker := time.NewTicker(liquidityCheckPeriod)
	defer liquidityTicker.Stop()
	defer c.log.Info("expiry checker closed")

	for {
		select {
		case <-c.closed:
			return
		case obs := <-c.inbox:
			c.observed[obs.chainID] = max(c.observed[obs.chainID], obs.block)
			c.pending = append(c.pending, obs.events...)
		case <-checkTicker.C:
			c.processEvents(ctx)
		case <-liquidityTicker.C:
			c.checkLiquidity(ctx)
		}
	}
}

// processEvents checks every pending event whose block is at or below the head, refunds first,
// keeping events that hit a transient error for the next round.
func (c *ExpiryChecker) processEvents(ctx context.Context) {
	heads := make(map[eth.ChainID]eth.BlockInfo)
	for id, client := range c.verifier.clients {
		head, err := client.InfoByLabel(ctx, c.verifier.head)
		if err != nil {
			c.log.Warn("failed to read head", "chain_id", id, "label", c.verifier.head, "err", err)
			continue
		}
		heads[id] = head
	}

	// Refunds release ETH, so they are checked before expiries.
	slices.SortStableFunc(c.pending, func(a, b expiryEvent) int {
		if a.check != b.check {
			if a.check == checkRefund {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.log.BlockNumber, b.log.BlockNumber)
	})

	remaining := c.pending[:0]
	for _, ev := range c.pending {
		if !c.processEvent(ctx, ev, heads[ev.chainID]) {
			remaining = append(remaining, ev)
		}
	}
	clear(c.pending[len(remaining):])
	c.pending = remaining

	c.recordLastVerified(heads)
}

// processEvent checks one event, and reports whether it is done with it.
func (c *ExpiryChecker) processEvent(ctx context.Context, ev expiryEvent, head eth.BlockInfo) bool {
	if head == nil || ev.log.BlockNumber > head.NumberU64() {
		return false
	}
	client := c.verifier.clients[ev.chainID]
	canonical, err := client.InfoByNumber(ctx, ev.log.BlockNumber)
	if err != nil {
		c.log.Warn("failed to read event block", "chain_id", ev.chainID, "block", ev.log.BlockNumber, "err", err)
		return false
	}
	if canonical.Hash() != ev.log.BlockHash {
		c.log.Info("dropping event from reorged block", "chain_id", ev.chainID, "check", ev.check,
			"block", ev.log.BlockNumber, "hash", ev.log.BlockHash, "canonical", canonical.Hash())
		return true
	}

	var violations []violation
	switch ev.check {
	case checkExpiry:
		violations, err = c.verifier.verifyExpiry(ctx, ev.chainID, &ev.log)
	case checkRefund:
		violations, err = c.verifier.verifyRefund(ctx, ev.chainID, &ev.log)
	}
	if err != nil {
		level := c.log.Warn
		if errors.Is(err, errNotReady) {
			level = c.log.Debug
		}
		level("check not finished, will retry", "chain_id", ev.chainID, "check", ev.check, "tx", ev.log.TxHash, "err", err)
		return false
	}

	c.recordVerdict(ev.chainID.String(), ev.check, violations, "tx", ev.log.TxHash, "block", ev.log.BlockNumber)
	if len(violations) == 0 {
		c.log.Info("verified", "chain_id", ev.chainID, "check", ev.check, "tx", ev.log.TxHash, "block", ev.log.BlockNumber)
	}
	return true
}

// recordLastVerified records, per chain, the highest block up to which every event the Finder
// passed on has been checked.
func (c *ExpiryChecker) recordLastVerified(heads map[eth.ChainID]eth.BlockInfo) {
	for id, observed := range c.observed {
		head, ok := heads[id]
		if !ok {
			continue
		}
		last := min(observed, head.NumberU64())
		for _, ev := range c.pending {
			if ev.chainID == id && ev.log.BlockNumber <= last {
				last = ev.log.BlockNumber - 1
			}
		}
		c.m.RecordExpiryLastVerifiedBlock(id.String(), last)
	}
}

func (c *ExpiryChecker) checkLiquidity(ctx context.Context) {
	reading, err := c.tripwire.check(ctx)
	if err != nil {
		c.log.Warn("liquidity check not finished, will retry", "err", err)
	}
	if reading == nil {
		return
	}
	for id, violations := range reading.violations {
		c.recordVerdict(id.String(), checkConservation, violations, "timestamp", reading.timestamp)
	}
	if reading.inFlight == nil {
		return
	}
	inFlight, _ := new(big.Float).Quo(new(big.Float).SetInt(reading.inFlight), big.NewFloat(params.Ether)).Float64()
	c.m.RecordETHInFlight(inFlight, reading.timestamp)
	var cluster []violation
	if reading.inFlight.Sign() < 0 {
		cluster = append(cluster, violationf(reasonNegativeInFlight, "the cluster relayed and refunded %s wei more than it sent", new(big.Int).Neg(reading.inFlight)))
	}
	c.recordVerdict(clusterChainID, checkConservation, cluster, "timestamp", reading.timestamp)
}

// recordVerdict logs and counts the outcome of one finished check.
func (c *ExpiryChecker) recordVerdict(chainID string, check string, violations []violation, ctx ...any) {
	c.m.RecordExpiryCheck(chainID, check, len(violations) == 0)
	for _, v := range violations {
		c.log.Error("interop expiry safety violation", append([]any{"chain_id", chainID, "check", check,
			"reason", v.reason, "detail", v.detail}, ctx...)...)
		c.m.RecordExpiryViolation(chainID, check, v.reason)
	}
}

// Stop cancels the checks in flight and waits for the checker to exit.
func (c *ExpiryChecker) Stop() error {
	close(c.closed)
	if c.cancel != nil {
		c.cancel()
		<-c.done
	}
	return nil
}

func (c *ExpiryChecker) Stopped() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}
