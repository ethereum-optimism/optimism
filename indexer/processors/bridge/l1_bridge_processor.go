package bridge

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/processors/contracts"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// L1ProcessInitiatedBridgeEvents will query the database for bridge events that have been initiated between
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
	if len(optimismPortalTxDeposits) > 0 {
		log.Info("detected transaction deposits", "size", len(optimismPortalTxDeposits))
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
		if err := db.BridgeTransactions.StoreL1TransactionDeposits(transactionDeposits); err != nil {
			return err
		}
	}

	// (2) L1CrossDomainMessenger
	crossDomainSentMessages, err := contracts.CrossDomainMessengerSentMessageEvents("l1", chainConfig.L1Contracts.L1CrossDomainMessengerProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainSentMessages) > 0 {
		log.Info("detected sent messages", "size", len(crossDomainSentMessages))
	}

	sentMessages := make(map[logKey]*contracts.CrossDomainMessengerSentMessageEvent, len(crossDomainSentMessages))
	l1BridgeMessages := make([]database.L1BridgeMessage, len(crossDomainSentMessages))
	for i := range crossDomainSentMessages {
		sentMessage := crossDomainSentMessages[i]
		sentMessages[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex}] = &sentMessage

		// extract the deposit hash from the previous TransactionDepositedEvent
		portalDeposit, ok := portalDeposits[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex - 1}]
		if !ok {
			log.Error("expected TransactionDeposit preceding SentMessage event", "tx_hash", sentMessage.Event.TransactionHash.String())
			return fmt.Errorf("expected TransactionDeposit preceding SentMessage event. tx_hash = %s", sentMessage.Event.TransactionHash.String())
		}

		l1BridgeMessages[i] = database.L1BridgeMessage{TransactionSourceHash: portalDeposit.DepositTx.SourceHash, BridgeMessage: sentMessage.BridgeMessage}
	}
	if len(l1BridgeMessages) > 0 {
		if err := db.BridgeMessages.StoreL1BridgeMessages(l1BridgeMessages); err != nil {
			return err
		}
	}

	// (3) L1StandardBridge
	initiatedBridges, err := contracts.StandardBridgeInitiatedEvents("l1", chainConfig.L1Contracts.L1StandardBridgeProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(initiatedBridges) > 0 {
		log.Info("detected bridge deposits", "size", len(initiatedBridges))
	}

	l1BridgeDeposits := make([]database.L1BridgeDeposit, len(initiatedBridges))
	for i := range initiatedBridges {
		initiatedBridge := initiatedBridges[i]

		// extract the cross domain message hash & deposit source hash from the following events
		portalDeposit, ok := portalDeposits[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex + 1}]
		if !ok {
			log.Error("expected TransactionDeposit following BridgeInitiated event", "tx_hash", initiatedBridge.Event.TransactionHash.String())
			return fmt.Errorf("expected TransactionDeposit following BridgeInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash.String())
		}
		sentMessage, ok := sentMessages[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex + 2}]
		if !ok {
			log.Error("expected SentMessage following TransactionDeposit event", "tx_hash", initiatedBridge.Event.TransactionHash.String())
			return fmt.Errorf("expected SentMessage following TransactionDeposit event. tx_hash = %s", initiatedBridge.Event.TransactionHash.String())
		}

		initiatedBridge.BridgeTransfer.CrossDomainMessageHash = &sentMessage.BridgeMessage.MessageHash
		l1BridgeDeposits[i] = database.L1BridgeDeposit{
			TransactionSourceHash: portalDeposit.DepositTx.SourceHash,
			BridgeTransfer:        initiatedBridge.BridgeTransfer,
		}
	}
	if len(l1BridgeDeposits) > 0 {
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
//  3. L1StandardBridge (no-op, since this is simply a wrapper over the L1CrossDomainMessenger)
func L1ProcessFinalizedBridgeEvents(log log.Logger, db *database.DB, chainConfig config.ChainConfig, fromHeight *big.Int, toHeight *big.Int) error {
	// (1) OptimismPortal (proven withdrawals)
	provenWithdrawals, err := contracts.OptimismPortalWithdrawalProvenEvents(chainConfig.L1Contracts.OptimismPortalProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(provenWithdrawals) > 0 {
		log.Info("detected proven withdrawals", "size", len(provenWithdrawals))
	}

	for i := range provenWithdrawals {
		proven := provenWithdrawals[i]
		withdrawal, err := db.BridgeTransactions.L2TransactionWithdrawal(proven.WithdrawalHash)
		if err != nil {
			return err
		} else if withdrawal == nil {
			log.Error("missing indexed withdrawal on proven event!", "tx_hash", proven.Event.TransactionHash.String())
			return fmt.Errorf("missing indexed withdrawal! tx_hash = %s", proven.Event.TransactionHash.String())
		}

		if err := db.BridgeTransactions.MarkL2TransactionWithdrawalProvenEvent(proven.WithdrawalHash, provenWithdrawals[i].Event.GUID); err != nil {
			log.Error("failed to mark withdrawal as proven", "err", err, "tx_hash", proven.Event.TransactionHash.String())
			return err
		}
	}

	// (2) OptimismPortal (finalized withdrawals)
	finalizedWithdrawals, err := contracts.OptimismPortalWithdrawalFinalizedEvents(chainConfig.L1Contracts.OptimismPortalProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(finalizedWithdrawals) > 0 {
		log.Info("detected finalized withdrawals", "size", len(finalizedWithdrawals))
	}

	for i := range finalizedWithdrawals {
		finalized := finalizedWithdrawals[i]
		withdrawal, err := db.BridgeTransactions.L2TransactionWithdrawal(finalized.WithdrawalHash)
		if err != nil {
			return err
		} else if withdrawal == nil {
			log.Error("missing indexed withdrawal on finalization event!", "tx_hash", finalized.Event.TransactionHash.String())
			return fmt.Errorf("missing indexed withdrawal on finalization! tx_hash: %s", finalized.Event.TransactionHash.String())
		}

		if err = db.BridgeTransactions.MarkL2TransactionWithdrawalFinalizedEvent(finalized.WithdrawalHash, finalized.Event.GUID, finalized.Success); err != nil {
			log.Error("failed to mark withdrawal as finalized", "err", err, "tx_hash", finalized.Event.TransactionHash.String())
			return err
		}
	}

	// (3) L1CrossDomainMessenger
	crossDomainRelayedMessages, err := contracts.CrossDomainMessengerRelayedMessageEvents("l1", chainConfig.L1Contracts.L1CrossDomainMessengerProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainRelayedMessages) > 0 {
		log.Info("detected relayed messages", "size", len(crossDomainRelayedMessages))
	}

	relayedMessages := make(map[logKey]*contracts.CrossDomainMessengerRelayedMessageEvent, len(crossDomainRelayedMessages))
	for i := range crossDomainRelayedMessages {
		relayed := crossDomainRelayedMessages[i]
		relayedMessages[logKey{BlockHash: relayed.Event.BlockHash, LogIndex: relayed.Event.LogIndex}] = &relayed
		message, err := db.BridgeMessages.L2BridgeMessage(relayed.MessageHash)
		if err != nil {
			return err
		} else if message == nil {
			log.Error("missing indexed L2CrossDomainMessenger message", "tx_hash", relayed.Event.TransactionHash.String())
			return fmt.Errorf("missing indexed L2CrossDomainMessager message. tx_hash %s", relayed.Event.TransactionHash.String())
		}

		if err := db.BridgeMessages.MarkRelayedL2BridgeMessage(relayed.MessageHash, relayed.Event.GUID); err != nil {
			log.Error("failed to relay cross domain message", "err", err, "tx_hash", relayed.Event.TransactionHash.String())
			return err
		}
	}

	// (4) L1StandardBridge
	finalizedBridges, err := contracts.StandardBridgeFinalizedEvents("l1", chainConfig.L1Contracts.L1StandardBridgeProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(finalizedBridges) > 0 {
		log.Info("detected finalized bridge withdrawals", "size", len(finalizedBridges))
	}

	for i := range finalizedBridges {
		// Nothing actionable on the database. However, we can treat the relayed message
		// as an invariant by ensuring we can query for a deposit by the same hash
		finalizedBridge := finalizedBridges[i]
		relayedMessage, ok := relayedMessages[logKey{finalizedBridge.Event.BlockHash, finalizedBridge.Event.LogIndex + 1}]
		if !ok {
			log.Error("expected RelayedMessage following BridgeFinalized event", "tx_hash", finalizedBridge.Event.TransactionHash.String())
			return fmt.Errorf("expected RelayedMessage following BridgeFinalized event. tx_hash = %s", finalizedBridge.Event.TransactionHash.String())
		}

		// Since the message hash is computed from the relayed message, this ensures the deposit fields must match. For good measure,
		// we may choose to make sure `withdrawal.BridgeTransfer` matches with the finalized bridge
		withdrawal, err := db.BridgeTransfers.L2BridgeWithdrawalWithFilter(database.BridgeTransfer{CrossDomainMessageHash: &relayedMessage.MessageHash})
		if err != nil {
			return err
		} else if withdrawal == nil {
			log.Error("missing L2StandardBridge withdrawal on L1 finalization", "tx_hash", finalizedBridge.Event.TransactionHash.String())
			return fmt.Errorf("missing L2StandardBridge withdrawal on L1 finalization. tx_hash: %s", finalizedBridge.Event.TransactionHash.String())
		}
	}

	// a-ok!
	return nil
}
