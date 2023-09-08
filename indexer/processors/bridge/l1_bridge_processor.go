package bridge

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/processors/contracts"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// L1ProcessInitiatedBridgeEvents will query the database for new bridge events that have been initiated between
// the specified block range. This covers every part of the multi-layered stack:
//  1. OptimismPortal
//  2. L1CrossDomainMessenger
//  3. L1StandardBridge
func L1ProcessInitiatedBridgeEvents(log log.Logger, db *database.DB, chainConfig config.ChainConfig, fromHeight *big.Int, toHeight *big.Int) error {
	// (1) OptimismPortal
	optimismPortalTxDeposits, err := contracts.OptimismPortalTransactionDepositEvents(chainConfig.L1Contracts.OptimismPortalProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}

	portalDeposits := make(map[logKey]*contracts.OptimismPortalTransactionDepositEvent, len(optimismPortalTxDeposits))
	transactionDeposits := make([]database.L1TransactionDeposit, len(optimismPortalTxDeposits))
	for i := range optimismPortalTxDeposits {
		depositTx := optimismPortalTxDeposits[i]
		portalDeposits[logKey{depositTx.Event.BlockHash, depositTx.Event.LogIndex}] = &depositTx
		transactionDeposits[i] = database.L1TransactionDeposit{
			SourceHash:           depositTx.DepositTx.SourceHash,
			L2TransactionHash:    types.NewTx(depositTx.DepositTx).Hash(),
			InitiatedL1EventGUID: depositTx.Event.GUID,
			GasLimit:             depositTx.GasLimit,
			Tx:                   depositTx.Tx,
		}
	}

	if len(transactionDeposits) > 0 {
		log.Info("detected transaction deposits", "size", len(transactionDeposits))
		if err := db.BridgeTransactions.StoreL1TransactionDeposits(transactionDeposits); err != nil {
			return err
		}
	}

	// (2) L1CrossDomainMessenger
	crossDomainSentMessages, err := contracts.CrossDomainMessengerSentMessageEvents("l1", chainConfig.L1Contracts.L1CrossDomainMessengerProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainSentMessages) > len(transactionDeposits) {
		return fmt.Errorf("missing transaction deposit for each cross-domain message. deposits: %d, messages: %d", len(transactionDeposits), len(crossDomainSentMessages))
	}

	sentMessages := make(map[logKey]*contracts.CrossDomainMessengerSentMessageEvent, len(crossDomainSentMessages))
	l1BridgeMessages := make([]database.L1BridgeMessage, len(crossDomainSentMessages))
	for i := range crossDomainSentMessages {
		sentMessage := crossDomainSentMessages[i]
		sentMessages[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex}] = &sentMessage

		// extract the deposit hash from the previous TransactionDepositedEvent
		portalDeposit, ok := portalDeposits[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex - 1}]
		if !ok {
			return fmt.Errorf("expected TransactionDeposit preceding SentMessage event. tx_hash = %s", sentMessage.Event.TransactionHash)
		}

		l1BridgeMessages[i] = database.L1BridgeMessage{TransactionSourceHash: portalDeposit.DepositTx.SourceHash, BridgeMessage: sentMessage.BridgeMessage}
	}

	if len(l1BridgeMessages) > 0 {
		log.Info("detected L1CrossDomainMessenger messages", "size", len(l1BridgeMessages))
		if err := db.BridgeMessages.StoreL1BridgeMessages(l1BridgeMessages); err != nil {
			return err
		}
	}

	// (3) L1StandardBridge
	initiatedBridges, err := contracts.StandardBridgeInitiatedEvents("l1", chainConfig.L1Contracts.L1StandardBridgeProxy, db, fromHeight, toHeight)
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
		portalDeposit, ok := portalDeposits[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex + 1}]
		if !ok {
			return fmt.Errorf("expected TransactionDeposit following BridgeInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		}
		sentMessage, ok := sentMessages[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex + 2}]
		if !ok {
			return fmt.Errorf("expected SentMessage following TransactionDeposit event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		}

		initiatedBridge.BridgeTransfer.CrossDomainMessageHash = &sentMessage.BridgeMessage.MessageHash
		l1BridgeDeposits[i] = database.L1BridgeDeposit{
			TransactionSourceHash: portalDeposit.DepositTx.SourceHash,
			BridgeTransfer:        initiatedBridge.BridgeTransfer,
		}
	}

	if len(l1BridgeDeposits) > 0 {
		log.Info("detected L1StandardBridge deposits", "size", len(l1BridgeDeposits))
		if err := db.BridgeTransfers.StoreL1BridgeDeposits(l1BridgeDeposits); err != nil {
			return err
		}
	}

	return nil
}

// L1ProcessFinalizedBridgeEvent will query the database for all the finalization markers for all initiated
// bridge events. This covers every part of the multi-layered stack:
//  1. OptimismPortal (Bedrock prove & finalize steps)
//  2. L1CrossDomainMessenger (relayMessage marker)
//  3. L1StandardBridge (no-op, since this is simply a wrapper over the L1CrossDomainMEssenger)
func L1ProcessFinalizedBridgeEvents(log log.Logger, db *database.DB, chainConfig config.ChainConfig, fromHeight *big.Int, toHeight *big.Int) error {
	// (1) OptimismPortal (proven withdrawals)
	provenWithdrawals, err := contracts.OptimismPortalWithdrawalProvenEvents(chainConfig.L1Contracts.OptimismPortalProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}

	for i := range provenWithdrawals {
		proven := provenWithdrawals[i]
		withdrawal, err := db.BridgeTransactions.L2TransactionWithdrawal(proven.WithdrawalHash)
		if err != nil {
			return err
		} else if withdrawal == nil {
			log.Error("missing indexed withdrawal on prove event!", "withdrawal_hash", proven.WithdrawalHash, "tx_hash", proven.Event.TransactionHash)
			return errors.New("missing indexed withdrawal")
		}

		if err := db.BridgeTransactions.MarkL2TransactionWithdrawalProvenEvent(proven.WithdrawalHash, provenWithdrawals[i].Event.GUID); err != nil {
			return err
		}
	}

	if len(provenWithdrawals) > 0 {
		log.Info("proven transaction withdrawals", "size", len(provenWithdrawals))
	}

	// (2) OptimismPortal (finalized withdrawals)
	finalizedWithdrawals, err := contracts.OptimismPortalWithdrawalFinalizedEvents(chainConfig.L1Contracts.OptimismPortalProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}

	for i := range finalizedWithdrawals {
		finalized := finalizedWithdrawals[i]
		withdrawal, err := db.BridgeTransactions.L2TransactionWithdrawal(finalized.WithdrawalHash)
		if err != nil {
			return err
		} else if withdrawal == nil {
			log.Error("missing indexed withdrawal on finalization event!", "withdrawal_hash", finalized.WithdrawalHash, "tx_hash", finalized.Event.TransactionHash)
			return errors.New("missing indexed withdrawal")
		}

		if err = db.BridgeTransactions.MarkL2TransactionWithdrawalFinalizedEvent(finalized.WithdrawalHash, finalized.Event.GUID, finalized.Success); err != nil {
			return err
		}
	}

	if len(finalizedWithdrawals) > 0 {
		log.Info("finalized transaction withdrawals", "size", len(finalizedWithdrawals))
	}

	// (3) L1CrossDomainMessenger
	crossDomainRelayedMessages, err := contracts.CrossDomainMessengerRelayedMessageEvents("l1", chainConfig.L1Contracts.L1CrossDomainMessengerProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}

	relayedMessages := make(map[logKey]*contracts.CrossDomainMessengerRelayedMessageEvent, len(crossDomainRelayedMessages))
	for i := range crossDomainRelayedMessages {
		relayed := crossDomainRelayedMessages[i]
		relayedMessages[logKey{BlockHash: relayed.Event.BlockHash, LogIndex: relayed.Event.LogIndex}] = &relayed
		message, err := db.BridgeMessages.L2BridgeMessage(relayed.MessageHash)
		if err != nil {
			return err
		} else if message == nil {
			log.Error("missing indexed L2CrossDomainMessenger message", "message_hash", relayed.MessageHash, "tx_hash", relayed.Event.TransactionHash)
			return fmt.Errorf("missing indexed L2CrossDomainMessager message")
		}

		if err := db.BridgeMessages.MarkRelayedL2BridgeMessage(relayed.MessageHash, relayed.Event.GUID); err != nil {
			return err
		}
	}

	if len(crossDomainRelayedMessages) > 0 {
		log.Info("relayed L2CrossDomainMessenger messages", "size", len(crossDomainRelayedMessages))
	}

	// (4) L1StandardBridge
	finalizedBridges, err := contracts.StandardBridgeFinalizedEvents("l1", chainConfig.L1Contracts.L1StandardBridgeProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(finalizedBridges) > len(crossDomainRelayedMessages) {
		return fmt.Errorf("missing cross-domain message for each finalized bridge event. messages: %d, bridges: %d", len(crossDomainRelayedMessages), len(finalizedBridges))
	}

	for i := range finalizedBridges {
		// Nothing actionable on the database. However, we can treat the relayed message
		// as an invariant by ensuring we can query for a deposit by the same hash
		finalizedBridge := finalizedBridges[i]
		relayedMessage, ok := relayedMessages[logKey{finalizedBridge.Event.BlockHash, finalizedBridge.Event.LogIndex + 1}]
		if !ok {
			return fmt.Errorf("expected RelayedMessage following BridgeFinalized event. tx_hash = %s", finalizedBridge.Event.TransactionHash)
		}

		// Since the message hash is computed from the relayed message, this ensures the deposit fields must match. For good measure,
		// we may choose to make sure `withdrawal.BridgeTransfer` matches with the finalized bridge
		withdrawal, err := db.BridgeTransfers.L2BridgeWithdrawalWithFilter(database.BridgeTransfer{CrossDomainMessageHash: &relayedMessage.MessageHash})
		if err != nil {
			return err
		} else if withdrawal == nil {
			log.Error("missing L2StandardBridge withdrawal on L1 finalization", "tx_hash", finalizedBridge.Event.TransactionHash)
			return errors.New("missing L2StandardBridge withdrawal on L1 finalization")
		}
	}

	// a-ok!
	return nil
}
