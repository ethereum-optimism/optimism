package processor

import (
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type L2Processor struct {
	processor
}

func NewL2Processor(ethClient node.EthClient, db *database.DB) (*L2Processor, error) {
	l2ProcessLog := log.New("processor", "l2")
	l2ProcessLog.Info("initializing processor")

	latestHeader, err := db.Blocks.FinalizedL2BlockHeader()
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
			fetcher:    node.NewFetcher(ethClient, fromL2Header),
			db:         db,
			processFn:  l2ProcessFn(ethClient),
			processLog: l2ProcessLog,
		},
	}

	return l2Processor, nil
}

func l2ProcessFn(ethClient node.EthClient) func(db *database.DB, headers []*types.Header) error {
	return func(db *database.DB, headers []*types.Header) error {

		// index all l2 blocks for now
		l2Headers := make([]*database.L2BlockHeader, len(headers))
		for i, header := range headers {
			l2Headers[i] = &database.L2BlockHeader{
				BlockHeader: database.BlockHeader{
					Hash:       header.Hash(),
					ParentHash: header.ParentHash,
					Number:     database.U256{Int: header.Number},
					Timestamp:  header.Time,
				},
			}
		}

		return db.Blocks.StoreL2BlockHeaders(l2Headers)
	}
}
