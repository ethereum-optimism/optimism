package processor

import (
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

		// L2 checkpoints posted on L1
		outputProposals := []*database.OutputProposal{}
		legacyStateBatches := []*database.LegacyStateBatch{}

		l1HeadersOfInterest := make(map[common.Hash]bool)
		l1ContractEvents := make([]*database.L1ContractEvent, len(logs))

		processedContractEvents := NewProcessedContractEvents()
		for i, log := range logs {
			header, ok := headerMap[log.BlockHash]
			if !ok {
				processLog.Error("contract event found with associated header not in the batch", "header", log.BlockHash, "log_index", log.Index)
				return errors.New("parsed log with a block hash not in this batch")
			}

			contractEvent := processedContractEvents.AddLog(&logs[i], header.Time)
			l1HeadersOfInterest[log.BlockHash] = true
			l1ContractEvents[i] = &database.L1ContractEvent{ContractEvent: *contractEvent}

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

			indexedL1Headers = append(indexedL1Headers, &database.L1BlockHeader{BlockHeader: database.BlockHeaderFromGethHeader(header)})
		}

		/** Update Database **/

		numIndexedL1Headers := len(indexedL1Headers)
		if numIndexedL1Headers > 0 {
			processLog.Info("saving l1 blocks with optimism logs", "size", numIndexedL1Headers, "batch_size", numHeaders)
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

			// forward along contract events to the bridge processor
			err = l1BridgeProcessContractEvents(processLog, db, ethClient, processedContractEvents, l1Contracts)
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

func l1BridgeProcessContractEvents(processLog log.Logger, db *database.DB, ethClient node.EthClient, events *ProcessedContractEvents, l1Contracts L1Contracts) error {
	rawEthClient := ethclient.NewClient(ethClient.RawRpcClient())

	// Process New Deposits
	initiatedDepositEvents, err := StandardBridgeInitiatedEvents(events)
	if err != nil {
		return err
	}

	deposits := make([]*database.Deposit, len(initiatedDepositEvents))
	for i, initiatedBridgeEvent := range initiatedDepositEvents {
		deposits[i] = &database.Deposit{
			GUID:                 uuid.New(),
			InitiatedL1EventGUID: initiatedBridgeEvent.RawEvent.GUID,
			SentMessageNonce:     database.U256{Int: initiatedBridgeEvent.CrossDomainMessengerNonce},
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

	if len(deposits) > 0 {
		processLog.Info("detected L1StandardBridge deposits", "num", len(deposits))
		err := db.Bridge.StoreDeposits(deposits)
		if err != nil {
			return err
		}
	}

	// Prove L2 Withdrawals
	provenWithdrawalEvents, err := OptimismPortalWithdrawalProvenEvents(events)
	if err != nil {
		return err
	}

	// we manually keep track since not every proven withdrawal is a standard bridge withdrawal
	numProvenWithdrawals := 0
	for _, provenWithdrawalEvent := range provenWithdrawalEvents {
		withdrawalHash := provenWithdrawalEvent.WithdrawalHash
		withdrawal, err := db.Bridge.WithdrawalByHash(withdrawalHash)
		if err != nil {
			return err
		}

		// Check if the L2Processor is behind or really has missed an event. We can compare against the
		// OptimismPortal#ProvenWithdrawal on-chain mapping relative to the latest indexed L2 height
		if withdrawal == nil {
			bridgeAddress := l1Contracts.L1StandardBridge
			portalAddress := l1Contracts.OptimismPortal
			if provenWithdrawalEvent.From != bridgeAddress || provenWithdrawalEvent.To != bridgeAddress {
				// non-bridge withdrawal
				continue
			}

			// Query for the the proven withdrawal on-chain
			provenWithdrawal, err := OptimismPortalQueryProvenWithdrawal(rawEthClient, portalAddress, withdrawalHash)
			if err != nil {
				return err
			}

			latestL2Header, err := db.Blocks.LatestL2BlockHeader()
			if err != nil {
				return err
			}

			if latestL2Header == nil || provenWithdrawal.L2OutputIndex.Cmp(latestL2Header.Number.Int) > 0 {
				processLog.Warn("behind on indexed L2 withdrawals")
				return errors.New("waiting for L2Processor to catch up")
			} else {
				processLog.Crit("missing indexed withdrawal for this proven event")
				return errors.New("missing withdrawal message")
			}
		}

		err = db.Bridge.MarkProvenWithdrawalEvent(withdrawal.GUID, provenWithdrawalEvent.RawEvent.GUID)
		if err != nil {
			return err
		}

		numProvenWithdrawals++
	}

	if numProvenWithdrawals > 0 {
		processLog.Info("proven L2StandardBridge withdrawals", "size", numProvenWithdrawals)
	}

	// Finalize Pending Withdrawals
	finalizedWithdrawalEvents, err := StandardBridgeFinalizedEvents(rawEthClient, events)
	if err != nil {
		return err
	}

	for _, finalizedBridgeEvent := range finalizedWithdrawalEvents {
		nonce := finalizedBridgeEvent.CrossDomainMessengerNonce
		withdrawal, err := db.Bridge.WithdrawalByMessageNonce(nonce)
		if err != nil {
			processLog.Error("error querying associated withdrawal messsage using nonce", "cross_domain_messenger_nonce", nonce)
			return err
		}

		// Since we have to prove the event on-chain first, we don't need to check if the processor is
		// behind. we're definitely in an error state if we cannot find the withdrawal when parsing this even
		if withdrawal == nil {
			processLog.Crit("missing indexed withdrawal for this finalization event")
			return errors.New("missing withdrawal message")
		}

		err = db.Bridge.MarkFinalizedWithdrawalEvent(withdrawal.GUID, finalizedBridgeEvent.RawEvent.GUID)
		if err != nil {
			processLog.Error("error finalizing withdrawal", "err", err)
			return err
		}
	}

	if len(finalizedWithdrawalEvents) > 0 {
		processLog.Info("finalized L2StandardBridge withdrawals", "num", len(finalizedWithdrawalEvents))
	}

	// a-ok!
	return nil
}
