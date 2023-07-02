package processor

import (
	"context"
	"errors"
	"reflect"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum"
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

	latestHeader, err := db.Blocks.FinalizedL1BlockHeader()
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
		// we shouldn't start from genesis with l1. Need a "genesis" height to be defined here
		l1ProcessLog.Info("no indexed state, starting from genesis")
		fromL1Header = nil
	}

	l1Processor := &L1Processor{
		processor: processor{
			fetcher:    node.NewFetcher(ethClient, fromL1Header),
			db:         db,
			processFn:  l1ProcessFn(l1ProcessLog, ethClient, l1Contracts),
			processLog: l1ProcessLog,
		},
	}

	return l1Processor, nil
}

func l1ProcessFn(processLog log.Logger, ethClient node.EthClient, l1Contracts L1Contracts) func(db *database.DB, headers []*types.Header) error {
	rawEthClient := ethclient.NewClient(ethClient.RawRpcClient())

	contractAddrs := l1Contracts.toSlice()
	processLog.Info("processor configured with contracts", "contracts", l1Contracts)

	return func(db *database.DB, headers []*types.Header) error {
		numHeaders := len(headers)
		l1HeaderMap := make(map[common.Hash]*types.Header)
		for _, header := range headers {
			l1HeaderMap[header.Hash()] = header
		}

		/** Watch for Contract Events **/

		logFilter := ethereum.FilterQuery{FromBlock: headers[0].Number, ToBlock: headers[numHeaders-1].Number, Addresses: contractAddrs}
		logs, err := rawEthClient.FilterLogs(context.Background(), logFilter)
		if err != nil {
			return err
		}

		numLogs := len(logs)
		l1ContractEvents := make([]*database.L1ContractEvent, numLogs)
		l1HeadersOfInterest := make(map[common.Hash]bool)
		for i, log := range logs {
			header, ok := l1HeaderMap[log.BlockHash]
			if !ok {
				processLog.Crit("contract event found with associated header not in the batch", "header", log.BlockHash, "log_index", log.Index)
				return errors.New("parsed log with a block hash not in this batch")
			}

			l1HeadersOfInterest[log.BlockHash] = true
			l1ContractEvents[i] = &database.L1ContractEvent{
				ContractEvent: database.ContractEvent{
					GUID:            uuid.New(),
					BlockHash:       log.BlockHash,
					TransactionHash: log.TxHash,
					EventSignature:  log.Topics[0],
					LogIndex:        uint64(log.Index),
					Timestamp:       header.Time,
				},
			}
		}

		/** Index L1 Blocks that have an optimism event **/

		// we iterate on the original array to maintain ordering. probably can find a more efficient
		// way to iterate over the `l1HeadersOfInterest` map while maintaining ordering
		indexedL1Header := []*database.L1BlockHeader{}
		for _, header := range headers {
			blockHash := header.Hash()
			_, hasLogs := l1HeadersOfInterest[blockHash]
			if !hasLogs {
				continue
			}

			indexedL1Header = append(indexedL1Header, &database.L1BlockHeader{
				BlockHeader: database.BlockHeader{
					Hash:       blockHash,
					ParentHash: header.ParentHash,
					Number:     database.U256{Int: header.Number},
					Timestamp:  header.Time,
				},
			})
		}

		/** Update Database **/

		numIndexedL1Headers := len(indexedL1Header)
		if numIndexedL1Headers > 0 {
			processLog.Info("saved l1 blocks of interest within batch", "num", numIndexedL1Headers, "batchSize", numHeaders)
			err = db.Blocks.StoreL1BlockHeaders(indexedL1Header)
			if err != nil {
				return err
			}

			// Since the headers to index are derived from the existence of logs, we know in this branch `numLogs > 0`
			processLog.Info("saving contract logs", "size", numLogs)
			err = db.ContractEvents.StoreL1ContractEvents(l1ContractEvents)
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
