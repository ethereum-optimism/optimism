package bridge

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/processors/contracts"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
)

// Legacy Bridge Initiation

func LegacyL1ProcessInitiatedBridgeEvents(log log.Logger, db *database.DB, chainConfig config.ChainConfig, fromHeight *big.Int, toHeight *big.Int) error {
	// (1) CanonicalTransactionChain
	ctcTxDepositEvents, err := contracts.LegacyCTCDepositEvents(chainConfig.L1Contracts.LegacyCanonicalTransactionChain, db, fromHeight, toHeight)
	if err != nil {
		return err
	}

	ctcTxDeposits := make(map[logKey]*contracts.LegacyCTCDepositEvent, len(ctcTxDepositEvents))
	transactionDeposits := make([]database.L1TransactionDeposit, len(ctcTxDepositEvents))
	for i := range ctcTxDepositEvents {
		deposit := ctcTxDepositEvents[i]
		ctcTxDeposits[logKey{deposit.Event.BlockHash, deposit.Event.LogIndex}] = &deposit
		transactionDeposits[i] = database.L1TransactionDeposit{
			SourceHash:           deposit.TxHash,
			L2TransactionHash:    deposit.TxHash, // TODO: check that this is right
			InitiatedL1EventGUID: deposit.Event.GUID,
			GasLimit:             deposit.GasLimit,
			Tx:                   deposit.Tx,
		}
	}

	if len(ctcTxDepositEvents) > 0 {
		log.Info("detected legacy transaction deposits", "size", len(transactionDeposits))
		if err := db.BridgeTransactions.StoreL1TransactionDeposits(transactionDeposits); err != nil {
			return err
		}
	}

	// (2) L1CrossDomainMessenger
	crossDomainSentMessages, err := contracts.CrossDomainMessengerSentMessageEvents("l1", chainConfig.L1Contracts.L1CrossDomainMessengerProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainSentMessages) < len(transactionDeposits) {
		return fmt.Errorf("missing transaction deposit for each cross-domain message. deposits: %d, messages: %d", len(transactionDeposits), len(crossDomainSentMessages))
	}
	sentMessages := make(map[logKey]*contracts.CrossDomainMessengerSentMessageEvent, len(crossDomainSentMessages))
	l1BridgeMessages := make([]database.L1BridgeMessage, len(crossDomainSentMessages))
	for i := range crossDomainSentMessages {
		sentMessage := crossDomainSentMessages[i]
		sentMessages[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex}] = &sentMessage

		// extract the deposit hash from the previous TransactionDepositedEvent
		ctcTxDeposit, ok := ctcTxDeposits[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex - 1}]
		if !ok {
			return fmt.Errorf("missing expected preceding TransactionEnqueued for SentMessage. tx_hash = %s", sentMessage.Event.TransactionHash)
		}

		l1BridgeMessages[i] = database.L1BridgeMessage{TransactionSourceHash: ctcTxDeposit.TxHash, BridgeMessage: sentMessage.BridgeMessage}
	}

	if len(l1BridgeMessages) > 0 {
		log.Info("detected legacy L1CrossDomainMessenger messages", "size", len(l1BridgeMessages))
		if err := db.BridgeMessages.StoreL1BridgeMessages(l1BridgeMessages); err != nil {
			return err
		}
	}

	// (3) L1StandardBridge
	initiatedBridges, err := contracts.L1StandardBridgeLegacyDepositInitiatedEvents(chainConfig.L1Contracts.L1StandardBridgeProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(initiatedBridges) > len(crossDomainSentMessages) {
		return fmt.Errorf("missing cross-domain message for each initiated bridge event. messages: %d, bridges: %d", len(crossDomainSentMessages), len(initiatedBridges))
	}

	l1BridgeDeposits := make([]database.L1BridgeDeposit, len(initiatedBridges))
	for i := range initiatedBridges {
		initiatedBridge := initiatedBridges[i]

		// extract the cross domain message hash & deposit source hash from the following events
		// Unlike bedrock, the bridge events are emitted AFTER sending the cross domain message
		// 	- Event Flow: TransactionEnqueued -> SentMessage -> DepositInitiated
		ctcTxDeposit, ok := ctcTxDeposits[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex - 2}]
		if !ok {
			return fmt.Errorf("expected TransactionEnqueued preceding BridgeInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		}
		sentMessage, ok := sentMessages[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex - 1}]
		if !ok {
			return fmt.Errorf("expected SentMessage preceding TransactionEnqeueud event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		}

		initiatedBridge.BridgeTransfer.CrossDomainMessageHash = &sentMessage.BridgeMessage.MessageHash
		l1BridgeDeposits[i] = database.L1BridgeDeposit{
			TransactionSourceHash: ctcTxDeposit.TxHash,
			BridgeTransfer:        initiatedBridge.BridgeTransfer,
		}
	}

	if len(l1BridgeDeposits) > 0 {
		log.Info("detected legacy L1StandardBridge deposits", "size", len(l1BridgeDeposits))
		if err := db.BridgeTransfers.StoreL1BridgeDeposits(l1BridgeDeposits); err != nil {
			return err
		}
	}

	// a-ok!
	return nil
}

func LegacyL2ProcessInitiatedBridgeEvents(log log.Logger, db *database.DB, chainConfig config.ChainConfig, fromHeight *big.Int, toHeight *big.Int) error {
	// (1) LegacyMessagePasser -> no-op since there's no emitted events an the public entrypoint is non-functional pre-bedrock. All withdrawals
	// be sent through the L2CrossDomainMessenger

	// (2) L2CrossDomainMessenger
	crossDomainSentMessages, err := contracts.CrossDomainMessengerSentMessageEvents("l2", predeploys.L2CrossDomainMessengerAddr, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	sentMessages := make(map[logKey]*contracts.CrossDomainMessengerSentMessageEvent, len(crossDomainSentMessages))
	l2BridgeMessages := make([]database.L2BridgeMessage, len(crossDomainSentMessages))
	l2TransactionWithdrawals := make([]database.L2TransactionWithdrawal, len(crossDomainSentMessages))
	for i := range crossDomainSentMessages {
		sentMessage := crossDomainSentMessages[i]
		sentMessages[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex}] = &sentMessage

		// To ensure consitency in our schema, we duplicate this as the "root" transaction withdrawal. We re-use the same withdrawal hash as the storage
		// key for the proof sha3(calldata + sender) can be derived from the fields. (NOTE: should we just use the storage key here?)
		l2BridgeMessages[i] = database.L2BridgeMessage{TransactionWithdrawalHash: sentMessage.BridgeMessage.MessageHash, BridgeMessage: sentMessage.BridgeMessage}
		l2TransactionWithdrawals[i] = database.L2TransactionWithdrawal{
			WithdrawalHash:       sentMessage.BridgeMessage.MessageHash,
			InitiatedL2EventGUID: sentMessage.Event.GUID,
			Nonce:                sentMessage.BridgeMessage.Nonce,
			GasLimit:             sentMessage.BridgeMessage.GasLimit,
			Tx: database.Transaction{
				FromAddress: sentMessage.BridgeMessage.Tx.FromAddress,
				ToAddress:   sentMessage.BridgeMessage.Tx.ToAddress,
				Amount:      big.NewInt(0),
				Data:        sentMessage.BridgeMessage.Tx.Data,
				Timestamp:   sentMessage.Event.Timestamp,
			},
		}
	}

	if len(l2BridgeMessages) > 0 {
		log.Info("detected legacy transaction withdrawals (via L2CrossDomainMessenger)", "size", len(l2BridgeMessages))
		if err := db.BridgeTransactions.StoreL2TransactionWithdrawals(l2TransactionWithdrawals); err != nil {
			return err
		}
		if err := db.BridgeMessages.StoreL2BridgeMessages(l2BridgeMessages); err != nil {
			return err
		}
	}

	// (3) L2StandardBridge
	initiatedBridges, err := contracts.L2StandardBridgeLegacyWithdrawalInitiatedEvents(predeploys.L2StandardBridgeAddr, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(initiatedBridges) > len(crossDomainSentMessages) {
		return fmt.Errorf("missing cross-domain message for each initiated bridge event. messages: %d, bridges: %d", len(crossDomainSentMessages), len(initiatedBridges))
	}

	l2BridgeWithdrawals := make([]database.L2BridgeWithdrawal, len(initiatedBridges))
	for i := range initiatedBridges {
		initiatedBridge := initiatedBridges[i]

		// extract the cross domain message hash & deposit source hash from the following events
		// Unlike bedrock, the bridge events are emitted AFTER sending the cross domain message
		// 	- Event Flow: TransactionEnqueued -> SentMessage -> DepositInitiated
		sentMessage, ok := sentMessages[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex - 1}]
		if !ok {
			return fmt.Errorf("expected SentMessage preceding TransactionEnqeueud event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		}

		initiatedBridge.BridgeTransfer.CrossDomainMessageHash = &sentMessage.BridgeMessage.MessageHash
		l2BridgeWithdrawals[i] = database.L2BridgeWithdrawal{
			TransactionWithdrawalHash: sentMessage.BridgeMessage.MessageHash,
			BridgeTransfer:            initiatedBridge.BridgeTransfer,
		}
	}

	if len(l2BridgeWithdrawals) > 0 {
		log.Info("detected legacy L2StandardBridge withdrawals", "size", len(l2BridgeWithdrawals))
		if err := db.BridgeTransfers.StoreL2BridgeWithdrawals(l2BridgeWithdrawals); err != nil {
			return err
		}
	}

	// a-ok
	return nil
}

// Legacy Bridge Finalization

func LegacyL1ProcessFinalizedBridgeEvents(log log.Logger, db *database.DB, chainConfig config.ChainConfig, fromHeight *big.Int, toHeight *big.Int) error {
	// (1) L1CrossDomainMessenger -> This is the root-most contract from which bridge events are finalized since withdrawals must be initiated from the
	// L2CrossDomainMessenger. Since there's no analogous bedrock two-step withdrawal process, we mark the transaction as proven/finalized in the same step
	crossDomainRelayedMessages, err := contracts.CrossDomainMessengerRelayedMessageEvents("l1", chainConfig.L1Contracts.L1CrossDomainMessengerProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}

	for i := range crossDomainRelayedMessages {
		relayedMessage := crossDomainRelayedMessages[i]

		// Mark the associated tx withdrawal as proven/finalized with the same event
		if err := db.BridgeTransactions.MarkL2TransactionWithdrawalProvenEvent(relayedMessage.MessageHash, relayedMessage.Event.GUID); err != nil {
			return err
		}
		if err := db.BridgeTransactions.MarkL2TransactionWithdrawalFinalizedEvent(relayedMessage.MessageHash, relayedMessage.Event.GUID, true); err != nil {
			return err
		}

		message, err := db.BridgeMessages.L2BridgeMessage(relayedMessage.MessageHash)
		if err != nil {
			return err
		} else if message == nil {
			log.Error("missing indexed legacy L2CrossDomainMessenger message", "message_hash", relayedMessage.MessageHash, "tx_hash", relayedMessage.Event.TransactionHash)
			return fmt.Errorf("missing indexed L2CrossDomainMessager message")
		}

		if err := db.BridgeMessages.MarkRelayedL2BridgeMessage(relayedMessage.MessageHash, relayedMessage.Event.GUID); err != nil {
			return err
		}
	}

	if len(crossDomainRelayedMessages) > 0 {
		log.Info("relayed legacy L2CrossDomainMessenger messages", "size", len(crossDomainRelayedMessages))
	}

	// (2) L1StandardBridge
	// no-op for now as there's nothing actionable to do here besides sanity checks that
	// we'll skip for now

	// a-ok!
	return nil
}

func LegacyL2ProcessFinalizedBridgeEvents(log log.Logger, db *database.DB, chainConfig config.ChainConfig, fromHeight *big.Int, toHeight *big.Int) error {
	// (1) L2CrossDomainMessenger
	crossDomainRelayedMessages, err := contracts.CrossDomainMessengerRelayedMessageEvents("l2", predeploys.L2CrossDomainMessengerAddr, db, fromHeight, toHeight)
	if err != nil {
		return err
	}

	for i := range crossDomainRelayedMessages {
		relayedMessage := crossDomainRelayedMessages[i]
		message, err := db.BridgeMessages.L1BridgeMessage(relayedMessage.MessageHash)
		if err != nil {
			return err
		} else if message == nil {
			log.Error("missing indexed legacy L1CrossDomainMessenger message", "message_hash", relayedMessage.MessageHash, "tx_hash", relayedMessage.Event.TransactionHash)
			return fmt.Errorf("missing indexed L1CrossDomainMessager message")
		}

		if err := db.BridgeMessages.MarkRelayedL1BridgeMessage(relayedMessage.MessageHash, relayedMessage.Event.GUID); err != nil {
			return err
		}
	}

	if len(crossDomainRelayedMessages) > 0 {
		log.Info("relayed legacy L1CrossDomainMessenger messages", "size", len(crossDomainRelayedMessages))
	}

	// (2) L2StandardBridge
	// no-op for now as there's nothing actionable to do here besides sanity checks that
	// we'll skip for now

	// a-ok!
	return nil
}
