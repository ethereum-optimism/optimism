package processor

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"reflect"

	"github.com/google/uuid"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	legacy_bindings "github.com/ethereum-optimism/optimism/op-bindings/legacy-bindings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ethAddress = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")
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

type checkpointAbi struct {
	l2OutputOracle             *abi.ABI
	legacyStateCommitmentChain *abi.ABI
}

func (c L1Contracts) toSlice() []common.Address {
	fields := reflect.VisibleFields(reflect.TypeOf(c))
	v := reflect.ValueOf(c)

	contracts := make([]common.Address, len(fields))
	for i, field := range fields {
		contracts[i] = (v.FieldByName(field.Name).Interface()).(common.Address)
	}

	return contracts
}

type L1Processor struct {
	processor
}

func NewL1Processor(ethClient node.EthClient, db *database.DB, l1Contracts L1Contracts) (*L1Processor, error) {
	l1ProcessLog := log.New("processor", "l1")
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

	contractAddrs := l1Contracts.toSlice()
	processLog.Info("processor configured with contracts", "contracts", l1Contracts)

	outputProposedEventSig := checkpointAbi.l2OutputOracle.Events["OutputProposed"].ID
	legacyStateBatchAppendedEventSig := checkpointAbi.legacyStateCommitmentChain.Events["StateBatchAppended"].ID

	return func(db *database.DB, headers []*types.Header) error {
		numHeaders := len(headers)
		headerMap := make(map[common.Hash]*types.Header)
		for _, header := range headers {
			headerMap[header.Hash()] = header
		}

		/** Watch for all Optimism Contract Events **/

		logFilter := ethereum.FilterQuery{FromBlock: headers[0].Number, ToBlock: headers[numHeaders-1].Number, Addresses: contractAddrs}
		logs, err := rawEthClient.FilterLogs(context.Background(), logFilter) // []types.Log
		if err != nil {
			return err
		}

		// L2 checkpoitns posted on L1
		outputProposals := []*database.OutputProposal{}
		legacyStateBatches := []*database.LegacyStateBatch{}

		numLogs := len(logs)
		logsByIndex := make(map[uint]*types.Log, numLogs)

		l1ContractEvents := make([]*database.L1ContractEvent, numLogs)
		l1ContractEventLogs := make(map[uuid.UUID]*types.Log)
		l1HeadersOfInterest := make(map[common.Hash]bool)
		for i, log := range logs {
			header, ok := headerMap[log.BlockHash]
			if !ok {
				processLog.Error("contract event found with associated header not in the batch", "header", log.BlockHash, "log_index", log.Index)
				return errors.New("parsed log with a block hash not in this batch")
			}

			logsByIndex[log.Index] = &logs[i]
			contractEvent := &database.L1ContractEvent{ContractEvent: database.ContractEventFromLog(&log, header.Time)}

			l1ContractEvents[i] = contractEvent
			l1ContractEventLogs[contractEvent.GUID] = &logs[i]
			l1HeadersOfInterest[log.BlockHash] = true

			// Track Checkpoint Events for L2
			switch contractEvent.EventSignature {
			case outputProposedEventSig:
				if len(log.Topics) != 4 {
					processLog.Error("parsed unexpected number of L2OutputOracle#OutputProposed log topics", "log_topics", log.Topics)
					return errors.New("parsed unexpected OutputProposed event")
				}

				outputProposals = append(outputProposals, &database.OutputProposal{
					OutputRoot:          log.Topics[1],
					L2BlockNumber:       database.U256{Int: new(big.Int).SetBytes(log.Topics[2].Bytes())},
					L1ContractEventGUID: contractEvent.GUID,
				})

			case legacyStateBatchAppendedEventSig:
				var stateBatchAppended legacy_bindings.StateCommitmentChainStateBatchAppended
				err := checkpointAbi.l2OutputOracle.UnpackIntoInterface(&stateBatchAppended, "StateBatchAppended", log.Data)
				if err != nil || len(log.Topics) != 2 {
					processLog.Error("unexpected StateCommitmentChain#StateBatchAppended log data or log topics", "log_topics", log.Topics, "log_data", hex.EncodeToString(log.Data), "err", err)
					return err
				}

				legacyStateBatches = append(legacyStateBatches, &database.LegacyStateBatch{
					Index:               new(big.Int).SetBytes(log.Topics[1].Bytes()).Uint64(),
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

			indexedL1Headers = append(
				indexedL1Headers,
				&database.L1BlockHeader{BlockHeader: database.BlockHeaderFromHeader(header)},
			)
		}

		/** Update Database **/

		numIndexedL1Headers := len(indexedL1Headers)
		if numIndexedL1Headers > 0 {
			processLog.Info("saved l1 blocks of interest within batch", "num", numIndexedL1Headers, "batchSize", numHeaders)
			err = db.Blocks.StoreL1BlockHeaders(indexedL1Headers)
			if err != nil {
				return err
			}

			// Since the headers to index are derived from the existence of logs, we know in this branch `numLogs > 0`
			processLog.Info("saving contract logs", "size", numLogs)
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

			// forward along contract events to the bridge processor
			err = l1BridgeProcessContractEvents(processLog, db, l1ContractEvents, l1ContractEventLogs, logsByIndex)
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

func l1BridgeProcessContractEvents(
	processLog log.Logger,
	db *database.DB,
	events []*database.L1ContractEvent,
	eventLogs map[uuid.UUID]*types.Log,
	logsByIndex map[uint]*types.Log,
) error {
	l1StandardBridgeABI, err := bindings.L1StandardBridgeMetaData.GetAbi()
	if err != nil {
		return err
	}

	l1CrossDomainMessengerABI, err := bindings.L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return err
	}

	l1StandardBridgeDeposits := []*database.Deposit{}
	ethBridgeInitiatedEventSig := l1StandardBridgeABI.Events["ETHBridgeInitiated"].ID
	sentMessageEventSig := l1CrossDomainMessengerABI.Events["SentMessage"].ID
	for _, contractEvent := range events {
		eventSig := contractEvent.EventSignature
		log := eventLogs[contractEvent.GUID]
		if eventSig == ethBridgeInitiatedEventSig {
			// (1) Deconstruct the bridge event
			var bridgeData bindings.L1StandardBridgeETHBridgeInitiated
			err = l1StandardBridgeABI.UnpackIntoInterface(&bridgeData, "ETHBridgeInitiated", log.Data)
			if err != nil || len(log.Topics) != 3 {
				processLog.Crit("unexpected ETHDepositInitiated log format", "tx", log.TxHash, "err", err)
				return err
			}

			// from/to must be retrieved from log topics
			bridgeData.From = common.BytesToAddress(log.Topics[1].Bytes())
			bridgeData.To = common.BytesToAddress(log.Topics[2].Bytes())

			// (2) Look for the sent message event to extract the associated messager nonce
			//       - The `SentMessage` event is the second after the bridge initiated event. BridgeInitiated -> Portal#DepositTransaction -> SentMesage ...
			sentMsgLog := logsByIndex[log.Index+2]
			if sentMsgLog.Topics[0] != sentMessageEventSig {
				processLog.Crit("expected CrossDomainMessenger#SentMessage to follow StandardBridge#EthBridgeInitiated event", "event_sig", sentMsgLog.Topics[0], "sent_message_sig", sentMessageEventSig)
				return errors.New("unexpected bridge event ordering")
			}

			expectedMsg, err := l1StandardBridgeABI.Pack("finalizeBridgeETH", bridgeData.From, bridgeData.To, bridgeData.Amount, bridgeData.ExtraData)
			if err != nil {
				processLog.Crit("unable to create bridge message")
				return err
			}

			var sentMsg bindings.L1CrossDomainMessengerSentMessage
			err = l1CrossDomainMessengerABI.UnpackIntoInterface(&sentMsg, "SentMessage", sentMsgLog.Data)
			if err != nil {
				processLog.Crit("unexpected SentMessage log format", "tx", log.TxHash, "err", err)
				return err
			} else if !bytes.Equal(sentMsg.Message, expectedMsg) {
				processLog.Crit("SentMessage message mismatch", "expected_bridge_msg", hex.EncodeToString(expectedMsg), "event_msg", hex.EncodeToString(sentMsg.Message))
				return errors.New("bridge message mismatch")
			}

			// (3) Record the deposit
			l1StandardBridgeDeposits = append(l1StandardBridgeDeposits, &database.Deposit{
				GUID:                 uuid.New(),
				InitiatedL1EventGUID: contractEvent.GUID,
				SentMessageNonce:     database.U256{Int: sentMsg.MessageNonce},
				TokenPair:            database.TokenPair{L1TokenAddress: ethAddress, L2TokenAddress: ethAddress},
				Tx: database.Transaction{
					FromAddress: common.BytesToAddress(log.Topics[1].Bytes()),
					ToAddress:   common.BytesToAddress(log.Topics[2].Bytes()),
					Amount:      database.U256{Int: bridgeData.Amount},
					Data:        bridgeData.ExtraData,
					Timestamp:   contractEvent.Timestamp,
				},
			})
		}
	}

	if len(l1StandardBridgeDeposits) > 0 {
		processLog.Info("detected L1StandardBridge deposits", "num", len(l1StandardBridgeDeposits))
		return db.Bridge.StoreDeposits(l1StandardBridgeDeposits)
	}

	// no-op
	return nil
}
