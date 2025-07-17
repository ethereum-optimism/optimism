package contract

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum/go-ethereum/common"
)

// TestRegularMessage checks that messages can be sent and relayed via L2ToL2CrossDomainMessenger
func TestRegularMessage(gt *testing.T) {
	gt.Skip() // TODO(#16731) Fix to handle the op-geth version bump to v1.101511.1
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	require := sys.T.Require()
	logger := t.Logger()
	rng := rand.New(rand.NewSource(1234))

	alice, bob := sys.FunderA.NewFundedEOA(eth.OneTenthEther), sys.FunderB.NewFundedEOA(eth.OneTenthEther)

	// deploy event logger at chain B
	eventLoggerAddress := bob.DeployEventLogger()
	// only use the binding to generate calldata
	eventLogger := bindings.NewBindings[bindings.EventLogger]()
	// manually build topics and data for EventLogger
	topics := []eth.Bytes32{}
	for range rng.Intn(5) {
		var topic [32]byte
		copy(topic[:], testutils.RandomData(rng, 32))
		topics = append(topics, topic)
	}
	data := testutils.RandomData(rng, rng.Intn(30))

	calldata, err := eventLogger.EmitLog(topics, data).EncodeInputLambda()
	require.NoError(err, "failed to prepare calldata")

	logger.Info("Send message", "address", eventLoggerAddress, "topicCnt", len(topics), "dataLen", len(data))
	trigger := &txintent.SendTrigger{
		Emitter:         constants.L2ToL2CrossDomainMessenger,
		DestChainID:     bob.ChainID(),
		Target:          eventLoggerAddress,
		RelayedCalldata: calldata,
	}
	// Intent to send message on chain A
	txA := txintent.NewIntent[*txintent.SendTrigger, *txintent.InteropOutput](alice.Plan())
	txA.Content.Set(trigger)

	sendMsgReceipt, err := txA.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err, "send msg receipt not found")
	require.Equal(1, len(sendMsgReceipt.Logs)) // SentMessage event
	require.Equal(constants.L2ToL2CrossDomainMessenger, sendMsgReceipt.Logs[0].Address)

	// Make sure supervisor syncs the chain A events
	sys.Supervisor.WaitForUnsafeHeadToAdvance(alice.ChainID(), 2)

	// Intent to relay message on chain B
	txB := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](bob.Plan())
	txB.Content.DependOn(&txA.Result)
	idx := 0
	txB.Content.Fn(txintent.RelayIndexed(constants.L2ToL2CrossDomainMessenger, &txA.Result, &txA.PlannedTx.Included, idx))

	relayMsgReceipt, err := txB.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err, "relay msg receipt not found")

	// ExecutingMessage, EventLogger, RelayedMessage Events
	require.Equal(3, len(relayMsgReceipt.Logs))
	for logIdx, addr := range []common.Address{constants.CrossL2Inbox, eventLoggerAddress, constants.L2ToL2CrossDomainMessenger} {
		require.Equal(addr, relayMsgReceipt.Logs[logIdx].Address)
	}
	// EventLogger topics and data
	eventLog := relayMsgReceipt.Logs[1]
	require.Equal(len(topics), len(eventLog.Topics))
	for topicIdx := range len(eventLog.Topics) {
		require.Equal(topics[topicIdx][:], eventLog.Topics[topicIdx].Bytes())
	}
	require.Equal(data, eventLog.Data)
}
