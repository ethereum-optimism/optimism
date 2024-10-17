package backend

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-supervisor/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/frontend"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type SupervisorBackend struct {
	started atomic.Bool
	logger  log.Logger
	m       Metrics
	dataDir string

	// RW lock to avoid concurrent map mutations.
	// Read = any chain may be used and mutated.
	// Write = set of chains is changing.
	mu sync.RWMutex

	depSet depset.DependencySet

	// db holds on to the DB indices for each chain
	db *db.ChainsDB

	// chainProcessors are notified of new unsafe blocks, and add the unsafe log events data into the events DB
	chainProcessors map[types.ChainID]*processors.ChainProcessor
}

var _ frontend.Backend = (*SupervisorBackend)(nil)

var errAlreadyStopped = errors.New("already stopped")

func NewSupervisorBackend(ctx context.Context, logger log.Logger, m Metrics, cfg *config.Config) (*SupervisorBackend, error) {
	// attempt to prepare the data directory
	if err := prepDataDir(cfg.Datadir); err != nil {
		return nil, err
	}

	// Load the dependency set
	depSet, err := cfg.DependencySetSource.LoadDependencySet(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load dependency set: %w", err)
	}

	// create the chains db
	chainsDB := db.NewChainsDB(logger)

	// create an empty map of chain monitors
	chainProcessors := make(map[types.ChainID]*processors.ChainProcessor, len(cfg.L2RPCs))

	// create the supervisor backend
	super := &SupervisorBackend{
		logger:          logger,
		m:               m,
		dataDir:         cfg.Datadir,
		depSet:          depSet,
		chainProcessors: chainProcessors,
		db:              chainsDB,
	}

	// from the RPC strings, have the supervisor backend create a chain monitor
	// don't start the monitor yet, as we will start all monitors at once when Start is called
	for _, rpc := range cfg.L2RPCs {
		err := super.addFromRPC(ctx, logger, rpc, false)
		if err != nil {
			return nil, fmt.Errorf("failed to add chain monitor for rpc %v: %w", rpc, err)
		}
	}
	return super, nil
}

// addFromRPC adds a chain monitor to the supervisor backend from an rpc endpoint
// it does not expect to be called after the backend has been started
// it will start the monitor if shouldStart is true
func (su *SupervisorBackend) addFromRPC(ctx context.Context, logger log.Logger, rpc string, _ bool) error {
	// create the rpc client, which yields the chain id
	rpcClient, chainID, err := clientForL2(ctx, logger, rpc)
	if err != nil {
		return err
	}
	su.logger.Info("adding from rpc connection", "rpc", rpc, "chainID", chainID)
	// create metrics and a logdb for the chain
	cm := newChainMetrics(chainID, su.m)
	path, err := prepLogDBPath(chainID, su.dataDir)
	if err != nil {
		return fmt.Errorf("failed to create datadir for chain %v: %w", chainID, err)
	}
	logDB, err := logs.NewFromFile(logger, cm, path, true)
	if err != nil {
		return fmt.Errorf("failed to create logdb for chain %v at %v: %w", chainID, path, err)
	}
	if su.chainProcessors[chainID] != nil {
		return fmt.Errorf("chain monitor for chain %v already exists", chainID)
	}
	// create a client like the monitor would have
	cl, err := processors.NewEthClient(
		ctx,
		logger,
		cm,
		rpc,
		rpcClient, 2*time.Second,
		false,
		sources.RPCKindStandard)
	if err != nil {
		return err
	}
	logProcessor := processors.NewLogProcessor(chainID, su.db)
	chainProcessor := processors.NewChainProcessor(logger, cl, chainID, logProcessor, su.db)
	su.chainProcessors[chainID] = chainProcessor
	su.db.AddLogDB(chainID, logDB)
	return nil
}

func clientForL2(ctx context.Context, logger log.Logger, rpc string) (client.RPC, types.ChainID, error) {
	ethClient, err := dial.DialEthClientWithTimeout(ctx, 10*time.Second, logger, rpc)
	if err != nil {
		return nil, types.ChainID{}, fmt.Errorf("failed to connect to rpc %v: %w", rpc, err)
	}
	chainID, err := ethClient.ChainID(ctx)
	if err != nil {
		return nil, types.ChainID{}, fmt.Errorf("failed to load chain id for rpc %v: %w", rpc, err)
	}
	return client.NewBaseRPCClient(ethClient.Client()), types.ChainIDFromBig(chainID), nil
}

func (su *SupervisorBackend) Start(ctx context.Context) error {
	su.mu.Lock()
	defer su.mu.Unlock()

	// ensure we only start once
	if !su.started.CompareAndSwap(false, true) {
		return errors.New("already started")
	}
	// initiate "ResumeFromLastSealedBlock" on the chains db,
	// which rewinds the database to the last block that is guaranteed to have been fully recorded
	if err := su.db.ResumeFromLastSealedBlock(); err != nil {
		return fmt.Errorf("failed to resume chains db: %w", err)
	}
	// TODO(#12423): init background processors, de-dup with constructor
	return nil
}

func (su *SupervisorBackend) Stop(ctx context.Context) error {
	su.mu.Lock()
	defer su.mu.Unlock()

	if !su.started.CompareAndSwap(true, false) {
		return errAlreadyStopped
	}
	// close all processors
	for id, processor := range su.chainProcessors {
		su.logger.Info("stopping chain processor", "chainID", id)
		processor.Close()
	}
	clear(su.chainProcessors)
	// close the databases
	return su.db.Close()
}

// AddL2RPC adds a new L2 chain to the supervisor backend
// it stops and restarts the backend to add the new chain
func (su *SupervisorBackend) AddL2RPC(ctx context.Context, rpc string) error {
	su.mu.Lock()
	defer su.mu.Unlock()

	// start the monitor immediately, as the backend is assumed to already be running
	return su.addFromRPC(ctx, su.logger, rpc, true)
}

// Query methods
// ----------------------------

func (su *SupervisorBackend) CheckMessage(identifier types.Identifier, payloadHash common.Hash) (types.SafetyLevel, error) {
	su.mu.RLock()
	defer su.mu.RUnlock()

	chainID := identifier.ChainID
	blockNum := identifier.BlockNumber
	logIdx := identifier.LogIndex
	_, err := su.db.Check(chainID, blockNum, uint32(logIdx), payloadHash)
	if errors.Is(err, entrydb.ErrFuture) {
		return types.LocalUnsafe, nil
	}
	if errors.Is(err, entrydb.ErrConflict) {
		return types.Invalid, nil
	}
	if err != nil {
		return types.Invalid, fmt.Errorf("failed to check log: %w", err)
	}
	return su.db.Safest(chainID, blockNum, uint32(logIdx))
}

func (su *SupervisorBackend) CheckMessages(
	messages []types.Message,
	minSafety types.SafetyLevel) error {
	su.mu.RLock()
	defer su.mu.RUnlock()

	for _, msg := range messages {
		safety, err := su.CheckMessage(msg.Identifier, msg.PayloadHash)
		if err != nil {
			return fmt.Errorf("failed to check message: %w", err)
		}
		if !safety.AtLeastAsSafe(minSafety) {
			return fmt.Errorf("message %v (safety level: %v) does not meet the minimum safety %v",
				msg.Identifier,
				safety,
				minSafety)
		}
	}
	return nil
}

func (su *SupervisorBackend) UnsafeView(ctx context.Context, chainID types.ChainID, unsafe types.ReferenceView) (types.ReferenceView, error) {
	su.mu.RLock()
	defer su.mu.RUnlock()

	head, err := su.db.LocalUnsafe(chainID)
	if err != nil {
		return types.ReferenceView{}, fmt.Errorf("failed to get local-unsafe head: %w", err)
	}
	cross, err := su.db.CrossUnsafe(chainID)
	if err != nil {
		return types.ReferenceView{}, fmt.Errorf("failed to get cross-unsafe head: %w", err)
	}

	// TODO(#11693): check `unsafe` input to detect reorg conflicts

	return types.ReferenceView{
		Local: head.ID(),
		Cross: cross.ID(),
	}, nil
}

func (su *SupervisorBackend) SafeView(ctx context.Context, chainID types.ChainID, safe types.ReferenceView) (types.ReferenceView, error) {
	su.mu.RLock()
	defer su.mu.RUnlock()

	_, localSafe, err := su.db.LocalSafe(chainID)
	if err != nil {
		return types.ReferenceView{}, fmt.Errorf("failed to get local-safe head: %w", err)
	}
	_, crossSafe, err := su.db.CrossSafe(chainID)
	if err != nil {
		return types.ReferenceView{}, fmt.Errorf("failed to get cross-safe head: %w", err)
	}

	// TODO(#11693): check `safe` input to detect reorg conflicts

	return types.ReferenceView{
		Local: localSafe.ID(),
		Cross: crossSafe.ID(),
	}, nil
}

func (su *SupervisorBackend) Finalized(ctx context.Context, chainID types.ChainID) (eth.BlockID, error) {
	su.mu.RLock()
	defer su.mu.RUnlock()

	return su.db.Finalized(chainID)
}

func (su *SupervisorBackend) DerivedFrom(ctx context.Context, chainID types.ChainID, derived eth.BlockID) (derivedFrom eth.BlockID, err error) {
	su.mu.RLock()
	defer su.mu.RUnlock()

	return su.db.DerivedFrom(chainID, derived)
}

// Update methods
// ----------------------------

func (su *SupervisorBackend) UpdateLocalUnsafe(chainID types.ChainID, head eth.BlockRef) error {
	su.mu.RLock()
	defer su.mu.RUnlock()
	ch, ok := su.chainProcessors[chainID]
	if !ok {
		return db.ErrUnknownChain
	}
	return ch.OnNewHead(head)
}

func (su *SupervisorBackend) UpdateLocalSafe(chainID types.ChainID, derivedFrom eth.BlockRef, lastDerived eth.BlockRef) error {
	su.mu.RLock()
	defer su.mu.RUnlock()

	return su.db.UpdateLocalSafe(chainID, derivedFrom, lastDerived)
}

func (su *SupervisorBackend) UpdateFinalizedL1(chainID types.ChainID, finalized eth.BlockRef) error {
	su.mu.RLock()
	defer su.mu.RUnlock()

	return su.db.UpdateFinalizedL1(finalized)
}
