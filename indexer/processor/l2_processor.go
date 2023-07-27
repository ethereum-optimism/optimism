package processor

import (
	"context"
	"errors"
	"reflect"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/google/uuid"

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

			// forward along contract events to the bridge processor
			err = l2BridgeProcessContractEvents(processLog, db, ethClient, processedContractEvents)
			if err != nil {
				return err
			}
		}

		// a-ok!
		return nil
	}
}

func l2BridgeProcessContractEvents(processLog log.Logger, db *database.DB, ethClient node.EthClient, events *ProcessedContractEvents) error {
	rawEthClient := ethclient.NewClient(ethClient.RawRpcClient())

	l2ToL1MessagePasserABI, err := bindings.L2ToL1MessagePasserMetaData.GetAbi()
	if err != nil {
		return err
	}

	messagePassedEventAbi := l2ToL1MessagePasserABI.Events["MessagePassed"]

	// Process New Withdrawals
	initiatedWithdrawalEvents, err := StandardBridgeInitiatedEvents(events)
	if err != nil {
		return err
	}

	withdrawals := make([]*database.Withdrawal, len(initiatedWithdrawalEvents))
	for i, initiatedBridgeEvent := range initiatedWithdrawalEvents {
		log := events.eventLog[initiatedBridgeEvent.RawEvent.GUID]

		// extract the withdrawal hash from the MessagePassed event
		var msgPassedData bindings.L2ToL1MessagePasserMessagePassed
		msgPassedLog := events.eventLog[events.eventByLogIndex[ProcessedContractEventLogIndexKey{log.BlockHash, log.Index + 1}].GUID]
		err := UnpackLog(&msgPassedData, msgPassedLog, messagePassedEventAbi.Name, l2ToL1MessagePasserABI)
		if err != nil {
			return err
		}

		withdrawals[i] = &database.Withdrawal{
			GUID:                 uuid.New(),
			InitiatedL2EventGUID: initiatedBridgeEvent.RawEvent.GUID,
			SentMessageNonce:     database.U256{Int: initiatedBridgeEvent.CrossDomainMessengerNonce},
			WithdrawalHash:       msgPassedData.WithdrawalHash,
			TokenPair:            database.TokenPair{L1TokenAddress: initiatedBridgeEvent.LocalToken, L2TokenAddress: initiatedBridgeEvent.RemoteToken},
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
		err := db.Bridge.StoreWithdrawals(withdrawals)
		if err != nil {
			return err
		}
	}

	// Finalize Deposits
	finalizationBridgeEvents, err := StandardBridgeFinalizedEvents(rawEthClient, events)
	if err != nil {
		return err
	}

	for _, finalizedBridgeEvent := range finalizationBridgeEvents {
		nonce := finalizedBridgeEvent.CrossDomainMessengerNonce

		deposit, err := db.Bridge.DepositByMessageNonce(nonce)
		if err != nil {
			processLog.Error("error querying associated deposit messsage using nonce", "cross_domain_messenger_nonce", nonce)
			return err
		} else if deposit == nil {
			latestNonce, err := db.Bridge.LatestDepositMessageNonce()
			if err != nil {
				return err
			}

			// Check if the L1Processor is behind or really has missed an event
			if latestNonce == nil || nonce.Cmp(latestNonce) > 0 {
				processLog.Warn("behind on indexed L1 deposits")
				return errors.New("waiting for L1Processor to catch up")
			} else {
				processLog.Crit("missing indexed deposit for this finalization event")
				return errors.New("missing deposit message")
			}
		}

		err = db.Bridge.MarkFinalizedDepositEvent(deposit.GUID, finalizedBridgeEvent.RawEvent.GUID)
		if err != nil {
			processLog.Error("error finalizing deposit", "err", err)
			return err
		}
	}

	if len(finalizationBridgeEvents) > 0 {
		processLog.Info("finalized L1StandardBridge deposits", "size", len(finalizationBridgeEvents))
	}

	// a-ok
	return nil
}
