package processor

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const (
	defaultLoopInterval     = 5 * time.Second
	defaultHeaderBufferSize = 500
)

type ETL struct {
	log log.Logger

	headerTraversal *node.HeaderTraversal
	ethClient       *ethclient.Client
	contracts       []common.Address

	etlBatches chan ETLBatch
}

// (RPC | DB) <-> ETL -- Starting from some height

type ETLBatch struct {
	Logger log.Logger

	Headers   []types.Header
	HeaderMap map[common.Hash]*types.Header

	Logs           []types.Log
	HeadersWithLog map[common.Hash]bool
}

// This ETL runs at HEAD for a given client and indexes the
// applicable blocks & logs. Should also gracefully handle
// reorgs that may occur.
//
// I wonder if we can use this same ETL for backfilling
//
//	-> When at HEAD, fetches and indexes blocks/logs from RPC
//	-> When backfilling, fetches blocks/logs from DB
func (etl *ETL) Start(ctx context.Context) error {
	done := ctx.Done()
	pollTicker := time.NewTicker(defaultLoopInterval)
	defer pollTicker.Stop()

	etl.log.Info("starting etl...")
	var headers []types.Header
	for {
		select {
		case <-done:
			etl.log.Info("stopping etl")
			return nil

		case <-pollTicker.C:
			if len(headers) == 0 {
				newHeaders, err := etl.headerTraversal.NextFinalizedHeaders(defaultHeaderBufferSize)
				if err != nil {
					etl.log.Error("error querying for headers", "err", err)
					continue
				}
				if len(newHeaders) == 0 {
					// Logged as an error since this loop should be operating at a longer interval than the provider
					etl.log.Error("no new headers. processor unexpectedly at head...")
					continue
				}

				headers = newHeaders
			} else {
				etl.log.Info("retrying previous batch")
			}

			firstHeader := headers[0]
			lastHeader := headers[len(headers)-1]
			batchLog := etl.log.New("batch_start_block_number", firstHeader.Number, "batch_end_block_number", lastHeader.Number, "batch_size", len(headers))
			batchLog.Info("extracting batch")

			headerMap := make(map[common.Hash]*types.Header, len(headers))
			for i := range headers {
				headerMap[headers[i].Hash()] = &headers[i]
			}

			headersWithLog := make(map[common.Hash]bool, len(headers))
			logFilter := ethereum.FilterQuery{FromBlock: firstHeader.Number, ToBlock: lastHeader.Number, Addresses: etl.contracts}
			logs, err := etl.ethClient.FilterLogs(context.Background(), logFilter)
			if err != nil {
				batchLog.Info("unable to extract logs within batch", "err", err)
				continue // spin and try again
			}
			for i := range logs {
				if _, ok := headerMap[logs[i].BlockHash]; !ok {
					// TODO. Dont error out? Or is this a terminal state. Can this happen due to a reorg?
					batchLog.Error("log found with block hash not in the batch", "block_hash", logs[i].BlockHash, "log_index", logs[i].Index)
					return errors.New("parsed log with a block hash not in the fetched batch")
				}
				headersWithLog[logs[i].BlockHash] = true
			}

			if len(logs) > 0 {
				batchLog.Info("detected logs", "size", len(logs))
			}

			// create a new reference such that when we clear `headers`
			// the slice passed to `ETLBatch` remains unchanged
			headersRef := headers
			headers = nil
			etl.etlBatches <- ETLBatch{Logger: batchLog, Headers: headersRef, HeaderMap: headerMap, Logs: logs, HeadersWithLog: headersWithLog}
		}
	}
}

// L1 ETL that persists blocks & logs on L1

type L1ETL struct {
	ETL

	db *database.DB
}

func NewL1ETL(log log.Logger, db *database.DB, client node.EthClient) (*L1ETL, error) {
	log = log.New("etl", "l1")

	// work only with devnet deployments for now
	l1Contracts := []common.Address{}
	config.L1Deployments.ForEach(func(name string, addr common.Address) {
		if strings.HasSuffix(name, "Proxy") {
			log.Info("configured contract", "name", name, "addr", addr)
			l1Contracts = append(l1Contracts, addr)
		}
	})

	var fromHeader *types.Header
	latestHeader, err := db.Blocks.LatestL1BlockHeader()
	if err != nil {
		return nil, err
	}
	if latestHeader != nil {
		log.Info("detected last indexed block", "number", latestHeader.Number.Int, "hash", latestHeader.Hash)
		fromHeader = latestHeader.RLPHeader.Header()
	} else {
		log.Info("no indexed state, starting from genesis")
	}

	etlBatches := make(chan ETLBatch)
	etl := ETL{
		log:             log,
		headerTraversal: node.NewHeaderTraversal(client, fromHeader),
		ethClient:       client.GethEthClient(),
		contracts:       l1Contracts,
		etlBatches:      etlBatches,
	}

	return &L1ETL{ETL: etl, db: db}, nil
}

func (l1Etl *L1ETL) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- l1Etl.ETL.Start(ctx)
	}()

	for {
		select {
		case err := <-errCh:
			return err

		// Index incoming batches
		case batch := <-l1Etl.etlBatches:
			err := l1Etl.db.Transaction(func(tx *database.DB) error {
				// pull out L1 blocks that have emitted a log ( <= batch.Headers )
				l1BlockHeaders := make([]*database.L1BlockHeader, 0, len(batch.Headers))
				for i := range batch.Headers {
					if _, ok := batch.HeadersWithLog[batch.Headers[i].Hash()]; ok {
						l1BlockHeaders = append(l1BlockHeaders, &database.L1BlockHeader{BlockHeader: database.BlockHeaderFromHeader(&batch.Headers[i])})
					}
				}

				l1ContractEvents := make([]*database.L1ContractEvent, len(batch.Logs))
				for i := range batch.Logs {
					timestamp := batch.HeaderMap[batch.Logs[i].BlockHash].Time
					l1ContractEvents[i] = &database.L1ContractEvent{ContractEvent: database.ContractEventFromLog(&batch.Logs[i], timestamp)}
				}

				// index blocks and logs
				if len(l1BlockHeaders) > 0 {
					if err := tx.Blocks.StoreL1BlockHeaders(l1BlockHeaders); err != nil {
						return err
					}
					if err := tx.ContractEvents.StoreL1ContractEvents(l1ContractEvents); err != nil {
						return err
					}
				}

				// a-ok
				batch.Logger.Info("indexed batch")
				return nil
			})

			// TODO: Retry mechanism for this batch
			if err != nil {
				panic(err)
			}
		}
	}
}

// L2 ETL that persists blocks & logs on L2

type L2ETL struct {
	ETL

	db *database.DB
}

func NewL2ETL(log log.Logger, db *database.DB, client node.EthClient) (*L2ETL, error) {
	log = log.New("etl", "l2")

	// allow predeploys to be overridable
	l2Contracts := []common.Address{}
	for name, addr := range predeploys.Predeploys {
		log.Info("configured contract", "name", name, "addr", addr)
		l2Contracts = append(l2Contracts, *addr)
	}

	var fromHeader *types.Header
	latestHeader, err := db.Blocks.LatestL2BlockHeader()
	if err != nil {
		return nil, err
	}
	if latestHeader != nil {
		log.Info("detected last indexed block", "number", latestHeader.Number.Int, "hash", latestHeader.Hash)
		fromHeader = latestHeader.RLPHeader.Header()
	} else {
		log.Info("no indexed state, starting from genesis")
	}

	etlBatches := make(chan ETLBatch)
	etl := ETL{
		log:             log,
		headerTraversal: node.NewHeaderTraversal(client, fromHeader),
		ethClient:       client.GethEthClient(),
		contracts:       l2Contracts,
		etlBatches:      etlBatches,
	}

	return &L2ETL{ETL: etl, db: db}, nil
}

func (l2Etl *L2ETL) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- l2Etl.ETL.Start(ctx)
	}()

	for {
		select {
		case err := <-errCh:
			return err

		// Index incoming batches
		case batch := <-l2Etl.etlBatches:
			err := l2Etl.db.Transaction(func(tx *database.DB) error {
				l2BlockHeaders := make([]*database.L2BlockHeader, len(batch.Headers))
				for i := range batch.Headers {
					l2BlockHeaders[i] = &database.L2BlockHeader{BlockHeader: database.BlockHeaderFromHeader(&batch.Headers[i])}
				}

				l2ContractEvents := make([]*database.L2ContractEvent, len(batch.Logs))
				for i := range batch.Logs {
					timestamp := batch.HeaderMap[batch.Logs[i].BlockHash].Time
					l2ContractEvents[i] = &database.L2ContractEvent{ContractEvent: database.ContractEventFromLog(&batch.Logs[i], timestamp)}
				}

				// We're indexing every L2 block.
				if err := tx.Blocks.StoreL2BlockHeaders(l2BlockHeaders); err != nil {
					return err
				}
				if len(l2ContractEvents) > 0 {
					if err := tx.ContractEvents.StoreL2ContractEvents(l2ContractEvents); err != nil {
						return err
					}
				}

				// a-ok
				batch.Logger.Info("indexed batch")
				return nil
			})

			// TODO: Retry mechanism for this batch
			if err != nil {
				panic(err)
			}
		}
	}
}

// Built on top of the L1/L2 ETL that indexes checkpointed
// outputs from L2 on L1. This provides a foundation for
// processors that need to ensure checkpointed state on L1.
type OptimismProcessor struct {
	log log.Logger

	l1Etl *ETL
	l2Etl *ETL
}

// TODO: Checkpoint OutputProposals and StateBatchAppended.

type BridgeProcessor struct {
	log log.Logger
	db  *database.DB

	// reorgs?
	// On each loop we check that the recorded hash exists? If not,
	// we assume a reorg and start from the latest known headers

	paused    bool
	LastEpoch *database.Epoch
}

func NewBridgeProcessor(log log.Logger, db *database.DB) (*BridgeProcessor, error) {
	log = log.New("processor", "bridge")

	// TODO: Detect where we should be starting from

	return &BridgeProcessor{log: log, db: db, paused: false, LastEpoch: nil}, nil
}

func (bridge *BridgeProcessor) Start(ctx context.Context) error {
	done := ctx.Done()
	pollTicker := time.NewTicker(defaultLoopInterval)
	defer pollTicker.Stop()

	// In order to ensure all seen bridge finalization events correspond with seen
	// bridge initiated events, we establish a shared marker between L1 and L2 when
	// processing events.
	//
	// As L1 and L2 blocks are indexed, the highest indexed L2 block starting a new
	// sequencing epoch and corresponding L1 origin that has also been indexed
	// serves as this shared marker.

	// TODOs:
	// 	  1. Fix Logging. Should be clear if we're looking at L1 or L2 side of things

	bridge.log.Info("starting bridge processor...")
	for {
		select {
		case <-done:
			bridge.log.Info("stopping bridge processor")
			return nil

		case <-pollTicker.C:
			latestEpoch, err := bridge.db.Blocks.LatestEpoch()
			if err != nil {
				return err
			}
			if latestEpoch == nil {
				bridge.log.Warn("no epochs indexed...")
				continue
			}

			if bridge.LastEpoch != nil && latestEpoch.L1BlockHeader.Hash == bridge.LastEpoch.L1BlockHeader.Hash {
				// Marked as a warning since the bridge should always be processing at least 1 new epoch
				bridge.log.Warn("no new epochs", "latest_epoch", bridge.LastEpoch.L1BlockHeader.Number.Int)
				continue
			}

			fromL1Height, fromL2Height := big.NewInt(0), big.NewInt(0)
			if bridge.LastEpoch != nil {
				fromL1Height = new(big.Int).Add(bridge.LastEpoch.L1BlockHeader.Number.Int, big.NewInt(1))
				fromL2Height = new(big.Int).Add(bridge.LastEpoch.L2BlockHeader.Number.Int, big.NewInt(1))
			}

			toL1Height, toL2Height := latestEpoch.L1BlockHeader.Number.Int, latestEpoch.L2BlockHeader.Number.Int
			batchLog := bridge.log.New("epoch_start", fromL1Height, "epoch_end", toL1Height)
			batchLog.Info("scanning bridge events")
			err = bridge.db.Transaction(func(tx *database.DB) error {
				// First, find all possible initiated bridge events
				if err := bridge.indexInitiatedL1BridgeEvents(tx, fromL1Height, toL1Height); err != nil {
					return err
				}
				if err := bridge.indexInitiatedL2BridgeEvents(tx, fromL2Height, toL2Height); err != nil {
					return err
				}

				// Now that all initiated events have been indexed, it is ensured that all
				// finalization events must be able to find their counterpart.
				if err := bridge.indexFinalizedL1BridgeEvents(tx, fromL1Height, toL1Height); err != nil {
					return err
				}
				if err := bridge.indexFinalizedL2BridgeEvents(tx, fromL1Height, toL1Height); err != nil {
					return err
				}

				// a-ok
				return nil
			})

			if err != nil {
				// todo: retry stuff
				panic(err)
			} else {
				batchLog.Info("done scanning bridge events", "latest_l1_block_number", toL1Height, "latest_l2_block_number", toL2Height)
			}

			bridge.LastEpoch = latestEpoch
		}
	}
}

func (bridge *BridgeProcessor) indexInitiatedL1BridgeEvents(tx *database.DB, fromHeight, toHeight *big.Int) error {
	type LogKey struct {
		BlockHash common.Hash
		LogIndex  uint64
	}

	// (1) OptimismPortal deposits
	optimismPortalTxDeposits, err := OptimismPortalTransactionDepositEvents(config.L1Deployments.OptimismPortalProxy, tx, fromHeight, toHeight)
	if err != nil {
		return err
	}
	ethDeposits := []*database.L1BridgeDeposit{}
	transactionDeposits := make([]*database.L1TransactionDeposit, len(optimismPortalTxDeposits))
	depositTransactions := make(map[LogKey]*types.DepositTx, len(optimismPortalTxDeposits))
	for i := range optimismPortalTxDeposits {
		depositTx := optimismPortalTxDeposits[i]
		depositTransactions[LogKey{depositTx.Event.BlockHash, depositTx.Event.LogIndex}] = depositTx.DepositTx
		transactionDeposits[i] = &database.L1TransactionDeposit{
			SourceHash:           depositTx.DepositTx.SourceHash,
			L2TransactionHash:    types.NewTx(depositTx.DepositTx).Hash(),
			InitiatedL1EventGUID: depositTx.Event.GUID,
			Version:              database.U256{Int: depositTx.Version},
			OpaqueData:           depositTx.OpaqueData,
			GasLimit:             database.U256{Int: new(big.Int).SetUint64(depositTx.DepositTx.Gas)},
			Tx: database.Transaction{
				FromAddress: depositTx.From,
				ToAddress:   depositTx.To,
				Amount:      database.U256{Int: depositTx.DepositTx.Value},
				Data:        depositTx.DepositTx.Data,
				Timestamp:   depositTx.Event.Timestamp,
			},
		}

		// catch ETH transfers to the portal contract.
		if len(depositTx.DepositTx.Data) == 0 && depositTx.DepositTx.Value.BitLen() > 0 {
			ethDeposits = append(ethDeposits, &database.L1BridgeDeposit{
				TransactionSourceHash: depositTx.DepositTx.SourceHash,
				BridgeTransfer:        database.BridgeTransfer{Tx: transactionDeposits[i].Tx, TokenPair: database.ETHTokenPair},
			})
		}
	}

	if len(transactionDeposits) > 0 {
		bridge.log.Info("detected transaction deposits", "size", len(transactionDeposits))
		if err := tx.BridgeTransactions.StoreL1TransactionDeposits(transactionDeposits); err != nil {
			return err
		}
		if len(ethDeposits) > 0 {
			bridge.log.Info("detected portal ETH transfers", "size", len(ethDeposits))
			if err := tx.BridgeTransfers.StoreL1BridgeDeposits(ethDeposits); err != nil {
				return err
			}
		}
	}

	// (2) L1CrossDomainMessenger SentMessages
	crossDomainSentMessages, err := CrossDomainMessengerSentMessageEvents(config.L1Deployments.L1CrossDomainMessengerProxy, "l1", tx, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainSentMessages) > len(transactionDeposits) {
		return fmt.Errorf("missing transaction deposit for each cross-domain message. deposits: %d, messages: %d", len(transactionDeposits), len(crossDomainSentMessages))
	}
	sentMessages := make(map[LogKey]*CrossDomainMessengerSentMessageEvent, len(crossDomainSentMessages))
	l1BridgeMessages := make([]*database.L1BridgeMessage, len(crossDomainSentMessages))
	for i := range crossDomainSentMessages {
		sentMessage := crossDomainSentMessages[i]
		sentMessages[LogKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex}] = &sentMessage

		// extract the deposit hash from the previous TransactionDepositedEvent
		depositTx, ok := depositTransactions[LogKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex - 1}]
		if !ok {
			return fmt.Errorf("missing expected preceding TransactionDeposit for SentMessage. tx_hash = %s", sentMessage.Event.TransactionHash)
		}
		l1BridgeMessages[i] = &database.L1BridgeMessage{
			TransactionSourceHash: depositTx.SourceHash,
			BridgeMessage: database.BridgeMessage{
				Nonce:                database.U256{Int: sentMessage.MessageNonce},
				MessageHash:          sentMessage.MessageHash,
				SentMessageEventGUID: sentMessage.Event.GUID,
				GasLimit:             database.U256{Int: sentMessage.GasLimit},
				Tx: database.Transaction{
					FromAddress: sentMessage.Sender,
					ToAddress:   sentMessage.Target,
					Amount:      database.U256{Int: sentMessage.Value},
					Data:        sentMessage.Message,
					Timestamp:   sentMessage.Event.Timestamp,
				},
			},
		}
	}

	if len(l1BridgeMessages) > 0 {
		bridge.log.Info("detected L1CrossDomainMessenger messages", "size", len(l1BridgeMessages))
		if err := tx.BridgeMessages.StoreL1BridgeMessages(l1BridgeMessages); err != nil {
			return err
		}
	}

	// (3) L1StandardBridge BridgeInitiated
	initiatedBridges, err := StandardBridgeInitiatedEvents(config.L1Deployments.L1StandardBridgeProxy, "l1", tx, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(initiatedBridges) > len(crossDomainSentMessages) {
		return fmt.Errorf("missing cross-domain message for each initiated bridge event. messages: %d, bridges: %d", len(crossDomainSentMessages), len(initiatedBridges))
	}
	l1BridgeDeposits := make([]*database.L1BridgeDeposit, len(initiatedBridges))
	for i := range initiatedBridges {
		initiatedBridge := initiatedBridges[i]

		// extract the cross domain message hash & deposit source hash from the following events
		depositTx, ok := depositTransactions[LogKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex + 1}]
		if !ok {
			return fmt.Errorf("missing expected following TransactionDeposit for BridgeInitiated. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		}
		sentMessage, ok := sentMessages[LogKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex + 2}]
		if !ok {
			return fmt.Errorf("missing expected following SentMessage for BridgeInitiated. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		}

		l1BridgeDeposits[i] = &database.L1BridgeDeposit{
			TransactionSourceHash: depositTx.SourceHash,
			BridgeTransfer: database.BridgeTransfer{
				CrossDomainMessageHash: &sentMessage.MessageHash,
				TokenPair:              database.TokenPair{L1TokenAddress: initiatedBridge.LocalToken, L2TokenAddress: initiatedBridge.RemoteToken},
				Tx: database.Transaction{
					FromAddress: initiatedBridge.From,
					ToAddress:   initiatedBridge.To,
					Amount:      database.U256{Int: initiatedBridge.Amount},
					Data:        initiatedBridge.ExtraData,
					Timestamp:   initiatedBridge.Event.Timestamp,
				},
			},
		}
	}

	if len(l1BridgeDeposits) > 0 {
		bridge.log.Info("detected L1StandardBridge deposits", "size", len(l1BridgeDeposits))
		if err := tx.BridgeTransfers.StoreL1BridgeDeposits(l1BridgeDeposits); err != nil {
			return err
		}
	}

	// a-ok
	return nil
}

func (bridge *BridgeProcessor) indexInitiatedL2BridgeEvents(tx *database.DB, fromHeight, toHeight *big.Int) error {
	type LogKey struct {
		BlockHash common.Hash
		LogIndex  uint64
	}

	// (1) L2ToL1MessagePasser withdrawals
	l2ToL1MPMessagesPassed, err := L2ToL1MessagePasserMessagePassedEvents(predeploys.L2ToL1MessagePasserAddr, tx, fromHeight, toHeight)
	if err != nil {
		return err
	}

	ethWithdrawals := []*database.L2BridgeWithdrawal{}
	messagesPassed := make(map[LogKey]*bindings.L2ToL1MessagePasserMessagePassed, len(l2ToL1MPMessagesPassed))
	transactionWithdrawals := make([]*database.L2TransactionWithdrawal, len(l2ToL1MPMessagesPassed))
	for i := range l2ToL1MPMessagesPassed {
		messagePassed := l2ToL1MPMessagesPassed[i]
		messagesPassed[LogKey{messagePassed.Event.BlockHash, messagePassed.Event.LogIndex}] = messagePassed.L2ToL1MessagePasserMessagePassed
		transactionWithdrawals[i] = &database.L2TransactionWithdrawal{
			WithdrawalHash:       messagePassed.WithdrawalHash,
			InitiatedL2EventGUID: messagePassed.Event.GUID,
			Nonce:                database.U256{Int: messagePassed.Nonce},
			GasLimit:             database.U256{Int: messagePassed.GasLimit},
			Tx: database.Transaction{
				FromAddress: messagePassed.Sender,
				ToAddress:   messagePassed.Target,
				Amount:      database.U256{Int: messagePassed.Value},
				Data:        messagePassed.Data,
				Timestamp:   messagePassed.Event.Timestamp,
			},
		}

		if len(messagePassed.Data) == 0 && messagePassed.Value.BitLen() > 0 {
			ethWithdrawals = append(ethWithdrawals, &database.L2BridgeWithdrawal{
				TransactionWithdrawalHash: messagePassed.WithdrawalHash,
				BridgeTransfer:            database.BridgeTransfer{Tx: transactionWithdrawals[i].Tx, TokenPair: database.ETHTokenPair},
			})
		}
	}

	if len(messagesPassed) > 0 {
		bridge.log.Info("detected transaction withdrawals", "size", len(transactionWithdrawals))
		if err := tx.BridgeTransactions.StoreL2TransactionWithdrawals(transactionWithdrawals); err != nil {
			return err
		}
		if len(ethWithdrawals) > 0 {
			bridge.log.Info("detected L2ToL1MessagePasser ETH transfers", "size", len(ethWithdrawals))
			if err := tx.BridgeTransfers.StoreL2BridgeWithdrawals(ethWithdrawals); err != nil {
				return err
			}
		}
	}

	// (2) L2CrosssDomainMessenger sentMessages
	crossDomainSentMessages, err := CrossDomainMessengerSentMessageEvents(predeploys.L2CrossDomainMessengerAddr, "l2", tx, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(crossDomainSentMessages) > len(messagesPassed) {
		return fmt.Errorf("missing L2ToL1MP withdrawal for each cross-domain message. withdrawals: %d, messages: %d", len(messagesPassed), len(crossDomainSentMessages))
	}
	sentMessages := make(map[LogKey]*CrossDomainMessengerSentMessageEvent, len(crossDomainSentMessages))
	l2BridgeMessages := make([]*database.L2BridgeMessage, len(crossDomainSentMessages))
	for i := range crossDomainSentMessages {
		sentMessage := crossDomainSentMessages[i]
		sentMessages[LogKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex}] = &sentMessage

		// extract the withdrawal hash from the previous MessagePassed event
		messagePassed, ok := messagesPassed[LogKey{sentMessage.Event.BlockHash, sentMessage.Event.LogIndex - 1}]
		if !ok {
			return fmt.Errorf("missing expected preceding MessagePassedEvent for SentMessage. tx_hash = %s", sentMessage.Event.TransactionHash)
		}
		l2BridgeMessages[i] = &database.L2BridgeMessage{
			TransactionWithdrawalHash: messagePassed.WithdrawalHash,
			BridgeMessage: database.BridgeMessage{
				Nonce:                database.U256{Int: sentMessage.MessageNonce},
				MessageHash:          sentMessage.MessageHash,
				SentMessageEventGUID: sentMessage.Event.GUID,
				GasLimit:             database.U256{Int: sentMessage.GasLimit},
				Tx: database.Transaction{
					FromAddress: sentMessage.Sender,
					ToAddress:   sentMessage.Target,
					Amount:      database.U256{Int: sentMessage.Value},
					Data:        sentMessage.Message,
					Timestamp:   sentMessage.Event.Timestamp,
				},
			},
		}
	}

	if len(l2BridgeMessages) > 0 {
		bridge.log.Info("detected L2CrossDomainMessenger messages", "size", len(l2BridgeMessages))
		if err := tx.BridgeMessages.StoreL2BridgeMessages(l2BridgeMessages); err != nil {
			return err
		}
	}

	// (3) L2StandardBridge bridgeInitiated
	initiatedBridges, err := StandardBridgeInitiatedEvents(predeploys.L2StandardBridgeAddr, "l2", tx, fromHeight, toHeight)
	if err != nil {
		return err
	}
	if len(initiatedBridges) > len(crossDomainSentMessages) {
		return fmt.Errorf("missing cross-domain message for each initiated bridge event. messages: %d, bridges: %d", len(crossDomainSentMessages), len(initiatedBridges))
	}
	l2BridgeWithdrawals := make([]*database.L2BridgeWithdrawal, len(initiatedBridges))
	for i := range initiatedBridges {
		initiatedBridge := initiatedBridges[i]

		// extract the cross domain message hash & deposit source hash from the following events
		messagePassed, ok := messagesPassed[LogKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex + 1}]
		if !ok {
			return fmt.Errorf("missing expected following MessagePassed for BridgeInitiated. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		}
		sentMessage, ok := sentMessages[LogKey{initiatedBridge.Event.BlockHash, initiatedBridge.Event.LogIndex + 2}]
		if !ok {
			return fmt.Errorf("missing expected following SentMessage for BridgeInitiated. tx_hash = %s", initiatedBridge.Event.TransactionHash)
		}

		l2BridgeWithdrawals[i] = &database.L2BridgeWithdrawal{
			TransactionWithdrawalHash: messagePassed.WithdrawalHash,
			BridgeTransfer: database.BridgeTransfer{
				CrossDomainMessageHash: &sentMessage.MessageHash,
				TokenPair:              database.TokenPair{L2TokenAddress: initiatedBridge.LocalToken, L1TokenAddress: initiatedBridge.RemoteToken},
				Tx: database.Transaction{
					FromAddress: initiatedBridge.From,
					ToAddress:   initiatedBridge.To,
					Amount:      database.U256{Int: initiatedBridge.Amount},
					Data:        initiatedBridge.ExtraData,
					Timestamp:   initiatedBridge.Event.Timestamp,
				},
			},
		}
	}

	if len(l2BridgeWithdrawals) > 0 {
		bridge.log.Info("detected L2StandardBridge withdrawals", "size", len(l2BridgeWithdrawals))
		if err := tx.BridgeTransfers.StoreL2BridgeWithdrawals(l2BridgeWithdrawals); err != nil {
			return err
		}
	}

	return nil
}

func (bridge *BridgeProcessor) indexFinalizedL1BridgeEvents(tx *database.DB, fromHeight, toHeight *big.Int) error {
	// (1) OptimismPortal proven withdrawals
	provenWithdrawals, err := OptimismPortalWithdrawalProvenEvents(config.L1Deployments.OptimismPortalProxy, tx, fromHeight, toHeight)
	if err != nil {
		return err
	}
	for i := range provenWithdrawals {
		proven := provenWithdrawals[i]
		withdrawal, err := tx.BridgeTransactions.L2TransactionWithdrawal(proven.WithdrawalHash)
		if err != nil {
			return err
		} else if withdrawal == nil {
			bridge.log.Crit("missing indexed withdrawal on prove event!", "withdrawl_hash", proven.WithdrawalHash, "tx_hash", proven.Event.TransactionHash)
			return errors.New("missing indexed withdrawal")
		}

		if err := tx.BridgeTransactions.MarkL2TransactionWithdrawalProvenEvent(proven.WithdrawalHash, provenWithdrawals[i].Event.GUID); err != nil {
			return err
		}
	}

	if len(provenWithdrawals) > 0 {
		bridge.log.Info("proven transaction withdrawals", "size", len(provenWithdrawals))
	}

	// (2) OptimismPortal finalized withdrawals
	finalizedWithdrawals, err := OptimismPortalWithdrawalFinalizedEvents(config.L1Deployments.OptimismPortalProxy, tx, fromHeight, toHeight)
	if err != nil {
		return err
	}
	for i := range finalizedWithdrawals {
		finalized := finalizedWithdrawals[i]
		withdrawal, err := tx.BridgeTransactions.L2TransactionWithdrawal(finalized.WithdrawalHash)
		if err != nil {
			return err
		} else if withdrawal == nil {
			bridge.log.Crit("missing indexed withdrawal on finalization event!", "withdrawl_hash", finalized.WithdrawalHash, "tx_hash", finalized.Event.TransactionHash)
			return errors.New("missing indexed withdrawal")
		}

		if err = tx.BridgeTransactions.MarkL2TransactionWithdrawalFinalizedEvent(finalized.WithdrawalHash, finalized.Event.GUID, finalized.Success); err != nil {
			return err
		}
	}

	if len(finalizedWithdrawals) > 0 {
		bridge.log.Info("finalized transaction withdrawals", "size", len(finalizedWithdrawals))
	}

	// (3) L1CrossDomainMessenger relayedMessage
	crossDomainRelayedMessages, err := CrossDomainMessengerRelayedMessageEvents(config.L1Deployments.L1CrossDomainMessengerProxy, "l1", tx, fromHeight, toHeight)
	if err != nil {
		return err
	}
	for i := range crossDomainRelayedMessages {
		relayed := crossDomainRelayedMessages[i]
		message, err := tx.BridgeMessages.L2BridgeMessage(relayed.MsgHash)
		if err != nil {
			return err
		} else if message == nil {
			bridge.log.Crit("missing indexed L2CrossDomainMessenger message", "message_hash", relayed.MsgHash, "tx_hash", relayed.Event.TransactionHash)
			return fmt.Errorf("missing indexed L2CrossDomainMessager message")
		}
		if err := tx.BridgeMessages.MarkRelayedL2BridgeMessage(relayed.MsgHash, relayed.Event.GUID); err != nil {
			return err
		}
	}

	if len(crossDomainRelayedMessages) > 0 {
		bridge.log.Info("relayed L2CrossDomainMessenger messages", "size", len(crossDomainRelayedMessages))
	}

	// (4) L1StandardBridge bridgeFinalized
	return nil
}

func (bridge *BridgeProcessor) indexFinalizedL2BridgeEvents(tx *database.DB, fromHeight, toHeight *big.Int) error {
	// (1) L2CrosssDomainMessenger relayedMessage
	crossDomainRelayedMessages, err := CrossDomainMessengerRelayedMessageEvents(predeploys.L2CrossDomainMessengerAddr, "l2", tx, fromHeight, toHeight)
	if err != nil {
		return err
	}
	for i := range crossDomainRelayedMessages {
		relayed := crossDomainRelayedMessages[i]
		message, err := tx.BridgeMessages.L1BridgeMessage(relayed.MsgHash)
		if err != nil {
			return err
		} else if message == nil {
			bridge.log.Crit("missing indexed L1CrossDomainMessenger message", "message_hash", relayed.MsgHash, "tx_hash", relayed.Event.TransactionHash)
			return fmt.Errorf("missing indexed L1CrossDomainMessager message")
		}
		if err := tx.BridgeMessages.MarkRelayedL1BridgeMessage(relayed.MsgHash, relayed.Event.GUID); err != nil {
			return err
		}
	}

	if len(crossDomainRelayedMessages) > 0 {
		bridge.log.Info("relayed L2CrossDomainMessenger messages", "size", len(crossDomainRelayedMessages))
	}

	// (2) L2StandardBridge bridgeFinalized
	return nil
}
