package op_reth_opgeth_peering

import (
	"testing"

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
	// after the reth 2.6 bump (reth#26924): a hash a peer announces while that many of its
	// announcements are still unfetched is dropped. The burst announces one fan-out of hashes
	// more than that within a few seconds. Whether op-reth ever had that many unfetched at once
	// is not observable through its RPC; op-reth fetches while the burst is still arriving.
	announcedHashCap  = 4096
	burstTransactions = announcedHashCap + fanOut
	// opGethTxMaxBroadcastSize is the largest transaction op-geth sends to a peer whole (its
	// txMaxBroadcastSize); it announces a larger one by hash only, so the peer has to fetch it.
	opGethTxMaxBroadcastSize = 4096
	// announceOnlyDataBytes is the calldata that puts a transaction over that size on its own.
	announceOnlyDataBytes = opGethTxMaxBroadcastSize
	// announceOnlyTxGas covers the EIP-7623 calldata floor of announceOnlyDataBytes zero bytes.
	announceOnlyTxGas = 80_000
	// burstSenderFundsGwei pays one announce-only transaction at the pinned fee cap plus its L1
	// data fee.
	burstSenderFundsGwei = 20_000_000
	// burstPoolAttempts bounds, in 1s polls, how long op-reth may take to fetch the burst into
	// its pool.
	burstPoolAttempts = 60
	// burstDrainAttempts bounds, in 1s polls, how long the resumed sequencer may take to include
	// the burst, which fills a handful of blocks at the devnet's gas limit.
	burstDrainAttempts = 90
	// syncAttempts bounds, in 2s polls, how long a follower may take to import the funding blocks.
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

// TestOpRethKeepsItsOpGethPeerThroughAnAnnouncementBurst has op-geth announce more transaction
// hashes to op-reth within a few seconds than op-reth tracks per peer after the reth 2.6 bump,
// while the sequencer is paused, and requires op-reth to fetch at least a full cap's worth of
// them into its pool, to keep its session with op-geth and to log no error meanwhile; and once
// the sequencer resumes, to see the burst included but for what a cap-sized fetch downstream can
// leave behind, and to keep following the chain. How many of the burst op-reth fetched, and how
// many each follower still holds afterwards, is logged.
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
	// funds; and op-reth's pool must be empty of the funding transfers before the burst, so the
	// burst is all it holds.
	dsl.CheckAll(t,
		opGeth.ReachedFn(eth.Unsafe, fundedAt, syncAttempts),
		opReth.ReachedFn(eth.Unsafe, fundedAt, syncAttempts),
	)
	sys.Sequencer.CL.StopSequencer()
	opReth.TxPoolDrained(syncAttempts)

	sessionBefore := opReth.PeerInfoOf(opGeth)
	errorLogs := opReth.RecordLogs(log.LevelError)
	opGeth.SendTransactions(burst)
	pooled := opReth.TxPoolReached(announcedHashCap, burstPoolAttempts)
	t.Logger().Info("op-reth fetched the burst", "announced", len(burst), "pooled", pooled)

	verifySessionKept(t, sessionBefore, opReth.PeerInfoOf(opGeth))
	errorLogs.VerifyEmpty("fetching the burst")

	sys.Sequencer.CL.StartSequencer()
	// op-reth passes the burst on to the sequencer, which fetches what op-reth only announced
	// to it under the same per-peer cap, so up to one fan-out of transactions can stay behind in
	// op-reth's pool.
	leftInOpReth := opReth.TxPoolHoldsAtMost(burstTransactions-announcedHashCap, burstDrainAttempts)
	opReth.Advanced(eth.Unsafe, 2)
	verifySessionKept(t, sessionBefore, opReth.PeerInfoOf(opGeth))
	opGeth.VerifyPeeredWith(opReth)
	t.Logger().Info("Burst included", "announced", len(burst), "left_in_op_geth", opGeth.TxPoolStatus(), "left_in_op_reth", leftInOpReth)
}
