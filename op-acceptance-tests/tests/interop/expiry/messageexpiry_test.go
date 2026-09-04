// Package expiry covers the interop message-expiry refund path: a cross-chain ETH send that is
// never relayed on its destination chain is proven undelivered there, carried to L1 through the
// withdrawal path, and forwarded back into the source chain through the deposit path, where the
// sending application refunds it.
package expiry

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	// expiryWindowSeconds is the interop message expiry window of the dependency set, and the
	// window the MessageExpiryRelay is initialized with on both chains. The relay's window must
	// be >= the protocol window, or an expiry could be consumed while delivery is still possible.
	expiryWindowSeconds = 12

	// relayMinGasLimit is the gas reserved for the cross-domain calls this test triggers: the
	// hub's receiveExpiryNotice on L1 and the relay's receiveExpiry (plus the bridge's refund
	// callback) on L2. Generous enough that neither lands in failed-messages on the first try.
	relayMinGasLimit = uint32(500_000)

	// Dispute-game timings. The refund's last leg is a deposit into chain A, and devstack time
	// travel advances L1 only: a large AdvanceTime jump would stall L2 origin adoption (and
	// therefore that deposit) for the size of the jump. So this test skips time travel entirely
	// and instead shrinks the L1 windows until every one of them can be waited out in
	// wall-clock time.
	//
	// The game contracts require
	// max(2*clockExtension, clockExtension+preimageOracleChallengePeriod) <= maxClockDuration,
	// enforced in the game constructor and again in initialize(), so all three move together.
	proofMaturityDelaySeconds       = 4
	disputeGameFinalityDelaySeconds = 4
	faultGameMaxClockDuration       = 40
	faultGameClockExtension         = 1
	preimageOracleChallengePeriod   = 10
)

var (
	// expiredSendAmount is refunded in full to its sender, which must make no further
	// transactions after the send so the balance assertion can be exact.
	expiredSendAmount   = eth.HalfEther
	deliveredSendAmount = eth.OneHundredthEther

	sentMessageRecordedTopic        = crypto.Keccak256Hash([]byte("SentMessageRecorded(bytes32,address,uint256,uint256)"))
	undeliveredMessageAttestedTopic = crypto.Keccak256Hash([]byte("UndeliveredMessageAttested(bytes32,uint256,uint256)"))
	expiredMessageRelayedTopic      = crypto.Keccak256Hash([]byte("ExpiredMessageRelayed(bytes32,address,uint256,uint256)"))
	refundETHTopic                  = crypto.Keccak256Hash([]byte("RefundETH(address,uint256,bytes32)"))

	messageDeliveredSelector = hexutil.Encode(crypto.Keccak256([]byte("MessageExpiryRelay_MessageDelivered()"))[:4])
)

// TestInteropMessageExpiryRefund proves the full expiry-refund loop end to end:
//
//	A: sendETH -> (never relayed on B) -> B: attestUndelivered -> L1 withdrawal
//	-> hub notice -> forwardExpiryNotice -> A deposit -> refund to the sender
//
// It also proves that ordinary delivery still works with the 1.1.0 SuperchainETHBridge and that
// a delivered message can never be attested as undelivered.
func TestInteropMessageExpiryRefund(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithMessageExpiryWindow(expiryWindowSeconds),
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
	t.Logger().Info("interop expiry system up", "chainA", chainA, "chainB", chainB)

	// ── 1. Deploy the MessageExpiryHub on L1 ────────────────────────────────────────────────
	// The hub is an ownerless singleton that no deployment pipeline installs yet.
	l1User := sys.FunderL1.NewFundedEOA(eth.OneEther)
	hubAddr := deployMessageExpiryHub(t, l1User)
	hub := bindings.NewBindings[bindings.MessageExpiryHub](
		bindings.WithClient(sys.L1EL.EthClient()),
		bindings.WithTo(hubAddr),
		bindings.WithTest(t))
	t.Logger().Info("deployed MessageExpiryHub", "address", hubAddr, "version", contract.Read(hub.Version()))

	// ── 2. Initialize the relay proxy on both L2s ───────────────────────────────────────────
	// The predeploy ships uninitialized by design; only the L2 ProxyAdmin owner can set it up.
	relayA := initializeRelay(t, sys.L2ChainA, sys.L2ELA, sys.FunderA, hubAddr)
	relayB := initializeRelay(t, sys.L2ChainB, sys.L2ELB, sys.FunderB, hubAddr)

	// ── 3. Register both chains with the hub (permissionless) ───────────────────────────────
	for _, chain := range []*dsl.L2Network{sys.L2ChainA, sys.L2ChainB} {
		systemConfig := chain.Escape().Deployment().SystemConfigProxyAddr()
		rcpt := contract.Write(l1User, hub.RegisterChain(systemConfig))
		require.Equal(types.ReceiptStatusSuccessful, rcpt.Status, "registerChain must succeed for %s", chain)
	}
	// Both chains share one ETHLockbox: it is the cluster identity the hub keys notices by.
	lockbox := ethLockboxOf(t, sys.L1EL, sys.L2ChainA)
	require.Equal(lockbox, ethLockboxOf(t, sys.L1EL, sys.L2ChainB),
		"interop cluster chains must share one ETHLockbox")
	require.Equal(sys.L2ChainB.Escape().Deployment().SystemConfigProxyAddr(),
		contract.Read(hub.RegisteredChains(lockbox, chainB)), "chain B must be registered")
	require.Equal(sys.L2ChainA.Escape().Deployment().SystemConfigProxyAddr(),
		contract.Read(hub.RegisteredChains(lockbox, chainA)), "chain A must be registered")

	bridgeA := ethBridge(t, sys.L2ELA)
	require.Equal("1.1.0", contract.Read(bridgeA.Version()), "expected the expiry-aware ETH bridge")

	// ── 4a. Regression guard: an ordinary A->B send still relays on B ───────────────────────
	// Done first so the delivered message is available for the negative assertion at the end.
	deliveredSender := sys.FunderA.NewFundedEOA(eth.OneEther)
	deliveredRecipient := sys.FunderB.NewFundedEOA(eth.ZeroWei)
	relayer := sys.FunderB.NewFundedEOA(eth.OneEther)
	deliveredHash := sendAndRelayETH(t, sys, deliveredSender, deliveredRecipient, relayer, deliveredSendAmount)
	deliveredRecipient.VerifyBalanceExact(deliveredSendAmount)
	// Delivery does not clear the source chain's pending record — nothing on A observes the
	// relay. Mutual exclusion comes from the destination chain refusing to attest a delivered
	// message, which step 11 checks.
	require.Equal(deliveredSender.Address(), contract.Read(bridgeA.PendingETHSends(deliveredHash)).From,
		"a delivered send keeps its pending record on the source chain")

	// ── 4b. The send that will expire: never relayed on B ───────────────────────────────────
	expiredSender := sys.FunderA.NewFundedEOA(eth.OneEther)
	expiredRecipient := sys.FunderB.NewFundedEOA(eth.ZeroWei)
	sendRcpt := contract.Write(expiredSender, bridgeA.SendETH(expiredRecipient.Address(), chainB),
		txplan.WithValue(expiredSendAmount))
	require.Equal(types.ReceiptStatusSuccessful, sendRcpt.Status, "sendETH must succeed")
	expiredHash := msgHashFromSend(t, sendRcpt)
	sendTimestamp := sys.L2ChainA.TimestampForBlockNum(bigs.Uint64Strict(sendRcpt.BlockNumber))
	// Balance after the send is the baseline: the sender must not transact again, so the refund
	// is the only thing that can move it.
	balanceBeforeRefund := expiredSender.GetBalance()
	t.Logger().Info("sent ETH that will expire", "msgHash", expiredHash, "sendTimestamp", sendTimestamp)

	pending := contract.Read(bridgeA.PendingETHSends(expiredHash))
	require.Equal(expiredSender.Address(), pending.From, "pending send must record the sender")
	require.Zero(expiredSendAmount.ToBig().Cmp(pending.Amount), "pending send must record the amount")
	record := contract.Read(relayA.SentMessageRecords(expiredHash))
	require.Equal(predeploys.SuperchainETHBridgeAddr, record.App, "the bridge must own the expiry record")
	require.Zero(chainB.ToBig().Cmp(record.Destination), "record must name chain B as the destination")
	require.Zero(record.RecordedAt.Cmp(new(big.Int).SetUint64(sendTimestamp)), "record must be stamped at the send")

	// ── 5. Wait past the expiry window, measured on chain B ─────────────────────────────────
	// attestUndelivered stamps chain B's timestamp; chain A accepts it once it exceeds the
	// recorded send time plus the relay's window. One extra B block of slack.
	blockTimeB := sys.L2ChainB.Escape().RollupConfig().BlockTime
	require.NotZero(blockTimeB, "chain B block time must be non-zero")
	attestAfter := sendTimestamp + expiryWindowSeconds + blockTimeB
	t.Logger().Info("waiting for chain B to pass the expiry window", "until", attestAfter)
	sys.L2ELB.WaitForTime(attestAfter)

	// ── 6. Attest non-delivery on B (permissionless) ────────────────────────────────────────
	attestor := sys.FunderB.NewFundedEOA(eth.OneEther)
	attestRcpt := contract.Write(attestor, relayB.AttestUndelivered(expiredHash, chainA, relayMinGasLimit))
	require.Equal(types.ReceiptStatusSuccessful, attestRcpt.Status, "attestUndelivered must succeed")
	require.NotNil(findLog(attestRcpt, predeploys.MessageExpiryRelayAddr, undeliveredMessageAttestedTopic, expiredHash),
		"attestUndelivered must emit UndeliveredMessageAttested")

	// ── 7. Prove + finalize the resulting withdrawal from B on L1 ───────────────────────────
	bridge := sys.StandardBridge(sys.L2ChainB)
	require.True(bridge.UsesSuperRoots(), "expected the interop system to use super roots")
	withdrawal := bridge.WithdrawalFromReceipt(attestRcpt)
	withdrawal.Prove(l1User)
	// No time travel here (see the timings block above): the game's chess clock is waited out in
	// wall-clock time, so the budget must exceed the max clock duration.
	t.Logger().Info("waiting for the dispute game to resolve", "gameResolutionDelay", bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolvedWithin(bridge.GameResolutionDelay() + time.Minute)
	withdrawal.Finalize(l1User)

	// ── 8. The hub has the notice ───────────────────────────────────────────────────────────
	// Notices are keyed by cluster (the shared lockbox), attestor chain, source chain and
	// message hash, so finding one under chain A's key is what says it routes back to A.
	notice := contract.Read(hub.Notices(lockbox, chainB, chainA, expiredHash))
	require.NotZero(notice.AttestedAt,
		"hub must hold the expiry notice; if the finalize's inner call ran out of gas the message "+
			"is still replayable on B's L1CrossDomainMessenger")
	require.Greater(notice.AttestedAt, sendTimestamp+expiryWindowSeconds,
		"attestation must be beyond the send time plus the expiry window")

	// ── 9. Forward the notice to chain A through the deposit path ───────────────────────────
	forwardRcpt := contract.Write(l1User, hub.ForwardExpiryNotice(lockbox, chainB, chainA, expiredHash, relayMinGasLimit))
	require.Equal(types.ReceiptStatusSuccessful, forwardRcpt.Status, "forwardExpiryNotice must succeed")
	depositRcpt := awaitDeposit(t, sys.L2ELA, forwardRcpt)
	require.Equal(types.ReceiptStatusSuccessful, depositRcpt.Status, "the forwarded deposit must execute on A")

	// ── 10. The sender got its ETH back, and the records are gone ───────────────────────────
	require.NotNil(findLog(depositRcpt, predeploys.MessageExpiryRelayAddr, expiredMessageRelayedTopic, expiredHash),
		"relay must emit ExpiredMessageRelayed; an undergassed relay would leave the message replayable")
	refundLog := findLog(depositRcpt, predeploys.SuperchainETHBridgeAddr, refundETHTopic, common.Hash{})
	require.NotNil(refundLog, "bridge must emit RefundETH")
	require.Equal(expiredHash, refundLog.Topics[2], "refund must name the expired message")
	require.Equal(expiredSender.Address(), common.BytesToAddress(refundLog.Topics[1].Bytes()),
		"refund must go to the sender, not the intended recipient")

	// Exactly the sent amount, back to the sender: the refund is a forced SafeSend, and the
	// sender has made no transaction since the send, so nothing else can have moved its balance.
	expiredSender.WaitForBalance(balanceBeforeRefund.Add(expiredSendAmount))
	expiredSender.VerifyBalanceExact(balanceBeforeRefund.Add(expiredSendAmount))
	expiredRecipient.VerifyBalanceExact(eth.ZeroWei)
	require.Equal(common.Address{}, contract.Read(bridgeA.PendingETHSends(expiredHash)).From,
		"the pending send must be consumed exactly once")
	require.Equal(common.Address{}, contract.Read(relayA.SentMessageRecords(expiredHash)).App,
		"the expiry record must be consumed exactly once")

	// ── 11. A delivered message can never be attested as undelivered ────────────────────────
	_, err := contractio.Read(relayB.AttestUndelivered(deliveredHash, chainA, relayMinGasLimit), t.Ctx(),
		txplan.WithSender(attestor.Address()))
	require.Error(err, "attesting a delivered message must revert")
	revertErr := errutil.TryAddRevertReason(err)
	t.Logger().Info("attesting a delivered message reverted", "err", revertErr)
	require.Contains(revertErr.Error(), messageDeliveredSelector,
		"expected MessageExpiryRelay_MessageDelivered (%s)", messageDeliveredSelector)
}

// deployMessageExpiryHub deploys the hub from its checked-in creation bytecode and returns its
// address. The hub takes no constructor arguments.
func deployMessageExpiryHub(t devtest.T, deployer *dsl.EOA) common.Address {
	tx := txplan.NewPlannedTx(deployer.Plan(), txplan.WithData(common.FromHex(bindings.MessageExpiryHubBin)))
	rcpt, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err, "failed to deploy MessageExpiryHub")
	t.Require().Equal(types.ReceiptStatusSuccessful, rcpt.Status, "MessageExpiryHub deployment reverted")
	t.Require().NotEqual(common.Address{}, rcpt.ContractAddress, "MessageExpiryHub has no address")
	return rcpt.ContractAddress
}

// initializeRelay points a chain's MessageExpiryRelay predeploy at the hub and gives it the
// expiry window. The proxy ships uninitialized, and initialize is gated to the L2 ProxyAdmin
// owner, so the devkey-derived owner is funded on the chain first.
func initializeRelay(
	t devtest.T,
	chain *dsl.L2Network,
	el *dsl.L2ELNode,
	funder *dsl.FunderEOA,
	hubAddr common.Address,
) bindings.MessageExpiryRelay {
	require := t.Require()
	owner := dsl.NewKey(t, chain.Escape().Keys().Secret(devkeys.L2ProxyAdminOwnerRole.Key(chain.ChainID().ToBig()))).User(el)
	proxyAdmin := bindings.NewBindings[bindings.ProxyAdmin](
		bindings.WithClient(el.EthClient()),
		bindings.WithTo(predeploys.ProxyAdminAddr),
		bindings.WithTest(t))
	require.Equal(owner.Address(), contract.Read(proxyAdmin.Owner()),
		"devkey-derived L2 proxy-admin owner must match the ProxyAdmin's owner on %s", chain)
	funder.FundAtLeast(owner, eth.OneTenthEther)

	relay := bindings.NewBindings[bindings.MessageExpiryRelay](
		bindings.WithClient(el.EthClient()),
		bindings.WithTo(predeploys.MessageExpiryRelayAddr),
		bindings.WithTest(t))
	require.Equal(common.Address{}, contract.Read(relay.Hub()), "relay proxy must ship uninitialized on %s", chain)

	rcpt := contract.Write(owner, relay.Initialize(hubAddr, big.NewInt(expiryWindowSeconds)))
	require.Equal(types.ReceiptStatusSuccessful, rcpt.Status, "relay initialize must succeed on %s", chain)
	require.Equal(hubAddr, contract.Read(relay.Hub()))
	// Invariant: the relay's window must be at least the dependency set's message expiry window,
	// or an expiry could be consumed while delivery on the destination chain is still possible.
	require.GreaterOrEqual(contract.Read(relay.ExpiryWindow()).Uint64(), uint64(expiryWindowSeconds))
	t.Logger().Info("initialized MessageExpiryRelay", "chain", chain, "hub", hubAddr, "owner", owner.Address())
	return relay
}

func ethBridge(t devtest.T, el *dsl.L2ELNode) bindings.SuperchainETHBridge {
	return bindings.NewBindings[bindings.SuperchainETHBridge](
		bindings.WithClient(el.EthClient()),
		bindings.WithTo(predeploys.SuperchainETHBridgeAddr),
		bindings.WithTest(t))
}

// ethLockboxOf reads the shared ETHLockbox authorizing a chain's portal. The hub uses it as the
// cluster identity that notices are namespaced by.
func ethLockboxOf(t devtest.T, l1EL *dsl.L1ELNode, chain *dsl.L2Network) common.Address {
	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(l1EL.EthClient()),
		bindings.WithTo(chain.DepositContractAddr()),
		bindings.WithTest(t))
	lockbox := contract.Read(portal.EthLockbox())
	t.Require().NotEqual(common.Address{}, lockbox, "chain %s has no ETHLockbox", chain)
	return lockbox
}

// rawCall is a txintent.Call over pre-encoded calldata, so an ordinary contract call can drive a
// txintent flow (which is what RelayIndexed needs to build the executing message).
type rawCall struct {
	to   common.Address
	data []byte
}

func (c *rawCall) To() (*common.Address, error)          { return &c.to, nil }
func (c *rawCall) EncodeInput() ([]byte, error)          { return c.data, nil }
func (c *rawCall) AccessList() (types.AccessList, error) { return nil, nil }

// sendAndRelayETH performs a full A->B ETH bridge, explicitly relaying the message on B (there is
// no auto-relayer), and returns the message hash.
func sendAndRelayETH(
	t devtest.T,
	sys *presets.SimpleInterop,
	sender *dsl.EOA,
	recipient *dsl.EOA,
	relayer *dsl.EOA,
	amount eth.ETH,
) common.Hash {
	require := t.Require()
	sendCall := ethBridge(t, sys.L2ELA).SendETH(recipient.Address(), recipient.ChainID())
	calldata, err := sendCall.EncodeInput()
	require.NoError(err, "failed to encode sendETH")

	sendTx := txintent.NewIntent[*rawCall, *txintent.InteropOutput](sender.Plan(), txplan.WithValue(amount))
	sendTx.Content.Set(&rawCall{to: predeploys.SuperchainETHBridgeAddr, data: calldata})
	sendRcpt, err := sendTx.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err, "sendETH receipt not found")
	require.Equal(types.ReceiptStatusSuccessful, sendRcpt.Status, "sendETH must succeed")
	msgHash := msgHashFromSend(t, sendRcpt)

	// One block of slack is all the supernode needs to index the initiating message (see
	// tests/interop/message). Waiting on super-root validation instead would blow the 12s
	// message expiry window this system runs with, and the message would expire mid-relay.
	sys.L2ChainA.WaitForBlock()

	// Log index 1 is the L2ToL2CrossDomainMessenger's SentMessage; see msgHashFromSend.
	relayTx := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](relayer.Plan())
	relayTx.Content.DependOn(&sendTx.Result)
	relayTx.Content.Fn(txintent.RelayIndexed(
		predeploys.L2toL2CrossDomainMessengerAddr,
		&sendTx.Result,
		&sendTx.PlannedTx.Included,
		1,
	))
	relayRcpt, err := relayTx.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err, "relayETH receipt not found")
	require.Equal(types.ReceiptStatusSuccessful, relayRcpt.Status, "relayETH must succeed")
	t.Logger().Info("relayed ETH", "msgHash", msgHash, "send", sendRcpt.TxHash, "relay", relayRcpt.TxHash)
	return msgHash
}

// msgHashFromSend reads the sent message hash out of a sendETH receipt. The 1.1.0 bridge emits,
// in order: ETHLiquidity burn, L2ToL2CrossDomainMessenger SentMessage, MessageExpiryRelay
// SentMessageRecorded, SuperchainETHBridge SendETH. The recorded event carries the hash as its
// first indexed topic, and its presence is itself proof that the bridge registered the send for
// expiry handling.
func msgHashFromSend(t devtest.T, rcpt *types.Receipt) common.Hash {
	t.Require().Len(rcpt.Logs, 4, "sendETH must emit burn, sendMessage, sentMessageRecorded and sendETH logs")
	for idx, addr := range []common.Address{
		predeploys.ETHLiquidityAddr,
		predeploys.L2toL2CrossDomainMessengerAddr,
		predeploys.MessageExpiryRelayAddr,
		predeploys.SuperchainETHBridgeAddr,
	} {
		t.Require().Equal(addr, rcpt.Logs[idx].Address, "unexpected emitter for sendETH log %d", idx)
	}
	recorded := findLog(rcpt, predeploys.MessageExpiryRelayAddr, sentMessageRecordedTopic, common.Hash{})
	t.Require().NotNil(recorded, "sendETH must record the message with the MessageExpiryRelay")
	return recorded.Topics[1]
}

// awaitDeposit follows the deposit an L1 transaction initiated through to its execution on L2.
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

// findLog returns the first log from emitter with the given event topic, optionally also matching
// topic[1]. It returns nil when there is no match.
func findLog(rcpt *types.Receipt, emitter common.Address, topic0 common.Hash, topic1 common.Hash) *types.Log {
	for _, l := range rcpt.Logs {
		if l.Address != emitter || len(l.Topics) == 0 || l.Topics[0] != topic0 {
			continue
		}
		if topic1 != (common.Hash{}) && (len(l.Topics) < 2 || l.Topics[1] != topic1) {
			continue
		}
		return l
	}
	return nil
}
