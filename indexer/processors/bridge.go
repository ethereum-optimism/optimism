package processors

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/processors/bridge"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type BridgeProcessor struct {
	log         log.Logger
	db          *database.DB
	chainConfig config.ChainConfig

	// NOTE: We'll need this processor to handle for reorgs events.

	LatestL1Header *types.Header
	LatestL2Header *types.Header
}

func NewBridgeProcessor(log log.Logger, db *database.DB, chainConfig config.ChainConfig) (*BridgeProcessor, error) {
	log = log.New("processor", "bridge")

	latestL1Header, err := bridge.L1LatestBridgeEventHeader(db, chainConfig)
	if err != nil {
		return nil, err
	}
	latestL2Header, err := bridge.L2LatestBridgeEventHeader(db)
	if err != nil {
		return nil, err
	}

	// Since the bridge processor indexes events based on epochs, there's
	// no scenario in which we have indexed L2 data with no L1 data.
	//
	// NOTE: Technically there is an exception if our bridging contracts are
	// used to bridges native from L2 and an op-chain happens to launch where
	// only L2 native bridge events have occurred. This is a rare situation now
	// and it's worth the assertion as an integrity check. We can revisit this
	// as more chains launch with primarily L2-native activity.
	if latestL1Header == nil && latestL2Header != nil {
		log.Error("detected indexed L2 bridge activity with no indexed L1 state", "l2_block_number", latestL2Header.Number)
		return nil, errors.New("detected indexed L2 bridge activity with no indexed L1 state")
	}

	if latestL1Header == nil && latestL2Header == nil {
		log.Info("no indexed state, starting from genesis")
	} else {
		log.Info("detected the latest indexed state", "l1_block_number", latestL1Header.Number, "l2_block_number", latestL2Header.Number)
	}

	return &BridgeProcessor{log, db, chainConfig, latestL1Header, latestL2Header}, nil
}

func (b *BridgeProcessor) Start(ctx context.Context) error {
	done := ctx.Done()

	// NOTE: This should run on same iterval as L1 ETL rather than as finding the
	// lasted epoch is constrained to how much L1 data we've indexed.
	pollTicker := time.NewTicker(5 * time.Second)
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

	b.log.Info("starting bridge processor...")
	for {
		select {
		case <-done:
			b.log.Info("stopping bridge processor")
			return nil

		case <-pollTicker.C:
			latestEpoch, err := b.db.Blocks.LatestEpoch()
			if err != nil {
				return err
			}
			if latestEpoch == nil {
				if b.LatestL1Header != nil {
					// Once we have some satte `latestEpoch` should never return nil.
					b.log.Error("started with indexed bridge state, but no blocks epochs returned", "latest_bridge_l1_block_number", b.LatestL1Header.Number)
					return errors.New("started with indexed bridge state, but no blocks epochs returned")
				} else {
					b.log.Warn("no indexed block state. waiting...")
					continue
				}
			}

			if b.LatestL1Header != nil && latestEpoch.L1BlockHeader.Hash == b.LatestL1Header.Hash() {
				// Marked as a warning since the bridge should always be processing at least 1 new epoch
				b.log.Warn("all available epochs indexed by the bridge", "latest_epoch_number", b.LatestL1Header.Number)
				continue
			}

			toL1Height, toL2Height := latestEpoch.L1BlockHeader.Number, latestEpoch.L2BlockHeader.Number
			fromL1Height, fromL2Height := big.NewInt(0), big.NewInt(0)
			if b.LatestL1Header != nil {
				// `NewBridgeProcessor` ensures that LatestL2Header must not be nil if LatestL1Header is set
				fromL1Height = new(big.Int).Add(b.LatestL1Header.Number, big.NewInt(1))
				fromL2Height = new(big.Int).Add(b.LatestL2Header.Number, big.NewInt(1))
			}

			batchLog := b.log.New("epoch_start_number", fromL1Height, "epoch_end_number", toL1Height)
			batchLog.Info("scanning bridge events")
			err = b.db.Transaction(func(tx *database.DB) error {
				l1BridgeLog := b.log.New("from_l1_block_number", fromL1Height, "to_l1_block_number", toL1Height)
				l2BridgeLog := b.log.New("from_l2_block_number", fromL2Height, "to_l2_block_number", toL2Height)

				// First, find all possible initiated bridge events
				if err := bridge.L1ProcessInitiatedBridgeEvents(l1BridgeLog, tx, b.chainConfig, fromL1Height, toL1Height); err != nil {
					return err
				}
				if err := bridge.L2ProcessInitiatedBridgeEvents(l2BridgeLog, tx, fromL2Height, toL2Height); err != nil {
					return err
				}

				// Now that all initiated events have been indexed, it is ensured that all finalization can find their counterpart.
				if err := bridge.L1ProcessFinalizedBridgeEvents(l1BridgeLog, tx, b.chainConfig, fromL1Height, toL1Height); err != nil {
					return err
				}
				if err := bridge.L2ProcessFinalizedBridgeEvents(l2BridgeLog, tx, fromL2Height, toL2Height); err != nil {
					return err
				}

				// a-ok
				return nil
			})

			if err != nil {
				// Try again on a subsequent interval
				batchLog.Error("unable to index new bridge events", "err", err)
			} else {
				batchLog.Info("done indexing new bridge events", "latest_l1_block_number", toL1Height, "latest_l2_block_number", toL2Height)
				b.LatestL1Header = latestEpoch.L1BlockHeader.RLPHeader.Header()
				b.LatestL2Header = latestEpoch.L2BlockHeader.RLPHeader.Header()
			}
		}
	}
}
