package processor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	legacy_bindings "github.com/ethereum-optimism/optimism/op-bindings/legacy-bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type L1Contracts struct {
	OptimismPortal         common.Address
	L2OutputOracle         common.Address
	L1CrossDomainMessenger common.Address
	L1StandardBridge       common.Address
	L1ERC721Bridge         common.Address

	// Some more contracts -- ProxyAdmin, SystemConfig, etcc
	// Ignore the auxiliary contracts?

	// Legacy contracts? We'll add this in to index the legacy chain.
	// Remove afterwards?
}

func (c L1Contracts) ToSlice() []common.Address {
	fields := reflect.VisibleFields(reflect.TypeOf(c))
	v := reflect.ValueOf(c)

	contracts := make([]common.Address, len(fields))
	for i, field := range fields {
		contracts[i] = (v.FieldByName(field.Name).Interface()).(common.Address)
	}

	return contracts
}

type checkpointAbi struct {
	l2OutputOracle             *abi.ABI
	legacyStateCommitmentChain *abi.ABI
}

type L1Processor struct {
	processor
}

func NewL1Processor(logger log.Logger, ethClient node.EthClient, db *database.DB, l1Contracts L1Contracts) (*L1Processor, error) {
	l1ProcessLog := logger.New("processor", "l1")
	l1ProcessLog.Info("initializing processor")

	l2OutputOracleABI, err := bindings.L2OutputOracleMetaData.GetAbi()
	if err != nil {
		l1ProcessLog.Error("unable to generate L2OutputOracle ABI", "err", err)
		return nil, err
	}
	legacyStateCommitmentChainABI, err := legacy_bindings.StateCommitmentChainMetaData.GetAbi()
	if err != nil {
		l1ProcessLog.Error("unable to generate legacy StateCommitmentChain ABI", "err", err)
		return nil, err
	}
	checkpointAbi := checkpointAbi{l2OutputOracle: l2OutputOracleABI, legacyStateCommitmentChain: legacyStateCommitmentChainABI}

	latestHeader, err := db.Blocks.LatestL1BlockHeader()
	if err != nil {
		return nil, err
	}

	var fromL1Header *types.Header
	if latestHeader != nil {
		l1ProcessLog.Info("detected last indexed block", "height", latestHeader.Number.Int, "hash", latestHeader.Hash)
		l1Header, err := ethClient.BlockHeaderByHash(latestHeader.Hash)
		if err != nil {
			l1ProcessLog.Error("unable to fetch header for last indexed block", "hash", latestHeader.Hash, "err", err)
			return nil, err
		}

		fromL1Header = l1Header
	} else {
		// we shouldn't start from genesis with l1. Need a "genesis" L1 height provided for the rollup
		l1ProcessLog.Info("no indexed state, starting from genesis")
		fromL1Header = nil
	}

	l1Processor := &L1Processor{
		processor: processor{
			headerTraversal: node.NewHeaderTraversal(ethClient, fromL1Header),
			db:              db,
			processFn:       l1ProcessFn(l1ProcessLog, ethClient, l1Contracts, checkpointAbi),
			processLog:      l1ProcessLog,
		},
	}

	return l1Processor, nil
}

func l1ProcessFn(processLog log.Logger, ethClient node.EthClient, l1Contracts L1Contracts, checkpointAbi checkpointAbi) ProcessFn {
	rawEthClient := ethclient.NewClient(ethClient.RawRpcClient())

	contractAddrs := l1Contracts.ToSlice()
	processLog.Info("processor configured with contracts", "contracts", l1Contracts)

	outputProposedEventName := "OutputProposed"
	outputProposedEventSig := checkpointAbi.l2OutputOracle.Events[outputProposedEventName].ID

	legacyStateBatchAppendedEventName := "StateBatchAppended"
	legacyStateBatchAppendedEventSig := checkpointAbi.legacyStateCommitmentChain.Events[legacyStateBatchAppendedEventName].ID

	return func(db *database.DB, headers []*types.Header) error {
		headerMap := make(map[common.Hash]*types.Header)
		for _, header := range headers {
			headerMap[header.Hash()] = header
		}

		/** Watch for all Optimism Contract Events **/

		logFilter := ethereum.FilterQuery{FromBlock: headers[0].Number, ToBlock: headers[len(headers)-1].Number, Addresses: contractAddrs}
		logs, err := rawEthClient.FilterLogs(context.Background(), logFilter) // []types.Log
		if err != nil {
			return err
		}

		// L2 checkpoints posted on L1
		outputProposals := []*database.OutputProposal{}
		legacyStateBatches := []*database.LegacyStateBatch{}

		l1HeadersOfInterest := make(map[common.Hash]bool)
		l1ContractEvents := make([]*database.L1ContractEvent, len(logs))

		processedContractEvents := NewProcessedContractEvents()
		for i := range logs {
			log := &logs[i]
			header, ok := headerMap[log.BlockHash]
			if !ok {
				processLog.Error("contract event found with associated header not in the batch", "header", log.BlockHash, "log_index", log.Index)
				return errors.New("parsed log with a block hash not in this batch")
			}

			contractEvent := processedContractEvents.AddLog(log, header.Time)
			l1HeadersOfInterest[log.BlockHash] = true
			l1ContractEvents[i] = &database.L1ContractEvent{ContractEvent: *contractEvent}

			// Track Checkpoint Events for L2
			switch contractEvent.EventSignature {
			case outputProposedEventSig:
				var outputProposed bindings.L2OutputOracleOutputProposed
				err := UnpackLog(&outputProposed, log, outputProposedEventName, checkpointAbi.l2OutputOracle)
				if err != nil {
					return err
				}

				outputProposals = append(outputProposals, &database.OutputProposal{
					OutputRoot:          outputProposed.OutputRoot,
					L2OutputIndex:       database.U256{Int: outputProposed.L2OutputIndex},
					L2BlockNumber:       database.U256{Int: outputProposed.L2BlockNumber},
					L1ContractEventGUID: contractEvent.GUID,
				})

			case legacyStateBatchAppendedEventSig:
				var stateBatchAppended legacy_bindings.StateCommitmentChainStateBatchAppended
				err := UnpackLog(&stateBatchAppended, log, legacyStateBatchAppendedEventName, checkpointAbi.legacyStateCommitmentChain)
				if err != nil {
					return err
				}

				legacyStateBatches = append(legacyStateBatches, &database.LegacyStateBatch{
					Index:               stateBatchAppended.BatchIndex.Uint64(),
					Root:                stateBatchAppended.BatchRoot,
					Size:                stateBatchAppended.BatchSize.Uint64(),
					PrevTotal:           stateBatchAppended.PrevTotalElements.Uint64(),
					L1ContractEventGUID: contractEvent.GUID,
				})
			}
		}

		/** Aggregate applicable L1 Blocks **/

		// we iterate on the original array to maintain ordering. probably can find a more efficient
		// way to iterate over the `l1HeadersOfInterest` map while maintaining ordering
		indexedL1Headers := []*database.L1BlockHeader{}
		for _, header := range headers {
			_, hasLogs := l1HeadersOfInterest[header.Hash()]
			if !hasLogs {
				continue
			}

			indexedL1Headers = append(indexedL1Headers, &database.L1BlockHeader{BlockHeader: database.BlockHeaderFromGethHeader(header)})
		}

		/** Update Database **/

		numIndexedL1Headers := len(indexedL1Headers)
		if numIndexedL1Headers > 0 {
			processLog.Info("saving l1 blocks with optimism logs", "size", numIndexedL1Headers, "batch_size", len(headers))
			err = db.Blocks.StoreL1BlockHeaders(indexedL1Headers)
			if err != nil {
				return err
			}

			// Since the headers to index are derived from the existence of logs, we know in this branch `numLogs > 0`
			processLog.Info("detected contract logs", "size", len(l1ContractEvents))
			err = db.ContractEvents.StoreL1ContractEvents(l1ContractEvents)
			if err != nil {
				return err
			}

			// Mark L2 checkpoints that have been recorded on L1 (L2OutputProposal & StateBatchAppended events)
			numLegacyStateBatches := len(legacyStateBatches)
			if numLegacyStateBatches > 0 {
				latestBatch := legacyStateBatches[numLegacyStateBatches-1]
				latestL2Height := latestBatch.PrevTotal + latestBatch.Size - 1
				processLog.Info("detected legacy state batches", "size", numLegacyStateBatches, "latest_l2_block_number", latestL2Height)
			}

			numOutputProposals := len(outputProposals)
			if numOutputProposals > 0 {
				latestL2Height := outputProposals[numOutputProposals-1].L2BlockNumber.Int
				processLog.Info("detected output proposals", "size", numOutputProposals, "latest_l2_block_number", latestL2Height)
				err := db.Blocks.StoreOutputProposals(outputProposals)
				if err != nil {
					return err
				}
			}

			// forward along contract events to bridge txs processor
			err = l1ProcessContractEventsBridgeTransactions(processLog, db, l1Contracts, processedContractEvents)
			if err != nil {
				return err
			}

			// forward along contract events to bridge messages processor
			err = l1ProcessContractEventsBridgeCrossDomainMessages(processLog, db, processedContractEvents)
			if err != nil {
				return err
			}

			// forward along contract events to standard bridge processor
			err = l1ProcessContractEventsStandardBridge(processLog, db, ethClient, processedContractEvents)
			if err != nil {
				return err
			}
		} else {
			processLog.Info("no l1 blocks of interest within batch")
		}

		// a-ok!
		return nil
	}
}

func l1ProcessContractEventsBridgeTransactions(processLog log.Logger, db *database.DB, l1Contracts L1Contracts, events *ProcessedContractEvents) error {
	// (1) Process New Deposits
	portalDeposits, err := OptimismPortalTransactionDepositEvents(events)
	if err != nil {
		return err
	}

	ethDeposits := []*database.L1BridgeDeposit{}
	transactionDeposits := make([]*database.L1TransactionDeposit, len(portalDeposits))
	for i, depositEvent := range portalDeposits {
		depositTx := depositEvent.DepositTx
		transactionDeposits[i] = &database.L1TransactionDeposit{
			SourceHash:           depositTx.SourceHash,
			L2TransactionHash:    types.NewTx(depositTx).Hash(),
			InitiatedL1EventGUID: depositEvent.RawEvent.GUID,
			Version:              database.U256{Int: depositEvent.Version},
			OpaqueData:           depositEvent.OpaqueData,
			GasLimit:             database.U256{Int: new(big.Int).SetUint64(depositTx.Gas)},
			Tx: database.Transaction{
				FromAddress: depositTx.From,
				ToAddress:   depositTx.From,
				Amount:      database.U256{Int: depositTx.Value},
				Data:        depositTx.Data,
				Timestamp:   depositEvent.RawEvent.Timestamp,
			},
		}

		// catch ETH transfers to the portal contract.
		if len(depositTx.Data) == 0 && depositTx.Value.BitLen() > 0 {
			ethDeposits = append(ethDeposits, &database.L1BridgeDeposit{
				TransactionSourceHash: depositTx.SourceHash,
				Tx:                    transactionDeposits[i].Tx,
				TokenPair: database.TokenPair{
					L1TokenAddress: predeploys.LegacyERC20ETHAddr,
					L2TokenAddress: predeploys.LegacyERC20ETHAddr,
				},
			})
		}
	}

	if len(transactionDeposits) > 0 {
		processLog.Info("detected transaction deposits", "size", len(transactionDeposits))
		err := db.BridgeTransactions.StoreL1TransactionDeposits(transactionDeposits)
		if err != nil {
			return err
		}

		if len(ethDeposits) > 0 {
			processLog.Info("detected portal ETH transfers", "size", len(ethDeposits))
			err := db.BridgeTransfers.StoreL1BridgeDeposits(ethDeposits)
			if err != nil {
				return err
			}
		}
	}

	// (2) Process Proven Withdrawals
	provenWithdrawals, err := OptimismPortalWithdrawalProvenEvents(events)
	if err != nil {
		return err
	}

	latestL2Header, err := db.Blocks.LatestL2BlockHeader()
	if err != nil {
		return nil
	} else if len(provenWithdrawals) > 0 && latestL2Header == nil {
		return errors.New("no indexed L2 headers to prove withdrawals. waiting for L2Processor to catch up")
	}

	for _, provenWithdrawal := range provenWithdrawals {
		withdrawalHash := provenWithdrawal.WithdrawalHash
		withdrawal, err := db.BridgeTransactions.L2TransactionWithdrawal(withdrawalHash)
		if err != nil {
			return err
		}

		if withdrawal == nil {
			// We need to ensure we are in a caught up state before claiming a missing event. Since L2 timestamps
			// are derived from L1, we can simply compare the timestamp of this event with the latest L2 header.
			if provenWithdrawal.RawEvent.Timestamp > latestL2Header.Timestamp {
				processLog.Warn("behind on indexed L2 withdrawals")
				return errors.New("waiting for L2Processor to catch up")
			} else {
				processLog.Crit("L2 withdrawal missing!", "withdrawal_hash", withdrawalHash)
				return errors.New("withdrawal missing!")
			}
		}

		err = db.BridgeTransactions.MarkL2TransactionWithdrawalProvenEvent(withdrawalHash, provenWithdrawal.RawEvent.GUID)
		if err != nil {
			return err
		}
	}

	if len(provenWithdrawals) > 0 {
		processLog.Info("proven transaction withdrawals", "size", len(provenWithdrawals))
	}

	// (2) Process Withdrawal Finalization
	finalizedWithdrawals, err := OptimismPortalWithdrawalFinalizedEvents(events)
	if err != nil {
		return err
	}

	for _, finalizedWithdrawal := range finalizedWithdrawals {
		withdrawalHash := finalizedWithdrawal.WithdrawalHash
		withdrawal, err := db.BridgeTransactions.L2TransactionWithdrawal(withdrawalHash)
		if err != nil {
			return err
		} else if withdrawal == nil {
			// since withdrawals must be proven first, we don't have to check on the L2Processor
			processLog.Crit("withdrawal missing!", "hash", withdrawalHash)
			return errors.New("withdrawal missing!")
		}

		err = db.BridgeTransactions.MarkL2TransactionWithdrawalFinalizedEvent(withdrawalHash, finalizedWithdrawal.RawEvent.GUID, finalizedWithdrawal.Success)
		if err != nil {
			return err
		}
	}

	if len(finalizedWithdrawals) > 0 {
		processLog.Info("finalized transaction withdrawals", "size", len(finalizedWithdrawals))
	}

	// a-ok
	return nil
}

func l1ProcessContractEventsBridgeCrossDomainMessages(processLog log.Logger, db *database.DB, events *ProcessedContractEvents) error {
	// (1) Process New Messages
	sentMessageEvents, err := CrossDomainMessengerSentMessageEvents(events)
	if err != nil {
		return err
	}

	sentMessages := make([]*database.L1BridgeMessage, len(sentMessageEvents))
	for i, sentMessageEvent := range sentMessageEvents {
		log := sentMessageEvent.RawEvent.GethLog

		// extract the deposit hash from the previous TransactionDepositedEvent
		transactionDepositedLog := events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index - 1}].GethLog
		depositTx, err := derive.UnmarshalDepositLogEvent(transactionDepositedLog)
		if err != nil {
			return err
		}

		sentMessages[i] = &database.L1BridgeMessage{
			TransactionSourceHash: depositTx.SourceHash,
			BridgeMessage: database.BridgeMessage{
				Nonce:                database.U256{Int: sentMessageEvent.MessageNonce},
				MessageHash:          sentMessageEvent.MessageHash,
				SentMessageEventGUID: sentMessageEvent.RawEvent.GUID,
				GasLimit:             database.U256{Int: sentMessageEvent.GasLimit},
				Tx: database.Transaction{
					FromAddress: sentMessageEvent.Sender,
					ToAddress:   sentMessageEvent.Target,
					Amount:      database.U256{Int: sentMessageEvent.Value},
					Data:        sentMessageEvent.Message,
					Timestamp:   sentMessageEvent.RawEvent.Timestamp,
				},
			},
		}
	}

	if len(sentMessages) > 0 {
		processLog.Info("detected L1CrossDomainMessenger messages", "size", len(sentMessages))
		err := db.BridgeMessages.StoreL1BridgeMessages(sentMessages)
		if err != nil {
			return err
		}
	}

	// (2) Process Relayed Messages.
	//
	// NOTE: Should we care about failed messages? A failed message can be
	// inferred via a finalized withdrawal that has not been marked as relayed.
	relayedMessageEvents, err := CrossDomainMessengerRelayedMessageEvents(events)
	if err != nil {
		return err
	}

	for _, relayedMessage := range relayedMessageEvents {
		message, err := db.BridgeMessages.L2BridgeMessageByHash(relayedMessage.MsgHash)
		if err != nil {
			return err
		} else if message == nil {
			// Since L2 withdrawals must be proven before being relayed, the transaction processor
			// ensures that we are in indexed state on L2 if we've seen this finalization event
			processLog.Crit("missing indexed L2CrossDomainMessenger sent message", "message_hash", relayedMessage.MsgHash)
			return fmt.Errorf("missing indexed L2CrossDomainMessager mesesage: 0x%x", relayedMessage.MsgHash)
		}

		err = db.BridgeMessages.MarkRelayedL2BridgeMessage(relayedMessage.MsgHash, relayedMessage.RawEvent.GUID)
		if err != nil {
			return err
		}
	}

	if len(relayedMessageEvents) > 0 {
		processLog.Info("relayed L2CrossDomainMessenger messages", "size", len(relayedMessageEvents))
	}

	// a-ok!
	return nil
}

func l1ProcessContractEventsStandardBridge(processLog log.Logger, db *database.DB, ethClient node.EthClient, events *ProcessedContractEvents) error {
	rawEthClient := ethclient.NewClient(ethClient.RawRpcClient())

	// (1) Process New Deposits
	initiatedDepositEvents, err := StandardBridgeInitiatedEvents(events)
	if err != nil {
		return err
	}

	deposits := make([]*database.L1BridgeDeposit, len(initiatedDepositEvents))
	for i, initiatedBridgeEvent := range initiatedDepositEvents {
		log := initiatedBridgeEvent.RawEvent.GethLog

		// extract the deposit hash from the following TransactionDeposited event
		transactionDepositedLog := events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index + 1}].GethLog
		depositTx, err := derive.UnmarshalDepositLogEvent(transactionDepositedLog)
		if err != nil {
			return err
		}

		deposits[i] = &database.L1BridgeDeposit{
			TransactionSourceHash:     depositTx.SourceHash,
			CrossDomainMessengerNonce: &database.U256{Int: initiatedBridgeEvent.CrossDomainMessengerNonce},
			TokenPair:                 database.TokenPair{L1TokenAddress: initiatedBridgeEvent.LocalToken, L2TokenAddress: initiatedBridgeEvent.RemoteToken},
			Tx: database.Transaction{
				FromAddress: initiatedBridgeEvent.From,
				ToAddress:   initiatedBridgeEvent.To,
				Amount:      database.U256{Int: initiatedBridgeEvent.Amount},
				Data:        initiatedBridgeEvent.ExtraData,
				Timestamp:   initiatedBridgeEvent.RawEvent.Timestamp,
			},
		}
	}

	if len(deposits) > 0 {
		processLog.Info("detected L1StandardBridge deposits", "size", len(deposits))
		err := db.BridgeTransfers.StoreL1BridgeDeposits(deposits)
		if err != nil {
			return err
		}
	}

	// (2) Process Finalized Withdrawals
	//  - We dont need do anything actionable on the database here as this is layered on top of the
	// bridge transaction & messages that have a tracked lifecyle. We simply walk through and ensure
	// that the corresponding initiated withdrawals exist and match as an integrity check
	finalizedWithdrawalEvents, err := StandardBridgeFinalizedEvents(rawEthClient, events)
	if err != nil {
		return err
	}

	for _, finalizedWithdrawalEvent := range finalizedWithdrawalEvents {
		withdrawal, err := db.BridgeTransfers.L2BridgeWithdrawalByCrossDomainMessengerNonce(finalizedWithdrawalEvent.CrossDomainMessengerNonce)
		if err != nil {
			return err
		} else if withdrawal == nil {
			processLog.Error("missing indexed L2StandardBridge withdrawal for finalization", "cross_domain_messenger_nonce", finalizedWithdrawalEvent.CrossDomainMessengerNonce)
			return errors.New("missing indexed L2StandardBridge withdrawal for finalization event")
		}

		// sanity check on the bridge fields
		if finalizedWithdrawalEvent.From != withdrawal.Tx.FromAddress || finalizedWithdrawalEvent.To != withdrawal.Tx.ToAddress ||
			finalizedWithdrawalEvent.Amount.Cmp(withdrawal.Tx.Amount.Int) != 0 || !bytes.Equal(finalizedWithdrawalEvent.ExtraData, withdrawal.Tx.Data) ||
			finalizedWithdrawalEvent.LocalToken != withdrawal.TokenPair.L1TokenAddress || finalizedWithdrawalEvent.RemoteToken != withdrawal.TokenPair.L2TokenAddress {
			processLog.Crit("bridge finalization fields mismatch with initiated fields!", "tx_withdrawal_hash", withdrawal.TransactionWithdrawalHash, "cross_domain_messenger_nonce", withdrawal.CrossDomainMessengerNonce.Int)
			return errors.New("bridge tx mismatch!")
		}
	}

	// a-ok!
	return nil
}
