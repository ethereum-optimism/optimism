package monitor

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Names of the checks, used as the `check` metric label.
const (
	checkExpiry       = "expiry"
	checkRefund       = "refund"
	checkConservation = "conservation"
)

// Reasons a check reports a violation, used as the `reason` metric label.
const (
	// The event could not be decoded.
	reasonMalformedEvent = "malformed_event"
	// The source chain has no send time for the message: it never sent it, or sent it before
	// send times were recorded, in which case it can never expire.
	reasonUnknownMessage = "unknown_message"
	// No SentMessage in the block at the recorded send time rebuilds to the message hash.
	reasonHashMismatch = "hash_mismatch"
	// The message is not marked expired at the block of its MessageExpired event.
	reasonExpiredFlagUnset = "expired_flag_unset"
	// The message expired on a fact from before the end of its expiry period.
	reasonEarlyExpiry = "early_expiry"
	// The message expired on a fact from after the source block that marked it.
	reasonUndeliveredAtInFuture = "undelivered_at_in_future"
	// The message's destination is not a monitored chain. Only a chain of the source's cluster
	// can export a fact the source accepts, so the monitor must watch every chain of the cluster.
	reasonUnknownDestination = "unknown_destination"
	// The destination relayed the message within the window.
	reasonRelayedInWindow = "relayed_in_window"
	// The destination's messenger records the message as relayed.
	reasonRelayedAtHead = "relayed_at_head"
	// The destination exported no undelivered fact for the message at the timestamp it expired on.
	reasonNoExport = "no_export"
	// A refund paid out for a message that is not marked expired.
	reasonRefundWithoutExpiry = "refund_without_expiry"
	// A refund paid out for a message that is not a SuperchainETHBridge send.
	reasonNotBridgeSend = "not_bridge_send"
	// A refund's sender or amount differs from the send's.
	reasonRefundMismatch = "refund_mismatch"
	// The send transaction burned no matching ETH.
	reasonNoBurn = "no_burn"
	// The send was refunded more than once.
	reasonDuplicateRefund = "duplicate_refund"
	// The cluster's bridge events show more ETH relayed and refunded than sent.
	reasonNegativeInFlight = "negative_in_flight"
	// ETHLiquidity's balance fell by more than its bridge events account for.
	reasonUnexplainedOutflow = "unexplained_outflow"
	// A transaction burned into ETHLiquidity without a matching SendETH.
	reasonUnpairedBurn = "unpaired_burn"
	// A transaction minted from ETHLiquidity without a matching RelayETH or RefundETH.
	reasonUnpairedMint = "unpaired_mint"
)

// violation is a broken safety rule found by a check.
type violation struct {
	reason string
	detail string
}

func violationf(reason string, format string, args ...any) violation {
	return violation{reason: reason, detail: fmt.Sprintf(format, args...)}
}

// errNotReady means a check needs chain data that is not safe yet; it is retried later.
var errNotReady = errors.New("chain data not ready")

// expiryVerifier holds the reads the expiry and refund checks share.
type expiryVerifier struct {
	clients             map[eth.ChainID]ExpiryClient
	messageExpiryWindow uint64
	// head is the label of the blocks the checks read up to: finalized, or safe.
	head eth.BlockLabel
}

// sentMessage is a SentMessage located by its hash on its source chain.
type sentMessage struct {
	msg     *l2ToL2Message
	block   eth.BlockInfo
	receipt *types.Receipt
}

func (v *expiryVerifier) readWord(ctx context.Context, chainID eth.ChainID, addr common.Address, slot common.Hash, block uint64) (*big.Int, error) {
	word, err := storageAt(ctx, v.clients[chainID], addr, slot, block)
	if err != nil {
		return nil, fmt.Errorf("read slot %s of %s on chain %s at block %d: %w", slot, addr, chainID, block, err)
	}
	return word.Big(), nil
}

// locateSentMessage finds the SentMessage of `hash` on `chainID`, sent at `sentAt`. It rebuilds
// the hash of every SentMessage in the block at that time, rather than trusting any stored
// mapping from hash to message.
func (v *expiryVerifier) locateSentMessage(ctx context.Context, chainID eth.ChainID, hash common.Hash, sentAt uint64, at eth.BlockInfo) (*sentMessage, *violation, error) {
	c := v.clients[chainID]
	block, err := blockAtOrBefore(ctx, c, sentAt, at)
	if err != nil {
		return nil, nil, fmt.Errorf("find block at send time %d on chain %s: %w", sentAt, chainID, err)
	}
	if block.Time() != sentAt {
		viol := violationf(reasonHashMismatch, "chain %s has no block at the recorded send time %d of message %s", chainID, sentAt, hash)
		return nil, &viol, nil
	}
	info, receipts, err := c.FetchReceiptsByNumber(ctx, block.NumberU64())
	if err != nil {
		return nil, nil, fmt.Errorf("fetch receipts of block %d on chain %s: %w", block.NumberU64(), chainID, err)
	}
	if info.Hash() != block.Hash() {
		return nil, nil, fmt.Errorf("block %d on chain %s changed while reading it", block.NumberU64(), chainID)
	}
	for _, rcpt := range receipts.Geth() {
		for _, l := range rcpt.Logs {
			if l.Address != predeploys.L2toL2CrossDomainMessengerAddr || len(l.Topics) == 0 || l.Topics[0] != sentMessageTopic {
				continue
			}
			msg, err := decodeSentMessage(l, chainID)
			if err != nil {
				continue
			}
			if msg.Hash() == hash {
				return &sentMessage{msg: msg, block: block, receipt: rcpt}, nil, nil
			}
		}
	}
	viol := violationf(reasonHashMismatch, "no SentMessage in block %d on chain %s rebuilds to message %s", block.NumberU64(), chainID, hash)
	return nil, &viol, nil
}

// verifyExpiry checks a MessageExpired event on `source`: the message was sent from `source`, its
// expiry period had passed at the destination time the fact names, its destination never relayed
// it, and its destination exported that fact.
func (v *expiryVerifier) verifyExpiry(ctx context.Context, source eth.ChainID, l *types.Log) ([]violation, error) {
	ev, err := decodeMessageExpired(l)
	if err != nil {
		return []violation{violationf(reasonMalformedEvent, "log %d in block %d: %v", l.Index, l.BlockNumber, err)}, nil
	}
	hash, t := ev.MessageHash, ev.UndeliveredAt
	c := v.clients[source]
	expiredIn, err := c.InfoByNumber(ctx, l.BlockNumber)
	if err != nil {
		return nil, err
	}
	var violations []violation

	// a. The source recorded when it sent the message.
	sentAtWord, err := v.readWord(ctx, source, predeploys.L2toL2CrossDomainMessengerAddr,
		mappingSlot(hash, sentMessageTimestampsSlot), l.BlockNumber)
	if err != nil {
		return nil, err
	}
	if sentAtWord.Sign() == 0 || !sentAtWord.IsUint64() {
		// expireMessage reverts for a message with no send time, including one sent before send
		// times were recorded, so this expiry should be impossible.
		return []violation{violationf(reasonUnknownMessage, "message %s expired on chain %s has send time %s", hash, source, sentAtWord)}, nil
	}
	sentAt := bigs.Uint64Strict(sentAtWord)
	expired, err := v.readWord(ctx, source, predeploys.L2toL2CrossDomainMessengerAddr,
		mappingSlot(hash, expiredMessagesSlot), l.BlockNumber)
	if err != nil {
		return nil, err
	}
	if expired.Sign() == 0 {
		violations = append(violations, violationf(reasonExpiredFlagUnset, "message %s is not marked expired at block %d on chain %s", hash, l.BlockNumber, source))
	}

	// b. A SentMessage at that time rebuilds to the hash.
	sent, viol, err := v.locateSentMessage(ctx, source, hash, sentAt, expiredIn)
	if err != nil {
		return nil, err
	}
	if viol != nil {
		return append(violations, *viol), nil
	}

	// c. The fact is from after the expiry period, and not from after the block that used it.
	period, err := v.readWord(ctx, source, predeploys.L2toL2CrossDomainMessengerAddr,
		common.BigToHash(big.NewInt(expiryPeriodSlot)), l.BlockNumber)
	if err != nil {
		return nil, err
	}
	if new(big.Int).SetUint64(t).Cmp(new(big.Int).Add(new(big.Int).SetUint64(sentAt), period)) <= 0 {
		violations = append(violations, violationf(reasonEarlyExpiry, "message %s sent at %d expired on undelivered time %d within expiry period %s", hash, sentAt, t, period))
	}
	if t > expiredIn.Time() {
		// The destination may never reach t, so report now rather than wait for it.
		return append(violations, violationf(reasonUndeliveredAtInFuture, "message %s expired at time %d on undelivered time %d", hash, expiredIn.Time(), t)), nil
	}

	dest := sent.msg.Destination
	cDest, ok := v.clients[dest]
	if !ok {
		return append(violations, violationf(reasonUnknownDestination, "message %s from chain %s is to unmonitored chain %s", hash, source, dest)), nil
	}
	destHead, err := cDest.InfoByLabel(ctx, v.head)
	if err != nil {
		return nil, err
	}
	// The export at t must be at or below the destination's head before the check can finish.
	if t > destHead.Time() {
		return nil, fmt.Errorf("%w: chain %s %s time %d is before undelivered time %d", errNotReady, dest, v.head, destHead.Time(), t)
	}

	// d. No RelayedMessage of the message on the destination within the window.
	relayFrom, err := blockAtOrBefore(ctx, cDest, sentAt, destHead)
	if err != nil {
		return nil, err
	}
	relayTo, err := blockAtOrBefore(ctx, cDest, min(sentAt+v.messageExpiryWindow, destHead.Time()), destHead)
	if err != nil {
		return nil, err
	}
	relays, err := filterLogsChunked(ctx, cDest, ethereum.FilterQuery{
		Addresses: []common.Address{predeploys.L2toL2CrossDomainMessengerAddr},
		Topics:    [][]common.Hash{{relayedMessageTopic}, {common.BigToHash(source.ToBig())}, nil, {hash}},
	}, relayFrom.NumberU64(), relayTo.NumberU64())
	if err != nil {
		return nil, err
	}
	if len(relays) > 0 {
		violations = append(violations, violationf(reasonRelayedInWindow, "expired message %s was relayed on chain %s in tx %s", hash, dest, relays[0].TxHash))
	}

	// e. The destination's messenger does not record the message as relayed.
	relayed, err := v.readWord(ctx, dest, predeploys.L2toL2CrossDomainMessengerAddr,
		mappingSlot(hash, successfulMessagesSlot), destHead.NumberU64())
	if err != nil {
		return nil, err
	}
	if relayed.Sign() != 0 {
		violations = append(violations, violationf(reasonRelayedAtHead, "expired message %s is marked relayed on chain %s at block %d", hash, dest, destHead.NumberU64()))
	}

	// f. The destination exported the fact, for this source, in its block at t.
	exportBlock, err := blockAtOrBefore(ctx, cDest, t, destHead)
	if err != nil {
		return nil, err
	}
	exported := false
	if exportBlock.Time() == t {
		info, receipts, err := cDest.FetchReceiptsByNumber(ctx, exportBlock.NumberU64())
		if err != nil {
			return nil, err
		}
		if info.Hash() != exportBlock.Hash() {
			return nil, fmt.Errorf("block %d on chain %s changed while reading it", exportBlock.NumberU64(), dest)
		}
		exported = hasExport(receipts.Geth(), hash, source, t)
	}
	if !exported {
		violations = append(violations, violationf(reasonNoExport, "chain %s exported no undelivered fact for message %s at time %d", dest, hash, t))
	}
	return violations, nil
}

func hasExport(receipts []*types.Receipt, hash common.Hash, source eth.ChainID, t uint64) bool {
	for _, rcpt := range receipts {
		for _, l := range rcpt.Logs {
			if l.Address != predeploys.UndeliveredMessageExporterAddr || len(l.Topics) == 0 || l.Topics[0] != undeliveredExpTopic {
				continue
			}
			export, err := decodeUndeliveredExport(l)
			if err != nil {
				continue
			}
			if export.MessageHash == hash && export.Source == source && export.UndeliveredAt.Cmp(new(big.Int).SetUint64(t)) == 0 {
				return true
			}
		}
	}
	return false
}

// verifyRefund checks a RefundETH event on `source`: the message is marked expired, it is a
// SuperchainETHBridge send of the refunded sender and amount, its transaction burned that amount,
// and it was refunded once.
func (v *expiryVerifier) verifyRefund(ctx context.Context, source eth.ChainID, l *types.Log) ([]violation, error) {
	ev, err := decodeRefundETH(l)
	if err != nil {
		return []violation{violationf(reasonMalformedEvent, "log %d in block %d: %v", l.Index, l.BlockNumber, err)}, nil
	}
	hash := ev.MessageHash
	c := v.clients[source]
	refundedIn, err := c.InfoByNumber(ctx, l.BlockNumber)
	if err != nil {
		return nil, err
	}

	// a. The message is marked expired.
	expired, err := v.readWord(ctx, source, predeploys.L2toL2CrossDomainMessengerAddr,
		mappingSlot(hash, expiredMessagesSlot), l.BlockNumber)
	if err != nil {
		return nil, err
	}
	if expired.Sign() == 0 {
		return []violation{violationf(reasonRefundWithoutExpiry, "refund of message %s in tx %s on chain %s, which is not expired", hash, l.TxHash, source)}, nil
	}

	// b. The message is a bridge send of the refunded sender and amount.
	sentAtWord, err := v.readWord(ctx, source, predeploys.L2toL2CrossDomainMessengerAddr,
		mappingSlot(hash, sentMessageTimestampsSlot), l.BlockNumber)
	if err != nil {
		return nil, err
	}
	if sentAtWord.Sign() == 0 || !sentAtWord.IsUint64() {
		return []violation{violationf(reasonUnknownMessage, "refunded message %s on chain %s has send time %s", hash, source, sentAtWord)}, nil
	}
	sent, viol, err := v.locateSentMessage(ctx, source, hash, bigs.Uint64Strict(sentAtWord), refundedIn)
	if err != nil {
		return nil, err
	}
	if viol != nil {
		return []violation{*viol}, nil
	}
	msg := sent.msg
	if msg.Sender != predeploys.SuperchainETHBridgeAddr || msg.Target != predeploys.SuperchainETHBridgeAddr {
		return []violation{violationf(reasonNotBridgeSend, "refunded message %s was sent by %s to %s", hash, msg.Sender, msg.Target)}, nil
	}
	from, to, amount, err := decodeRelayETH(msg.Message)
	if err != nil {
		return []violation{violationf(reasonNotBridgeSend, "refunded message %s: %v", hash, err)}, nil
	}
	var violations []violation
	if from != ev.From || amount.Cmp(ev.Amount) != 0 {
		violations = append(violations, violationf(reasonRefundMismatch, "refund of message %s paid %s to %s, but the send was %s from %s", hash, ev.Amount, ev.From, amount, from))
	}

	// c. The send transaction burned the amount.
	if !hasBurn(sent.receipt, from, to, amount, msg.Destination) {
		violations = append(violations, violationf(reasonNoBurn, "send tx %s of message %s burned no matching %s wei", sent.receipt.TxHash, hash, amount))
	}

	// d. Exactly one refund of the message, from the send up to this refund.
	refunds, err := filterLogsChunked(ctx, c, ethereum.FilterQuery{
		Addresses: []common.Address{predeploys.SuperchainETHBridgeAddr},
		Topics:    [][]common.Hash{{refundETHTopic}, nil, {hash}},
	}, sent.block.NumberU64(), l.BlockNumber)
	if err != nil {
		return nil, err
	}
	switch {
	case len(refunds) == 0:
		return nil, fmt.Errorf("refund log of message %s missing from logs of chain %s", hash, source)
	case len(refunds) > 1:
		violations = append(violations, violationf(reasonDuplicateRefund, "message %s was refunded %d times on chain %s, up to tx %s", hash, len(refunds), source, l.TxHash))
	}
	return violations, nil
}

// hasBurn reports whether a send transaction emitted a SendETH matching the message and an
// ETHLiquidity burn of the amount.
func hasBurn(rcpt *types.Receipt, from, to common.Address, amount *big.Int, dest eth.ChainID) bool {
	sendETH, burned := false, false
	for _, l := range rcpt.Logs {
		if l.Address == predeploys.SuperchainETHBridgeAddr && len(l.Topics) > 0 && l.Topics[0] == sendETHTopic {
			ev, err := decodeSendETH(l)
			if err == nil && ev.From == from && ev.To == to && ev.Amount.Cmp(amount) == 0 && ev.Destination.Cmp(dest.ToBig()) == 0 {
				sendETH = true
			}
		}
		if value, ok := liquidityAmount(l, liquidityBurnedTopic); ok && value.Cmp(amount) == 0 &&
			common.BytesToAddress(l.Topics[1].Bytes()) == predeploys.SuperchainETHBridgeAddr {
			burned = true
		}
	}
	return sendETH && burned
}
