package processors

import (
	"context"
	"errors"
	"math/big"

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

	// NOTE: We'll need this processor to handle for reorgs events.

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
		l1Height, l2Height := big.NewInt(0), big.NewInt(0)
		if latestL1Header != nil {
			l1Height = latestL1Header.Number
			l1Header = latestL1Header.RLPHeader.Header()
		}
		if latestL2Header != nil {
			l2Height = latestL2Header.Number
			l2Header = latestL2Header.RLPHeader.Header()
		}
		log.Info("detected latest indexed state", "l1_block_number", l1Height, "l2_block_number", l2Height)
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
	b.log.Info("starting bridge processor...")
	for {
		select {
		case <-done:
			b.log.Info("stopping bridge processor")
			return nil

		case <-l1EtlUpdates:
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
				b.log.Error("non-increasing l1 block height observed", "latest_bridge_l1_block_number", b.LatestL1Header.Number, "latest_epoch_number", latestEpoch.L1BlockHeader.Number)
				return errors.New("non-increasing l1 block heght observed")
			}
			if b.LatestL2Header != nil && latestEpoch.L2BlockHeader.Number.Cmp(b.LatestL2Header.Number) <= 0 {
				b.log.Error("non-increasing l2 block height observed", "latest_bridge_l2_block_number", b.LatestL2Header.Number, "latest_epoch_number", latestEpoch.L2BlockHeader.Number)
				return errors.New("non-increasing l2 block heght observed")
			}

			// Process Bridge Events

			toL1Height, toL2Height := latestEpoch.L1BlockHeader.Number, latestEpoch.L2BlockHeader.Number
			fromL1Height, fromL2Height := big.NewInt(0), big.NewInt(0)
			if b.LatestL1Header != nil {
				fromL1Height = new(big.Int).Add(b.LatestL1Header.Number, big.NewInt(1))
			}
			if b.LatestL2Header != nil {
				fromL2Height = new(big.Int).Add(b.LatestL2Header.Number, big.NewInt(1))
			}

			batchLog := b.log.New("epoch_start_number", fromL1Height, "epoch_end_number", toL1Height)
			batchLog.Info("scanning for new bridge events")
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
				batchLog.Info("done indexing bridge events", "latest_l1_block_number", toL1Height, "latest_l2_block_number", toL2Height)
				b.LatestL1Header = latestEpoch.L1BlockHeader.RLPHeader.Header()
				b.LatestL2Header = latestEpoch.L2BlockHeader.RLPHeader.Header()
			}
		}
	}
}
