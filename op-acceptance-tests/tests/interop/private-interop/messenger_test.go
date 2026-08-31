package privateinterop

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
)

// TestPrivateInteropMessengerBothDirections is the gate: a messenger message crossing INTO the
// private chain and back OUT of it, through the whole pipeline.
//
// The two directions are not symmetric, and the asymmetry is the point.
//
// INTO the private chain (counterparty -> private) is written with the stock devstack helpers
// because the initiating message is on an ordinary public chain and its identifier is an ordinary
// receipt position. This half is `interop/contract`'s TestRegularMessage with chain B swapped for a
// pair, and it passes unmodified in substance.
//
// OUT of the private chain (private -> counterparty) is the seam -- and it is written with the SAME
// stock helpers, which is what changed with T2. The initiating message's identity is its position
// on the RENDERING, not on the private chain: same block number and timestamp, different log index,
// because the rendering carries only the emitter-set logs while the private block carries all of
// them. Before the resolver this test resolved that position by hand; now
// txintent.InteropOutput.FromReceipt does it centrally, and the outbound half is line-for-line the
// inbound half with the chains swapped. What the test still does by hand is CHECK the resolver:
// it sends the outbound message in a block position the private chain and the rendering
// deliberately disagree about, and asserts the identifier that came back names the rendering's.
func TestPrivateInteropMessengerBothDirections(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0, presets.WithPrivateInteropChain())
	require := sys.T.Require()
	logger := t.Logger()
	rng := rand.New(rand.NewSource(1234))

	alice := sys.FunderA.NewFundedEOA(eth.OneTenthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneTenthEther)

	// The EventLogger is deployed at runtime on the PRIVATE chain, which works because the private
	// chain is an ordinary sequenced chain from its own side: a mempool, a nonce, a receipt.
	//
	// Note what this does NOT do: it does not make the deployed logger replayable. A runtime deploy
	// lands at an EOA-nonce-derived address, and the rendering's EventReplayer is a fixed genesis
	// address -- so a message whose TARGET is this contract is fine (targets live on the executing
	// side) while a message EMITTED by it would not be. The fixed-address policy that closes that
	// gap is T2's, and neither direction below needs it.
	eventLoggerB := bob.DeployEventLogger()
	eventLoggerA := alice.DeployEventLogger()
	eventLogger := bindings.NewBindings[bindings.EventLogger]()

	// ---------------------------------------------------------------------------------------
	// Direction 1: counterparty -> private. Stock helpers, no resolution.
	// ---------------------------------------------------------------------------------------
	topicsIn, dataIn := randomEvent(rng)
	calldataIn, err := eventLogger.EmitLog(topicsIn, dataIn).EncodeInputLambda()
	require.NoError(err, "failed to prepare calldata for the inbound message")

	logger.Info("Sending a message INTO the private chain", "target", eventLoggerB)
	txA := txintent.NewIntent[*txintent.SendTrigger, *txintent.InteropOutput](alice.Plan())
	txA.Content.Set(&txintent.SendTrigger{
		Emitter:         predeploys.L2toL2CrossDomainMessengerAddr,
		DestChainID:     bob.ChainID(),
		Target:          eventLoggerB,
		RelayedCalldata: calldataIn,
	})
	sendIn, err := txA.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err, "the inbound send did not land on the counterparty")
	require.Len(sendIn.Logs, 1, "a send emits exactly the SentMessage event")

	// Let the counterparty advance so the supernode indexes the initiating message.
	sys.L2A.WaitForBlock()

	txB := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](bob.Plan())
	txB.Content.DependOn(&txA.Result)
	txB.Content.Fn(txintent.RelayIndexed(predeploys.L2toL2CrossDomainMessengerAddr, &txA.Result, &txA.PlannedTx.Included, 0))
	relayIn, err := txB.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err, "the inbound relay did not land on the private chain")
	// ExecutingMessage, the EventLogger's own event, RelayedMessage.
	require.Len(relayIn.Logs, 3, "the relay emits the executing message, the target's event and the receipt")
	for i, addr := range []common.Address{predeploys.CrossL2InboxAddr, eventLoggerB, predeploys.L2toL2CrossDomainMessengerAddr} {
		require.Equal(addr, relayIn.Logs[i].Address, "unexpected emitter at relay log %d", i)
	}
	logger.Info("The private chain imported a message from its counterparty",
		"private_block", relayIn.BlockNumber, "tx", relayIn.TxHash)

	// ---------------------------------------------------------------------------------------
	// Direction 2: private -> counterparty. The identifier comes from the rendering, and the
	// resolver is what puts it there.
	// ---------------------------------------------------------------------------------------
	topicsOut, dataOut := randomEvent(rng)
	calldataOut, err := eventLogger.EmitLog(topicsOut, dataOut).EncodeInputLambda()
	require.NoError(err, "failed to prepare calldata for the outbound message")

	// The outbound send goes out BEHIND a log the export policy does not publish, in one
	// transaction, so that the message's private position and its public one are guaranteed to
	// differ. eventLoggerB is a runtime deploy at an EOA-nonce address: it is in nobody's emitter
	// set, so the rendering drops its log and the SentMessage that follows moves from private index
	// 1 to public index 0. A resolver that did nothing would hand the counterparty index 1 -- a
	// position the rendering does not carry -- and the judge would replace the block that executed
	// it, which is exactly what this test's last assertion checks.
	sentinelTopics, sentinelData := randomEvent(rng)
	logger.Info("Sending a message OUT of the private chain", "target", eventLoggerA)
	txOut := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](bob.Plan())
	txOut.Content.Set(&txintent.MultiTrigger{
		Emitter: predeploys.MultiCall3Addr,
		Calls: []txintent.Call{
			&txintent.InitTrigger{
				Emitter:    eventLoggerB,
				Topics:     asTopics(sentinelTopics),
				OpaqueData: sentinelData,
			},
			&txintent.SendTrigger{
				Emitter:         predeploys.L2toL2CrossDomainMessengerAddr,
				DestChainID:     alice.ChainID(),
				Target:          eventLoggerA,
				RelayedCalldata: calldataOut,
			},
		},
	})
	sendOut, err := txOut.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err, "the outbound send did not land on the private chain")
	require.Len(sendOut.Logs, 2, "the transaction emits the unpublished event and then the SentMessage")
	require.Equal(eventLoggerB, sendOut.Logs[0].Address, "the first log is the one the export policy drops")
	require.Equal(predeploys.L2toL2CrossDomainMessengerAddr, sendOut.Logs[1].Address, "the second log is the message")
	const outIdx = 1
	sentLog := sendOut.Logs[outIdx]
	privateBlock := sendOut.BlockNumber.Uint64()

	// Evaluating the result is what runs the resolver: it waits for the rendering to derive this
	// height and rewrites the identifier to the position the rendering carries. Nothing in this
	// test asks it to; it is what the stock helper does now.
	out, err := txOut.Result.Eval(t.Ctx())
	require.NoError(err, "the outbound message's public position did not resolve")
	require.Len(out.Entries, 2, "one entry per log, so an index into the entries still means the log it always did")
	resolved := out.Entries[outIdx].Identifier
	logger.Info("The stock helper resolved the outbound message's public position",
		"private_log_index", sentLog.Index, "public_log_index", resolved.LogIndex, "block", privateBlock)
	require.Less(resolved.LogIndex, uint32(sentLog.Index),
		"the rendering drops the unpublished log ahead of the message, so the public index must be lower than the private one; "+
			"an unchanged index is a resolver that did not run")
	require.Equal(privateBlock, resolved.BlockNumber, "block-for-block: the number does not move")
	require.Equal(predeploys.L2toL2CrossDomainMessengerAddr, resolved.Origin,
		"a messenger export is replayed at the standard messenger predeploy, which is what every stock consumer expects")

	require.NotNil(sys.PrivateInterop.Invariant, "the standing invariant checker should be on")
	renderedBlock := sys.L2BSupernodeEL.BlockRefByNumber(privateBlock)
	privateAtSame := sys.L2ELB.BlockRefByNumber(privateBlock)
	require.Equal(privateAtSame.Time, renderedBlock.Time, "block-for-block correspondence at the message's height")
	require.NotEqual(privateAtSame.Hash, renderedBlock.Hash, "the two halves are different chains")
	require.Equal(resolved.Timestamp, renderedBlock.Time, "the resolved identifier's timestamp is the rendering block's")

	// The rendering's own copy of the message, at the position the resolver named.
	renderedLog := renderedLogAt(t, sys, renderedBlock.Hash, int(resolved.LogIndex))
	require.Equal(predeploys.L2toL2CrossDomainMessengerAddr, renderedLog.Address,
		"the replay re-emits at the standard messenger predeploy, which is what every stock consumer expects")
	require.Equal(sentLog.Topics, renderedLog.Topics, "the replayed SentMessage is byte-identical in its topics")
	require.Equal(sentLog.Data, renderedLog.Data, "the replayed SentMessage is byte-identical in its payload")

	// And the counterparty executes it -- through the same stock helper the inbound direction used,
	// which is the whole claim of the resolver: a test does not know which side of it is private.
	txRelayOut := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](alice.Plan())
	txRelayOut.Content.DependOn(&txOut.Result)
	txRelayOut.Content.Fn(txintent.RelayIndexed(predeploys.L2toL2CrossDomainMessengerAddr, &txOut.Result, &txOut.PlannedTx.Included, outIdx))
	relayOut, err := txRelayOut.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err, "the outbound relay did not land on the counterparty")
	require.Len(relayOut.Logs, 3, "the relay emits the executing message, the target's event and the receipt")
	for i, addr := range []common.Address{predeploys.CrossL2InboxAddr, eventLoggerA, predeploys.L2toL2CrossDomainMessengerAddr} {
		require.Equal(addr, relayOut.Logs[i].Address, "unexpected emitter at outbound relay log %d", i)
	}
	relayedEvent := relayOut.Logs[1]
	require.Len(relayedEvent.Topics, len(topicsOut), "the target saw the message's topics")
	for i := range relayedEvent.Topics {
		require.Equal(topicsOut[i][:], relayedEvent.Topics[i].Bytes())
	}
	require.Equal(dataOut, relayedEvent.Data, "the target saw the message's data")

	logger.Info("A message left the private chain and executed on its counterparty",
		"private_block", privateBlock, "public_log_index", resolved.LogIndex,
		"counterparty_block", relayOut.BlockNumber, "tx", relayOut.TxHash)

	// The execution has to SURVIVE the judge, which is the assertion the whole pipeline exists for.
	//
	// The supernode judges chain A's executing message against the RENDERING's message database --
	// never against a private receipt, which it has never seen and by design never will. An
	// identifier naming a position the rendering does not carry is an invalid executing message, and
	// an invalid executing message gets its block REPLACED with a deposits-only one. So the check is
	// that the block is still there, unchanged, some blocks later.
	//
	// Paired with a liveness counter, per house discipline: a chain that stopped producing would
	// also never replace anything, and would pass a bare "was not replaced" check for the wrong
	// reason.
	relayOutBlock := sys.L2ELA.BlockRefByHash(relayOut.BlockHash)
	before := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	dsl.CheckAll(t, sys.L2ELA.ReachedFn(eth.Unsafe, before.Number+judgeSettleBlocks, 60))
	after := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	require.GreaterOrEqual(after.Number, before.Number+judgeSettleBlocks,
		"the counterparty must keep producing, or 'the block was not replaced' means nothing")
	stillThere := sys.L2ELA.BlockRefByNumber(relayOutBlock.Number)
	require.Equal(relayOutBlock.Hash, stillThere.Hash,
		"the counterparty's block carrying the executing message was replaced: the supernode judged the message invalid, "+
			"which means the identifier did not name the position the rendering actually carries")

	// "Not replaced" is the fast answer. CROSS-SAFE is the whole answer: the counterparty's safe
	// frontier only passes this block once the supernode has validated the executing message against
	// the rendering's message database and carried both chains' local-safe frontiers past it. It
	// therefore proves the pipeline end to end -- the range built, its claim landed, the rendering
	// derived it, the message database indexed the replayed SentMessage at the position the
	// identifier names, and the judge agreed.
	//
	// This is what the T1 lane had to park. It needs the RENDERING's own safety to keep up with the
	// counterparty's clock, which it cannot do while only the first range is buildable; with the
	// rendering follower's block decode fixed, ranges follow one another and this is an ordinary
	// wait. Generous budget and a liveness-aware wait, because a range is a discrete step: the
	// frontier moves in jumps, and the thing that must never stall is block production.
	dsl.CheckAll(t, sys.L2ASupernodeCL.ReachedWithProgressFn(
		safety.CrossSafe, safety.LocalUnsafe, relayOutBlock.Number, 6*time.Minute, 90*time.Second))

	status, err := sys.L2ASupernodeCL.Escape().RollupAPI().SyncStatus(t.Ctx())
	require.NoError(err, "reading the counterparty's sync status")
	require.GreaterOrEqual(status.SafeL2.Number, relayOutBlock.Number,
		"the counterparty's cross-safe frontier must pass the block that executed the private chain's message")
	logger.Info("The counterparty's frontier after the judge settled",
		"cross_safe", status.SafeL2.Number, "local_safe", status.LocalSafeL2.Number,
		"relay_block", relayOutBlock.Number, "rendering_safe", sys.PrivateInterop.Invariant.RenderingSafeHead())

	// And the super-root at the relay block's timestamp: the cluster's single view of both chains at
	// one instant, which cannot exist until every chain in the dependency set -- the rendering
	// included -- has a cross-safe block at or past it.
	sys.Supernode.AwaitValidatedTimestamp(relayOutBlock.Time)
}

// judgeSettleBlocks is how many counterparty blocks to let pass before concluding that the executing
// message was accepted. The supernode judges cross-unsafe within a block or two of the initiating
// message being known; this is several times that.
const judgeSettleBlocks = 8

// renderedLogAt reads the rendering's k-th emitter-set log out of a derived block.
//
// The test reads the rendering here for one reason only: to check the resolver's answer against the
// chain a judge will actually read. Resolving a position is no longer a test's job -- that is what
// the identifier resolver does, centrally, in the helper every test already calls (testing plan
// section 2).
func renderedLogAt(t devtest.T, sys *presets.TwoL2SupernodeInterop, blockHash common.Hash, index int) *types.Log {
	_, receipts, err := sys.L2BSupernodeEL.Escape().L2EthClient().FetchReceipts(t.Ctx(), blockHash)
	t.Require().NoError(err, "reading the rendering block's receipts")

	var logs []*types.Log
	for _, r := range receipts.Geth() {
		logs = append(logs, r.Logs...)
	}
	rendered := render.RenderedLogs(logs, render.EmitterSet{})
	t.Require().Greaterf(len(rendered), index,
		"the rendering block has %d emitter-set logs, so there is no log %d; the rendering's log sequence must equal RenderedLogs of the private block exactly",
		len(rendered), index)
	return rendered[index].Log
}

// randomEvent builds an EventLogger event of random shape, so that a passing run is not passing on
// one fixed payload.
func randomEvent(rng *rand.Rand) ([]eth.Bytes32, []byte) {
	topics := []eth.Bytes32{}
	for range 1 + rng.Intn(4) {
		var topic [32]byte
		copy(topic[:], testutils.RandomData(rng, 32))
		topics = append(topics, topic)
	}
	return topics, testutils.RandomData(rng, 1+rng.Intn(30))
}

// asTopics adapts randomEvent's topics to the shape the raw EventLogger trigger takes.
func asTopics(topics []eth.Bytes32) [][32]byte {
	out := make([][32]byte, len(topics))
	for i, topic := range topics {
		out[i] = topic
	}
	return out
}
