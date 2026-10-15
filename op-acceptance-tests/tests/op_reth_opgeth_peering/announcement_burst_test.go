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
	// after the reth 2.6 bump (reth#26924): a peer announcing more while those are unfetched has
	// the excess dropped. The burst announces one more fan-out of hashes than the cap.
	announcedHashCap  = 4096
	burstTransactions = announcedHashCap + fanOut
	// announceOnlyDataBytes is the calldata that makes a transaction larger than the 4096 bytes
	// op-geth sends whole to a peer (its txMaxBroadcastSize): op-geth announces a larger one by
	// hash only, so the peer has to fetch it.
	announceOnlyDataBytes = 4096
	// announceOnlyTxGas covers the EIP-7623 calldata floor of announceOnlyDataBytes zero bytes.
	announceOnlyTxGas = 80_000
	// burstSenderFunds pays one announce-only transaction at the pinned fee cap plus its L1 fee.
	burstSenderFundsGwei = 20_000_000
	// burstPoolAttempts bounds, in 1s polls, how long op-reth may take to fetch the burst into
	// its pool.
	burstPoolAttempts = 60
	// burstDrainAttempts bounds, in 1s polls, how long the resumed sequencer may take to include
	// the burst: at 60M gas per block it fills five blocks.
	burstDrainAttempts = 90
	// syncAttempts bounds, in 2s polls, how long op-geth may take to import the funding blocks.
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
	t.Require().Greater(tx.Size(), uint64(announceOnlyDataBytes), "the transaction must be larger than what op-geth sends whole")
	return tx
}

// verifySessionKept requires after to describe the same session as before: a different remote
// address means the session was dropped and the peer re-dialed from a new port.
func verifySessionKept(t devtest.T, before, after sysgo.ELPeerInfo) {
	t.Require().Equal(before.RemoteAddress, after.RemoteAddress, "op-reth must keep its session with op-geth; a new remote port means the session was dropped and re-dialed")
	t.Require().Equal(before.EthVersion, after.EthVersion, "the session must keep its negotiated eth version")
}

// TestOpRethKeepsItsOpGethPeerThroughAnAnnouncementBurst has op-geth announce more transaction
// hashes to op-reth at once than op-reth tracks per peer after the reth 2.6 bump, while the
// sequencer is paused, and requires op-reth to fetch at least a full cap's worth of them into its
// pool, to keep its session with op-geth and to log no error meanwhile; and once the sequencer
// resumes, to see every transaction it fetched included and to keep following the chain. How
// many of the burst op-reth fetched, and how many op-geth still holds afterwards, is logged.
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
	// op-geth admits a transaction against its own state, which must hold the senders' funds.
	opGeth.Reached(eth.Unsafe, fundedAt, syncAttempts)

	sys.Sequencer.CL.StopSequencer()
	sessionBefore := opReth.PeerInfoOf(opGeth)
	errorLogs := opReth.RecordLogs(log.LevelError)
	opGeth.SendTransactions(burst)
	pooled := opReth.TxPoolReached(announcedHashCap, burstPoolAttempts)
	t.Logger().Info("op-reth fetched the burst", "announced", len(burst), "pooled", pooled)

	verifySessionKept(t, sessionBefore, opReth.PeerInfoOf(opGeth))
	opGeth.VerifyPeeredWith(opReth)
	errorLogs.VerifyEmpty("fetching the burst")

	sys.Sequencer.CL.StartSequencer()
	opReth.TxPoolDrained(burstDrainAttempts)
	opReth.Advanced(eth.Unsafe, 2)
	verifySessionKept(t, sessionBefore, opReth.PeerInfoOf(opGeth))
	t.Logger().Info("Burst included", "announced", len(burst), "left_in_op_geth", opGeth.TxPoolStatus(), "left_in_op_reth", opReth.TxPoolStatus())
}
