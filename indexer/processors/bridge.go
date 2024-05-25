package processors

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"gorm.io/gorm"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/bigint"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/etl"
	"github.com/ethereum-optimism/optimism/indexer/processors/bridge"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
)

var blocksLimit = 10_000

type BridgeProcessor struct {
	log     log.Logger
	db      *database.DB
	metrics bridge.Metricer

	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group

	l1Etl       *etl.L1ETL
	l2Etl       *etl.L2ETL
	chainConfig config.ChainConfig

	LastL1Header *database.L1BlockHeader
	LastL2Header *database.L2BlockHeader

	LastFinalizedL1Header *database.L1BlockHeader
	LastFinalizedL2Header *database.L2BlockHeader
}

func NewBridgeProcessor(log log.Logger, db *database.DB, metrics bridge.Metricer, l1Etl *etl.L1ETL, l2Etl *etl.L2ETL,
	chainConfig config.ChainConfig, shutdown context.CancelCauseFunc) (*BridgeProcessor, error) {
	log = log.New("processor", "bridge")

	latestL1Header, err := db.BridgeTransactions.L1LatestBlockHeader()
	if err != nil {
		return nil, err
	}
	latestL2Header, err := db.BridgeTransactions.L2LatestBlockHeader()
	if err != nil {
		return nil, err
	}

	latestFinalizedL1Header, err := db.BridgeTransactions.L1LatestFinalizedBlockHeader()
	if err != nil {
		return nil, err
	}
	latestFinalizedL2Header, err := db.BridgeTransactions.L2LatestFinalizedBlockHeader()
	if err != nil {
		return nil, err
	}

	log.Info("detected indexed bridge state",
		"l1_block", latestL1Header, "l2_block", latestL2Header,
		"finalized_l1_block", latestFinalizedL1Header, "finalized_l2_block", latestFinalizedL2Header)

	resCtx, resCancel := context.WithCancel(context.Background())
	return &BridgeProcessor{
		log:                   log,
		db:                    db,
		metrics:               metrics,
		l1Etl:                 l1Etl,
		l2Etl:                 l2Etl,
		resourceCtx:           resCtx,
		resourceCancel:        resCancel,
		chainConfig:           chainConfig,
		LastL1Header:          latestL1Header,
		LastL2Header:          latestL2Header,
		LastFinalizedL1Header: latestFinalizedL1Header,
		LastFinalizedL2Header: latestFinalizedL2Header,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in bridge processor: %w", err))
		}},
	}, nil
}

func (b *BridgeProcessor) Start() error {
	b.log.Info("starting bridge processor...")

	// start L1 worker
	b.tasks.Go(func() error {
		l1EtlUpdates := b.l1Etl.Notify()
		for range l1EtlUpdates {
			done := b.metrics.RecordL1Interval()
			done(b.onL1Data())
		}
		b.log.Info("no more l1 etl updates. shutting down l1 task")
		return nil
	})
	// start L2 worker
	b.tasks.Go(func() error {
		l2EtlUpdates := b.l2Etl.Notify()
		for range l2EtlUpdates {
			done := b.metrics.RecordL2Interval()
			done(b.onL2Data())
		}
		b.log.Info("no more l2 etl updates. shutting down l2 task")
		return nil
	})
	return nil
}

func (b *BridgeProcessor) Close() error {
	// signal that we can stop any ongoing work
	b.resourceCancel()
	// await the work to stop
	return b.tasks.Wait()
}

// onL1Data will index new bridge events for the unvisited L1 state. As new L1 bridge events
// are processed, bridge finalization events can be processed on L2 in this same window.
func (b *BridgeProcessor) onL1Data() error {
	latestL1Header := b.l1Etl.LatestHeader
	b.log.Info("notified of new L1 state", "l1_etl_block_number", latestL1Header.Number)

	var errs error
	if err := b.processInitiatedL1Events(); err != nil {
		b.log.Error("failed to process initiated L1 events", "err", err)
		errs = errors.Join(errs, err)
	}

	// `LastFinalizedL2Header` and `LastL1Header` are mutated by the same routine and can
	// safely be read without needing any sync primitives
	if b.LastFinalizedL2Header == nil || b.LastFinalizedL2Header.Timestamp < b.LastL1Header.Timestamp {
		if err := b.processFinalizedL2Events(); err != nil {
			b.log.Error("failed to process finalized L2 events", "err", err)
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

// onL2Data will index new bridge events for the unvisited L2 state. As new L2 bridge events
// are processed, bridge finalization events can be processed on L1 in this same window.
func (b *BridgeProcessor) onL2Data() error {
	if b.l2Etl.LatestHeader.Number.Cmp(bigint.Zero) == 0 {
		return nil // skip genesis
	}
	b.log.Info("notified of new L2 state", "l2_etl_block_number", b.l2Etl.LatestHeader.Number)

	var errs error
	if err := b.processInitiatedL2Events(); err != nil {
		b.log.Error("failed to process initiated L2 events", "err", err)
		errs = errors.Join(errs, err)
	}

	// `LastFinalizedL1Header` and `LastL2Header` are mutated by the same routine and can
	// safely be read without needing any sync primitives
	if b.LastFinalizedL1Header == nil || b.LastFinalizedL1Header.Timestamp < b.LastL2Header.Timestamp {
		if err := b.processFinalizedL1Events(); err != nil {
			b.log.Error("failed to process finalized L1 events", "err", err)
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

// Process Initiated Bridge Events

func (b *BridgeProcessor) processInitiatedL1Events() error {
	l1BridgeLog := b.log.New("bridge", "l1", "kind", "initiated")
	lastL1BlockNumber := big.NewInt(int64(b.chainConfig.L1StartingHeight) - 1)
	if b.LastL1Header != nil {
		lastL1BlockNumber = b.LastL1Header.Number
	}

	// Latest unobserved L1 state bounded by `blockLimits` blocks. Since this process is driven on new L1 data,
	// we always expect this query to return a new result
	latestL1HeaderScope := func(db *gorm.DB) *gorm.DB {
		newQuery := db.Session(&gorm.Session{NewDB: true}) // fresh subquery
		headers := newQuery.Model(database.L1BlockHeader{}).Where("number > ?", lastL1BlockNumber)
		return db.Where("number = (?)", newQuery.Table("(?) as block_numbers", headers.Order("number ASC").Limit(blocksLimit)).Select("MAX(number)"))
	}
	latestL1Header, err := b.db.Blocks.L1BlockHeaderWithScope(latestL1HeaderScope)
	if err != nil {
		return fmt.Errorf("failed to query new L1 state: %w", err)
	} else if latestL1Header == nil {
		return fmt.Errorf("no new L1 state found")
	}

	fromL1Height, toL1Height := new(big.Int).Add(lastL1BlockNumber, bigint.One), latestL1Header.Number
	if err := b.db.Transaction(func(tx *database.DB) error {
		l1BedrockStartingHeight := big.NewInt(int64(b.chainConfig.L1BedrockStartingHeight))
		if l1BedrockStartingHeight.Cmp(fromL1Height) > 0 { // OP Mainnet & OP Goerli Only.
			legacyFromL1Height, legacyToL1Height := fromL1Height, toL1Height
			if l1BedrockStartingHeight.Cmp(toL1Height) <= 0 {
				legacyToL1Height = new(big.Int).Sub(l1BedrockStartingHeight, bigint.One)
			}

			legacyBridgeLog := l1BridgeLog.New("mode", "legacy", "from_block_number", legacyFromL1Height, "to_block_number", legacyToL1Height)
			legacyBridgeLog.Info("scanning for initiated bridge events")
			if err := bridge.LegacyL1ProcessInitiatedBridgeEvents(legacyBridgeLog, tx, b.metrics, b.chainConfig.L1Contracts, legacyFromL1Height, legacyToL1Height); err != nil {
				return err
			} else if legacyToL1Height.Cmp(toL1Height) == 0 {
				return nil // a-ok! Entire range was legacy blocks
			}
			legacyBridgeLog.Info("detected switch to bedrock", "bedrock_block_number", l1BedrockStartingHeight)
			fromL1Height = l1BedrockStartingHeight
		}

		l1BridgeLog = l1BridgeLog.New("from_block_number", fromL1Height, "to_block_number", toL1Height)
		l1BridgeLog.Info("scanning for initiated bridge events")
		return bridge.L1ProcessInitiatedBridgeEvents(l1BridgeLog, tx, b.metrics, b.chainConfig.L1Contracts, fromL1Height, toL1Height)
	}); err != nil {
		return err
	}

	b.LastL1Header = latestL1Header
	b.metrics.RecordL1LatestHeight(latestL1Header.Number)
	return nil
}

func (b *BridgeProcessor) processInitiatedL2Events() error {
	l2BridgeLog := b.log.New("bridge", "l2", "kind", "initiated")
	lastL2BlockNumber := bigint.Zero
	if b.LastL2Header != nil {
		lastL2BlockNumber = b.LastL2Header.Number
	}

	// Latest unobserved L2 state bounded by `blockLimits` blocks. Since this process is driven by new L2 data,
	// we always expect this query to return a new result
	latestL2HeaderScope := func(db *gorm.DB) *gorm.DB {
		newQuery := db.Session(&gorm.Session{NewDB: true}) // fresh subquery
		headers := newQuery.Model(database.L2BlockHeader{}).Where("number > ?", lastL2BlockNumber)
		return db.Where("number = (?)", newQuery.Table("(?) as block_numbers", headers.Order("number ASC").Limit(blocksLimit)).Select("MAX(number)"))
	}
	latestL2Header, err := b.db.Blocks.L2BlockHeaderWithScope(latestL2HeaderScope)
	if err != nil {
		return fmt.Errorf("failed to query new L2 state: %w", err)
	} else if latestL2Header == nil {
		return fmt.Errorf("no new L2 state found")
	}

	fromL2Height, toL2Height := new(big.Int).Add(lastL2BlockNumber, bigint.One), latestL2Header.Number
	if err := b.db.Transaction(func(tx *database.DB) error {
		l2BedrockStartingHeight := big.NewInt(int64(b.chainConfig.L2BedrockStartingHeight))
		if l2BedrockStartingHeight.Cmp(fromL2Height) > 0 { // OP Mainnet & OP Goerli Only
			legacyFromL2Height, legacyToL2Height := fromL2Height, toL2Height
			if l2BedrockStartingHeight.Cmp(toL2Height) <= 0 {
				legacyToL2Height = new(big.Int).Sub(l2BedrockStartingHeight, bigint.One)
			}

			legacyBridgeLog := l2BridgeLog.New("mode", "legacy", "from_block_number", legacyFromL2Height, "to_block_number", legacyToL2Height)
			legacyBridgeLog.Info("scanning for initiated bridge events")
			if err := bridge.LegacyL2ProcessInitiatedBridgeEvents(legacyBridgeLog, tx, b.metrics, b.chainConfig.Preset, b.chainConfig.L2Contracts, legacyFromL2Height, legacyToL2Height); err != nil {
				return err
			} else if legacyToL2Height.Cmp(toL2Height) == 0 {
				return nil // a-ok! Entire range was legacy blocks
			}
			legacyBridgeLog.Info("detected switch to bedrock")
			fromL2Height = l2BedrockStartingHeight
		}

		l2BridgeLog = l2BridgeLog.New("from_block_number", fromL2Height, "to_block_number", toL2Height)
		l2BridgeLog.Info("scanning for initiated bridge events")
		return bridge.L2ProcessInitiatedBridgeEvents(l2BridgeLog, tx, b.metrics, b.chainConfig.L2Contracts, fromL2Height, toL2Height)
	}); err != nil {
		return err
	}

	b.LastL2Header = latestL2Header
	b.metrics.RecordL2LatestHeight(latestL2Header.Number)
	return nil
}

// Process Finalized Bridge Events

func (b *BridgeProcessor) processFinalizedL1Events() error {
	l1BridgeLog := b.log.New("bridge", "l1", "kind", "finalization")
	lastFinalizedL1BlockNumber := big.NewInt(int64(b.chainConfig.L1StartingHeight) - 1)
	if b.LastFinalizedL1Header != nil {
		lastFinalizedL1BlockNumber = b.LastFinalizedL1Header.Number
	}

	// Latest unfinalized L1 state bounded by `blockLimit` blocks that have had L2 bridge events indexed. Since L1 data
	// is indexed independently of L2, there may not be new L1 state to finalized
	latestL1HeaderScope := func(db *gorm.DB) *gorm.DB {
		newQuery := db.Session(&gorm.Session{NewDB: true}) // fresh subquery
		headers := newQuery.Model(database.L1BlockHeader{}).Where("number > ? AND timestamp <= ?", lastFinalizedL1BlockNumber, b.LastL2Header.Timestamp)
		return db.Where("number = (?)", newQuery.Table("(?) as block_numbers", headers.Order("number ASC").Limit(blocksLimit)).Select("MAX(number)"))
	}
	latestL1Header, err := b.db.Blocks.L1BlockHeaderWithScope(latestL1HeaderScope)
	if err != nil {
		return fmt.Errorf("failed to query for latest unfinalized L1 state: %w", err)
	} else if latestL1Header == nil {
		l1BridgeLog.Debug("no new l1 state to finalize", "last_finalized_block_number", lastFinalizedL1BlockNumber)
		return nil
	}

	fromL1Height, toL1Height := new(big.Int).Add(lastFinalizedL1BlockNumber, bigint.One), latestL1Header.Number
	if err := b.db.Transaction(func(tx *database.DB) error {
		l1BedrockStartingHeight := big.NewInt(int64(b.chainConfig.L1BedrockStartingHeight))
		if l1BedrockStartingHeight.Cmp(fromL1Height) > 0 {
			legacyFromL1Height, legacyToL1Height := fromL1Height, toL1Height
			if l1BedrockStartingHeight.Cmp(toL1Height) <= 0 {
				legacyToL1Height = new(big.Int).Sub(l1BedrockStartingHeight, bigint.One)
			}

			legacyBridgeLog := l1BridgeLog.New("mode", "legacy", "from_block_number", legacyFromL1Height, "to_block_number", legacyToL1Height)
			legacyBridgeLog.Info("scanning for finalized bridge events")
			if err := bridge.LegacyL1ProcessFinalizedBridgeEvents(legacyBridgeLog, tx, b.metrics, b.chainConfig.L1Contracts, legacyFromL1Height, legacyToL1Height); err != nil {
				return err
			} else if legacyToL1Height.Cmp(toL1Height) == 0 {
				return nil // a-ok! Entire range was legacy blocks
			}
			legacyBridgeLog.Info("detected switch to bedrock")
			fromL1Height = l1BedrockStartingHeight
		}

		l1BridgeLog = l1BridgeLog.New("from_block_number", fromL1Height, "to_block_number", toL1Height)
		l1BridgeLog.Info("scanning for finalized bridge events")
		return bridge.L1ProcessFinalizedBridgeEvents(l1BridgeLog, tx, b.metrics, b.chainConfig.L1Contracts, fromL1Height, toL1Height)
	}); err != nil {
		return err
	}

	b.LastFinalizedL1Header = latestL1Header
	b.metrics.RecordL1LatestFinalizedHeight(latestL1Header.Number)
	return nil
}

func (b *BridgeProcessor) processFinalizedL2Events() error {
	l2BridgeLog := b.log.New("bridge", "l2", "kind", "finalization")
	lastFinalizedL2BlockNumber := bigint.Zero
	if b.LastFinalizedL2Header != nil {
		lastFinalizedL2BlockNumber = b.LastFinalizedL2Header.Number
	}

	// Latest unfinalized L2 state bounded by `blockLimit` blocks that have had L1 bridge events indexed. Since L2 data
	// is indexed independently of L1, there may not be new L2 state to finalized
	latestL2HeaderScope := func(db *gorm.DB) *gorm.DB {
		newQuery := db.Session(&gorm.Session{NewDB: true}) // fresh subquery
		headers := newQuery.Model(database.L2BlockHeader{}).Where("number > ? AND timestamp <= ?", lastFinalizedL2BlockNumber, b.LastL1Header.Timestamp)
		return db.Where("number = (?)", newQuery.Table("(?) as block_numbers", headers.Order("number ASC").Limit(blocksLimit)).Select("MAX(number)"))
	}
	latestL2Header, err := b.db.Blocks.L2BlockHeaderWithScope(latestL2HeaderScope)
	if err != nil {
		return fmt.Errorf("failed to query for latest unfinalized L2 state: %w", err)
	} else if latestL2Header == nil {
		l2BridgeLog.Debug("no new l2 state to finalize", "last_finalized_block_number", lastFinalizedL2BlockNumber)
		return nil
	}

	fromL2Height, toL2Height := new(big.Int).Add(lastFinalizedL2BlockNumber, bigint.One), latestL2Header.Number
	if err := b.db.Transaction(func(tx *database.DB) error {
		l2BedrockStartingHeight := big.NewInt(int64(b.chainConfig.L2BedrockStartingHeight))
		if l2BedrockStartingHeight.Cmp(fromL2Height) > 0 {
			legacyFromL2Height, legacyToL2Height := fromL2Height, toL2Height
			if l2BedrockStartingHeight.Cmp(toL2Height) <= 0 {
				legacyToL2Height = new(big.Int).Sub(l2BedrockStartingHeight, bigint.One)
			}

			legacyBridgeLog := l2BridgeLog.New("mode", "legacy", "from_block_number", legacyFromL2Height, "to_block_number", legacyToL2Height)
			legacyBridgeLog.Info("scanning for finalized bridge events")
			if err := bridge.LegacyL2ProcessFinalizedBridgeEvents(legacyBridgeLog, tx, b.metrics, b.chainConfig.L2Contracts, legacyFromL2Height, legacyToL2Height); err != nil {
				return err
			} else if legacyToL2Height.Cmp(toL2Height) == 0 {
				return nil // a-ok! Entire range was legacy blocks
			}
			legacyBridgeLog.Info("detected switch to bedrock", "bedrock_block_number", l2BedrockStartingHeight)
			fromL2Height = l2BedrockStartingHeight
		}

		l2BridgeLog = l2BridgeLog.New("from_block_number", fromL2Height, "to_block_number", toL2Height)
		l2BridgeLog.Info("scanning for finalized bridge events")
		return bridge.L2ProcessFinalizedBridgeEvents(l2BridgeLog, tx, b.metrics, b.chainConfig.L2Contracts, fromL2Height, toL2Height)
	}); err != nil {
		return err
	}

	b.LastFinalizedL2Header = latestL2Header
	b.metrics.RecordL2LatestFinalizedHeight(latestL2Header.Number)
	return nil
}
