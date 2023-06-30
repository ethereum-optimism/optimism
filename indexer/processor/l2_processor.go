package processor

import (
	"context"
	"errors"
	"math/big"
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
			headerTraversal: node.NewBufferedHeaderTraversal(ethClient, fromL2Header),
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
	return func(db *database.DB, headers []*types.Header) (*types.Header, error) {
		numHeaders := len(headers)

		latestOutput, err := db.Blocks.LatestOutputProposed()
		if err != nil {
			return nil, err
		} else if latestOutput == nil {
			processLog.Warn("no checkpointed outputs found. waiting...")
			return nil, errors.New("no checkpointed l2 outputs")
		}

		// check if any of these blocks have been published to L1
		latestOutputHeight := latestOutput.L2BlockNumber.Int
		if headers[0].Number.Cmp(latestOutputHeight) > 0 {
			processLog.Warn("entire batch exceeds the latest output", "latest_output_block_number", latestOutputHeight)
			return nil, errors.New("entire batch exceeds latest output")
		}

		// check if we need to partially process this batch
		if headers[numHeaders-1].Number.Cmp(latestOutputHeight) > 0 {
			processLog.Info("reducing batch", "latest_output_block_number", latestOutputHeight)

			// reduce the batch size
			lastHeaderIndex := new(big.Int).Sub(latestOutputHeight, headers[0].Number).Uint64()

			// update markers (including `lastHeaderIndex`)
			headers = headers[:lastHeaderIndex+1]
			numHeaders = len(headers)
		}

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
			return nil, err
		}

		numLogs := len(logs)
		l2ContractEvents := make([]*database.L2ContractEvent, numLogs)
		for i, log := range logs {
			header, ok := l2HeaderMap[log.BlockHash]
			if !ok {
				processLog.Error("contract event found with associated header not in the batch", "header", header, "log_index", log.Index)
				return nil, errors.New("parsed log with a block hash not in this batch")
			}

			l2ContractEvents[i] = &database.L2ContractEvent{
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

		/** Update Database **/

		processLog.Info("saving l2 blocks", "size", numHeaders)
		err = db.Blocks.StoreL2BlockHeaders(l2Headers)
		if err != nil {
			return nil, err
		}

		if numLogs > 0 {
			processLog.Info("detected contract logs", "size", numLogs)
			err = db.ContractEvents.StoreL2ContractEvents(l2ContractEvents)
			if err != nil {
				return nil, err
			}
		}

		// a-ok!
		return headers[numHeaders-1], nil
	}
}
