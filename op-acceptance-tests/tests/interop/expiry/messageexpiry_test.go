// Package expiry covers the interop message expiry path: a message that its destination never
// relays is exported as undelivered through the destination's withdrawal path, to the source
// chain's L1CrossDomainMessenger, which passes it on to the source chain as a deposit.
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
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	// minGasLimit is the gas reserved for the source L1CrossDomainMessenger call on L1.
	minGasLimit = uint32(500_000)

	// The last leg is a deposit, and devstack time travel advances L1 only, which would
	// stall L2 origin adoption. So the L1 windows are shrunk until they can be waited out in
	// wall-clock time. The games require
	// max(2*clockExtension, clockExtension+preimageOracleChallengePeriod) <= maxClockDuration.
	proofMaturityDelaySeconds       = 4
	disputeGameFinalityDelaySeconds = 4
	faultGameMaxClockDuration       = 40
	faultGameClockExtension         = 1
	preimageOracleChallengePeriod   = 10
)

var (
	sentMessageTopic          = crypto.Keccak256Hash([]byte("SentMessage(uint256,address,uint256,address,bytes)"))
	failedRelayedMessageTopic = crypto.Keccak256Hash([]byte("FailedRelayedMessage(bytes32)"))
	expireMessageSelector     = crypto.Keccak256([]byte("expireMessage(bytes32,uint256)"))[:4]
	messageNotExpiredSelector = crypto.Keccak256([]byte("MessageNotExpired()"))[:4]
	messageAlreadyRelayed     = hexutil.Encode(crypto.Keccak256([]byte("MessageAlreadyRelayed()"))[:4])
)

// TestInteropMessageExpiry runs every leg of the expiry path for real:
//
//	A: sendETH -> (never relayed on B) -> B: exportUndeliveredMessage -> L1 withdrawal
//	-> A's L1CrossDomainMessenger.relayUndeliveredMessage -> A deposit -> expireMessage
//
// The 7-day expiry window cannot pass in this system, so A must reject the word, and the send
// must not be refundable. It also checks that ordinary delivery still works and that a delivered
// message cannot be exported as undelivered.
func TestInteropMessageExpiry(gt *testing.T) {
	t := devtest.SerialT(gt)
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
	sys.L1Network.WaitForOnline()
	chainA, chainB := sys.L2ChainA.ChainID(), sys.L2ChainB.ChainID()

	l1User := sys.FunderL1.NewFundedEOA(eth.OneEther)
	systemConfigA := bindings.NewSystemConfig(bindings.WithClient(sys.L1EL.EthClient()),
		bindings.WithTo(sys.L2ChainA.Escape().Deployment().SystemConfigProxyAddr()), bindings.WithTest(t))
	l1MessengerA := contract.Read(systemConfigA.L1CrossDomainMessenger())
	messengerA := messenger(t, sys.L2ELA)
	messengerB := messenger(t, sys.L2ELB)
	bridgeA := bindings.NewBindings[bindings.SuperchainETHBridge](
		bindings.WithClient(sys.L2ELA.EthClient()), bindings.WithTo(predeploys.SuperchainETHBridgeAddr), bindings.WithTest(t))

	// An ordinary send still relays.
	deliveredRecipient := sys.FunderB.NewFundedEOA(eth.ZeroWei)
	delivered := sendAndRelayETH(t, sys, sys.FunderA.NewFundedEOA(eth.OneEther), deliveredRecipient,
		sys.FunderB.NewFundedEOA(eth.OneEther), eth.OneHundredthEther)
	deliveredRecipient.VerifyBalanceExact(eth.OneHundredthEther)

	// The send that is never relayed on B.
	sender := sys.FunderA.NewFundedEOA(eth.OneEther)
	recipient := sys.FunderB.NewFundedEOA(eth.ZeroWei)
	sendRcpt := contract.Write(sender, bridgeA.SendETH(recipient.Address(), chainB), txplan.WithValue(eth.HalfEther))
	sent := sentMessageFrom(t, sendRcpt, chainA)

	// B exports that it never relayed the message, to A's L1CrossDomainMessenger.
	exporter := sys.FunderB.NewFundedEOA(eth.OneEther)
	exportRcpt := contract.Write(exporter, messengerB.ExportUndeliveredMessage(
		l1MessengerA, chainA, sent.nonce, sent.sender, sent.target, sent.message, minGasLimit))

	// Finalizing the withdrawal relays it to A's L1CrossDomainMessenger, which deposits it into A.
	bridge := sys.StandardBridge(sys.L2ChainB)
	withdrawal := bridge.WithdrawalFromReceipt(exportRcpt)
	withdrawal.Prove(l1User)
	withdrawal.WaitForDisputeGameResolvedWithin(bridge.GameResolutionDelay() + time.Minute)
	withdrawal.Finalize(l1User)

	// A's messenger rejects it: 7 days have not passed.
	depositRcpt := awaitDeposit(t, sys.L2ELA, withdrawal.FinalizeReceipt())
	require.Equal(types.ReceiptStatusSuccessful, depositRcpt.Status, "the forwarded deposit must execute on A")
	require.True(hasLog(depositRcpt, predeploys.L2CrossDomainMessengerAddr, failedRelayedMessageTopic),
		"the L2CrossDomainMessenger must keep the rejected call as a failed message")
	requireExpireRejected(t, sys.L2ELA, depositRcpt.TxHash)
	require.False(contract.Read(messengerA.ExpiredMessages(sent.hash)), "the message must not expire early")
	_, err := contractio.Read(bridgeA.RefundETH(chainB, sent.nonce, sender.Address(), recipient.Address(), eth.HalfEther.ToBig()), t.Ctx())
	require.Error(err, "an unexpired send must not be refundable")

	// A delivered message cannot be exported as undelivered.
	_, err = contractio.Read(messengerB.ExportUndeliveredMessage(
		l1MessengerA, chainA, delivered.nonce, delivered.sender, delivered.target, delivered.message, minGasLimit), t.Ctx())
	require.Error(err, "exporting a delivered message must revert")
	require.Contains(errutil.TryAddRevertReason(err).Error(), messageAlreadyRelayed)
}

// sentMessage is a message read from a SentMessage event.
type sentMessage struct {
	hash    common.Hash
	nonce   *big.Int
	sender  common.Address
	target  common.Address
	message []byte
}

// sentMessageFrom reads the L2ToL2CrossDomainMessenger's SentMessage event from a receipt and
// computes the message hash.
func sentMessageFrom(t devtest.T, rcpt *types.Receipt, source eth.ChainID) sentMessage {
	for _, l := range rcpt.Logs {
		if l.Address != predeploys.L2toL2CrossDomainMessengerAddr || l.Topics[0] != sentMessageTopic {
			continue
		}
		data, err := abi.Arguments{{Type: abiType(t, "address")}, {Type: abiType(t, "bytes")}}.Unpack(l.Data)
		t.Require().NoError(err)
		destination := l.Topics[1].Big()
		m := sentMessage{
			nonce:   l.Topics[3].Big(),
			sender:  data[0].(common.Address),
			target:  common.BytesToAddress(l.Topics[2].Bytes()),
			message: data[1].([]byte),
		}
		encoded, err := abi.Arguments{
			{Type: abiType(t, "uint256")}, {Type: abiType(t, "uint256")}, {Type: abiType(t, "uint256")},
			{Type: abiType(t, "address")}, {Type: abiType(t, "address")}, {Type: abiType(t, "bytes")},
		}.Pack(destination, source.ToBig(), m.nonce, m.sender, m.target, m.message)
		t.Require().NoError(err)
		m.hash = crypto.Keccak256Hash(encoded)
		return m
	}
	t.Require().Fail("no SentMessage event in the receipt")
	return sentMessage{}
}

func abiType(t devtest.T, name string) abi.Type {
	typ, err := abi.NewType(name, "", nil)
	t.Require().NoError(err)
	return typ
}

func messenger(t devtest.T, el *dsl.L2ELNode) bindings.L2ToL2CrossDomainMessenger {
	return bindings.NewBindings[bindings.L2ToL2CrossDomainMessenger](
		bindings.WithClient(el.EthClient()), bindings.WithTo(predeploys.L2toL2CrossDomainMessengerAddr), bindings.WithTest(t))
}

// rawCall is a txintent.Call over pre-encoded calldata, so a contract call can drive the
// txintent flow that RelayIndexed builds the executing message from.
type rawCall struct {
	to   common.Address
	data []byte
}

func (c *rawCall) To() (*common.Address, error)          { return &c.to, nil }
func (c *rawCall) EncodeInput() ([]byte, error)          { return c.data, nil }
func (c *rawCall) AccessList() (types.AccessList, error) { return nil, nil }

// sendAndRelayETH bridges ETH from A to B and relays the message on B.
func sendAndRelayETH(t devtest.T, sys *presets.SimpleInterop, sender, recipient, relayer *dsl.EOA, amount eth.ETH) sentMessage {
	bridgeA := bindings.NewBindings[bindings.SuperchainETHBridge](
		bindings.WithClient(sys.L2ELA.EthClient()), bindings.WithTo(predeploys.SuperchainETHBridgeAddr), bindings.WithTest(t))
	sendCall := bridgeA.SendETH(recipient.Address(), recipient.ChainID())
	calldata, err := sendCall.EncodeInput()
	t.Require().NoError(err)

	sendTx := txintent.NewIntent[*rawCall, *txintent.InteropOutput](sender.Plan(), txplan.WithValue(amount))
	sendTx.Content.Set(&rawCall{to: predeploys.SuperchainETHBridgeAddr, data: calldata})
	sendRcpt, err := sendTx.PlannedTx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "sendETH receipt not found")
	sent := sentMessageFrom(t, sendRcpt, sys.L2ChainA.ChainID())

	// One block lets the supernode index the initiating message.
	sys.L2ChainA.WaitForBlock()

	// Log index 1 is the SentMessage, after the ETHLiquidity burn.
	relayTx := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](relayer.Plan())
	relayTx.Content.DependOn(&sendTx.Result)
	relayTx.Content.Fn(txintent.RelayIndexed(
		predeploys.L2toL2CrossDomainMessengerAddr, &sendTx.Result, &sendTx.PlannedTx.Included, 1))
	relayRcpt, err := relayTx.PlannedTx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "relay receipt not found")
	t.Require().Equal(types.ReceiptStatusSuccessful, relayRcpt.Status, "relay must succeed")
	return sent
}

// awaitDeposit follows the deposit an L1 transaction made through to its execution on L2.
func awaitDeposit(t devtest.T, el *dsl.L2ELNode, l1Rcpt *types.Receipt) *types.Receipt {
	var depositTxHash common.Hash
	for _, l := range l1Rcpt.Logs {
		if depositTx, err := derive.UnmarshalDepositLogEvent(l); err == nil {
			depositTxHash = depositTx.Hash()
			break
		}
	}
	t.Require().NotEqual(common.Hash{}, depositTxHash, "no TransactionDeposited event in the L1 receipt")
	el.WaitL1OriginReached(eth.Unsafe, bigs.Uint64Strict(l1Rcpt.BlockNumber), 120)
	return el.WaitForReceipt(depositTxHash)
}

func hasLog(rcpt *types.Receipt, emitter common.Address, topic0 common.Hash) bool {
	for _, l := range rcpt.Logs {
		if l.Address == emitter && len(l.Topics) > 0 && l.Topics[0] == topic0 {
			return true
		}
	}
	return false
}

// callFrame is a frame of a callTracer trace.
type callFrame struct {
	To     common.Address `json:"to"`
	Input  hexutil.Bytes  `json:"input"`
	Output hexutil.Bytes  `json:"output"`
	Calls  []callFrame    `json:"calls"`
}

// requireExpireRejected traces the deposit and checks that the messenger's expireMessage was
// reached, past its sender check, and reverted with MessageNotExpired.
func requireExpireRejected(t devtest.T, el *dsl.L2ELNode, txHash common.Hash) {
	var trace callFrame
	err := el.EthClient().RPC().CallContext(t.Ctx(), &trace, "debug_traceTransaction", txHash,
		map[string]any{"tracer": "callTracer", "tracerConfig": map[string]any{}})
	t.Require().NoError(err, "failed to trace the deposit")
	var find func(f callFrame) bool
	find = func(f callFrame) bool {
		if f.To == predeploys.L2toL2CrossDomainMessengerAddr && bytes.HasPrefix(f.Input, expireMessageSelector) {
			return bytes.Equal(f.Output, messageNotExpiredSelector)
		}
		for _, c := range f.Calls {
			if find(c) {
				return true
			}
		}
		return false
	}
	t.Require().True(find(trace), "expireMessage must revert with MessageNotExpired")
}
