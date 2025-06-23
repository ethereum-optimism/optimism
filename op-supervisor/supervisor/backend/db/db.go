package db

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-supervisor/metrics"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/fromda"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type LogStorage interface {
	io.Closer

	IsEmpty() bool

	AddLog(logHash common.Hash, parentBlock eth.BlockID,
		logIdx uint32, execMsg *types.ExecutingMessage) error

	SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error

	Rewind(inv reads.Invalidator, newHead eth.BlockID) error

	// FirstSealedBlock() (block types.BlockSeal, err error)
	LatestSealedBlock() (id eth.BlockID, ok bool)

	// FindSealedBlock finds the requested block by number, to check if it exists,
	// returning the block seal if it was found.
	// returns ErrFuture if the block is too new to be able to tell.
	FindSealedBlock(number uint64) (block types.BlockSeal, err error)

	// Contains returns no error iff the specified logHash is recorded in the specified blockNum and logIdx.
	// If the log is out of reach, then ErrFuture is returned.
	// If the log is determined to conflict with the canonical chain, then ErrConflict is returned.
	// logIdx is the index of the log in the array of all logs in the block.
	// This can be used to check the validity of cross-chain interop events.
	// The block-seal of the blockNum block, that the log was included in, is returned.
	// This seal may be fully zeroed, without error, if the block isn't fully known yet.
	Contains(query types.ContainsQuery) (includedIn types.BlockSeal, err error)

	IteratorStartingAt(sealedNum uint64, logsSince uint32) (logs.Iterator, error)

	// OpenBlock accumulates the ExecutingMessage events for a block and returns them
	OpenBlock(blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*types.ExecutingMessage, err error)
}

type DerivationStorage interface {
	// basic info
	First() (pair types.DerivedBlockSealPair, err error)
	Last() (pair types.DerivedBlockSealPair, err error)

	// mapping from source<>derived
	DerivedToFirstSource(derived eth.BlockID, revision types.Revision) (source types.BlockSeal, err error)
	SourceToLastDerived(source eth.BlockID) (derived types.BlockSeal, err error)

	// traversal
	NextSource(source eth.BlockID) (nextSource types.BlockSeal, err error)

	Candidate(afterSource eth.BlockID, afterDerived eth.BlockID, revision types.Revision) (pair types.DerivedBlockRefPair, err error)

	PreviousSource(source eth.BlockID) (prevSource types.BlockSeal, err error)

	// Warning: only safe to use on cross-DB
	PreviousDerived(derived eth.BlockID, revision types.Revision) (prevDerived types.BlockSeal, err error)

	// type-specific
	Invalidated() (pair types.DerivedBlockSealPair, err error)
	ContainsDerived(derived eth.BlockID, revision types.Revision) error

	// DerivedToRevision is only safe to use on the cross-safe DB.
	DerivedToRevision(derived eth.BlockID) (types.Revision, error)

	LastRevision() (revision types.Revision, err error)
	SourceToRevision(source eth.BlockID) (types.Revision, error)

	// writing

	// AddDerived adds a derived block to the database. The first entry to be added may
	// have zero parent hashes.
	AddDerived(source eth.BlockRef, derived eth.BlockRef, revision types.Revision) error
	ReplaceInvalidatedBlock(inv reads.Invalidator, replacementDerived eth.BlockRef, invalidated common.Hash) (out types.DerivedBlockRefPair, err error)

	// rewinding
	RewindAndInvalidate(inv reads.Invalidator, invalidated types.DerivedBlockRefPair) error
	RewindToScope(inv reads.Invalidator, scope eth.BlockID) error
	RewindToFirstDerived(inv reads.Invalidator, v eth.BlockID, revision types.Revision) error
}

var _ DerivationStorage = (*fromda.DB)(nil)

var _ LogStorage = (*logs.DB)(nil)

type Metrics interface {
	RecordCrossUnsafe(chainID eth.ChainID, seal types.BlockSeal)
	RecordCrossSafe(chainID eth.ChainID, seal types.BlockSeal)
	RecordLocalSafe(chainID eth.ChainID, seal types.BlockSeal)
	RecordLocalUnsafe(chainID eth.ChainID, seal types.BlockSeal)
}

// ChainsDB is a database that stores logs and derived-from data for multiple chains.
// it implements the LogStorage interface, as well as several DB interfaces needed by the cross package.
type ChainsDB struct {
	// unsafe info: the sequence of block seals and events
	logDBs locks.RWMap[eth.ChainID, LogStorage]

	// initLocks: used to prevent certain database calls until initialization is signaled
	// uninitialized chains won't have values in the map
	initialized locks.RWMap[eth.ChainID, struct{}]

	// cross-unsafe: how far we have processed the unsafe data.
	// If present but set to a zeroed value the cross-unsafe will fallback to cross-safe.
	crossUnsafe locks.RWMap[eth.ChainID, *locks.RWValue[types.BlockSeal]]

	// local-safe: index of what we optimistically know about L2 blocks being derived from L1
	localDBs locks.RWMap[eth.ChainID, DerivationStorage]

	// cross-safe: index of L2 blocks we know to only have cross-L2 valid dependencies
	crossDBs locks.RWMap[eth.ChainID, DerivationStorage]

	// finalized: the L1 finality progress. This can be translated into what may be considered as finalized in L2.
	// It is initially zeroed, and the L2 finality query will return
	// an error until it has this L1 finality to work with.
	finalizedL1 locks.RWValue[eth.L1BlockRef]

	// readRegistry tracks what is actively being read,
	// so we can invalidate reads that are affected by rewinds/reorgs.
	readRegistry *reads.Registry

	// depSet is the dependency set, used to determine what may be tracked,
	// what is missing, and to provide it to DB users.
	depSet depset.DependencySet

	logger log.Logger

	// emitter used to signal when the DB changes, for other modules to react to
	emitter event.Emitter

	m Metrics

	rootCtx       context.Context
	rootCtxCancel context.CancelFunc
}

var _ event.AttachEmitter = (*ChainsDB)(nil)

func NewChainsDB(l log.Logger, depSet depset.DependencySet, m Metrics) *ChainsDB {
	if m == nil {
		m = metrics.NoopMetrics
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &ChainsDB{
		logger:        l,
		depSet:        depSet,
		m:             m,
		readRegistry:  reads.NewRegistry(l),
		rootCtx:       ctx,
		rootCtxCancel: cancel,
	}
}

func (db *ChainsDB) AttachEmitter(em event.Emitter) {
	db.emitter = em
}

func (db *ChainsDB) OnEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case superevents.UnsafeActivationBlockEvent:
		if !db.isInitialized(x.ChainID) {
			db.logger.Info("Initializing logs DB from unsafe activation block",
				"chain", x.ChainID, "block", x.Unsafe)
			// Note that isInitialized is only true after full initialization,
			// not only the logs db.
			if err := db.maybeInitFromUnsafe(x.ChainID, x.Unsafe); err != nil {
				db.logger.Error("Error initializing logs DB from unsafe activation block",
					"chain", x.ChainID, "block", x.Unsafe, "err", err)
				return false
			}
			return true
		} else {
			db.logger.Warn("Received unsafe activation block on initialized DB",
				"chain", x.ChainID, "block", x.Unsafe)
			// TODO(#15774): handle reorg
		}
		return false
	case superevents.SafeActivationBlockEvent:
		if !db.isInitialized(x.ChainID) {
			db.logger.Info("Initializing full DB from safe activation block",
				"chain", x.ChainID, "block", x.Safe)
			// Note that isInitialized is only true after full initialization,
			// not only the logs db.
			db.initFromAnchor(x.ChainID, x.Safe)
			return true
		} else {
			db.logger.Warn("Received safe activation block on initialized DB",
				"chain", x.ChainID, "block", x.Safe)
			// TODO(#15774): handle reorg
		}
		return false
	case superevents.LocalDerivedEvent:
		if !db.isInitialized(x.ChainID) {
			// Initialization is handled by SafeActivationBlockEvent, which will probably only be
			// received by the ChainsDB after this event here. So we need to skip processing this
			// event here.
			db.logger.Debug("Received derived event before DB is initialized (expected for activation block)",
				"chain", x.ChainID, "derived", x.Derived, "node", x.NodeID)
			return false
		}
		db.UpdateLocalSafe(x.ChainID, x.Derived.Source, x.Derived.Derived, x.NodeID)
	case superevents.FinalizedL1RequestEvent:
		db.onFinalizedL1(x.FinalizedL1)
	case superevents.ReplaceBlockEvent:
		db.onReplaceBlock(x.ChainID, x.Replacement.Replacement, x.Replacement.Invalidated)
	default:
		return false
	}
	return true
}

func (db *ChainsDB) AddLogDB(chainID eth.ChainID, logDB LogStorage) {
	if db.logDBs.Has(chainID) {
		db.logger.Warn("overwriting existing log DB for chain", "chain", chainID)
	}

	db.logDBs.Set(chainID, logDB)
}

func (db *ChainsDB) AddLocalDerivationDB(chainID eth.ChainID, dfDB DerivationStorage) {
	if db.localDBs.Has(chainID) {
		db.logger.Warn("overwriting existing local derived-from DB for chain", "chain", chainID)
	}

	db.localDBs.Set(chainID, dfDB)
}

func (db *ChainsDB) AddCrossDerivationDB(chainID eth.ChainID, dfDB DerivationStorage) {
	if db.crossDBs.Has(chainID) {
		db.logger.Warn("overwriting existing cross derived-from DB for chain", "chain", chainID)
	}

	db.crossDBs.Set(chainID, dfDB)
}

func (db *ChainsDB) AddCrossUnsafeTracker(chainID eth.ChainID) {
	if db.crossUnsafe.Has(chainID) {
		db.logger.Warn("overwriting existing cross-unsafe tracker for chain", "chain", chainID)
	}
	db.crossUnsafe.Set(chainID, &locks.RWValue[types.BlockSeal]{})
}

// ResumeFromLastSealedBlock prepares the chains db to resume recording events after a restart.
// It rewinds the database to the last block that is guaranteed to have been fully recorded to the database,
// to ensure it can resume recording from the first log of the next block.
func (db *ChainsDB) ResumeFromLastSealedBlock() error {
	var result error
	db.logDBs.Range(func(chain eth.ChainID, logStore LogStorage) bool {
		head, ok := logStore.LatestSealedBlock()
		if !ok {
			// db must be empty, nothing to rewind to
			db.logger.Info("Resuming, but found no DB contents", "chain", chain)
			return true
		}
		db.logger.Info("Resuming, starting from last sealed block", "chain", chain, "head", head)
		if err := logStore.Rewind(db.readRegistry, head); err != nil {
			result = fmt.Errorf("%w: failed to rewind chain %s to sealed block %d", types.ErrRewindFailed, chain, head)
			return false
		}
		return true
	})
	return result
}

func (db *ChainsDB) DependencySet() depset.DependencySet {
	return db.depSet
}

func (db *ChainsDB) Close() error {
	db.rootCtxCancel()
	var combined error
	db.logDBs.Range(func(id eth.ChainID, logDB LogStorage) bool {
		if err := logDB.Close(); err != nil {
			combined = errors.Join(combined, fmt.Errorf("failed to close log db for chain %v: %w", id, err))
		}
		return true
	})
	return combined
}

var _ reads.Acquirer = (*ChainsDB)(nil)

func (db *ChainsDB) AcquireHandle() reads.Handle {
	return db.readRegistry.AcquireHandle()
}
