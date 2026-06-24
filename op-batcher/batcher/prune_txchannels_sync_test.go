package batcher

import (
	"io"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derivetest "github.com/ethereum-optimism/optimism/op-node/rollup/derive/test"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/queue"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

// pruneTestManager builds a channelManager configured so that a single tx-bearing
// block produces several calldata txs (1 frame per tx). This lets us register many
// tx IDs into txChannels through the real TxData pipeline.
func pruneTestManager(t *testing.T, l log.Logger) *channelManager {
	cfg := channelManagerTestConfig(derive.FrameV0OverHeadSize+1, derive.SingularBatchType)
	cfg.ChannelTimeout = 2
	cfg.InitRatioCompressor(1, derive.Zlib)
	m := NewChannelManager(l, metrics.NoopMetrics, cfg, defaultTestRollupConfig)
	m.Clear(eth.BlockID{})
	return m
}

// buildOneChannel loads one real tx-bearing block (chained from parent) and drives the
// real TxData pipeline until drained, returning the newly created channel and the tx IDs
// registered into txChannels.
func buildOneChannel(t *testing.T, m *channelManager, seed int64, txCount int, parent *types.Block) (*channel, []txID, *types.Block) {
	rng := rand.New(rand.NewSource(seed))
	blk := derivetest.RandomL2BlockWithChainId(rng, txCount, defaultTestRollupConfig.L2ChainID)
	h := blk.Header()
	if parent != nil {
		h.Number = new(big.Int).Add(parent.Number(), big.NewInt(1))
		h.ParentHash = parent.Hash()
		h.Time = parent.Time() + 2
	}
	blk = types.NewBlock(h, blk.Body(), nil, trie.NewStackTrie(nil), types.DefaultBlockConfig)

	require.NoError(t, m.AddL2Block(blk))

	before := len(m.channelQueue)
	var ids []txID
	for {
		tx, err := m.TxData(eth.BlockID{}, false, pubInfo{})
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if tx.Len() > 0 {
			ids = append(ids, tx.ID())
		}
	}
	require.GreaterOrEqual(t, len(m.channelQueue), before)
	ch := m.channelQueue[len(m.channelQueue)-1]
	return ch, ids, blk
}

// assertNoOrphans asserts the coupled-state invariant: every txChannels entry maps to a
// channel still present in channelQueue (or the currentChannel).
func assertNoOrphans(t *testing.T, m *channelManager, ctx string) {
	t.Helper()
	present := make(map[*channel]bool, len(m.channelQueue)+1)
	for _, ch := range m.channelQueue {
		present[ch] = true
	}
	if m.currentChannel != nil {
		present[m.currentChannel] = true
	}
	for id, ch := range m.txChannels {
		require.Truef(t, present[ch],
			"%s: orphaned txChannels entry %s -> channel not in channelQueue/currentChannel", ctx, id)
	}
}


// TestPruneKeepsTxChannelsCoupled builds several channels, registers their txs, then
// PruneChannels(k) for various k and asserts no orphaned txChannels entries remain.
// Before the fix this failed; with the fix it holds for every k.
func TestPruneKeepsTxChannelsCoupled(t *testing.T) {
	for _, k := range []int{0, 1, 2, 3} {
		t.Run("prune"+string(rune('0'+k)), func(t *testing.T) {
			l := testlog.Logger(t, log.LevelCrit)
			m := pruneTestManager(t, l)

			var parent *types.Block
			var channels []*channel
			allIDs := map[string]*channel{}
			for i := 0; i < 3; i++ {
				ch, ids, blk := buildOneChannel(t, m, int64(1000+i), 30, parent)
				parent = blk
				channels = append(channels, ch)
				for _, id := range ids {
					allIDs[id.String()] = ch
				}
			}
			require.Len(t, m.channelQueue, 3)
			for id, ch := range allIDs {
				require.Same(t, ch, m.txChannels[id])
			}

			m.PruneChannels(k)

			require.Len(t, m.channelQueue, 3-k)
			assertNoOrphans(t, m, "after PruneChannels")

			prunedSet := map[*channel]bool{}
			for i := 0; i < k; i++ {
				prunedSet[channels[i]] = true
			}
			for id, ch := range m.txChannels {
				require.Falsef(t, prunedSet[ch], "tx %s still maps to a pruned channel", id)
			}
		})
	}
}


// TestPrunedLateConfirmNoPanic reproduces the validated crash scenario for a channel
// that is FULLY pruned (all its blocks safe): real computeSyncActions selects
// channelsToPrune>=1 (clearState==nil); we apply the prune exactly as syncAndPrune's
// non-clear branch; then a LATE confirmation through the REAL TxConfirmed must NOT panic
// and must be safely ignored (channel already pruned => unknown tx).
func TestPrunedLateConfirmNoPanic(t *testing.T) {
	l := testlog.Logger(t, log.LevelCrit)
	m := pruneTestManager(t, l)

	rng := rand.New(rand.NewSource(1234))
	blk := derivetest.RandomL2BlockWithChainId(rng, 40, defaultTestRollupConfig.L2ChainID)
	h := blk.Header()
	h.Number = big.NewInt(101)
	blk = types.NewBlock(h, blk.Body(), nil, trie.NewStackTrie(nil), types.DefaultBlockConfig)
	require.NoError(t, m.AddL2Block(blk))

	var txIDs []txID
	for {
		tx, err := m.TxData(eth.BlockID{}, false, pubInfo{})
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if tx.Len() > 0 {
			txIDs = append(txIDs, tx.ID())
		}
	}
	require.GreaterOrEqual(t, len(txIDs), 2)
	require.Len(t, m.channelQueue, 1)
	C := m.channelQueue[0]
	require.Equal(t, uint64(101), C.LatestL2().Number)

	m.TxConfirmed(txIDs[0], eth.BlockID{Number: 100})
	require.False(t, C.isTimedOut())

	syncStatus := eth.SyncStatus{
		HeadL1:      eth.BlockRef{Number: 5},
		CurrentL1:   eth.BlockRef{Number: 2},
		LocalSafeL2: eth.L2BlockRef{Number: 101, Hash: blk.Hash()},
		UnsafeL2:    eth.L2BlockRef{Number: 109},
	}
	actions, outOfSync := computeSyncActions(syncStatus, eth.BlockRef{Number: 1}, m.blocks, m.channelQueue, l)
	t.Logf("driver actions: %s outOfSync=%v", actions.TerminalString(), outOfSync)
	require.False(t, outOfSync)
	require.Nil(t, actions.clearState)
	require.GreaterOrEqual(t, actions.channelsToPrune, 1)

	m.PruneSafeBlocks(actions.blocksToPrune)
	m.PruneChannels(actions.channelsToPrune)

	require.Zero(t, m.blocks.Len())
	require.Empty(t, m.channelQueue)
	assertNoOrphans(t, m, "after driver-dictated prune")

	lateID := txIDs[len(txIDs)-1]
	require.NotContains(t, m.txChannels, lateID.String(), "fix: pruned channel's tx removed from txChannels")

	require.NotPanics(t, func() {
		m.TxConfirmed(lateID, eth.BlockID{Number: 100 + C.cfg.ChannelTimeout + 5})
	}, "late confirmation of a fully-pruned channel must not crash the batcher")

	assertNoOrphans(t, m, "after late confirmation")
	require.Empty(t, m.channelQueue)
	require.Zero(t, m.blocks.Len())
}


// Each round drives the REAL computeSyncActions to decide how many blocks/channels to
// prune (so we never fabricate states the driver can't produce), applies them exactly as
// syncAndPrune's non-clear branch, then confirms a random subset of txs (including
// pruned-channel txs) at random inclusion blocks (some crossing ChannelTimeout). Asserts
// (a) no panic ever and (b) the no-orphan invariant after each prune. Fails loudly with
// the seed on any violation.
func TestPruneAdversarialFuzz(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fuzz in short mode")
	}
	// Time-bounded: run as many randomized rounds as fit in the budget. Each round drives
	// the real compression + sync-action pipeline, so rounds are not free.
	budget := 8 * time.Second
	deadline := time.Now().Add(budget)
	rounds := 0
	stats := fuzzStats{}
	for round := 0; time.Now().Before(deadline); round++ {
		runDriverConsistentFuzzRound(t, int64(0xBADC0DE^round), &stats)
		rounds++
	}
	// The fuzz is only meaningful if it actually exercises the dangerous paths: pruning a
	// boundary-spanning channel's leading block AND timing such a channel out (which
	// reaches handleChannelInvalidated -> rewindToBlock, the clamp path the fix added).
	require.Positive(t, stats.boundarySpanningPrunes, "fuzz must exercise boundary-spanning prunes")
	require.Positive(t, stats.boundaryTimeoutsAfterPrune, "fuzz must time out a boundary-spanning channel (reach the rewindToBlock clamp)")
	t.Logf("adversarial fuzz completed: %d rounds in ~%s, 0 panics, invariant held every round; "+
		"boundary-spanning prunes=%d, boundary-spanning timeouts reaching rewindToBlock=%d, total channel timeouts=%d",
		rounds, budget, stats.boundarySpanningPrunes, stats.boundaryTimeoutsAfterPrune, stats.totalTimeouts)
}

type fuzzStats struct {
	boundarySpanningPrunes     int
	boundaryTimeoutsAfterPrune int
	totalTimeouts              int
}

// fuzzChannelManager uses a small frame size + no compressor with multi-tx mini-blocks so
// that channels are BOTH multi-block (a randomly chosen safe head can land mid-channel)
// AND multi-tx (so confirmations can be spread to actually time a channel out, reaching
// handleChannelInvalidated -> rewindToBlock).
func fuzzChannelManager(t *testing.T, l log.Logger) *channelManager {
	cfg := channelManagerTestConfig(400, derive.SingularBatchType)
	cfg.ChannelTimeout = 2
	cfg.MaxChannelDuration = 0
	cfg.InitNoneCompressor()
	m := NewChannelManager(l, metrics.NoopMetrics, cfg, defaultTestRollupConfig)
	m.Clear(eth.BlockID{})
	return m
}

// runDriverConsistentFuzzRound builds several MULTI-block, MULTI-tx channels, lets the
// REAL computeSyncActions decide the prune (with a safe head that can land mid-channel),
// applies it, then confirms txs in a way that deliberately spreads inclusion blocks so
// channels (including boundary-spanning ones whose leading block was pruned) time out and
// reach handleChannelInvalidated -> rewindToBlock. Asserts no panic (deferred recover)
// and the no-orphan invariant after every mutation.
func runDriverConsistentFuzzRound(t *testing.T, seed int64, stats *fuzzStats) {
	rng := rand.New(rand.NewSource(seed))
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("PANIC in driver-consistent fuzz round seed=%d: %v", seed, r)
		}
	}()

	l := testlog.Logger(t, log.LevelCrit)
	m := fuzzChannelManager(t, l)

	numChannels := 1 + rng.Intn(4)
	type chanRec struct {
		ch        *channel
		ids       []txID
		firstNum  uint64
		latestNum uint64
	}
	var recs []chanRec
	var parent *types.Block
	nextNum := int64(101)
	for i := 0; i < numChannels; i++ {
		blocksInChannel := 1 + rng.Intn(4) // 1..4 blocks per channel -> can straddle a safe head
		ch, ids, blk, first, latest := buildMultiBlockChannel(t, m, seed*131+int64(i), blocksInChannel, parent, nextNum, rng)
		parent = blk
		nextNum = int64(latest) + 1
		recs = append(recs, chanRec{ch: ch, ids: ids, firstNum: first, latestNum: latest})
	}

	// Optionally confirm an early tx of some channels before pruning (sets a LOW
	// minInclusionBlock so a later high confirmation crosses ChannelTimeout).
	for _, rec := range recs {
		if len(rec.ids) >= 2 && rng.Intn(2) == 0 {
			m.TxConfirmed(rec.ids[0], eth.BlockID{Number: 100})
		}
	}
	assertNoOrphansSeed(t, m, "fuzz: after pre-prune confirms", seed)

	// Snapshot the queue BEFORE pruning to classify channels as pruned vs kept.
	queueBefore := append([]*channel(nil), m.channelQueue...)

	boundarySpanning := map[*channel]bool{}

	// Pick a random safe head across the block range — explicitly including mid-channel
	// positions — and let the REAL driver decide the actions.
	if m.blocks.Len() > 0 {
		newestNum := m.blocks[m.blocks.Len()-1].NumberU64()
		oldestNum := m.blocks[0].NumberU64()
		safeNum := oldestNum - 1 + uint64(rng.Intn(int(newestNum-oldestNum+2))) // [oldest-1 .. newest]
		var safeHash [32]byte
		for i := 0; i < m.blocks.Len(); i++ {
			if m.blocks[i].NumberU64() == safeNum {
				safeHash = m.blocks[i].Hash()
				break
			}
		}
		ss := eth.SyncStatus{
			HeadL1:      eth.BlockRef{Number: 5},
			CurrentL1:   eth.BlockRef{Number: 2},
			LocalSafeL2: eth.L2BlockRef{Number: safeNum, Hash: safeHash},
			UnsafeL2:    eth.L2BlockRef{Number: newestNum + 5},
		}
		actions, oos := computeSyncActions(ss, eth.BlockRef{Number: 1}, m.blocks, m.channelQueue, l)
		if !oos && actions.clearState == nil {
			// Classify boundary-spanning: kept channel (not among pruned) whose first block
			// is <= safeNum (leading block pruned) but latest block > safeNum.
			keptSet := map[*channel]bool{}
			for i := actions.channelsToPrune; i < len(queueBefore); i++ {
				keptSet[queueBefore[i]] = true
			}
			for _, rec := range recs {
				if keptSet[rec.ch] && rec.firstNum <= safeNum && rec.latestNum > safeNum {
					boundarySpanning[rec.ch] = true
					stats.boundarySpanningPrunes++
				}
			}

			m.PruneSafeBlocks(actions.blocksToPrune)
			m.PruneChannels(actions.channelsToPrune)
			assertNoOrphansSeed(t, m, "fuzz: after driver-consistent prune", seed)
		}
	}

	// Confirm txs in a way that deliberately TIMES CHANNELS OUT: confirm the LAST tx of a
	// channel at a HIGH inclusion block. Combined with the low pre-prune confirm (or
	// another low confirm here), the inclusion spread crosses ChannelTimeout, so the
	// channel times out -> handleChannelInvalidated -> rewindToBlock. For boundary-spanning
	// channels this is exactly the clamp path. None of this may panic.
	for _, rec := range recs {
		// Ensure a low anchor exists for channels we will time out.
		if len(rec.ids) >= 2 {
			// Confirm a low one if it is still tracked & pending (no-op if already gone).
			m.TxConfirmed(rec.ids[0], eth.BlockID{Number: 100})
			assertNoOrphansSeed(t, m, "fuzz: after low confirm", seed)
		}
		// Late high confirmation to cross ChannelTimeout.
		wasBoundary := boundarySpanning[rec.ch]
		wasTimedOutBefore := rec.ch.isTimedOut()
		for _, id := range rec.ids {
			inc := eth.BlockID{Number: 100 + uint64(2+rng.Intn(8))} // >= ChannelTimeout above the low anchor
			m.TxConfirmed(id, inc)
			assertNoOrphansSeed(t, m, "fuzz: after high confirm", seed)
		}
		if !wasTimedOutBefore && rec.ch.isTimedOut() {
			stats.totalTimeouts++
			if wasBoundary {
				stats.boundaryTimeoutsAfterPrune++
			}
		}
	}
}

// buildMultiBlockChannel adds `numBlocks` chained multi-tx mini-blocks (starting at
// startNum) and force-publishes them so they pack into a single channel that produces
// several calldata txs. Returns the channel, its tx IDs, the last block, and the
// channel's first and latest block numbers.
func buildMultiBlockChannel(t *testing.T, m *channelManager, seed int64, numBlocks int, parent *types.Block, startNum int64, rng *rand.Rand) (*channel, []txID, *types.Block, uint64, uint64) {
	before := len(m.channelQueue)
	if parent == nil {
		parent = types.NewBlockWithHeader(&types.Header{Number: big.NewInt(startNum - 1)})
	}
	var last *types.Block
	for j := 0; j < numBlocks; j++ {
		numTx := 2 + rng.Intn(3)
		blk := newMiniL2BlockWithNumberParent(numTx, big.NewInt(startNum+int64(j)), parent.Hash())
		require.NoError(t, m.AddL2Block(blk))
		parent = blk
		last = blk
	}

	var ids []txID
	for {
		tx, err := m.TxData(eth.BlockID{Number: 50}, false, pubInfo{forcePublish: true})
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if tx.Len() > 0 {
			ids = append(ids, tx.ID())
		}
	}
	require.Greater(t, len(m.channelQueue), before, "expected a new channel")
	ch := m.channelQueue[len(m.channelQueue)-1]
	first := ch.ChannelBuilder.blocks[0].NumberU64()
	latest := ch.LatestL2().Number
	return ch, ids, last, first, latest
}

func assertNoOrphansSeed(t *testing.T, m *channelManager, ctx string, seed int64) {
	t.Helper()
	present := make(map[*channel]bool, len(m.channelQueue)+1)
	for _, ch := range m.channelQueue {
		present[ch] = true
	}
	if m.currentChannel != nil {
		present[m.currentChannel] = true
	}
	for id, ch := range m.txChannels {
		if !present[ch] {
			t.Fatalf("%s (seed=%d): orphaned txChannels entry %s -> pruned channel", ctx, seed, id)
		}
	}
}



func TestPruneDecisionUnchanged(t *testing.T) {
	l := testlog.Logger(t, log.LevelCrit)

	block101 := SizedBlock{Block: types.NewBlockWithHeader(&types.Header{Number: big.NewInt(101)})}
	block102 := SizedBlock{Block: types.NewBlockWithHeader(&types.Header{Number: big.NewInt(102)})}
	block103 := SizedBlock{Block: types.NewBlockWithHeader(&types.Header{Number: big.NewInt(103)})}

	channelPendingTx := testChannelStatuser{
		latestL2:       eth.ToBlockID(block103),
		inclusionBlock: 1,
		fullySubmitted: false,
		timedOut:       false,
	}
	syncStatus := eth.SyncStatus{
		HeadL1:      eth.BlockRef{Number: 5},
		CurrentL1:   eth.BlockRef{Number: 2},
		LocalSafeL2: eth.L2BlockRef{Number: 103, Hash: block103.Hash()},
		UnsafeL2:    eth.L2BlockRef{Number: 109},
	}
	blocks := queue.Queue[SizedBlock]{block101, block102, block103}

	result, outOfSync := computeSyncActions(syncStatus, eth.BlockRef{Number: 1}, blocks, []channelStatuser{channelPendingTx}, l)
	require.False(t, outOfSync)
	require.Nil(t, result.clearState, "prune decision must be unchanged: no Clear")
	require.GreaterOrEqual(t, result.channelsToPrune, 1, "prune decision must be unchanged: still prunes pending-tx channel")
	require.Equal(t, 3, result.blocksToPrune)
}


// A channel that SPANS the safe-head boundary (first block(s) safe, last block(s)
// unsafe) is NOT pruned by computeSyncActions, while its leading block IS pruned from
// s.blocks by PruneSafeBlocks. Before the rewindToBlock guard, a late confirmation that
// times this channel out reached handleChannelInvalidated -> rewindToBlock, where
// idx := block.Number - s.blocks[0].Number() underflowed (unsigned) and panicked.
//
// With the guard, rewindToBlock clamps the target to the oldest pending block (the
// already-safe leading blocks need no resubmission) instead of underflowing. This test
// drives the REAL computeSyncActions (which returns {blocksToPrune>=1, channelsToPrune:0,
// clearState:nil}) and the REAL TxConfirmed path, and asserts NO panic plus a sane
// post-state where the channelManager remains usable.
func TestBoundarySpanningChannelNoPanic(t *testing.T) {
	l := testlog.Logger(t, log.LevelCrit)

	// Frame size that packs two mini-blocks into one channel while still producing
	// several calldata txs.
	cfg := channelManagerTestConfig(300, derive.SingularBatchType)
	cfg.ChannelTimeout = 2
	cfg.MaxChannelDuration = 0
	cfg.InitNoneCompressor()
	m := NewChannelManager(l, metrics.NoopMetrics, cfg, defaultTestRollupConfig)
	m.Clear(eth.BlockID{})

	parent := types.NewBlockWithHeader(&types.Header{Number: big.NewInt(100)})
	var blks []*types.Block
	for i := 0; i < 2; i++ {
		blk := newMiniL2BlockWithNumberParent(3, big.NewInt(int64(101+i)), parent.Hash())
		require.NoError(t, m.AddL2Block(blk))
		parent = blk
		blks = append(blks, blk)
	}
	var txIDs []txID
	for {
		tx, err := m.TxData(eth.BlockID{Number: 50}, false, pubInfo{forcePublish: true})
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if tx.Len() > 0 {
			txIDs = append(txIDs, tx.ID())
		}
	}
	require.Len(t, m.channelQueue, 1, "two blocks packed into a single channel")
	C := m.channelQueue[0]
	require.Equal(t, uint64(101), C.ChannelBuilder.blocks[0].NumberU64())
	require.Equal(t, uint64(102), C.LatestL2().Number, "channel spans 101..102")
	require.GreaterOrEqual(t, len(txIDs), 2)

	// Confirm one tx early (not timed out).
	m.TxConfirmed(txIDs[0], eth.BlockID{Number: 100})
	require.False(t, C.isTimedOut())

	// REAL driver with safeL2 = 101 (mid-channel).
	ss := eth.SyncStatus{
		HeadL1:      eth.BlockRef{Number: 5},
		CurrentL1:   eth.BlockRef{Number: 2},
		LocalSafeL2: eth.L2BlockRef{Number: 101, Hash: blks[0].Hash()},
		UnsafeL2:    eth.L2BlockRef{Number: 109},
	}
	actions, oos := computeSyncActions(ss, eth.BlockRef{Number: 1}, m.blocks, m.channelQueue, l)
	t.Logf("driver (boundary-spanning): %s outOfSync=%v", actions.TerminalString(), oos)
	require.False(t, oos)
	require.Nil(t, actions.clearState)
	require.Equal(t, 0, actions.channelsToPrune, "boundary-spanning channel is NOT pruned")
	require.Equal(t, 1, actions.blocksToPrune, "but its leading (safe) block IS pruned")

	m.PruneSafeBlocks(actions.blocksToPrune)
	m.PruneChannels(actions.channelsToPrune) // no-op (0)

	// Channel + txs still tracked, but s.blocks no longer starts at the channel's first block.
	require.Len(t, m.channelQueue, 1)
	require.Equal(t, uint64(102), m.blocks[0].NumberU64())
	lateID := txIDs[len(txIDs)-1]
	require.Same(t, C, m.txChannels[lateID.String()])

	// Late confirmation times the channel out -> rewindToBlock(101) on s.blocks[0]==102.
	// With the guard this clamps instead of underflowing: NO panic.
	require.NotPanics(t, func() {
		m.TxConfirmed(lateID, eth.BlockID{Number: 100 + C.cfg.ChannelTimeout + 5})
	}, "boundary-spanning channel timeout must not crash the batcher")
	require.True(t, C.isTimedOut(), "channel timed out, so the rewindToBlock path was reached")

	// Sane post-state: the timed-out channel was handled (removed from the queue by
	// handleChannelInvalidated), no orphaned txChannels entries remain, and the cursor
	// is within bounds so the manager is still usable.
	assertNoOrphans(t, m, "after boundary-spanning timeout")
	require.NotContains(t, m.channelQueue, C, "timed-out channel removed from queue")
	require.LessOrEqual(t, m.blockCursor, m.blocks.Len(), "blockCursor stays within bounds")
	require.GreaterOrEqual(t, m.blockCursor, 0)

	// The manager remains usable: a fresh block can still be added and pipelined.
	next := newMiniL2BlockWithNumberParent(3, big.NewInt(103), blks[1].Hash())
	require.NoError(t, m.AddL2Block(next))
	require.NotPanics(t, func() {
		_, _ = m.TxData(eth.BlockID{Number: 60}, false, pubInfo{forcePublish: true})
	}, "channelManager remains usable after a boundary-spanning timeout")
}


// Directly exercises the three guard branches added to rewindToBlock so a regression in
// any one is caught at the function level: (a) empty blocks queue, (b) blockCursor==0,
// (c) target block already pruned (below s.blocks[0]) -> clamp to oldest pending block
// instead of unsigned underflow. Also keeps a positive case (normal rewind) and the
// genuine not-in-state panic.
func TestRewindToBlockGuards(t *testing.T) {
	mkBlock := func(num uint64) SizedBlock {
		return SizedBlock{Block: types.NewBlockWithHeader(&types.Header{Number: new(big.Int).SetUint64(num)})}
	}
	newM := func(blocks queue.Queue[SizedBlock], cursor int) *channelManager {
		l := testlog.Logger(t, log.LevelCrit)
		cfg := channelManagerTestConfig(100, derive.SingularBatchType)
		cfg.InitNoneCompressor()
		m := NewChannelManager(l, metrics.NoopMetrics, cfg, defaultTestRollupConfig)
		m.blocks = blocks
		m.blockCursor = cursor
		return m
	}

	t.Run("empty queue returns without panic", func(t *testing.T) {
		m := newM(queue.Queue[SizedBlock]{}, 0)
		require.NotPanics(t, func() {
			m.rewindToBlock(eth.BlockID{Number: 101, Hash: common.HexToHash("0xaa")})
		})
		require.Equal(t, 0, m.blockCursor)
	})

	t.Run("cursor zero returns without panic", func(t *testing.T) {
		b := mkBlock(200)
		m := newM(queue.Queue[SizedBlock]{b}, 0)
		require.NotPanics(t, func() {
			m.rewindToBlock(eth.BlockID{Number: 200, Hash: b.Hash()})
		})
		require.Equal(t, 0, m.blockCursor)
	})

	t.Run("target below oldest clamps instead of underflowing", func(t *testing.T) {
		b200, b201 := mkBlock(200), mkBlock(201)
		m := newM(queue.Queue[SizedBlock]{b200, b201}, 2)
		// Target block 101 was already pruned (below s.blocks[0]==200). Before the fix this
		// underflowed (101-200) and panicked; now it clamps to the oldest pending block.
		require.NotPanics(t, func() {
			m.rewindToBlock(eth.BlockID{Number: 101, Hash: common.HexToHash("0xdead")})
		})
		require.Equal(t, 0, m.blockCursor, "clamped to oldest pending block index")
	})

	t.Run("normal rewind still works", func(t *testing.T) {
		b200, b201, b202 := mkBlock(200), mkBlock(201), mkBlock(202)
		m := newM(queue.Queue[SizedBlock]{b200, b201, b202}, 3)
		require.NotPanics(t, func() {
			m.rewindToBlock(eth.ToBlockID(b201))
		})
		require.Equal(t, 1, m.blockCursor, "cursor rewound to block 201's index")
	})

	t.Run("genuinely-absent block still panics", func(t *testing.T) {
		b200, b201 := mkBlock(200), mkBlock(201)
		m := newM(queue.Queue[SizedBlock]{b200, b201}, 2)
		// Target number is in-range (201) but hash mismatches -> not in state -> panic.
		require.Panics(t, func() {
			m.rewindToBlock(eth.BlockID{Number: 201, Hash: common.HexToHash("0xfeed")})
		})
	})
}
