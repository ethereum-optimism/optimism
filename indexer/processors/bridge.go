package processors

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/bigint"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/etl"
	"github.com/ethereum-optimism/optimism/indexer/processors/bridge"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type BridgeProcessor struct {
	log         log.Logger
	db          *database.DB
	l1Etl       *etl.L1ETL
	chainConfig config.ChainConfig

	LatestL1Header *types.Header
	LatestL2Header *types.Header
}

func NewBridgeProcessor(log log.Logger, db *database.DB, l1Etl *etl.L1ETL, chainConfig config.ChainConfig) (*BridgeProcessor, error) {
	log = log.New("processor", "bridge")

	latestL1Header, err := db.BridgeTransactions.L1LatestBlockHeader()
	if err != nil {
		return nil, err
	}
	latestL2Header, err := db.BridgeTransactions.L2LatestBlockHeader()
	if err != nil {
		return nil, err
	}

	var l1Header, l2Header *types.Header
	if latestL1Header == nil && latestL2Header == nil {
		log.Info("no indexed state, starting from rollup genesis")
	} else {
		l1Height, l2Height := bigint.Zero, bigint.Zero
		if latestL1Header != nil {
			l1Height = latestL1Header.Number
			l1Header = latestL1Header.RLPHeader.Header()
		}
		if latestL2Header != nil {
			l2Height = latestL2Header.Number
			l2Header = latestL2Header.RLPHeader.Header()
		}
		log.Info("detected latest indexed bridge state", "l1_block_number", l1Height, "l2_block_number", l2Height)
	}

	return &BridgeProcessor{log, db, l1Etl, chainConfig, l1Header, l2Header}, nil
}

func (b *BridgeProcessor) Start(ctx context.Context) error {
	done := ctx.Done()

	// In order to ensure all seen bridge finalization events correspond with seen
	// bridge initiated events, we establish a shared marker between L1 and L2 when
	// processing events.
	//
	// As L1 and L2 blocks are indexed, the highest indexed L2 block starting a new
	// sequencing epoch and corresponding L1 origin that has also been indexed
	// serves as this shared marker.

	l1EtlUpdates := b.l1Etl.Notify()
	startup := make(chan interface{}, 1)
	startup <- nil

	b.log.Info("starting bridge processor...")
	for {
		select {
		case <-done:
			b.log.Info("stopping bridge processor")
			return nil

		// Fire off independently on startup to check for any
		// new data or if we've indexed new L1 data.
		case <-startup:
		case <-l1EtlUpdates:
		}

		latestEpoch, err := b.db.Blocks.LatestEpoch()
		if err != nil {
			return err
		} else if latestEpoch == nil {
			if b.LatestL1Header != nil || b.LatestL2Header != nil {
				// Once we have some indexed state `latestEpoch` can never return nil
				b.log.Error("bridge events indexed, but no indexed epoch returned", "latest_bridge_l1_block_number", b.LatestL1Header.Number)
				return errors.New("bridge events indexed, but no indexed epoch returned")
			}

			b.log.Warn("no indexed epochs available. waiting...")
			continue
		}

		// Integrity Checks

		if b.LatestL1Header != nil && latestEpoch.L1BlockHeader.Hash == b.LatestL1Header.Hash() {
			b.log.Warn("all available epochs indexed", "latest_bridge_l1_block_number", b.LatestL1Header.Number)
			continue
		}
		if b.LatestL1Header != nil && latestEpoch.L1BlockHeader.Number.Cmp(b.LatestL1Header.Number) <= 0 {
			b.log.Error("decreasing l1 block height observed", "latest_bridge_l1_block_number", b.LatestL1Header.Number, "latest_epoch_number", latestEpoch.L1BlockHeader.Number)
			return errors.New("decreasing l1 block heght observed")
		}
		if b.LatestL2Header != nil && latestEpoch.L2BlockHeader.Number.Cmp(b.LatestL2Header.Number) <= 0 {
			b.log.Error("decreasing l2 block height observed", "latest_bridge_l2_block_number", b.LatestL2Header.Number, "latest_epoch_number", latestEpoch.L2BlockHeader.Number)
			return errors.New("decreasing l2 block heght observed")
		}

		// Process Bridge Events

		toL1Height, toL2Height := latestEpoch.L1BlockHeader.Number, latestEpoch.L2BlockHeader.Number
		fromL1Height, fromL2Height := big.NewInt(int64(b.chainConfig.L1StartingHeight)), bigint.Zero
		if b.LatestL1Header != nil {
			fromL1Height = new(big.Int).Add(b.LatestL1Header.Number, bigint.One)
		}
		if b.LatestL2Header != nil {
			fromL2Height = new(big.Int).Add(b.LatestL2Header.Number, bigint.One)
		}

		l1BedrockStartingHeight := big.NewInt(int64(b.chainConfig.L1BedrockStartingHeight))
		l2BedrockStartingHeight := big.NewInt(int64(b.chainConfig.L2BedrockStartingHeight))

		batchLog := b.log.New("epoch_start_number", fromL1Height, "epoch_end_number", toL1Height)
		batchLog.Info("unobserved epochs")
		err = b.db.Transaction(func(tx *database.DB) error {
			// In the event where we have a large number of un-observed blocks, group the block range
			// on the order of 10k blocks at a time. If this turns out to be a bottleneck, we can
			// parallelize these operations
			maxBlockRange := uint64(10_000)
			l1BridgeLog := b.log.New("bridge", "l1")
			l2BridgeLog := b.log.New("bridge", "l2")

			// FOR OP-MAINNET, OP-GOERLI ONLY! Specially handle the existence of pre-bedrock blocks
			if l1BedrockStartingHeight.Cmp(fromL1Height) > 0 {
				l1BridgeLog := l1BridgeLog.New("mode", "legacy")
				l2BridgeLog := l2BridgeLog.New("mode", "legacy")

				legacyFromL1Height, legacyToL1Height := fromL1Height, toL1Height
				legacyFromL2Height, legacyToL2Height := fromL2Height, toL2Height
				if l1BedrockStartingHeight.Cmp(toL1Height) <= 0 {
					legacyToL1Height = new(big.Int).Sub(l1BedrockStartingHeight, big.NewInt(1))
					legacyToL2Height = new(big.Int).Sub(l2BedrockStartingHeight, big.NewInt(1))
				}

				// First, find all possible initiated bridge events
				l1BlockGroups := bigint.Grouped(legacyFromL1Height, legacyToL1Height, maxBlockRange)
				l2BlockGroups := bigint.Grouped(legacyFromL2Height, legacyToL2Height, maxBlockRange)
				for _, group := range l1BlockGroups {
					log := l1BridgeLog.New("from_l1_block_number", group.Start, "to_l1_block_number", group.End)
					log.Info("scanning for initiated bridge events")
					if err := bridge.LegacyL1ProcessInitiatedBridgeEvents(log, tx, b.chainConfig.L1Contracts, group.Start, group.End); err != nil {
						return err
					}
				}
				for _, group := range l2BlockGroups {
					log := l2BridgeLog.New("from_l2_block_number", group.Start, "to_l2_block_number", group.End)
					log.Info("scanning for initiated bridge events")
					if err := bridge.LegacyL2ProcessInitiatedBridgeEvents(log, tx, b.chainConfig.L2Contracts, group.Start, group.End); err != nil {
						return err
					}
				}

				// Now that all initiated events have been indexed, it is ensured that all finalization can find their counterpart.
				for _, group := range l1BlockGroups {
					log := l1BridgeLog.New("from_l1_block_number", group.Start, "to_l1_block_number", group.End)
					log.Info("scanning for finalized bridge events")
					if err := bridge.LegacyL1ProcessFinalizedBridgeEvents(log, tx, b.l1Etl.EthClient, b.chainConfig.L1Contracts, group.Start, group.End); err != nil {
						return err
					}
				}
				for _, group := range l2BlockGroups {
					log := l2BridgeLog.New("from_l2_block_number", group.Start, "to_l2_block_number", group.End)
					log.Info("scanning for finalized bridge events")
					if err := bridge.LegacyL2ProcessFinalizedBridgeEvents(log, tx, b.chainConfig.L2Contracts, group.Start, group.End); err != nil {
						return err
					}
				}

				if legacyToL1Height.Cmp(toL1Height) == 0 {
					// a-ok! entire batch was legacy blocks
					return nil
				}

				batchLog.Info("detected switch to bedrock", "l1_bedrock_starting_height", l1BedrockStartingHeight, "l2_bedrock_starting_height", l2BedrockStartingHeight)
				fromL1Height = l1BedrockStartingHeight
				fromL2Height = l2BedrockStartingHeight
			}

			// First, find all possible initiated bridge events
			l1BlockGroups := bigint.Grouped(fromL1Height, toL1Height, maxBlockRange)
			l2BlockGroups := bigint.Grouped(fromL2Height, toL2Height, maxBlockRange)
			for _, group := range l1BlockGroups {
				log := l1BridgeLog.New("from_block_number", group.Start, "to_block_number", group.End)
				log.Info("scanning for initiated bridge events")
				if err := bridge.L1ProcessInitiatedBridgeEvents(log, tx, b.chainConfig.L1Contracts, group.Start, group.End); err != nil {
					return err
				}
			}
			for _, group := range l2BlockGroups {
				log := l2BridgeLog.New("from_block_number", group.Start, "to_block_number", group.End)
				log.Info("scanning for initiated bridge events")
				if err := bridge.L2ProcessInitiatedBridgeEvents(log, tx, b.chainConfig.L2Contracts, group.Start, group.End); err != nil {
					return err
				}
			}

			// Now all finalization events can find their counterpart.
			for _, group := range l1BlockGroups {
				log := l1BridgeLog.New("from_block_number", group.Start, "to_block_number", group.End)
				log.Info("scanning for finalized bridge events")
				if err := bridge.L1ProcessFinalizedBridgeEvents(log, tx, b.chainConfig.L1Contracts, group.Start, group.End); err != nil {
					return err
				}
			}
			for _, group := range l2BlockGroups {
				log := l2BridgeLog.New("from_block_number", group.Start, "to_block_number", group.End)
				log.Info("scanning for finalized bridge events")
				if err := bridge.L2ProcessFinalizedBridgeEvents(log, tx, b.chainConfig.L2Contracts, group.Start, group.End); err != nil {
					return err
				}
			}

			// a-ok
			return nil
		})

		if err != nil {
			// Try again on a subsequent interval
			batchLog.Error("failed to index bridge events", "err", err)
		} else {
			batchLog.Info("indexed bridge events", "latest_l1_block_number", toL1Height, "latest_l2_block_number", toL2Height)
			b.LatestL1Header = latestEpoch.L1BlockHeader.RLPHeader.Header()
			b.LatestL2Header = latestEpoch.L2BlockHeader.RLPHeader.Header()
		}
	}

}
