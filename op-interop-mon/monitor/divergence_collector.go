package monitor

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// replicaRPCTimeout bounds each per-tick fan-out of replica RPCs so a single
// hung replica cannot block the collector goroutine indefinitely.
const replicaRPCTimeout = 10 * time.Second

// DivergenceMetrics is the metrics surface the divergence collector needs.
type DivergenceMetrics interface {
	RecordReplicaDivergence(diverged bool, comparedReplicas int, atTimestamp uint64)
}

// replicaSuperRoot is one replica's super root (and its dependency-set chain
// IDs) at the comparison timestamp.
type replicaSuperRoot struct {
	endpoint  string
	superRoot eth.Bytes32
	chainIDs  []eth.ChainID
}

// divergenceResult is the outcome of comparing replicas at one timestamp.
type divergenceResult struct {
	timestamp uint64
	compared  int
	diverged  bool
	// groups maps each distinct super root to the endpoints reporting it.
	// len(groups) > 1 means divergence.
	groups map[eth.Bytes32][]string
}

// ReplicaDivergenceCollector polls a set of supernode replicas and compares
// their super root at a common, fully-finalized timestamp. Two correct replicas
// of the same chains compute identical super roots (verification is a pure
// function of L2 data and canonical L1), so any disagreement is a determinism
// violation that the local halt guards may not catch — this is the cross-replica
// detector for exactly that case.
type ReplicaDivergenceCollector struct {
	clients         []ReplicaClient
	interval        time.Duration
	log             log.Logger
	m               DivergenceMetrics
	failsafeClients []FailsafeClient
	triggerFailsafe bool

	ctx    context.Context
	cancel context.CancelFunc
	closed chan struct{}
}

func NewReplicaDivergenceCollector(
	clients []ReplicaClient,
	interval time.Duration,
	log log.Logger,
	m DivergenceMetrics,
	failsafeClients []FailsafeClient,
	triggerFailsafe bool,
) *ReplicaDivergenceCollector {
	if interval <= 0 {
		interval = time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &ReplicaDivergenceCollector{
		clients:         clients,
		interval:        interval,
		log:             log,
		m:               m,
		failsafeClients: failsafeClients,
		triggerFailsafe: triggerFailsafe,
		ctx:             ctx,
		cancel:          cancel,
		closed:          make(chan struct{}),
	}
}

func (c *ReplicaDivergenceCollector) Start() error {
	c.log.Info("Starting replica divergence collector", "replicas", len(c.clients))
	go c.Run()
	return nil
}

func (c *ReplicaDivergenceCollector) Run() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-c.closed:
			return
		case <-ticker.C:
			c.collectOnce(c.ctx)
		}
	}
}

func (c *ReplicaDivergenceCollector) Stop() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
		c.cancel() // unblock any in-flight replica RPC
	}
	return nil
}

// collectOnce runs one comparison round.
func (c *ReplicaDivergenceCollector) collectOnce(parent context.Context) {
	// Fewer than two replicas: nothing to compare.
	if len(c.clients) < 2 {
		return
	}

	ctx, cancel := context.WithTimeout(parent, replicaRPCTimeout)
	defer cancel()

	// Phase 1: discover each replica's finalized frontier. The comparison point
	// is the minimum across replicas that have finalized something, so every
	// compared replica has already finalized (hence verified) it — a replica
	// merely lagging is not flagged as diverged. Replicas reporting 0 (not yet
	// finalized / booting) are excluded so they cannot drag the min to 0 and
	// blind the whole round.
	minFinalized, withData := c.minFinalizedTimestamp(ctx)
	if withData < 2 || minFinalized == 0 {
		return
	}

	// Phase 2: fetch each replica's super root at the common timestamp.
	roots := c.collectSuperRoots(ctx, minFinalized)
	if len(roots) < 2 {
		return // not enough comparable responses this round
	}

	// Replicas configured with different dependency sets legitimately compute
	// different super roots; that is a configuration difference, not a
	// consensus divergence, and must not trip the failsafe. Compare only when
	// every replica reports the same chain set.
	if !sameChainSets(roots) {
		c.log.Warn("supernode replicas report different dependency sets; skipping super-root comparison",
			"timestamp", minFinalized, "sets", formatChainSets(roots))
		return
	}

	result := compareSuperRoots(minFinalized, roots)
	c.m.RecordReplicaDivergence(result.diverged, result.compared, result.timestamp)
	if !result.diverged {
		return
	}

	c.log.Error("CROSS-REPLICA SUPER-ROOT DIVERGENCE DETECTED",
		"timestamp", result.timestamp,
		"distinctRoots", len(result.groups),
		"groups", formatGroups(result.groups))
	c.maybeTriggerFailsafe()
}

// minFinalizedTimestamp returns the minimum finalized L2 timestamp across
// replicas that have finalized something (timestamp > 0), and how many such
// replicas responded. Calls fan out concurrently and are bounded by ctx.
func (c *ReplicaDivergenceCollector) minFinalizedTimestamp(ctx context.Context) (uint64, int) {
	type res struct {
		ts  uint64
		err error
		ep  string
	}
	results := make([]res, len(c.clients))
	var wg sync.WaitGroup
	wg.Add(len(c.clients))
	for i, cl := range c.clients {
		go func(i int, cl ReplicaClient) {
			defer wg.Done()
			st, err := cl.SyncStatus(ctx)
			results[i] = res{ts: st.FinalizedTimestamp, err: err, ep: cl.Endpoint()}
		}(i, cl)
	}
	wg.Wait()

	var minTS uint64
	withData := 0
	for _, r := range results {
		if r.err != nil {
			c.log.Debug("replica sync status failed, skipping this round", "endpoint", r.ep, "err", r.err)
			continue
		}
		if r.ts == 0 {
			continue // not yet finalized — nothing to compare against
		}
		if withData == 0 || r.ts < minTS {
			minTS = r.ts
		}
		withData++
	}
	return minTS, withData
}

// collectSuperRoots fetches each replica's super root at ts concurrently.
// Replicas that error or report no data at ts (still catching up) are skipped,
// not flagged. Bounded by ctx.
func (c *ReplicaDivergenceCollector) collectSuperRoots(ctx context.Context, ts uint64) []replicaSuperRoot {
	out := make([]*replicaSuperRoot, len(c.clients))
	var wg sync.WaitGroup
	wg.Add(len(c.clients))
	for i, cl := range c.clients {
		go func(i int, cl ReplicaClient) {
			defer wg.Done()
			resp, err := cl.SuperRootAtTimestamp(ctx, ts)
			if err != nil {
				c.log.Debug("replica super root fetch failed, skipping", "endpoint", cl.Endpoint(), "ts", ts, "err", err)
				return
			}
			if resp.Data == nil {
				// Replica reports no verified data at ts despite ts <= min
				// finalized: a transient inconsistency, not a confirmed divergence.
				c.log.Debug("replica reports no data at finalized timestamp, skipping", "endpoint", cl.Endpoint(), "ts", ts)
				return
			}
			out[i] = &replicaSuperRoot{endpoint: cl.Endpoint(), superRoot: resp.Data.SuperRoot, chainIDs: resp.ChainIDs}
		}(i, cl)
	}
	wg.Wait()

	roots := make([]replicaSuperRoot, 0, len(c.clients))
	for _, r := range out {
		if r != nil {
			roots = append(roots, *r)
		}
	}
	return roots
}

func (c *ReplicaDivergenceCollector) maybeTriggerFailsafe() {
	if !c.triggerFailsafe {
		return
	}
	// Use a FRESH timeout, not the per-tick context: the diagnostic RPCs above
	// may have nearly exhausted that budget, and tripping the failsafe is the
	// most safety-critical action here — it must not be cancelled because
	// divergence detection was slow.
	ctx, cancel := context.WithTimeout(c.ctx, replicaRPCTimeout)
	defer cancel()
	for _, fc := range c.failsafeClients {
		if err := fc.SetFailsafeEnabled(ctx, true); err != nil {
			c.log.Error("failed to enable failsafe after replica divergence", "err", err)
		}
	}
}

// compareSuperRoots groups replicas by their reported super root. More than one
// distinct root means the replicas disagree. Pure function — unit-tested.
func compareSuperRoots(ts uint64, roots []replicaSuperRoot) divergenceResult {
	groups := make(map[eth.Bytes32][]string)
	for _, r := range roots {
		groups[r.superRoot] = append(groups[r.superRoot], r.endpoint)
	}
	return divergenceResult{
		timestamp: ts,
		compared:  len(roots),
		diverged:  len(groups) > 1,
		groups:    groups,
	}
}

// sameChainSets reports whether every replica's super root covers an identical
// dependency set (chain IDs). Order-independent.
func sameChainSets(roots []replicaSuperRoot) bool {
	if len(roots) < 2 {
		return true
	}
	want := chainSetKey(roots[0].chainIDs)
	for _, r := range roots[1:] {
		if chainSetKey(r.chainIDs) != want {
			return false
		}
	}
	return true
}

// chainSetKey is an order-independent key for a chain-ID set.
func chainSetKey(ids []eth.ChainID) string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = id.String()
	}
	sort.Strings(s)
	return strings.Join(s, ",")
}

func formatChainSets(roots []replicaSuperRoot) string {
	parts := make([]string, 0, len(roots))
	for _, r := range roots {
		parts = append(parts, r.endpoint+"=["+chainSetKey(r.chainIDs)+"]")
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

// formatGroups renders the divergence groups deterministically for logging,
// as "<root>=<ep1>,<ep2> <root2>=<ep3>" sorted by root then endpoint.
func formatGroups(groups map[eth.Bytes32][]string) string {
	parts := make([]string, 0, len(groups))
	for root, eps := range groups {
		sorted := append([]string(nil), eps...)
		sort.Strings(sorted)
		parts = append(parts, root.String()+"="+strings.Join(sorted, ","))
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}
