// Package expiry covers interop message expiry: a message that its destination never relays is
// exported as undelivered by the destination's UndeliveredMessageExporter, through the
// withdrawal path, to the source chain's L1CrossDomainMessenger, which passes it on to the source
// chain as a deposit.
package expiry

import (
	"slices"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	// exportL1GasLimit is the gas the export reserves for the source L1CrossDomainMessenger call
	// on L1. Most of it is the resource-metering burn of the deposit that call makes, about 460k
	// at a 1 gwei L1 base fee and more when deposits are congested; an undergassed call is kept as
	// a failed message on L1 and can be replayed.
	exportL1GasLimit = uint32(1_000_000)

	// exportGasLimit is the L2 gas limit of the deposit that forces the export in on B.
	exportGasLimit = uint64(500_000)

	// The last leg is a deposit, and devstack time travel advances L1 only, which would stall L2
	// origin adoption. So the L1 windows are shrunk until they can be waited out in wall-clock
	// time. The games require
	// max(2*clockExtension, clockExtension+preimageOracleChallengePeriod) <= maxClockDuration.
	proofMaturityDelaySeconds       = 4
	disputeGameFinalityDelaySeconds = 4
	faultGameMaxClockDuration       = 40
	faultGameClockExtension         = 1
	preimageOracleChallengePeriod   = 10

	// testMessageExpiryWindow and testExpiryPeriod shorten the protocol's message expiry window and
	// the messenger's expiry period so a message can expire within a test. The period must exceed
	// the window, as the production 8 days exceed the protocol's 7, so an expired message stays
	// unrelayable.
	testMessageExpiryWindow = 12
	testExpiryPeriod        = 30
)

// productionExpiryPeriod is the messenger's expiry period on production networks.
const productionExpiryPeriod = 8 * 24 * 60 * 60

var (
	// undeliveredMessageExporterAddr is the UndeliveredMessageExporter predeploy.
	undeliveredMessageExporterAddr = common.HexToAddress("0x4200000000000000000000000000000000000030")

	failedRelayedMessageTopic = crypto.Keccak256Hash([]byte("FailedRelayedMessage(bytes32)"))
	messageExpiredTopic       = crypto.Keccak256Hash([]byte("MessageExpired(bytes32,uint256)"))
)

// shortClocks shrinks the L1 dispute windows and game clocks until they can be waited out in
// wall-clock time.
func shortClocks() presets.Option {
	return presets.WithDeployerOptions(
		sysgo.WithProofMaturityDelaySeconds(proofMaturityDelaySeconds),
		sysgo.WithDisputeGameFinalityDelaySeconds(disputeGameFinalityDelaySeconds),
		sysgo.WithFaultGameMaxClockDuration(faultGameMaxClockDuration),
		sysgo.WithFaultGameClockExtension(faultGameClockExtension),
		sysgo.WithPreimageOracleChallengePeriod(preimageOracleChallengePeriod),
	)
}

// TestShortGameClocksReachDeployedGames checks that the deployer options the expiry tests use to
// shorten the game clocks reach the deployed games.
func TestShortGameClocksReachDeployedGames(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t, shortClocks())
	bridge := sys.StandardBridge(sys.L2ChainB)
	t.Require().Equal(faultGameMaxClockDuration*time.Second, bridge.GameResolutionDelay())
	t.Require().Equal(proofMaturityDelaySeconds*time.Second, bridge.WithdrawalDelay())
	t.Require().Equal(disputeGameFinalityDelaySeconds*time.Second, bridge.DisputeGameFinalityDelay())
}

// TestUnrelayedMessageCannotExpireBeforeExpiryPeriod runs every leg of the expiry path: A sends
// ETH that B never relays; B's exporter, forced in as a deposit, tells A's L1CrossDomainMessenger
// through a withdrawal; A's L1CrossDomainMessenger deposits the word into A. The 8-day expiry
// period cannot pass in this system, so A must reject the word.
func TestUnrelayedMessageCannotExpireBeforeExpiryPeriod(gt *testing.T) {
	gt.Skip("requires the UndeliveredMessageExporter predeploy and the L1 messengers' undelivered message relay")
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t, shortClocks())
	require := t.Require()
	l1User := sys.FunderL1.NewFundedEOA(eth.OneEther)

	sender := sys.FunderA.NewFundedEOA(eth.OneEther)
	recipient := sys.FunderB.NewFundedEOA(eth.ZeroWei)
	send := dsl.SendETH(sender, recipient.Address(), sys.L2ChainB.ChainID(), eth.HalfEther)

	messengerA := sys.L2ELA.L2ToL2CrossDomainMessenger()
	require.Equal(uint64(productionExpiryPeriod), bigs.Uint64Strict(contract.Read(messengerA.ExpiryPeriod())),
		"A's messenger must use the production expiry period")

	// The export is forced in through B's portal, as it would be if B's sequencer censored it.
	exportRcpt, exportedAt := exportAsDeposit(sys, l1User, send.Message)

	deposit := finalizeExport(sys, l1User, exportRcpt)
	require.Equal(types.ReceiptStatusSuccessful, deposit.Status, "the deposit into A must execute")
	require.True(slices.ContainsFunc(deposit.Logs, func(l *types.Log) bool {
		return l.Address == predeploys.L2CrossDomainMessengerAddr && len(l.Topics) > 0 &&
			l.Topics[0] == failedRelayedMessageTopic
	}), "A's L2CrossDomainMessenger must keep the rejected word as a failed message")
	expire := sys.L2ELA.ExpireMessageCall(deposit.TxHash)
	require.Equal(dsl.ErrorSelector("L2ToL2CrossDomainMessenger_MessageNotExpired()"), expire.Output,
		"expireMessage must reject the word as not yet expired")
	require.Equal(send.Message.Hash, expire.MessageHash, "the word must carry the message's hash")
	require.Equal(exportedAt, expire.UndeliveredAt, "the word must carry the time B exported at")

	require.Equal(send.BlockTime, bigs.Uint64Strict(contract.Read(messengerA.SentMessageTimestamps(send.Message.Hash))),
		"A must have recorded the message at its send time")
	require.False(contract.Read(messengerA.ExpiredMessages(send.Message.Hash)), "the message must not expire early")
}

// TestUnrelayedMessageExpires runs the expiry path to its end: A sends ETH that B never relays;
// once B is past the expiry period, B's exporter, forced in as a deposit, tells A through L1, and
// A marks the message expired. The system deploys the messenger with a short expiry period so it
// can pass within the test.
func TestUnrelayedMessageExpires(gt *testing.T) {
	gt.Skip("requires the L2ToL2CrossDomainMessenger's expireMessage and an expiry period set at deploy")
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t, shortClocks(),
		presets.WithMessageExpiryWindow(testMessageExpiryWindow),
		presets.WithL2ToL2MessageExpiryPeriod(testExpiryPeriod))
	require := t.Require()
	l1User := sys.FunderL1.NewFundedEOA(eth.OneEther)
	messengerA := sys.L2ELA.L2ToL2CrossDomainMessenger()
	require.Equal(uint64(testExpiryPeriod), bigs.Uint64Strict(contract.Read(messengerA.ExpiryPeriod())),
		"A's messenger must use the test expiry period")

	sender := sys.FunderA.NewFundedEOA(eth.OneEther)
	recipient := sys.FunderB.NewFundedEOA(eth.ZeroWei)
	send := dsl.SendETH(sender, recipient.Address(), sys.L2ChainB.ChainID(), eth.HalfEther)

	// B exports only once its clock is past the send time plus the expiry period.
	expiresAfter := send.BlockTime + testExpiryPeriod
	sys.L2ELB.WaitForTime(expiresAfter + 1)
	exportRcpt, exportedAt := exportAsDeposit(sys, l1User, send.Message)
	require.Greater(exportedAt, expiresAfter, "B must export after the expiry period")

	deposit := finalizeExport(sys, l1User, exportRcpt)
	require.Equal(types.ReceiptStatusSuccessful, deposit.Status, "the deposit into A must execute")
	require.True(slices.ContainsFunc(deposit.Logs, func(l *types.Log) bool {
		return l.Address == predeploys.L2toL2CrossDomainMessengerAddr && len(l.Topics) > 1 &&
			l.Topics[0] == messageExpiredTopic && l.Topics[1] == send.Message.Hash
	}), "A's messenger must emit MessageExpired for the message")
	require.True(contract.Read(messengerA.ExpiredMessages(send.Message.Hash)), "the message must be expired on A")
}

// TestRelayedMessageCannotBeExportedAsUndelivered checks that a destination's exporter refuses to
// speak for a message it relayed.
func TestRelayedMessageCannotBeExportedAsUndelivered(gt *testing.T) {
	gt.Skip("requires the UndeliveredMessageExporter predeploy")
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	require := t.Require()

	recipient := sys.FunderB.NewFundedEOA(eth.ZeroWei)
	send := dsl.SendETH(sys.FunderA.NewFundedEOA(eth.OneEther), recipient.Address(), sys.L2ChainB.ChainID(),
		eth.OneHundredthEther)
	send.Relay(sys.FunderB.NewFundedEOA(eth.OneEther), sys.Supernode())

	exporterB := bindings.NewBindings[bindings.UndeliveredMessageExporter](bindings.WithClient(sys.L2ELB.EthClient()),
		bindings.WithTo(undeliveredMessageExporterAddr), bindings.WithTest(t))
	m := send.Message
	_, err := contractio.Read(exporterB.ExportUndeliveredMessage(sys.L2ChainA.L1CrossDomainMessengerAddr(), m.Source,
		m.Nonce, m.Sender, m.Target, m.Message, exportL1GasLimit), t.Ctx())
	require.ErrorContains(errutil.TryAddRevertReason(err),
		hexutil.Encode(dsl.ErrorSelector("UndeliveredMessageExporter_MessageRelayed()")),
		"a relayed message must not be exportable")
}

// exportAsDeposit forces B's export of the message, to A's L1CrossDomainMessenger, in through B's
// portal, and returns the receipt and B's timestamp at the export.
func exportAsDeposit(sys *presets.SimpleInterop, l1User *dsl.EOA, m dsl.SentMessage) (*types.Receipt, uint64) {
	return sys.FunderB.NewFundedEOA(eth.ZeroWei).ViaDepositTx(l1User, sys.L2ELB, sys.L2ChainB).
		ExportUndeliveredMessage(undeliveredMessageExporterAddr, sys.L2ChainA.L1CrossDomainMessengerAddr(), m,
			exportL1GasLimit, func(o *dsl.DepositTxOpts) { o.GasLimit = exportGasLimit })
}

// finalizeExport proves and finalizes the withdrawal B's export made, against a game with the
// shortened clock, and returns the deposit A's L1CrossDomainMessenger made into A in response.
func finalizeExport(sys *presets.SimpleInterop, l1User *dsl.EOA, exportRcpt *types.Receipt) *types.Receipt {
	withdrawal := sys.StandardBridge(sys.L2ChainB).WithdrawalFromReceipt(exportRcpt)
	withdrawal.ProveAndFinalize(l1User, faultGameMaxClockDuration*time.Second)
	return sys.L2ELA.WaitForDeposit(sys.L2ChainA.DepositContractAddr(), withdrawal.FinalizeReceipt())
}
