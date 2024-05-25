package bridge

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/bigint"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/processors/contracts"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// L2ProcessInitiatedBridgeEvents will query the database for bridge events that have been initiated between
// the specified block range. This covers every part of the multi-layered stack:
//  1. OptimismPortal
//  2. L2CrossDomainMessenger
//  3. L2StandardBridge
func L2ProcessInitiatedBridgeEvents(log log.Logger, db *database.DB, metrics L2Metricer, l2Contracts config.L2Contracts, fromHeight, toHeight *big.Int) error {
	// (1) L2ToL1MessagePasser
	l2ToL1MPMessagesPassed, err := contracts.L2ToL1MessagePasserMessagePassedEvents(l2Contracts.L2ToL1MessagePasser, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(l2ToL1MPMessagesPassed) > 0 {
		log.Info("detected transaction withdrawals", "size", len(l2ToL1MPMessagesPassed))
	}

	withdrawnWEI := bigint.Zero
	messagesPassed := make(map[logKey]*contracts.L2ToL1MessagePasserMessagePassed, len(l2ToL1MPMessagesPassed))
	transactionWithdrawals := make([]database.L2TransactionWithdrawal, len(l2ToL1MPMessagesPassed))
	for i := range l2ToL1MPMessagesPassed {
		messagePassed := l2ToL1MPMessagesPassed[i]
		messagesPassed[logKey{messagePassed.Event.BlockHash, messagePassed.Event.LogIndex}] = &messagePassed
		withdrawnWEI = new(big.Int).Add(withdrawnWEI, messagePassed.Tx.Amount)

		transactionWithdrawals[i] = database.L2TransactionWithdrawal{
			WithdrawalHash:       messagePassed.WithdrawalHash,
			InitiatedL2EventGUID: messagePassed.Event.GUID,
			Nonce:                messagePassed.Nonce,
			GasLimit:             messagePassed.GasLimit,
			Tx:                   messagePassed.Tx,
		}
	}
	if len(messagesPassed) > 0 {
		if err := db.BridgeTransactions.StoreL2TransactionWithdrawals(transactionWithdrawals); err != nil {
			return err
		}

		// Convert the withdrawn WEI to ETH
		withdrawnETH, _ := bigint.WeiToETH(withdrawnWEI).Float64()
		metrics.RecordL2TransactionWithdrawals(len(transactionWithdrawals), withdrawnETH)
	}

	// (2) L2CrossDomainMessenger
	crossDomainSentMessages, err := contracts.CrossDomainMessengerSentMessageEvents("l2", l2Contracts.L2CrossDomainMessenger, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainSentMessages) > 0 {
		log.Info("detected sent messages", "size", len(crossDomainSentMessages))
	}

	sentMessages := make(map[logKey]*contracts.CrossDomainMessengerSentMessageEvent, len(crossDomainSentMessages))
	bridgeMessages := make([]database.L2BridgeMessage, len(crossDomainSentMessages))
	for i := range crossDomainSentMessages {
		sentMessage := crossDomainSentMessages[i]
		sentMessages[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex}] = &sentMessage

		// extract the withdrawal hash from the previous MessagePassed event
		messagePassed, ok := messagesPassed[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex - 1}]
		if !ok {
			return fmt.Errorf("expected MessagePassedEvent preceding SentMessage. tx_hash = %s", sentMessage.Event.TransactionHash)
		} else if messagePassed.Event.TransactionHash != sentMessage.Event.TransactionHash {
			return fmt.Errorf("correlated events tx hash mismatch. message_tx_hash = %s, withdraw_tx_hash = %s", sentMessage.Event.TransactionHash, messagePassed.Event.TransactionHash)
		}

		bridgeMessages[i] = database.L2BridgeMessage{TransactionWithdrawalHash: messagePassed.WithdrawalHash, BridgeMessage: sentMessage.BridgeMessage}
	}
	if len(bridgeMessages) > 0 {
		if err := db.BridgeMessages.StoreL2BridgeMessages(bridgeMessages); err != nil {
			return err
		}
		metrics.RecordL2CrossDomainSentMessages(len(bridgeMessages))
	}

	// (3) L2StandardBridge
	initiatedBridges, err := contracts.StandardBridgeInitiatedEvents("l2", l2Contracts.L2StandardBridge, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(initiatedBridges) > 0 {
		log.Info("detected bridge withdrawals", "size", len(initiatedBridges))
	}

	bridgedTokens := make(map[common.Address]int)
	bridgeWithdrawals := make([]database.L2BridgeWithdrawal, len(initiatedBridges))
	for i := range initiatedBridges {
		initiatedBridge := initiatedBridges[i]

		// extract the cross domain message hash & withdraw hash from the following events
		messagePassed, ok := messagesPassed[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex + 1}]
		if !ok {
			return fmt.Errorf("expected MessagePassed following BridgeInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		} else if messagePassed.Event.TransactionHash != initiatedBridge.Event.TransactionHash {
			return fmt.Errorf("correlated events tx hash mismatch. bridge_tx_hash = %s, withdraw_tx_hash = %s", initiatedBridge.Event.TransactionHash, messagePassed.Event.TransactionHash)
		}

		sentMessage, ok := sentMessages[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex + 2}]
		if !ok {
			return fmt.Errorf("expected SentMessage following BridgeInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		} else if sentMessage.Event.TransactionHash != initiatedBridge.Event.TransactionHash {
			return fmt.Errorf("correlated events tx hash mismatch. bridge_tx_hash = %s, message_tx_hash = %s", initiatedBridge.Event.TransactionHash, sentMessage.Event.TransactionHash)
		}

		bridgedTokens[initiatedBridge.BridgeTransfer.TokenPair.LocalTokenAddress]++

		initiatedBridge.BridgeTransfer.CrossDomainMessageHash = &sentMessage.BridgeMessage.MessageHash
		bridgeWithdrawals[i] = database.L2BridgeWithdrawal{
			TransactionWithdrawalHash: messagePassed.WithdrawalHash,
			BridgeTransfer:            initiatedBridge.BridgeTransfer,
		}
	}
	if len(bridgeWithdrawals) > 0 {
		if err := db.BridgeTransfers.StoreL2BridgeWithdrawals(bridgeWithdrawals); err != nil {
			return err
		}
		for tokenAddr, size := range bridgedTokens {
			metrics.RecordL2InitiatedBridgeTransfers(tokenAddr, size)
		}
	}

	// a-ok!
	return nil
}

// L2ProcessFinalizedBridgeEvent will query the database for all the finalization markers for all initiated
// bridge events. This covers every part of the multi-layered stack:
//  1. L2CrossDomainMessenger (relayMessage marker)
//  2. L2StandardBridge (no-op, since this is simply a wrapper over the L2CrossDomainMEssenger)
//
// NOTE: Unlike L1, there's no L2ToL1MessagePasser stage since transaction deposits are apart of the block derivation process.
func L2ProcessFinalizedBridgeEvents(log log.Logger, db *database.DB, metrics L2Metricer, l2Contracts config.L2Contracts, fromHeight, toHeight *big.Int) error {
	// (1) L2CrossDomainMessenger
	crossDomainRelayedMessages, err := contracts.CrossDomainMessengerRelayedMessageEvents("l2", l2Contracts.L2CrossDomainMessenger, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainRelayedMessages) > 0 {
		log.Info("detected relayed messages", "size", len(crossDomainRelayedMessages))
	}

	for i := range crossDomainRelayedMessages {
		relayed := crossDomainRelayedMessages[i]
		message, err := db.BridgeMessages.L1BridgeMessage(relayed.MessageHash)
		if err != nil {
			return err
		} else if message == nil {
			return fmt.Errorf("missing indexed L1CrossDomainMessager message! tx_hash = %s", relayed.Event.TransactionHash)
		}

		if err := db.BridgeMessages.MarkRelayedL1BridgeMessage(relayed.MessageHash, relayed.Event.GUID); err != nil {
			return fmt.Errorf("failed to relay cross domain message. tx_hash = %s: %w", relayed.Event.TransactionHash, err)
		}
	}
	if len(crossDomainRelayedMessages) > 0 {
		metrics.RecordL2CrossDomainRelayedMessages(len(crossDomainRelayedMessages))
	}

	// (2) L2StandardBridge
	// - Nothing actionable on the database. Since the StandardBridge is layered ontop of the
	// CrossDomainMessenger, there's no need for any sanity or invariant checks as the previous step
	// ensures a relayed message (finalized bridge) can be linked with a sent message (initiated bridge).
	finalizedBridges, err := contracts.StandardBridgeFinalizedEvents("l2", l2Contracts.L2StandardBridge, db, fromHeight, toHeight)
	if err != nil {
		return err
	}

	finalizedTokens := make(map[common.Address]int)
	for i := range finalizedBridges {
		finalizedBridge := finalizedBridges[i]
		finalizedTokens[finalizedBridge.BridgeTransfer.TokenPair.LocalTokenAddress]++
	}
	if len(finalizedBridges) > 0 {
		log.Info("detected finalized bridge deposits", "size", len(finalizedBridges))
		for tokenAddr, size := range finalizedTokens {
			metrics.RecordL2FinalizedBridgeTransfers(tokenAddr, size)
		}
	}

	// a-ok!
	return nil
}
