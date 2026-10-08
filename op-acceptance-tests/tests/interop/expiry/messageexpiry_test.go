// Package expiry covers interop message expiry: a message that its destination never relays is
// exported as undelivered by the destination's UndeliveredMessageExporter, through the
// withdrawal path, to the source chain's L1CrossDomainMessenger, which passes it on to the source
// chain as a deposit.
package expiry

import (
	"bytes"
	"math/big"
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
	"github.com/ethereum/go-ethereum/accounts/abi"
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

	// gameResolutionTimeout bounds the wait for the withdrawal's dispute game. An unchallenged
	// game resolves once the defender's chess clock (faultGameMaxClockDuration) runs out; the
	// budget covers twice that plus a minute for blocks and the proposer.
	gameResolutionTimeout = 2*faultGameMaxClockDuration*time.Second + time.Minute
)

var (
	expireMessageSelector = crypto.Keccak256([]byte("expireMessage(bytes32,uint256)"))[:4]
	messageNotExpired     = crypto.Keccak256([]byte("L2ToL2CrossDomainMessenger_MessageNotExpired()"))[:4]
	refundNotExpired      = hexutil.Encode(crypto.Keccak256([]byte("SuperchainETHBridge_MessageNotExpired()"))[:4])
	messageRelayed        = hexutil.Encode(crypto.Keccak256([]byte("UndeliveredMessageExporter_MessageRelayed()"))[:4])
)

// TestUnrelayedMessageCannotExpireBeforeExpiryPeriod runs every leg of the expiry path: A sends
// ETH that B never relays; B's exporter, forced in as a deposit, tells A's L1CrossDomainMessenger
// through a withdrawal; A's L1CrossDomainMessenger deposits the word into A. The 8-day expiry
// period cannot pass in this system, so A must reject the word, and the send must not be
// refundable.
func TestUnrelayedMessageCannotExpireBeforeExpiryPeriod(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithDeployerOptions(
			sysgo.WithProofMaturityDelaySeconds(proofMaturityDelaySeconds),
			sysgo.WithDisputeGameFinalityDelaySeconds(disputeGameFinalityDelaySeconds),
			sysgo.WithFaultGameMaxClockDuration(faultGameMaxClockDuration),
			sysgo.WithFaultGameClockExtension(faultGameClockExtension),
			sysgo.WithPreimageOracleChallengePeriod(preimageOracleChallengePeriod),
		),
	)
	require := t.Require()
	l1User := sys.FunderL1.NewFundedEOA(eth.OneEther)

	sender := sys.FunderA.NewFundedEOA(eth.OneEther)
	recipient := sys.FunderB.NewFundedEOA(eth.ZeroWei)
	send := dsl.SendETH(sender, recipient.Address(), sys.L2ChainB.ChainID(), eth.HalfEther)

	// The export is forced in through B's portal, as it would be if B's sequencer censored it.
	exporter := sys.FunderB.NewFundedEOA(eth.ZeroWei).ViaDepositTx(l1User, sys.L2ELB, sys.L2ChainB)
	exportRcpt := exporter.DepositTx(predeploys.UndeliveredMessageExporterAddr, exportCalldata(t, sys, send.Message),
		func(o *dsl.DepositTxOpts) { o.GasLimit = exportGasLimit })
	exportBlock := sys.L2ELB.BlockRefByNumber(bigs.Uint64Strict(exportRcpt.BlockNumber))

	withdrawal := sys.StandardBridge(sys.L2ChainB).WithdrawalFromReceipt(exportRcpt)
	withdrawal.Prove(l1User)
	withdrawal.WaitForDisputeGameResolved(func(o *dsl.WaitForDisputeGameOpts) { o.Timeout = gameResolutionTimeout })
	withdrawal.Finalize(l1User)

	deposit := sys.L2ELA.WaitForDeposit(withdrawal.FinalizeReceipt())
	require.Equal(types.ReceiptStatusSuccessful, deposit.Status, "the deposit into A must execute")
	expire, found := sys.L2ELA.TraceCalls(deposit.TxHash).Find(func(f dsl.CallFrame) bool {
		return f.To == predeploys.L2toL2CrossDomainMessengerAddr && bytes.HasPrefix(f.Input, expireMessageSelector)
	})
	require.True(found, "the deposit into A must call expireMessage on its L2ToL2CrossDomainMessenger")
	require.Equal(messageNotExpired, []byte(expire.Output), "expireMessage must reject the word as not yet expired")
	args, err := abi.Arguments{{Type: abiType(t, "bytes32")}, {Type: abiType(t, "uint256")}}.Unpack(expire.Input[4:])
	require.NoError(err)
	require.Equal(send.Message.Hash, common.Hash(args[0].([32]byte)), "the word must carry the message's hash")
	require.Equal(new(big.Int).SetUint64(exportBlock.Time), args[1].(*big.Int), "the word must carry the time B exported at")

	messengerA := bindings.NewBindings[bindings.L2ToL2CrossDomainMessenger](bindings.WithClient(sys.L2ELA.EthClient()),
		bindings.WithTo(predeploys.L2toL2CrossDomainMessengerAddr), bindings.WithTest(t))
	require.False(contract.Read(messengerA.ExpiredMessages(send.Message.Hash)), "the message must not expire early")
	bridgeA := bindings.NewBindings[bindings.SuperchainETHBridge](bindings.WithClient(sys.L2ELA.EthClient()),
		bindings.WithTo(predeploys.SuperchainETHBridgeAddr), bindings.WithTest(t))
	_, err = contractio.Read(bridgeA.RefundETH(sys.L2ChainB.ChainID(), send.Message.Nonce, sender.Address(),
		recipient.Address(), eth.HalfEther.ToBig()), t.Ctx())
	require.ErrorContains(errutil.TryAddRevertReason(err), refundNotExpired, "an unexpired send must not be refundable")
}

// TestRelayedMessageCannotBeExportedAsUndelivered checks that a destination's exporter refuses to
// speak for a message it relayed.
func TestRelayedMessageCannotBeExportedAsUndelivered(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	require := t.Require()

	recipient := sys.FunderB.NewFundedEOA(eth.ZeroWei)
	send := dsl.SendETH(sys.FunderA.NewFundedEOA(eth.OneEther), recipient.Address(), sys.L2ChainB.ChainID(),
		eth.OneHundredthEther)
	send.Relay(sys.FunderB.NewFundedEOA(eth.OneEther), sys.Supernode())

	exporterB := bindings.NewBindings[bindings.UndeliveredMessageExporter](bindings.WithClient(sys.L2ELB.EthClient()),
		bindings.WithTo(predeploys.UndeliveredMessageExporterAddr), bindings.WithTest(t))
	m := send.Message
	_, err := contractio.Read(exporterB.ExportUndeliveredMessage(l1MessengerA(t, sys), m.Source, m.Nonce, m.Sender,
		m.Target, m.Message, exportL1GasLimit), t.Ctx())
	require.ErrorContains(errutil.TryAddRevertReason(err), messageRelayed, "a relayed message must not be exportable")
}

// l1MessengerA returns chain A's L1CrossDomainMessenger, where B's exports are sent.
func l1MessengerA(t devtest.T, sys *presets.SimpleInterop) common.Address {
	systemConfigA := bindings.NewSystemConfig(bindings.WithClient(sys.L1EL.EthClient()),
		bindings.WithTo(sys.L2ChainA.Escape().Deployment().SystemConfigProxyAddr()), bindings.WithTest(t))
	return contract.Read(systemConfigA.L1CrossDomainMessenger())
}

// exportCalldata encodes the exporter call that tells A, through its L1CrossDomainMessenger, that
// the message was not relayed on B.
func exportCalldata(t devtest.T, sys *presets.SimpleInterop, m dsl.SentMessage) []byte {
	exporter := bindings.NewBindings[bindings.UndeliveredMessageExporter](bindings.WithTest(t))
	call := exporter.ExportUndeliveredMessage(l1MessengerA(t, sys), m.Source, m.Nonce, m.Sender, m.Target, m.Message,
		exportL1GasLimit)
	data, err := call.EncodeInput()
	t.Require().NoError(err)
	return data
}

func abiType(t devtest.T, name string) abi.Type {
	typ, err := abi.NewType(name, "", nil)
	t.Require().NoError(err)
	return typ
}
