package processor

import (
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type L1Processor struct {
	processor
}

func NewL1Processor(ethClient node.EthClient, db *database.DB) (*L1Processor, error) {
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
			processFn:  l1ProcessFn(ethClient),
			processLog: l1ProcessLog,
		},
	}

	return l1Processor, nil
}

func l1ProcessFn(ethClient node.EthClient) func(db *database.DB, headers []*types.Header) error {
	return func(db *database.DB, headers []*types.Header) error {

		// index all l2 blocks for now
		l1Headers := make([]*database.L1BlockHeader, len(headers))
		for i, header := range headers {
			l1Headers[i] = &database.L1BlockHeader{
				BlockHeader: database.BlockHeader{
					Hash:       header.Hash(),
					ParentHash: header.ParentHash,
					Number:     database.U256{Int: header.Number},
					Timestamp:  header.Time,
				},
			}
		}

		return db.Blocks.StoreL1BlockHeaders(l1Headers)
	}
}
