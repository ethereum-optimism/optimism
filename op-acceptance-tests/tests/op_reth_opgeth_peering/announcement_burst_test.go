package op_reth_opgeth_peering

import (
	"maps"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

const (
	// announcedHashCap is how many announced transaction hashes op-reth tracks per peer at most
	// (reth#26924): a hash a peer announces while that many of its announcements are still
	// unfetched is dropped. op-reth fetches while the burst is still arriving, so it stays under
	// the cap only if it fetches as fast as op-geth announces. Whether it ever reached the cap is
	// not observable through its RPC.
	announcedHashCap = 4096
	// burstTransactions is how many transactions op-geth takes in to announce to op-reth. op-reth
	// is offered all of them only if op-geth's sends to it keep up with opGethMaxQueuedTxAnns.
	burstTransactions = 2 * announcedHashCap

	// opGethMaxQueuedTxAnns is how many hashes op-geth queues for announcement to one peer at
	// most (its maxQueuedTxAnns): when more are queued it drops the oldest, so it always announces
	// the last opGethMaxQueuedTxAnns it queued, and earlier ones only if it sent them before.
	opGethMaxQueuedTxAnns = 4096
	// opGethTxMaxBroadcastSize is the largest transaction op-geth sends to a peer whole (its
	// txMaxBroadcastSize); it announces a larger one by hash only, so the peer has to fetch it.
	opGethTxMaxBroadcastSize = 4096

	// announceOnlyDataBytes is the calldata that alone puts a transaction over
	// opGethTxMaxBroadcastSize.
	announceOnlyDataBytes = opGethTxMaxBroadcastSize
	// announceOnlyTxGas covers the EIP-7623 calldata floor of announceOnlyDataBytes zero bytes.
	announceOnlyTxGas = 80_000
	// burstSenderFundsGwei pays one announce-only transaction at the pinned fee cap plus its L1
	// data fee.
	burstSenderFundsGwei = 20_000_000

	// burstPoolMaxCount is op-reth's pending and basefee sub-pool size, in transactions, and how
	// many transactions op-geth's pool holds at most. Their defaults, 10000 for reth and 6144 for
	// op-geth, are too close to or below the burst.
	burstPoolMaxCount = 2 * burstTransactions
	// burstPoolMaxSizeMB is op-reth's pending and basefee sub-pool size, in MB. The burst takes
	// about 34 MB of pool space, more than reth's 20 MB default.
	burstPoolMaxSizeMB = 100

	// burstDrainAttempts bounds, in 1s polls, how long the resumed sequencer may take to include a
	// cap's worth of the burst, which fills about five blocks at the devnet's gas limit.
	burstDrainAttempts = 90
	// burstLeadAttempts bounds, in 1s polls, how long op-geth may take to hold the burst's first
	// transaction as pending.
	burstLeadAttempts = 30
	// burstPoolAttempts bounds, in 1s polls, how long op-reth may take to fetch a cap's worth of
	// the burst into its pool.
	burstPoolAttempts = 60
	// includedDrainAttempts bounds, in 1s polls, how long op-reth may take to clear transactions
	// from its pool once it has imported the blocks that include them.
	includedDrainAttempts = 60
	// syncAttempts bounds, in 2s polls, how long a follower may take to import blocks the
	// sequencer has already built.
	syncAttempts = 30
)

// announceOnlyTransaction signs the transaction a fresh sender contributes to the burst: too
// large for op-geth to send it whole, so it is announced by hash.
func announceOnlyTransaction(t devtest.T, sender *dsl.EOA) *types.Transaction {
	to := sender.Address()
	tx := sender.Sign(
		sender.PlanOffline(),
		txplan.WithStaticNonce(0),
		txplan.WithTo(&to),
		txplan.WithData(make([]byte, announceOnlyDataBytes)),
		txplan.WithGasLimit(announceOnlyTxGas),
		txplan.WithGasFeeCap(feeCap),
		txplan.WithGasTipCap(tip),
	)
	t.Require().Greater(tx.Size(), uint64(opGethTxMaxBroadcastSize), "the transaction must be larger than what op-geth sends whole")
	return tx
}

// verifySessionKept requires after to describe the same session as before, seen from the
// accepting side: a dropped session that the dialing side re-dials shows up there as a new
// remote port.
func verifySessionKept(t devtest.T, before, after sysgo.ELPeerInfo) {
	t.Require().True(before.Inbound, "a re-dial shows a new remote port on the accepting side only")
	t.Require().Equal(before.RemoteAddress, after.RemoteAddress, "op-reth must keep its session with op-geth; a new remote port means the session was dropped and re-dialed")
	t.Require().Equal(before.EthVersion, after.EthVersion, "the session must keep its negotiated eth version")
}

// transactionsWithHashes returns those of txs whose hash is in hashes.
func transactionsWithHashes(txs []*types.Transaction, hashes map[common.Hash]struct{}) []*types.Transaction {
	var selected []*types.Transaction
	for _, tx := range txs {
		if _, ok := hashes[tx.Hash()]; ok {
			selected = append(selected, tx)
		}
	}
	return selected
}

// verifyBurstReconciled requires every burst transaction op-reth pooled to be either included in
// a block or still in its pool.
func verifyBurstReconciled(t devtest.T, pooled, included, stillPooled map[common.Hash]struct{}) {
	var inNeither []common.Hash
	for hash := range pooled {
		_, isIncluded := included[hash]
		_, isStillPooled := stillPooled[hash]
		if !isIncluded && !isStillPooled {
			inNeither = append(inNeither, hash)
		}
	}
	t.Require().Empty(inNeither, "every burst transaction op-reth pooled must be in a block or still in its pool; %d of %d are in neither", len(inNeither), len(pooled))
}

// TestOpRethKeepsItsOpGethPeerThroughAnAnnouncementBurst hands op-geth twice as many transactions
// to announce to op-reth within a few seconds as op-reth tracks announced hashes per peer, sent so
// that op-reth is offered more hashes than that whatever op-geth drops from its own announcement
// queue, while the sequencer is paused. op-reth has to fetch at least a full cap's worth of them
// into its pool, keep its session with op-geth and log no error meanwhile. Once the sequencer
// resumes, op-reth has to keep following the chain until it includes a cap's worth of the burst.
// With the sequencer paused again, op-reth has to clear the burst transactions in blocks up to
// the paused head from its pool, and every burst transaction it was seen pooling has to be either
// in such a block or still in its pool; its pool may keep what a cap-sized fetch downstream leaves
// behind. How many of the burst op-reth pooled, how many were included, how many op-geth offered
// op-reth at least, and how many each follower still holds afterwards, is logged.
func TestOpRethKeepsItsOpGethPeerThroughAnAnnouncementBurst(gt *testing.T) {
	t := devtest.ParallelT(gt)
	skipUnsupported(t)
	sys := newPairSystem(t, sysgo.MixedL2ELOpGeth)
	opGeth, opReth := sys.Sender.EL, sys.Receiver.EL
	senders, fundedAt := fundAccounts(t, sys, burstTransactions, eth.GWei(burstSenderFundsGwei))
	burst := make([]*types.Transaction, len(senders))
	for i, sender := range senders {
		burst[i] = announceOnlyTransaction(t, sender)
	}
	// Both followers admit a transaction against their own state, which must hold the senders'
	// funds.
	dsl.CheckAll(t,
		opGeth.ReachedFn(eth.Unsafe, fundedAt, syncAttempts),
		opReth.ReachedFn(eth.Unsafe, fundedAt, syncAttempts),
	)
	sys.Sequencer.CL.StopSequencer()

	sessionBefore := opReth.PeerInfoOf(opGeth)
	errorLogs := opReth.RecordLogs(log.LevelError)
	// op-geth announces at least the last opGethMaxQueuedTxAnns hashes it queues for op-reth. The
	// lead transaction, pending in op-geth before the rest is sent, is queued alone and taken for
	// announcement before the rest is queued, as op-geth has nothing else to announce to op-reth,
	// its only peer: op-reth is offered more than announcedHashCap hashes of the burst whatever
	// op-geth drops.
	t.Require().GreaterOrEqual(opGethMaxQueuedTxAnns, announcedHashCap, "op-geth must announce at least a cap's worth of what it queues")
	lead, rest := burst[:1], burst[1:]
	opGeth.SendTransactions(lead)
	opGeth.WaitForPendingTransactions(lead, burstLeadAttempts)
	opGeth.SendTransactions(rest)
	pooled := opReth.WaitForPooledTransactions(burst, announcedHashCap, burstPoolAttempts)

	verifySessionKept(t, sessionBefore, opReth.PeerInfoOf(opGeth))
	errorLogs.VerifyEmpty("fetching the burst")
	t.Logger().Info("op-reth fetched the burst", "burst", len(burst), "pooled", len(pooled))

	sys.Sequencer.CL.StartSequencer()
	// op-reth passes the burst on to the sequencer, which fetches what op-reth only announced
	// to it under the same per-peer cap: at least a cap's worth of the burst reaches a block, and
	// up to as many transactions as the burst exceeds the cap by can stay behind in op-reth's pool.
	// The burst is sent once both followers are past fundedAt, so no block up to it holds any.
	included, nextBlock := opReth.WaitForIncludedTransactions(burst, fundedAt+1, announcedHashCap, burstDrainAttempts)
	opReth.Advanced(eth.Unsafe, 2)
	// Pausing the sequencer again holds op-reth's chain still, so its blocks and its pool are read
	// against the same head.
	pausedHead := sys.Sequencer.EL.BlockRefByHash(sys.Sequencer.CL.StopSequencer())
	opReth.Reached(eth.Unsafe, pausedHead.Number, syncAttempts)
	maps.Copy(included, opReth.IncludedTransactions(burst, nextBlock, pausedHead.Number))
	opReth.WaitForTransactionsOutOfPool(transactionsWithHashes(burst, included), includedDrainAttempts)
	stillPooled := opReth.PooledTransactions(burst)
	t.Require().Equal(pausedHead.Hash, opReth.BlockRefByLabel(eth.Unsafe).Hash, "op-reth's head must stay at the paused sequencer's head while the burst is reconciled")
	verifyBurstReconciled(t, pooled, included, stillPooled)

	verifySessionKept(t, sessionBefore, opReth.PeerInfoOf(opGeth))
	opGeth.VerifyPeeredWith(opReth)
	// op-reth learns of a burst transaction only from op-geth, so those it pooled or that reached a
	// block are a lower bound on how many op-geth offered it.
	offeredAtLeast := maps.Clone(pooled)
	maps.Copy(offeredAtLeast, included)
	maps.Copy(offeredAtLeast, stillPooled)
	t.Logger().Info("Burst included", "burst", len(burst), "offered_at_least", len(offeredAtLeast), "pooled", len(pooled), "included", len(included), "left_in_op_geth", opGeth.TxPoolStatus(), "left_in_op_reth", len(stillPooled))
}
