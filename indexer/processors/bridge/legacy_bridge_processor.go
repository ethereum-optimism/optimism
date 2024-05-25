package bridge

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/bigint"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/processors/bridge/ovm1"
	"github.com/ethereum-optimism/optimism/indexer/processors/contracts"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
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

	mintedWEI := bigint.Zero
	ctcTxDeposits := make(map[logKey]*contracts.LegacyCTCDepositEvent, len(ctcTxDepositEvents))
	transactionDeposits := make([]database.L1TransactionDeposit, len(ctcTxDepositEvents))
	for i := range ctcTxDepositEvents {
		deposit := ctcTxDepositEvents[i]
		ctcTxDeposits[logKey{deposit.Event.BlockHash, deposit.Event.LogIndex}] = &deposit
		mintedWEI = new(big.Int).Add(mintedWEI, deposit.Tx.Amount)

		// We re-use the L2 Transaction hash as the source hash to remain consistent in the schema.
		transactionDeposits[i] = database.L1TransactionDeposit{
			SourceHash:           deposit.TxHash,
			L2TransactionHash:    deposit.TxHash,
			InitiatedL1EventGUID: deposit.Event.GUID,
			GasLimit:             deposit.GasLimit,
			Tx:                   deposit.Tx,
		}
	}
	if len(ctcTxDepositEvents) > 0 {
		if err := db.BridgeTransactions.StoreL1TransactionDeposits(transactionDeposits); err != nil {
			return err
		}

		mintedETH, _ := bigint.WeiToETH(mintedWEI).Float64()
		metrics.RecordL1TransactionDeposits(len(transactionDeposits), mintedETH)
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
			return fmt.Errorf("expected TransactionEnqueued preceding SentMessage event. tx_hash = %s", sentMessage.Event.TransactionHash)
		} else if ctcTxDeposit.Event.TransactionHash != sentMessage.Event.TransactionHash {
			return fmt.Errorf("correlated events tx hash mismatch. deposit_tx_hash = %s, message_tx_hash = %s", ctcTxDeposit.Event.TransactionHash, sentMessage.Event.TransactionHash)
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
			return fmt.Errorf("expected SentMessage preceding DepositInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		} else if sentMessage.Event.TransactionHash != initiatedBridge.Event.TransactionHash {
			return fmt.Errorf("correlated events tx hash mismatch. bridge_tx_hash = %s, message_tx_hash = %s", initiatedBridge.Event.TransactionHash, sentMessage.Event.TransactionHash)
		}

		ctcTxDeposit, ok := ctcTxDeposits[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex - 2}]
		if !ok {
			return fmt.Errorf("expected TransactionEnqueued preceding BridgeInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		} else if ctcTxDeposit.Event.TransactionHash != initiatedBridge.Event.TransactionHash {
			return fmt.Errorf("correlated events tx hash mismatch. bridge_tx_hash = %s, deposit_tx_hash = %s", initiatedBridge.Event.TransactionHash, ctcTxDeposit.Event.TransactionHash)
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
func LegacyL2ProcessInitiatedBridgeEvents(log log.Logger, db *database.DB, metrics L2Metricer, preset int, l2Contracts config.L2Contracts, fromHeight, toHeight *big.Int) error {
	// (1) L2CrossDomainMessenger
	crossDomainSentMessages, err := contracts.CrossDomainMessengerSentMessageEvents("l2", l2Contracts.L2CrossDomainMessenger, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainSentMessages) > 0 {
		log.Info("detected legacy transaction withdrawals (via L2CrossDomainMessenger)", "size", len(crossDomainSentMessages))
	}

	type sentMessageEvent struct {
		*contracts.CrossDomainMessengerSentMessageEvent
		WithdrawalHash common.Hash
	}

	withdrawnWEI := bigint.Zero
	sentMessages := make(map[logKey]sentMessageEvent, len(crossDomainSentMessages))
	bridgeMessages := make([]database.L2BridgeMessage, len(crossDomainSentMessages))
	versionedMessageHashes := make([]database.L2BridgeMessageVersionedMessageHash, len(crossDomainSentMessages))
	transactionWithdrawals := make([]database.L2TransactionWithdrawal, len(crossDomainSentMessages))
	for i := range crossDomainSentMessages {
		sentMessage := crossDomainSentMessages[i]
		withdrawnWEI = new(big.Int).Add(withdrawnWEI, sentMessage.BridgeMessage.Tx.Amount)

		// Since these message can be relayed in bedrock, we utilize the migrated withdrawal hash
		// and also store the v1 version of the message hash such that the bedrock l1 finalization
		// processor works as expected. There will be some entries relayed pre-bedrock that will not
		// require the entry but we'll store it anyways as a large % of these withdrawals are not relayed
		// pre-bedrock.

		v1MessageHash, err := legacyBridgeMessageV1MessageHash(&sentMessage.BridgeMessage)
		if err != nil {
			return fmt.Errorf("failed to compute versioned message hash: %w", err)
		}

		withdrawalHash, err := legacyBridgeMessageWithdrawalHash(preset, &sentMessage.BridgeMessage)
		if err != nil {
			return fmt.Errorf("failed to construct migrated withdrawal hash: %w", err)
		}

		transactionWithdrawals[i] = database.L2TransactionWithdrawal{
			WithdrawalHash:       withdrawalHash,
			InitiatedL2EventGUID: sentMessage.Event.GUID,
			Nonce:                sentMessage.BridgeMessage.Nonce,
			GasLimit:             sentMessage.BridgeMessage.GasLimit,
			Tx: database.Transaction{
				FromAddress: sentMessage.BridgeMessage.Tx.FromAddress,
				ToAddress:   sentMessage.BridgeMessage.Tx.ToAddress,
				Amount:      bigint.Zero,
				Data:        sentMessage.BridgeMessage.Tx.Data,
				Timestamp:   sentMessage.Event.Timestamp,
			},
		}

		sentMessages[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex}] = sentMessageEvent{&sentMessage, withdrawalHash}
		bridgeMessages[i] = database.L2BridgeMessage{TransactionWithdrawalHash: withdrawalHash, BridgeMessage: sentMessage.BridgeMessage}
		versionedMessageHashes[i] = database.L2BridgeMessageVersionedMessageHash{MessageHash: sentMessage.BridgeMessage.MessageHash, V1MessageHash: v1MessageHash}
	}
	if len(bridgeMessages) > 0 {
		if err := db.BridgeTransactions.StoreL2TransactionWithdrawals(transactionWithdrawals); err != nil {
			return err
		}
		if err := db.BridgeMessages.StoreL2BridgeMessages(bridgeMessages); err != nil {
			return err
		}
		if err := db.BridgeMessages.StoreL2BridgeMessageV1MessageHashes(versionedMessageHashes); err != nil {
			return err
		}

		withdrawnETH, _ := bigint.WeiToETH(withdrawnWEI).Float64()
		metrics.RecordL2TransactionWithdrawals(len(transactionWithdrawals), withdrawnETH)
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
			return fmt.Errorf("expected SentMessage preceding BridgeInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		} else if sentMessage.Event.TransactionHash != initiatedBridge.Event.TransactionHash {
			return fmt.Errorf("correlated events tx hash mismatch. bridge_tx_hash = %s, message_tx_hash = %s", initiatedBridge.Event.TransactionHash, sentMessage.Event.TransactionHash)
		}

		bridgedTokens[initiatedBridge.BridgeTransfer.TokenPair.LocalTokenAddress]++
		initiatedBridge.BridgeTransfer.CrossDomainMessageHash = &sentMessage.BridgeMessage.MessageHash
		l2BridgeWithdrawals[i] = database.L2BridgeWithdrawal{
			TransactionWithdrawalHash: sentMessage.WithdrawalHash,
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
func LegacyL1ProcessFinalizedBridgeEvents(log log.Logger, db *database.DB, metrics L1Metricer, l1Contracts config.L1Contracts, fromHeight, toHeight *big.Int) error {
	// (1) L1CrossDomainMessenger -> This is the root-most contract from which bridge events are finalized since withdrawals must be initiated from the
	// L2CrossDomainMessenger. Since there's no two-step withdrawal process, we mark the transaction as proven/finalized in the same step
	crossDomainRelayedMessages, err := contracts.CrossDomainMessengerRelayedMessageEvents("l1", l1Contracts.L1CrossDomainMessengerProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainRelayedMessages) > 0 {
		log.Info("detected relayed messages", "size", len(crossDomainRelayedMessages))
	}

	skippedOVM1Messages := 0
	for i := range crossDomainRelayedMessages {
		relayedMessage := crossDomainRelayedMessages[i]
		message, err := db.BridgeMessages.L2BridgeMessage(relayedMessage.MessageHash)
		if err != nil {
			return err
		} else if message == nil {
			if _, ok := ovm1.L1RelayedMessages[relayedMessage.MessageHash]; ok {
				skippedOVM1Messages++
				continue
			}

			return fmt.Errorf("missing indexed L2CrossDomainMessenger message! tx_hash %s", relayedMessage.Event.TransactionHash.String())
		}

		if err := db.BridgeMessages.MarkRelayedL2BridgeMessage(relayedMessage.MessageHash, relayedMessage.Event.GUID); err != nil {
			return fmt.Errorf("failed to relay cross domain message. tx_hash = %s: %w", relayedMessage.Event.TransactionHash, err)
		}

		// Mark the associated tx withdrawal as proven/finalized with the same event.
		if err := db.BridgeTransactions.MarkL2TransactionWithdrawalProvenEvent(message.TransactionWithdrawalHash, relayedMessage.Event.GUID); err != nil {
			return fmt.Errorf("failed to mark withdrawal as proven. tx_hash = %s: %w", relayedMessage.Event.TransactionHash, err)
		}
		if err := db.BridgeTransactions.MarkL2TransactionWithdrawalFinalizedEvent(message.TransactionWithdrawalHash, relayedMessage.Event.GUID, true); err != nil {
			return fmt.Errorf("failed to mark withdrawal as finalized. tx_hash = %s: %w", relayedMessage.Event.TransactionHash, err)
		}
	}
	if len(crossDomainRelayedMessages) > 0 {
		metrics.RecordL1CrossDomainRelayedMessages(len(crossDomainRelayedMessages))
		if skippedOVM1Messages > 0 { // Logged as a warning just for visibility
			metrics.RecordL1SkippedOVM1CrossDomainRelayedMessages(skippedOVM1Messages)
			log.Info("skipped OVM 1.0 relayed L2CrossDomainMessenger withdrawals", "size", skippedOVM1Messages)
		}
	}

	// (2) L1StandardBridge
	// 	- Nothing actionable on the database. Since the StandardBridge is layered ontop of the
	// CrossDomainMessenger, there's no need for any sanity or invariant checks as the previous step
	// ensures a relayed message (finalized bridge) can be linked with a sent message (initiated bridge).

	//  - NOTE: Ignoring metrics for pre-bedrock transfers

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
			return fmt.Errorf("missing indexed L1CrossDomainMessager message! tx_hash = %s", relayedMessage.Event.TransactionHash)
		}

		if err := db.BridgeMessages.MarkRelayedL1BridgeMessage(relayedMessage.MessageHash, relayedMessage.Event.GUID); err != nil {
			return fmt.Errorf("failed to relay cross domain message: %w", err)
		}
	}
	if len(crossDomainRelayedMessages) > 0 {
		metrics.RecordL2CrossDomainRelayedMessages(len(crossDomainRelayedMessages))
	}

	// (2) L2StandardBridge
	// 	- Nothing actionable on the database. Since the StandardBridge is layered ontop of the
	// CrossDomainMessenger, there's no need for any sanity or invariant checks as the previous step
	// ensures a relayed message (finalized bridge) can be linked with a sent message (initiated bridge).

	//  - NOTE: Ignoring metrics for pre-bedrock transfers

	// a-ok!
	return nil
}

// Utils

func legacyBridgeMessageWithdrawalHash(preset int, msg *database.BridgeMessage) (common.Hash, error) {
	l1Cdm := config.Presets[preset].ChainConfig.L1Contracts.L1CrossDomainMessengerProxy
	legacyWithdrawal := crossdomain.NewLegacyWithdrawal(predeploys.L2CrossDomainMessengerAddr, msg.Tx.ToAddress, msg.Tx.FromAddress, msg.Tx.Data, msg.Nonce)
	migratedWithdrawal, err := crossdomain.MigrateWithdrawal(legacyWithdrawal, &l1Cdm, big.NewInt(int64(preset)))
	if err != nil {
		return common.Hash{}, err
	}

	return migratedWithdrawal.Hash()
}

func legacyBridgeMessageV1MessageHash(msg *database.BridgeMessage) (common.Hash, error) {
	legacyWithdrawal := crossdomain.NewLegacyWithdrawal(predeploys.L2CrossDomainMessengerAddr, msg.Tx.ToAddress, msg.Tx.FromAddress, msg.Tx.Data, msg.Nonce)
	value, err := legacyWithdrawal.Value()
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to extract ETH value from legacy bridge message: %w", err)
	}

	// Note: GasLimit is always zero. Only the GasLimit for the withdrawal transaction was migrated
	return crossdomain.HashCrossDomainMessageV1(msg.Nonce, msg.Tx.FromAddress, msg.Tx.ToAddress, value, new(big.Int), msg.Tx.Data)
}
