package bridge

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/indexer/processors/contracts"
)

// Legacy Bridge Initiation

// LegacyL1ProcessInitiatedEvents will query the data for bridge events within the specified block range
// according the pre-bedrock protocol. This follows:
//  1. CanonicalTransactionChain
//  2. L1CrossDomainMessenger
//  3. L1StandardBridge
func LegacyL1ProcessInitiatedBridgeEvents(log log.Logger, db *database.DB, metrics L1Metricer, l1Contracts config.L1Contracts, fromHeight, toHeight *big.Int) error {
	// (1) CanonicalTransactionChain
	ctcTxDepositEvents, err := contracts.LegacyCTCDepositEvents(l1Contracts.LegacyCanonicalTransactionChain, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(ctcTxDepositEvents) > 0 {
		log.Info("detected legacy transaction deposits", "size", len(ctcTxDepositEvents))
	}

	ctcTxDeposits := make(map[logKey]*contracts.LegacyCTCDepositEvent, len(ctcTxDepositEvents))
	transactionDeposits := make([]database.L1TransactionDeposit, len(ctcTxDepositEvents))
	for i := range ctcTxDepositEvents {
		deposit := ctcTxDepositEvents[i]
		ctcTxDeposits[logKey{deposit.Event.BlockHash, deposit.Event.LogIndex}] = &deposit
		transactionDeposits[i] = database.L1TransactionDeposit{
			// We re-use the L2 Transaction hash as the source hash
			// to remain consistent in the schema.
			SourceHash:        deposit.TxHash,
			L2TransactionHash: deposit.TxHash,

			InitiatedL1EventGUID: deposit.Event.GUID,
			GasLimit:             deposit.GasLimit,
			Tx:                   deposit.Tx,
		}
	}
	if len(ctcTxDepositEvents) > 0 {
		if err := db.BridgeTransactions.StoreL1TransactionDeposits(transactionDeposits); err != nil {
			return err
		}
		metrics.RecordL1TransactionDeposits(len(transactionDeposits))
	}

	// (2) L1CrossDomainMessenger
	crossDomainSentMessages, err := contracts.CrossDomainMessengerSentMessageEvents("l1", l1Contracts.L1CrossDomainMessengerProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainSentMessages) > 0 {
		log.Info("detected legacy sent messages", "size", len(crossDomainSentMessages))
	}

	sentMessages := make(map[logKey]*contracts.CrossDomainMessengerSentMessageEvent, len(crossDomainSentMessages))
	bridgeMessages := make([]database.L1BridgeMessage, len(crossDomainSentMessages))
	for i := range crossDomainSentMessages {
		sentMessage := crossDomainSentMessages[i]
		sentMessages[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex}] = &sentMessage

		// extract the deposit hash from the previous TransactionDepositedEvent
		ctcTxDeposit, ok := ctcTxDeposits[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex - 1}]
		if !ok {
			log.Error("missing transaction deposit for cross domain message", "tx_hash", sentMessage.Event.TransactionHash.String())
			return fmt.Errorf("missing preceding TransactionEnqueued for SentMessage event. tx_hash = %s", sentMessage.Event.TransactionHash.String())
		}

		bridgeMessages[i] = database.L1BridgeMessage{TransactionSourceHash: ctcTxDeposit.TxHash, BridgeMessage: sentMessage.BridgeMessage}
	}
	if len(bridgeMessages) > 0 {
		if err := db.BridgeMessages.StoreL1BridgeMessages(bridgeMessages); err != nil {
			return err
		}
		metrics.RecordL1CrossDomainSentMessages(len(bridgeMessages))
	}

	// (3) L1StandardBridge
	initiatedBridges, err := contracts.L1StandardBridgeLegacyDepositInitiatedEvents(l1Contracts.L1StandardBridgeProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(initiatedBridges) > 0 {
		log.Info("detected iegacy bridge deposits", "size", len(initiatedBridges))
	}

	bridgedTokens := make(map[common.Address]int)
	bridgeDeposits := make([]database.L1BridgeDeposit, len(initiatedBridges))
	for i := range initiatedBridges {
		initiatedBridge := initiatedBridges[i]

		// extract the cross domain message hash & deposit source hash from the following events
		// Unlike bedrock, the bridge events are emitted AFTER sending the cross domain message
		// 	- Event Flow: TransactionEnqueued -> SentMessage -> DepositInitiated
		sentMessage, ok := sentMessages[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex - 1}]
		if !ok {
			log.Error("missing cross domain message for bridge transfer", "tx_hash", initiatedBridge.Event.TransactionHash.String())
			return fmt.Errorf("expected SentMessage preceding DepositInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash.String())
		}
		ctcTxDeposit, ok := ctcTxDeposits[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex - 2}]
		if !ok {
			log.Error("missing transaction deposit for bridge transfer", "tx_hash", initiatedBridge.Event.TransactionHash.String())
			return fmt.Errorf("expected TransactionEnqueued preceding DepostInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash.String())
		}

		initiatedBridge.BridgeTransfer.CrossDomainMessageHash = &sentMessage.BridgeMessage.MessageHash
		bridgedTokens[initiatedBridge.BridgeTransfer.TokenPair.LocalTokenAddress]++
		bridgeDeposits[i] = database.L1BridgeDeposit{
			TransactionSourceHash: ctcTxDeposit.TxHash,
			BridgeTransfer:        initiatedBridge.BridgeTransfer,
		}
	}
	if len(bridgeDeposits) > 0 {
		if err := db.BridgeTransfers.StoreL1BridgeDeposits(bridgeDeposits); err != nil {
			return err
		}
		for tokenAddr, size := range bridgedTokens {
			metrics.RecordL1InitiatedBridgeTransfers(tokenAddr, size)
		}
	}

	// a-ok!
	return nil
}

// LegacyL2ProcessInitiatedEvents will query the data for bridge events within the specified block range
// according the pre-bedrock protocol. This follows:
//  1. L2CrossDomainMessenger - The LegacyMessagePasser contract cannot be used as entrypoint to bridge transactions from L2. The protocol
//     only allows the L2CrossDomainMessenger as the sole sender when relaying a bridged message.
//  2. L2StandardBridge
func LegacyL2ProcessInitiatedBridgeEvents(log log.Logger, db *database.DB, metrics L2Metricer, l2Contracts config.L2Contracts, fromHeight, toHeight *big.Int) error {
	// (1) L2CrossDomainMessenger
	crossDomainSentMessages, err := contracts.CrossDomainMessengerSentMessageEvents("l2", l2Contracts.L2CrossDomainMessenger, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainSentMessages) > 0 {
		log.Info("detected legacy transaction withdrawals (via L2CrossDomainMessenger)", "size", len(crossDomainSentMessages))
	}

	sentMessages := make(map[logKey]*contracts.CrossDomainMessengerSentMessageEvent, len(crossDomainSentMessages))
	bridgeMessages := make([]database.L2BridgeMessage, len(crossDomainSentMessages))
	transactionWithdrawals := make([]database.L2TransactionWithdrawal, len(crossDomainSentMessages))
	for i := range crossDomainSentMessages {
		sentMessage := crossDomainSentMessages[i]
		sentMessages[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex}] = &sentMessage

		// To ensure consistency in the schema, we duplicate this as the "root" transaction withdrawal. The storage key in the message
		// passer contract is sha3(calldata + sender). The sender always being the L2CrossDomainMessenger pre-bedrock.
		withdrawalHash := crypto.Keccak256Hash(append(sentMessage.MessageCalldata, l2Contracts.L2CrossDomainMessenger[:]...))
		transactionWithdrawals[i] = database.L2TransactionWithdrawal{
			WithdrawalHash:       withdrawalHash,
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

		bridgeMessages[i] = database.L2BridgeMessage{
			TransactionWithdrawalHash: withdrawalHash,
			BridgeMessage:             sentMessage.BridgeMessage,
		}
	}
	if len(bridgeMessages) > 0 {
		if err := db.BridgeTransactions.StoreL2TransactionWithdrawals(transactionWithdrawals); err != nil {
			return err
		}
		if err := db.BridgeMessages.StoreL2BridgeMessages(bridgeMessages); err != nil {
			return err
		}
		metrics.RecordL2TransactionWithdrawals(len(transactionWithdrawals))
		metrics.RecordL2CrossDomainSentMessages(len(bridgeMessages))
	}

	// (2) L2StandardBridge
	initiatedBridges, err := contracts.L2StandardBridgeLegacyWithdrawalInitiatedEvents(l2Contracts.L2StandardBridge, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(initiatedBridges) > 0 {
		log.Info("detected legacy bridge withdrawals", "size", len(initiatedBridges))
	}

	bridgedTokens := make(map[common.Address]int)
	l2BridgeWithdrawals := make([]database.L2BridgeWithdrawal, len(initiatedBridges))
	for i := range initiatedBridges {
		initiatedBridge := initiatedBridges[i]

		// extract the cross domain message hash & deposit source hash from the following events
		// Unlike bedrock, the bridge events are emitted AFTER sending the cross domain message
		// 	- Event Flow: TransactionEnqueued -> SentMessage -> DepositInitiated
		sentMessage, ok := sentMessages[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex - 1}]
		if !ok {
			log.Error("expected SentMessage preceding DepositInitiated event", "tx_hash", initiatedBridge.Event.TransactionHash.String())
			return fmt.Errorf("expected SentMessage preceding DepositInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		}
		initiatedBridge.BridgeTransfer.CrossDomainMessageHash = &sentMessage.BridgeMessage.MessageHash

		bridgedTokens[initiatedBridge.BridgeTransfer.TokenPair.LocalTokenAddress]++
		l2BridgeWithdrawals[i] = database.L2BridgeWithdrawal{
			TransactionWithdrawalHash: sentMessage.BridgeMessage.MessageHash,
			BridgeTransfer:            initiatedBridge.BridgeTransfer,
		}
	}
	if len(l2BridgeWithdrawals) > 0 {
		if err := db.BridgeTransfers.StoreL2BridgeWithdrawals(l2BridgeWithdrawals); err != nil {
			return err
		}
		for tokenAddr, size := range bridgedTokens {
			metrics.RecordL2InitiatedBridgeTransfers(tokenAddr, size)
		}
	}

	// a-ok
	return nil
}

// Legacy Bridge Finalization

// LegacyL1ProcessFinalizedBridgeEvents will query for bridge events within the specified block range
// according to the pre-bedrock protocol. This follows:
//  1. L1CrossDomainMessenger
//  2. L1StandardBridge
func LegacyL1ProcessFinalizedBridgeEvents(log log.Logger, db *database.DB, metrics L1Metricer, l1Client node.EthClient, l1Contracts config.L1Contracts, fromHeight, toHeight *big.Int) error {
	// (1) L1CrossDomainMessenger -> This is the root-most contract from which bridge events are finalized since withdrawals must be initiated from the
	// L2CrossDomainMessenger. Since there's no two-step withdrawal process, we mark the transaction as proven/finalized in the same step
	crossDomainRelayedMessages, err := contracts.CrossDomainMessengerRelayedMessageEvents("l1", l1Contracts.L1CrossDomainMessengerProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainRelayedMessages) > 0 {
		log.Info("detected relayed messages", "size", len(crossDomainRelayedMessages))
	}

	skippedPreRegenesisMessages := 0
	for i := range crossDomainRelayedMessages {
		relayedMessage := crossDomainRelayedMessages[i]
		message, err := db.BridgeMessages.L2BridgeMessage(relayedMessage.MessageHash)
		if err != nil {
			return err
		} else if message == nil {
			// Before surfacing an error about a missing withdrawal, we need to handle an edge case
			// for OP-Mainnet pre-regensis withdrawals that no longer exist on L2.
			tx, err := l1Client.TxByHash(relayedMessage.Event.TransactionHash)
			if err != nil {
				return err
			} else if tx == nil {
				log.Error("missing tx for relayed message", "tx_hash", relayedMessage.Event.TransactionHash.String())
				return fmt.Errorf("missing tx for relayed message. tx_hash = %s", relayedMessage.Event.TransactionHash.String())
			}

			relayMessageData := tx.Data()[4:]
			inputs, err := contracts.CrossDomainMessengerLegacyRelayMessageEncoding.Inputs.Unpack(relayMessageData)
			if err != nil || inputs == nil {
				log.Error("failed to extract XDomainCallData from relayMessage transaction", "err", err, "tx_hash", relayedMessage.Event.TransactionHash.String())
				return fmt.Errorf("unable to extract XDomainCallData from relayMessage transaction. err = %w. tx_hash = %s", err, relayedMessage.Event.TransactionHash.String())
			}

			// NOTE: Since OP-Mainnet is the only network to go through a regensis we can simply harcode the
			// the starting message nonce at genesis (100k). Any relayed withdrawal on L1 with a lesser nonce
			// is a clear indicator of a pre-regenesis withdrawal.
			if inputs[3].(*big.Int).Int64() < 100_000 {
				// skip pre-regenesis withdrawals
				skippedPreRegenesisMessages++
				continue
			} else {
				log.Error("missing indexed legacy L2CrossDomainMessenger message", "tx_hash", relayedMessage.Event.TransactionHash.String())
				return fmt.Errorf("missing indexed L2CrossDomainMessager message. tx_hash %s", relayedMessage.Event.TransactionHash.String())
			}
		}

		// Mark the associated tx withdrawal as proven/finalized with the same event
		if err := db.BridgeTransactions.MarkL2TransactionWithdrawalProvenEvent(relayedMessage.MessageHash, relayedMessage.Event.GUID); err != nil {
			log.Error("failed to mark withdrawal as proven", "err", err)
			return err
		}
		if err := db.BridgeTransactions.MarkL2TransactionWithdrawalFinalizedEvent(relayedMessage.MessageHash, relayedMessage.Event.GUID, true); err != nil {
			log.Error("failed to mark withdrawal as finalzed", "err", err)
			return err
		}
		if err := db.BridgeMessages.MarkRelayedL2BridgeMessage(relayedMessage.MessageHash, relayedMessage.Event.GUID); err != nil {
			log.Error("failed to relay cross domain message", "err", err)
			return err
		}
	}
	if len(crossDomainRelayedMessages) > 0 {
		metrics.RecordL1ProvenWithdrawals(len(crossDomainRelayedMessages))
		metrics.RecordL1FinalizedWithdrawals(len(crossDomainRelayedMessages))
		metrics.RecordL1CrossDomainRelayedMessages(len(crossDomainRelayedMessages))
	}
	if skippedPreRegenesisMessages > 0 {
		// Logged as a warning just for visibility
		log.Warn("skipped pre-regensis relayed L2CrossDomainMessenger withdrawals", "size", skippedPreRegenesisMessages)
	}

	// (2) L2StandardBridge -- no-op for now as there's nothing actionable to do here besides
	// santiy checks which is not important for legacy code. Not worth extra code pre-bedrock.
	// The message status is already tracked via the relayed bridge messed through the cross domain messenger.
	//  - NOTE: This means we dont increment metrics for finalized bridge transfers

	// a-ok!
	return nil
}

// LegacyL2ProcessFinalizedBridgeEvents will query for bridge events within the specified block range
// according to the pre-bedrock protocol. This follows:
//  1. L2CrossDomainMessenger
//  2. L2StandardBridge
func LegacyL2ProcessFinalizedBridgeEvents(log log.Logger, db *database.DB, metrics L2Metricer, l2Contracts config.L2Contracts, fromHeight, toHeight *big.Int) error {
	// (1) L2CrossDomainMessenger
	crossDomainRelayedMessages, err := contracts.CrossDomainMessengerRelayedMessageEvents("l2", l2Contracts.L2CrossDomainMessenger, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainRelayedMessages) > 0 {
		log.Info("detected relayed legacy messages", "size", len(crossDomainRelayedMessages))
	}

	for i := range crossDomainRelayedMessages {
		relayedMessage := crossDomainRelayedMessages[i]
		message, err := db.BridgeMessages.L1BridgeMessage(relayedMessage.MessageHash)
		if err != nil {
			return err
		} else if message == nil {
			log.Error("missing indexed legacy L1CrossDomainMessenger message", "tx_hash", relayedMessage.Event.TransactionHash.String())
			return fmt.Errorf("missing indexed L1CrossDomainMessager message. tx_hash = %s", relayedMessage.Event.TransactionHash.String())
		}

		if err := db.BridgeMessages.MarkRelayedL1BridgeMessage(relayedMessage.MessageHash, relayedMessage.Event.GUID); err != nil {
			log.Error("failed to relay cross domain message", "err", err)
			return err
		}
	}
	if len(crossDomainRelayedMessages) > 0 {
		metrics.RecordL2CrossDomainRelayedMessages(len(crossDomainRelayedMessages))
	}

	// (2) L2StandardBridge -- no-op for now as there's nothing actionable to do here besides
	// santiy checks which is not important for legacy code. Not worth the extra code pre-bedorck.
	// The message status is already tracked via the relayed bridge messed through the cross domain messenger.
	//  - NOTE: This means we dont increment metrics for finalized bridge transfers

	// a-ok!
	return nil
}
