package interop

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/remote"
)

// defaultRemotePollInterval bounds how often a remote node is polled for its next
// finalized block when the adapter does not report a block time.
const defaultRemotePollInterval = 2 * time.Second

// defaultMaxBlocksPerCycle caps how many blocks a single ingest cycle drains before
// yielding to the ticker. One block per tick would only ever match the remote chain's
// production rate, so a node that falls behind (downtime, a slow adapter, a chain that
// finalizes in bursts) could never catch up. Draining lets it catch up; the cap keeps a
// cycle bounded so shutdown stays responsive and a slow adapter is not hammered
// indefinitely within one iteration.
const defaultMaxBlocksPerCycle = 256

// remoteNode is the finalized-only analogue of a driving virtual node. Instead of
// running a consensus client, it polls a remote.Adapter (in production, an HTTP client
// pointed at a remote endpoint) for finalized blocks and seals their initiating messages
// into the chain's LogsDB, so driven chains can reference them as executing messages.
type remoteNode struct {
	log     log.Logger
	adapter remote.Adapter
	db      LogsDB
	poll    time.Duration
	// maxPerCycle caps the number of blocks ingested by a single run-loop iteration.
	maxPerCycle int
}

// AddRemoteNode registers a remote chain represented by the given adapter. It opens a
// LogsDB for the adapter's chain (keyed by chain ID, the same map executing-message
// validation reads from) and prepares an ingester that Start launches. It must be called
// after New and before Start. The chain must not already be a driven chain.
func (i *Interop) AddRemoteNode(adapter remote.Adapter) error {
	chainID := adapter.ChainID()
	if _, ok := i.chains[chainID]; ok {
		return fmt.Errorf("chain %s is already a driven chain", chainID)
	}
	if _, ok := i.remoteNodes[chainID]; ok {
		return fmt.Errorf("remote node for chain %s already registered", chainID)
	}
	db, err := openLogsDB(i.log, chainID, i.dataDir)
	if err != nil {
		return fmt.Errorf("open logsDB for remote chain %s: %w", chainID, err)
	}
	poll := defaultRemotePollInterval
	if bt := adapter.BlockTime(); bt > 0 {
		poll = time.Duration(bt) * time.Second
	}
	i.logsDBs[chainID] = db
	i.remoteNodes[chainID] = &remoteNode{
		log:         i.log.New("remote_chain", chainID.String()),
		adapter:     adapter,
		db:          db,
		poll:        poll,
		maxPerCycle: defaultMaxBlocksPerCycle,
	}
	i.log.Info("registered remote node", "chain", chainID, "blockTime", adapter.BlockTime())
	return nil
}

// startRemoteNodes launches one background ingester per registered remote node. Each
// runs until i.ctx is cancelled (Stop), then exits; the LogsDBs they write to are closed
// by Stop's logsDBs cleanup.
func (i *Interop) startRemoteNodes() {
	for chainID, node := range i.remoteNodes {
		i.log.Info("starting remote node ingester", "chain", chainID)
		go node.run(i.ctx)
	}
}

// run drains the adapter on a ticker until the context is cancelled.
func (n *remoteNode) run(ctx context.Context) {
	ticker := time.NewTicker(n.poll)
	defer ticker.Stop()
	for {
		if err := n.ingestCycle(ctx); err != nil && ctx.Err() == nil {
			n.log.Warn("remote node ingest failed, will retry", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// ingestCycle drains finalized blocks from the adapter until it reports none left, the
// per-cycle cap is reached, the context is cancelled, or an error occurs. Draining is
// what lets a lagging node catch up: at one block per tick the ingester could only ever
// match the remote chain's production rate and would stay behind forever after any gap.
func (n *remoteNode) ingestCycle(ctx context.Context) error {
	maxPerCycle := n.maxPerCycle
	if maxPerCycle <= 0 {
		maxPerCycle = defaultMaxBlocksPerCycle
	}
	for i := 0; i < maxPerCycle; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		ingested, err := n.ingestOnce(ctx)
		if err != nil {
			return err
		}
		if !ingested {
			return nil
		}
	}
	n.log.Info("remote node hit per-cycle ingest cap, still catching up",
		"cap", maxPerCycle)
	return nil
}

// ingestOnce pulls the next finalized block from the adapter and seals it into the
// LogsDB. It returns (true, nil) if a block was ingested, (false, nil) if no new block
// is finalized yet, or (false, err) on failure. The resume cursor is the LogsDB's latest
// sealed block, so ingestion is idempotent across restarts.
//
// Anchoring: when the LogsDB is empty there is no history to be contiguous with, so the
// ingester accepts whatever height the adapter serves for after=0 and makes it the
// anchor of this chain's local history. This mirrors how a driven chain's logsDB starts
// at an arbitrary height (see processBlockLogs, and the --interop.log-backfill-depth
// window that decides where that history begins): the LogsDB records its first block
// number and answers ErrSkipped for anything below it, so a non-genesis-rooted history
// is a first-class state, not a special case. Without this, a remote chain whose head is
// at block N could only be ingested by replaying all N blocks from genesis. Once an
// anchor exists, strict contiguity is enforced exactly as before.
func (n *remoteNode) ingestOnce(ctx context.Context) (bool, error) {
	var after uint64
	latest, anchored := n.db.LatestSealedBlock()
	if anchored {
		after = latest.Number
	}
	blk, ok, err := n.adapter.NextFinalized(ctx, after)
	if err != nil {
		return false, fmt.Errorf("fetch next finalized after %d: %w", after, err)
	}
	if !ok {
		return false, nil
	}
	if anchored && blk.Number != after+1 {
		return false, fmt.Errorf("adapter returned non-contiguous block %d, expected %d", blk.Number, after+1)
	}
	if err := sealRemoteBlock(n.db, blk); err != nil {
		return false, fmt.Errorf("seal remote block %d: %w", blk.Number, err)
	}
	if !anchored {
		n.log.Info("anchored remote chain history at first served block",
			"number", blk.Number, "hash", blk.Hash, "timestamp", blk.Timestamp,
			"messages", len(blk.Messages))
		return true, nil
	}
	n.log.Debug("ingested finalized remote block",
		"number", blk.Number, "messages", len(blk.Messages), "timestamp", blk.Timestamp)
	return true, nil
}

// sealRemoteBlock writes a remote chain's finalized block into its LogsDB. It mirrors
// processBlockLogs but takes pre-extracted, normalized initiating messages instead of
// receipts, and records no executing messages (a remote chain contributes initiating
// messages only). Any block number >= 1 is accepted as the first seal (see ingestOnce's
// anchoring note); block 0 is the implicit genesis and can never carry logs.
func sealRemoteBlock(db LogsDB, blk *remote.FinalizedBlock) error {
	if blk.Number == 0 {
		return fmt.Errorf("remote block number must be >= 1 (block 0 is genesis)")
	}
	parentBlock := eth.BlockID{Hash: blk.ParentHash, Number: blk.Number - 1}
	for _, m := range blk.Messages {
		if err := db.AddLog(m.LogHash, parentBlock, m.LogIndex, nil); err != nil {
			return fmt.Errorf("add log %d: %w", m.LogIndex, err)
		}
	}
	if err := db.SealBlock(blk.ParentHash, eth.BlockID{Hash: blk.Hash, Number: blk.Number}, blk.Timestamp); err != nil {
		return fmt.Errorf("seal block: %w", err)
	}
	return nil
}

// initiatingBlockTime returns the block time of a chain that can originate initiating
// messages — either a driven chain or a registered remote node — and whether such a
// chain is known.
func (i *Interop) initiatingBlockTime(chainID eth.ChainID) (uint64, bool) {
	if c, ok := i.chains[chainID]; ok {
		return c.BlockTime(), true
	}
	if node, ok := i.remoteNodes[chainID]; ok {
		return node.adapter.BlockTime(), true
	}
	return 0, false
}
