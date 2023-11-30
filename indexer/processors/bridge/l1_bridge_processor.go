package bridge

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/bigint"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/processors/bridge/ovm1"
	"github.com/ethereum-optimism/optimism/indexer/processors/contracts"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// L1ProcessInitiatedBridgeEvents will query the database for bridge events that have been initiated between
// the specified block range. This covers every part of the multi-layered stack:
//  1. OptimismPortal
//  2. L1CrossDomainMessenger
//  3. L1StandardBridge
func L1ProcessInitiatedBridgeEvents(log log.Logger, db *database.DB, metrics L1Metricer, l1Contracts config.L1Contracts, fromHeight, toHeight *big.Int) error {
	// (1) OptimismPortal
	optimismPortalTxDeposits, err := contracts.OptimismPortalTransactionDepositEvents(l1Contracts.OptimismPortalProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(optimismPortalTxDeposits) > 0 {
		log.Info("detected transaction deposits", "size", len(optimismPortalTxDeposits))
	}

	mintedGWEI := bigint.Zero
	portalDeposits := make(map[logKey]*contracts.OptimismPortalTransactionDepositEvent, len(optimismPortalTxDeposits))
	transactionDeposits := make([]database.L1TransactionDeposit, len(optimismPortalTxDeposits))
	for i := range optimismPortalTxDeposits {
		depositTx := optimismPortalTxDeposits[i]
		portalDeposits[logKey{depositTx.Event.BlockHash, depositTx.Event.LogIndex}] = &depositTx
		mintedGWEI = new(big.Int).Add(mintedGWEI, depositTx.Tx.Amount)

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

		// Convert to from wei to eth
		mintedETH, _ := bigint.WeiToETH(mintedGWEI).Float64()
		metrics.RecordL1TransactionDeposits(len(transactionDeposits), mintedETH)
	}

	// (2) L1CrossDomainMessenger
	crossDomainSentMessages, err := contracts.CrossDomainMessengerSentMessageEvents("l1", l1Contracts.L1CrossDomainMessengerProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainSentMessages) > 0 {
		log.Info("detected sent messages", "size", len(crossDomainSentMessages))
	}

	sentMessages := make(map[logKey]*contracts.CrossDomainMessengerSentMessageEvent, len(crossDomainSentMessages))
	bridgeMessages := make([]database.L1BridgeMessage, len(crossDomainSentMessages))
	for i := range crossDomainSentMessages {
		sentMessage := crossDomainSentMessages[i]
		sentMessages[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex}] = &sentMessage

		// extract the deposit hash from the previous TransactionDepositedEvent
		portalDeposit, ok := portalDeposits[logKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex - 1}]
		if !ok {
			return fmt.Errorf("expected TransactionDeposit preceding SentMessage event. tx_hash = %s", sentMessage.Event.TransactionHash.String())
		} else if portalDeposit.Event.TransactionHash != sentMessage.Event.TransactionHash {
			return fmt.Errorf("correlated events tx hash mismatch. deposit_tx_hash = %s, message_tx_hash = %s", portalDeposit.Event.TransactionHash, sentMessage.Event.TransactionHash)
		}

		bridgeMessages[i] = database.L1BridgeMessage{
			TransactionSourceHash: portalDeposit.DepositTx.SourceHash,
			BridgeMessage:         sentMessage.BridgeMessage,
		}
	}
	if len(bridgeMessages) > 0 {
		if err := db.BridgeMessages.StoreL1BridgeMessages(bridgeMessages); err != nil {
			return err
		}
		metrics.RecordL1CrossDomainSentMessages(len(bridgeMessages))
	}

	// (3) L1StandardBridge
	initiatedBridges, err := contracts.StandardBridgeInitiatedEvents("l1", l1Contracts.L1StandardBridgeProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(initiatedBridges) > 0 {
		log.Info("detected bridge deposits", "size", len(initiatedBridges))
	}

	bridgedTokens := make(map[common.Address]int)
	bridgeDeposits := make([]database.L1BridgeDeposit, len(initiatedBridges))
	for i := range initiatedBridges {
		initiatedBridge := initiatedBridges[i]

		// extract the cross domain message hash & deposit source hash from the following events
		portalDeposit, ok := portalDeposits[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex + 1}]
		if !ok {
			return fmt.Errorf("expected TransactionDeposit following BridgeInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		} else if portalDeposit.Event.TransactionHash != initiatedBridge.Event.TransactionHash {
			return fmt.Errorf("correlated events tx hash mismatch, bridge_tx_hash = %s, deposit_tx_hash = %s", initiatedBridge.Event.TransactionHash, portalDeposit.Event.TransactionHash)
		}

		sentMessage, ok := sentMessages[logKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex + 2}]
		if !ok {
			return fmt.Errorf("expected SentMessage following BridgeInitiated event. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		} else if sentMessage.Event.TransactionHash != initiatedBridge.Event.TransactionHash {
			return fmt.Errorf("correlated events tx hash mismatch. bridge_tx_hash = %s, message_tx_hash = %s", initiatedBridge.Event.TransactionHash, sentMessage.Event.TransactionHash)
		}

		bridgedTokens[initiatedBridge.BridgeTransfer.TokenPair.LocalTokenAddress]++

		initiatedBridge.BridgeTransfer.CrossDomainMessageHash = &sentMessage.BridgeMessage.MessageHash
		bridgeDeposits[i] = database.L1BridgeDeposit{
			TransactionSourceHash: portalDeposit.DepositTx.SourceHash,
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

	return nil
}

// L1ProcessFinalizedBridgeEvent will query the database for all the finalization markers for all initiated
// bridge events. This covers every part of the multi-layered stack:
//  1. OptimismPortal (Bedrock prove & finalize steps)
//  2. L1CrossDomainMessenger (relayMessage marker)
//  3. L1StandardBridge (no-op, since this is simply a wrapper over the L1CrossDomainMessenger)
func L1ProcessFinalizedBridgeEvents(log log.Logger, db *database.DB, metrics L1Metricer, l1Contracts config.L1Contracts, fromHeight, toHeight *big.Int) error {
	// (1) OptimismPortal (proven withdrawals)
	provenWithdrawals, err := contracts.OptimismPortalWithdrawalProvenEvents(l1Contracts.OptimismPortalProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(provenWithdrawals) > 0 {
		log.Info("detected proven withdrawals", "size", len(provenWithdrawals))
	}

	skippedOVM1ProvenWithdrawals := 0
	for i := range provenWithdrawals {
		provenWithdrawal := provenWithdrawals[i]
		withdrawal, err := db.BridgeTransactions.L2TransactionWithdrawal(provenWithdrawal.WithdrawalHash)
		if err != nil {
			return err
		} else if withdrawal == nil {
			if _, ok := ovm1.PortalWithdrawalTransactions[provenWithdrawal.WithdrawalHash]; ok {
				skippedOVM1ProvenWithdrawals++
				continue
			}
			return fmt.Errorf("missing indexed withdrawal! tx_hash = %s", provenWithdrawal.Event.TransactionHash)
		}

		if err := db.BridgeTransactions.MarkL2TransactionWithdrawalProvenEvent(provenWithdrawal.WithdrawalHash, provenWithdrawals[i].Event.GUID); err != nil {
			return fmt.Errorf("failed to mark withdrawal as proven. tx_hash = %s: %w", provenWithdrawal.Event.TransactionHash, err)
		}
	}
	if len(provenWithdrawals) > 0 {
		metrics.RecordL1ProvenWithdrawals(len(provenWithdrawals))
		if skippedOVM1ProvenWithdrawals > 0 {
			metrics.RecordL1SkippedOVM1ProvenWithdrawals(skippedOVM1ProvenWithdrawals)
			log.Info("skipped OVM 1.0 proven withdrawals", "size", skippedOVM1ProvenWithdrawals)
		}
	}

	// (2) OptimismPortal (finalized withdrawals)
	finalizedWithdrawals, err := contracts.OptimismPortalWithdrawalFinalizedEvents(l1Contracts.OptimismPortalProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(finalizedWithdrawals) > 0 {
		log.Info("detected finalized withdrawals", "size", len(finalizedWithdrawals))
	}

	skippedOVM1FinalizedWithdrawals := 0
	for i := range finalizedWithdrawals {
		finalizedWithdrawal := finalizedWithdrawals[i]
		withdrawal, err := db.BridgeTransactions.L2TransactionWithdrawal(finalizedWithdrawal.WithdrawalHash)
		if err != nil {
			return err
		} else if withdrawal == nil {
			if _, ok := ovm1.PortalWithdrawalTransactions[finalizedWithdrawal.WithdrawalHash]; ok {
				skippedOVM1FinalizedWithdrawals++
				continue
			}
			return fmt.Errorf("missing indexed withdrawal on finalization! tx_hash = %s", finalizedWithdrawal.Event.TransactionHash.String())
		}

		if err = db.BridgeTransactions.MarkL2TransactionWithdrawalFinalizedEvent(finalizedWithdrawal.WithdrawalHash, finalizedWithdrawal.Event.GUID, finalizedWithdrawal.Success); err != nil {
			return fmt.Errorf("failed to mark withdrawal as finalized. tx_hash = %s: %w", finalizedWithdrawal.Event.TransactionHash, err)
		}
	}
	if len(finalizedWithdrawals) > 0 {
		metrics.RecordL1FinalizedWithdrawals(len(finalizedWithdrawals))
		if skippedOVM1ProvenWithdrawals > 0 {
			metrics.RecordL1SkippedOVM1FinalizedWithdrawals(skippedOVM1FinalizedWithdrawals)
			log.Info("skipped OVM 1.0 finalized withdrawals", "size", skippedOVM1FinalizedWithdrawals)
		}
	}

	// (3) L1CrossDomainMessenger
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
			return fmt.Errorf("missing indexed L2CrossDomainMessager message! tx_hash = %s", relayedMessage.Event.TransactionHash.String())
		}

		if err := db.BridgeMessages.MarkRelayedL2BridgeMessage(relayedMessage.MessageHash, relayedMessage.Event.GUID); err != nil {
			return fmt.Errorf("failed to relay cross domain message. tx_hash = %s: %w", relayedMessage.Event.TransactionHash, err)
		}
	}
	if len(crossDomainRelayedMessages) > 0 {
		metrics.RecordL1CrossDomainRelayedMessages(len(crossDomainRelayedMessages))
		if skippedOVM1Messages > 0 { // Logged as a warning just for visibility
			metrics.RecordL1SkippedOVM1CrossDomainRelayedMessages(skippedOVM1Messages)
			log.Info("skipped OVM 1.0 relayed L2CrossDomainMessenger withdrawals", "size", skippedOVM1Messages)
		}
	}

	// (4) L1StandardBridge
	// - Nothing actionable on the database. Since the StandardBridge is layered ontop of the
	// CrossDomainMessenger, there's no need for any sanity or invariant checks as the previous step
	// ensures a relayed message (finalized bridge) can be linked with a sent message (initiated bridge).
	finalizedBridges, err := contracts.StandardBridgeFinalizedEvents("l1", l1Contracts.L1StandardBridgeProxy, db, fromHeight, toHeight)
	if err != nil {
		return err
	}

	finalizedTokens := make(map[common.Address]int)
	for i := range finalizedBridges {
		finalizedBridge := finalizedBridges[i]
		finalizedTokens[finalizedBridge.BridgeTransfer.TokenPair.LocalTokenAddress]++
	}
	if len(finalizedBridges) > 0 {
		log.Info("detected finalized bridge withdrawals", "size", len(finalizedBridges))
		for tokenAddr, size := range finalizedTokens {
			metrics.RecordL1FinalizedBridgeTransfers(tokenAddr, size)
		}
	}

	// a-ok!
	return nil
}
