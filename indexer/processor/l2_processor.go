package processor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type L2Contracts struct {
	L2CrossDomainMessenger common.Address
	L2StandardBridge       common.Address
	L2ERC721Bridge         common.Address
	L2ToL1MessagePasser    common.Address

	// Some more contracts -- ProxyAdmin, SystemConfig, etcc
	// Ignore the auxiliary contracts?

	// Legacy Contracts? We'll add this in to index the legacy chain.
	// Remove afterwards?
}

func L2ContractPredeploys() L2Contracts {
	return L2Contracts{
		L2CrossDomainMessenger: common.HexToAddress("0x4200000000000000000000000000000000000007"),
		L2StandardBridge:       common.HexToAddress("0x4200000000000000000000000000000000000010"),
		L2ERC721Bridge:         common.HexToAddress("0x4200000000000000000000000000000000000014"),
		L2ToL1MessagePasser:    common.HexToAddress("0x4200000000000000000000000000000000000016"),
	}
}

func (c L2Contracts) ToSlice() []common.Address {
	fields := reflect.VisibleFields(reflect.TypeOf(c))
	v := reflect.ValueOf(c)

	contracts := make([]common.Address, len(fields))
	for i, field := range fields {
		contracts[i] = (v.FieldByName(field.Name).Interface()).(common.Address)
	}

	return contracts
}

type L2Processor struct {
	processor
}

func NewL2Processor(logger log.Logger, ethClient node.EthClient, db *database.DB, l2Contracts L2Contracts) (*L2Processor, error) {
	l2ProcessLog := logger.New("processor", "l2")
	l2ProcessLog.Info("initializing processor")

	latestHeader, err := db.Blocks.LatestL2BlockHeader()
	if err != nil {
		return nil, err
	}

	var fromL2Header *types.Header
	if latestHeader != nil {
		l2ProcessLog.Info("detected last indexed block", "height", latestHeader.Number.Int, "hash", latestHeader.Hash)
		l2Header, err := ethClient.BlockHeaderByHash(latestHeader.Hash)
		if err != nil {
			l2ProcessLog.Error("unable to fetch header for last indexed block", "hash", latestHeader.Hash, "err", err)
			return nil, err
		}

		fromL2Header = l2Header
	} else {
		l2ProcessLog.Info("no indexed state, starting from genesis")
		fromL2Header = nil
	}

	l2Processor := &L2Processor{
		processor: processor{
			headerTraversal: node.NewHeaderTraversal(ethClient, fromL2Header),
			db:              db,
			processFn:       l2ProcessFn(l2ProcessLog, ethClient, l2Contracts),
			processLog:      l2ProcessLog,
		},
	}

	return l2Processor, nil
}

func l2ProcessFn(processLog log.Logger, ethClient node.EthClient, l2Contracts L2Contracts) ProcessFn {
	rawEthClient := ethclient.NewClient(ethClient.RawRpcClient())

	contractAddrs := l2Contracts.ToSlice()
	processLog.Info("processor configured with contracts", "contracts", l2Contracts)
	return func(db *database.DB, headers []*types.Header) error {
		numHeaders := len(headers)

		/** Index all L2 blocks **/

		l2Headers := make([]*database.L2BlockHeader, len(headers))
		l2HeaderMap := make(map[common.Hash]*types.Header)
		for i, header := range headers {
			blockHash := header.Hash()
			l2Headers[i] = &database.L2BlockHeader{
				BlockHeader: database.BlockHeader{
					Hash:       blockHash,
					ParentHash: header.ParentHash,
					Number:     database.U256{Int: header.Number},
					Timestamp:  header.Time,
				},
			}

			l2HeaderMap[blockHash] = header
		}

		/** Watch for Contract Events **/

		logFilter := ethereum.FilterQuery{FromBlock: headers[0].Number, ToBlock: headers[numHeaders-1].Number, Addresses: contractAddrs}
		logs, err := rawEthClient.FilterLogs(context.Background(), logFilter)
		if err != nil {
			return err
		}

		l2ContractEvents := make([]*database.L2ContractEvent, len(logs))
		processedContractEvents := NewProcessedContractEvents()
		for i := range logs {
			log := &logs[i]
			header, ok := l2HeaderMap[log.BlockHash]
			if !ok {
				processLog.Error("contract event found with associated header not in the batch", "header", header, "log_index", log.Index)
				return errors.New("parsed log with a block hash not in this batch")
			}

			contractEvent := processedContractEvents.AddLog(log, header.Time)
			l2ContractEvents[i] = &database.L2ContractEvent{ContractEvent: *contractEvent}
		}

		/** Update Database **/

		processLog.Info("saving l2 blocks", "size", numHeaders)
		err = db.Blocks.StoreL2BlockHeaders(l2Headers)
		if err != nil {
			return err
		}

		numLogs := len(l2ContractEvents)
		if numLogs > 0 {
			processLog.Info("detected contract logs", "size", numLogs)
			err = db.ContractEvents.StoreL2ContractEvents(l2ContractEvents)
			if err != nil {
				return err
			}

			// forward along contract events to bridge txs processor
			err = l2ProcessContractEventsBridgeTransactions(processLog, db, processedContractEvents)
			if err != nil {
				return err
			}

			err = l2ProcessContractEventsBridgeCrossDomainMessages(processLog, db, processedContractEvents)
			if err != nil {
				return err
			}

			// forward along contract events to standard bridge processor
			err = l2ProcessContractEventsStandardBridge(processLog, db, ethClient, processedContractEvents)
			if err != nil {
				return err
			}
		}

		// a-ok!
		return nil
	}
}

func l2ProcessContractEventsBridgeTransactions(processLog log.Logger, db *database.DB, events *ProcessedContractEvents) error {
	// (1) Process New Withdrawals
	messagesPassed, err := L2ToL1MessagePasserMessagesPassed(events)
	if err != nil {
		return err
	}

	ethWithdrawals := []*database.L2BridgeWithdrawal{}
	transactionWithdrawals := make([]*database.L2TransactionWithdrawal, len(messagesPassed))
	for i, withdrawalEvent := range messagesPassed {
		transactionWithdrawals[i] = &database.L2TransactionWithdrawal{
			WithdrawalHash:       withdrawalEvent.WithdrawalHash,
			InitiatedL2EventGUID: withdrawalEvent.RawEvent.GUID,
			Nonce:                database.U256{Int: withdrawalEvent.Nonce},
			GasLimit:             database.U256{Int: withdrawalEvent.GasLimit},
			Tx: database.Transaction{
				FromAddress: withdrawalEvent.Sender,
				ToAddress:   withdrawalEvent.Target,
				Amount:      database.U256{Int: withdrawalEvent.Value},
				Data:        withdrawalEvent.Data,
				Timestamp:   withdrawalEvent.RawEvent.Timestamp,
			},
		}

		if len(withdrawalEvent.Data) == 0 && withdrawalEvent.Value.BitLen() > 0 {
			ethWithdrawals = append(ethWithdrawals, &database.L2BridgeWithdrawal{
				TransactionWithdrawalHash: withdrawalEvent.WithdrawalHash,
				Tx:                        transactionWithdrawals[i].Tx,
				TokenPair: database.TokenPair{
					L1TokenAddress: predeploys.LegacyERC20ETHAddr,
					L2TokenAddress: predeploys.LegacyERC20ETHAddr,
				},
			})
		}
	}

	if len(transactionWithdrawals) > 0 {
		processLog.Info("detected transaction withdrawals", "size", len(transactionWithdrawals))
		err := db.BridgeTransactions.StoreL2TransactionWithdrawals(transactionWithdrawals)
		if err != nil {
			return err
		}

		if len(ethWithdrawals) > 0 {
			processLog.Info("detected L2ToL1MessagePasser ETH transfers", "size", len(ethWithdrawals))
			err := db.BridgeTransfers.StoreL2BridgeWithdrawals(ethWithdrawals)
			if err != nil {
				return err
			}
		}
	}

	// (2) Process Deposit Finalization
	//  - Since L2 deposits are apart of the block derivation processes, we dont track finalization as it's too tricky
	// to do so purely from the L2-side since there is not a way to easily identify deposit transactions on L2 without walking
	// the transaction list of every L2 epoch.

	// a-ok!
	return nil
}

func l2ProcessContractEventsBridgeCrossDomainMessages(processLog log.Logger, db *database.DB, events *ProcessedContractEvents) error {
	l2ToL1MessagePasserABI, err := bindings.NewL2ToL1MessagePasser(common.Address{}, nil)
	if err != nil {
		return err
	}

	// (2) Process New Messages
	sentMessageEvents, err := CrossDomainMessengerSentMessageEvents(events)
	if err != nil {
		return err
	}

	sentMessages := make([]*database.L2BridgeMessage, len(sentMessageEvents))
	for i, sentMessageEvent := range sentMessageEvents {
		log := events.eventLog[sentMessageEvent.RawEvent.GUID]

		// extract the withdrawal hash from the previous MessagePassed event
		msgPassedLog := events.eventLog[events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index - 1}].GUID]
		msgPassedEvent, err := l2ToL1MessagePasserABI.ParseMessagePassed(*msgPassedLog)
		if err != nil {
			return err
		}

		sentMessages[i] = &database.L2BridgeMessage{
			TransactionWithdrawalHash: msgPassedEvent.WithdrawalHash,
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
		processLog.Info("detected L2CrossDomainMessenger messages", "size", len(sentMessages))
		err := db.BridgeMessages.StoreL2BridgeMessages(sentMessages)
		if err != nil {
			return err
		}
	}

	// (2) Process Relayed Messages.
	//
	// NOTE: Should we care about failed messages? A failed message can be
	// inferred via an included deposit on L2 that has not been marked as relayed.
	relayedMessageEvents, err := CrossDomainMessengerRelayedMessageEvents(events)
	if err != nil {
		return err
	}

	latestL1Header, err := db.Blocks.LatestL1BlockHeader()
	if err != nil {
		return err
	} else if len(relayedMessageEvents) > 0 && latestL1Header == nil {
		return errors.New("no indexed L1 headers to relay messages. waiting for L1Processor to catch up")
	}

	for _, relayedMessage := range relayedMessageEvents {
		message, err := db.BridgeMessages.L1BridgeMessageByHash(relayedMessage.MsgHash)
		if err != nil {
			return err
		}

		if message == nil {
			// Since the transaction processor running prior does not ensure the deposit inclusion, we need to
			// ensure we are in a caught up state before claiming a missing event. Since L2 timestamps are derived
			// from L1, we can simply compare the timestamp of this event with the latest L1 header.
			if latestL1Header == nil || relayedMessage.RawEvent.Timestamp > latestL1Header.Timestamp {
				processLog.Warn("waiting for L1Processor to catch up on L1CrossDomainMessages")
				return errors.New("waiting for L1Processor to catch up")
			} else {
				processLog.Crit("missing indexed L1CrossDomainMessenger message", "message_hash", relayedMessage.MsgHash)
				return fmt.Errorf("missing indexed L1CrossDomainMessager mesesage: 0x%x", relayedMessage.MsgHash)
			}
		}

		err = db.BridgeMessages.MarkRelayedL1BridgeMessage(relayedMessage.MsgHash, relayedMessage.RawEvent.GUID)
		if err != nil {
			return err
		}
	}

	if len(relayedMessageEvents) > 0 {
		processLog.Info("relayed L1CrossDomainMessenger messages", "size", len(relayedMessageEvents))
	}

	// a-ok!
	return nil
}

func l2ProcessContractEventsStandardBridge(processLog log.Logger, db *database.DB, ethClient node.EthClient, events *ProcessedContractEvents) error {
	rawEthClient := ethclient.NewClient(ethClient.RawRpcClient())

	l2ToL1MessagePasserABI, err := bindings.NewL2ToL1MessagePasser(common.Address{}, nil)
	if err != nil {
		return err
	}

	// (1) Process New Withdrawals
	initiatedWithdrawalEvents, err := StandardBridgeInitiatedEvents(events)
	if err != nil {
		return err
	}

	withdrawals := make([]*database.L2BridgeWithdrawal, len(initiatedWithdrawalEvents))
	for i, initiatedBridgeEvent := range initiatedWithdrawalEvents {
		log := events.eventLog[initiatedBridgeEvent.RawEvent.GUID]

		// extract the withdrawal hash from the following MessagePassed event
		msgPassedLog := events.eventLog[events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index + 1}].GUID]
		msgPassedEvent, err := l2ToL1MessagePasserABI.ParseMessagePassed(*msgPassedLog)
		if err != nil {
			return err
		}

		withdrawals[i] = &database.L2BridgeWithdrawal{
			TransactionWithdrawalHash: msgPassedEvent.WithdrawalHash,
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

	if len(withdrawals) > 0 {
		processLog.Info("detected L2StandardBridge withdrawals", "num", len(withdrawals))
		err := db.BridgeTransfers.StoreL2BridgeWithdrawals(withdrawals)
		if err != nil {
			return err
		}
	}

	// (2) Process Finalized Deposits
	//  - We dont need do anything actionable on the database here as this is layered on top of the
	// bridge transaction & messages that have a tracked lifecyle. We simply walk through and ensure
	// that the corresponding initiated deposits exist as an integrity check

	finalizedDepositEvents, err := StandardBridgeFinalizedEvents(rawEthClient, events)
	if err != nil {
		return err
	}

	for _, finalizedDepositEvent := range finalizedDepositEvents {
		deposit, err := db.BridgeTransfers.L1BridgeDepositByCrossDomainMessengerNonce(finalizedDepositEvent.CrossDomainMessengerNonce)
		if err != nil {
			return err
		} else if deposit == nil {
			// Indexed CrossDomainMessenger messages ensure we're in a caught up state here
			processLog.Error("missing indexed L1StandardBridge deposit on finalization", "cross_domain_messenger_nonce", finalizedDepositEvent.CrossDomainMessengerNonce)
			return errors.New("missing indexed L1StandardBridge deposit on finalization")
		}

		// sanity check on the bridge fields
		if finalizedDepositEvent.From != deposit.Tx.FromAddress || finalizedDepositEvent.To != deposit.Tx.ToAddress ||
			finalizedDepositEvent.Amount.Cmp(deposit.Tx.Amount.Int) != 0 || !bytes.Equal(finalizedDepositEvent.ExtraData, deposit.Tx.Data) ||
			finalizedDepositEvent.LocalToken != deposit.TokenPair.L1TokenAddress || finalizedDepositEvent.RemoteToken != deposit.TokenPair.L2TokenAddress {
			processLog.Error("bridge finalization fields mismatch with initiated fields!", "tx_source_hash", deposit.TransactionSourceHash, "cross_domain_messenger_nonce", deposit.CrossDomainMessengerNonce.Int)
			return errors.New("bridge tx mismatch")
		}
	}

	// a-ok!
	return nil
}
