package processor

import (
	"bytes"
	"context"
	"errors"
	"math/big"
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

func (c L2Contracts) toSlice() []common.Address {
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

func NewL2Processor(ethClient node.EthClient, db *database.DB, l2Contracts L2Contracts) (*L2Processor, error) {
	l2ProcessLog := log.New("processor", "l2")
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

	contractAddrs := l2Contracts.toSlice()
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

		numLogs := len(logs)
		logsByIndex := make(map[uint]*types.Log, numLogs)

		l2ContractEvents := make([]*database.L2ContractEvent, numLogs)
		l2ContractEventLogs := make(map[uuid.UUID]*types.Log)
		for i, log := range logs {
			header, ok := l2HeaderMap[log.BlockHash]
			if !ok {
				processLog.Error("contract event found with associated header not in the batch", "header", header, "log_index", log.Index)
				return errors.New("parsed log with a block hash not in this batch")
			}

			logsByIndex[log.Index] = &logs[i]
			contractEvent := &database.L2ContractEvent{ContractEvent: database.ContractEventFromLog(&log, header.Time)}

			l2ContractEvents[i] = contractEvent
			l2ContractEventLogs[contractEvent.GUID] = &logs[i]
		}

		/** Update Database **/

		processLog.Info("saving l2 blocks", "size", numHeaders)
		err = db.Blocks.StoreL2BlockHeaders(l2Headers)
		if err != nil {
			return err
		}

		if numLogs > 0 {
			processLog.Info("detected contract logs", "size", numLogs)
			err = db.ContractEvents.StoreL2ContractEvents(l2ContractEvents)
			if err != nil {
				return err
			}

			// forward along contract events to the bridge processor
			err = l2BridgeProcessContractEvents(processLog, db, ethClient, l2ContractEvents, l2ContractEventLogs, logsByIndex)
			if err != nil {
				return err
			}
		}

		// a-ok!
		return nil
	}
}

func l2BridgeProcessContractEvents(
	processLog log.Logger,
	db *database.DB,
	ethClient node.EthClient,
	events []*database.L2ContractEvent,
	eventLogs map[uuid.UUID]*types.Log,
	logsByIndex map[uint]*types.Log,
) error {
	rawEthClient := ethclient.NewClient(ethClient.RawRpcClient())

	l2StandardBridgeABI, err := bindings.L2StandardBridgeMetaData.GetAbi()
	if err != nil {
		return err
	}

	l2CrossDomainMessengerABI, err := bindings.L2CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return err
	}

	type BridgeData struct {
		From      common.Address
		To        common.Address
		Amount    *big.Int
		ExtraData []byte
	}

	numFinalizedDeposits := 0
	ethBridgeFinalizedEventSig := l2StandardBridgeABI.Events["ETHBridgeFinalized"].ID
	relayedMessageEventSig := l2CrossDomainMessengerABI.Events["RelayedMessage"].ID
	relayMessageMethod := l2CrossDomainMessengerABI.Methods["relayMessage"]
	for _, contractEvent := range events {
		eventSig := contractEvent.EventSignature
		log := eventLogs[contractEvent.GUID]
		if eventSig == ethBridgeFinalizedEventSig {
			// (1) Ensure the RelayedMessage follows the log right after the bridge event
			relayedMsgLog := logsByIndex[log.Index+1]
			if relayedMsgLog.Topics[0] != relayedMessageEventSig {
				processLog.Crit("expected CrossDomainMessenger#RelayedMessage following StandardBridge#EthBridgeFinalized event", "event_sig", relayedMsgLog.Topics[0], "relayed_message_sig", relayedMessageEventSig)
				return errors.New("unexpected bridge event ordering")
			}

			// unfortunately there's no way to extract the nonce on the relayed message event. we can
			// extract the nonce by unpacking the transaction input for the `relayMessage` transaction
			tx, isPending, err := rawEthClient.TransactionByHash(context.Background(), relayedMsgLog.TxHash)
			if err != nil || isPending {
				processLog.Crit("CrossDomainMessager#relayeMessage transaction query err or found pending", err, "err", "isPending", isPending)
				return errors.New("unable to query relayMessage tx")
			}

			txData := tx.Data()
			fnSelector := txData[:4]
			if !bytes.Equal(fnSelector, relayMessageMethod.ID) {
				processLog.Crit("expected relayMessage function selector")
				return errors.New("RelayMessage log does not match relayMessage transaction")
			}

			fnData := txData[4:]
			inputsMap := make(map[string]interface{})
			err = relayMessageMethod.Inputs.UnpackIntoMap(inputsMap, fnData)
			if err != nil {
				processLog.Crit("unable to unpack CrossDomainMessenger#relayMessage function data", "err", err)
				return err
			}

			nonce, ok := inputsMap["_nonce"].(*big.Int)
			if !ok {
				processLog.Crit("unable to extract _nonce from CrossDomainMessenger#relayMessage function call")
				return errors.New("unable to extract relayMessage nonce")
			}

			// (2) Mark initiated L1 deposit as finalized
			deposit, err := db.Bridge.DepositByMessageNonce(nonce)
			if err != nil {
				processLog.Error("error querying initiated deposit messsage using nonce", "nonce", nonce)
				return err
			} else if deposit == nil {
				latestNonce, err := db.Bridge.LatestDepositMessageNonce()
				if err != nil {
					return err
				}

				// check if the the L1Processor is behind or really has missed an event
				if latestNonce == nil || nonce.Cmp(latestNonce) > 0 {
					processLog.Warn("behind on indexed L1 deposits", "deposit_message_nonce", nonce, "latest_deposit_message_nonce", latestNonce)
					return errors.New("waiting for L1Processor to catch up")
				} else {
					processLog.Crit("missing indexed deposit for this finalization event", "deposit_message_nonce", nonce, "tx_hash", log.TxHash, "log_index", log.Index)
					return errors.New("missing deposit message")
				}
			}

			err = db.Bridge.MarkFinalizedDepositEvent(deposit.GUID, contractEvent.GUID)
			if err != nil {
				processLog.Error("error finalizing deposit", "err", err)
				return err
			}

			numFinalizedDeposits++
		}
	}

	// a-ok!
	if numFinalizedDeposits > 0 {
		processLog.Info("finalized deposits", "num", numFinalizedDeposits)
	}

	return nil
}
