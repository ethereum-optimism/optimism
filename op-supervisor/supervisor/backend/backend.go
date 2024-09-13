package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-supervisor/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/source"
	backendTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/frontend"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

type SupervisorBackend struct {
	started atomic.Bool
	logger  log.Logger
	m       Metrics
	dataDir string

	chainMonitors map[types.ChainID]*source.ChainMonitor
	db            *db.ChainsDB

	maintenanceCancel context.CancelFunc
}

var _ frontend.Backend = (*SupervisorBackend)(nil)

var _ io.Closer = (*SupervisorBackend)(nil)

func NewSupervisorBackend(ctx context.Context, logger log.Logger, m Metrics, cfg *config.Config) (*SupervisorBackend, error) {
	// attempt to prepare the data directory
	if err := prepDataDir(cfg.Datadir); err != nil {
		return nil, err
	}

	// create the head tracker
	headTracker, err := heads.NewHeadTracker(filepath.Join(cfg.Datadir, "heads.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load existing heads: %w", err)
	}

	// create the chains db
	db := db.NewChainsDB(map[types.ChainID]db.LogStorage{}, headTracker)

	// create an empty map of chain monitors
	chainMonitors := make(map[types.ChainID]*source.ChainMonitor, len(cfg.L2RPCs))

	// create the supervisor backend
	super := &SupervisorBackend{
		logger:        logger,
		m:             m,
		dataDir:       cfg.Datadir,
		chainMonitors: chainMonitors,
		db:            db,
	}

	// from the RPC strings, have the supervisor backend create a chain monitor
	for _, rpc := range cfg.L2RPCs {
		err := super.addFromRPC(ctx, logger, rpc)
		if err != nil {
			return nil, fmt.Errorf("failed to add chain monitor for rpc %v: %w", rpc, err)
		}
	}
	return super, nil
}

// addFromRPC adds a chain monitor to the supervisor backend from an rpc endpoint
// it does not expect to be called after the backend has been started
func (su *SupervisorBackend) addFromRPC(ctx context.Context, logger log.Logger, rpc string) error {
	// create the rpc client, which yields the chain id
	rpcClient, chainID, err := createRpcClient(ctx, logger, rpc)
	if err != nil {
		return err
	}
	// create metrics and a logdb for the chain
	cm := newChainMetrics(chainID, su.m)
	path, err := prepLogDBPath(chainID, su.dataDir)
	if err != nil {
		return fmt.Errorf("failed to create datadir for chain %v: %w", chainID, err)
	}
	logDB, err := logs.NewFromFile(logger, cm, path)
	if err != nil {
		return fmt.Errorf("failed to create logdb for chain %v at %v: %w", chainID, path, err)
	}
	monitor, err := source.NewChainMonitor(ctx, logger, cm, chainID, rpc, rpcClient, su.db)
	if err != nil {
		return fmt.Errorf("failed to create monitor for rpc %v: %w", rpc, err)
	}
	if su.chainMonitors[chainID] != nil {
		return fmt.Errorf("chain monitor for chain %v already exists", chainID)
	}
	su.chainMonitors[chainID] = monitor
	su.db.AddLogDB(chainID, logDB)
	return nil
}

func createRpcClient(ctx context.Context, logger log.Logger, rpc string) (client.RPC, types.ChainID, error) {
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
	// ensure we only start once
	if !su.started.CompareAndSwap(false, true) {
		return errors.New("already started")
	}
	// initiate "Resume" on the chains db, which rewinds the database to the last block that is guaranteed to have been fully recorded
	if err := su.db.Resume(); err != nil {
		return fmt.Errorf("failed to resume chains db: %w", err)
	}
	// start chain monitors
	for _, monitor := range su.chainMonitors {
		if err := monitor.Start(); err != nil {
			return fmt.Errorf("failed to start chain monitor: %w", err)
		}
	}
	// start db maintenance loop
	maintinenceCtx, cancel := context.WithCancel(ctx)
	su.db.StartCrossHeadMaintenance(maintinenceCtx)
	su.maintenanceCancel = cancel
	return nil
}

func (su *SupervisorBackend) Stop(ctx context.Context) error {
	if !su.started.CompareAndSwap(true, false) {
		return errors.New("already stopped")
	}
	// signal the maintenance loop to stop
	su.maintenanceCancel()
	// collect errors from stopping chain monitors
	var errs error
	for _, monitor := range su.chainMonitors {
		if err := monitor.Stop(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop chain monitor: %w", err))
		}
	}
	// close the database
	if err := su.db.Close(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to close database: %w", err))
	}
	return errs
}

func (su *SupervisorBackend) Close() error {
	// TODO(protocol-quest#288): close logdb of all chains
	return nil
}

// AddL2RPC adds a new L2 chain to the supervisor backend
// it stops and restarts the backend to add the new chain
func (su *SupervisorBackend) AddL2RPC(ctx context.Context, rpc string) error {
	if err := su.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop backend: %w", err)
	}
	su.logger.Info("temporarily stopped the backend to add a new L2 RPC", "rpc", rpc)
	if err := su.addFromRPC(ctx, su.logger, rpc); err != nil {
		return fmt.Errorf("failed to add chain monitor: %w", err)
	}
	su.logger.Info("added the new L2 RPC, starting supervisor again", "rpc", rpc)
	return su.Start(ctx)
}

func (su *SupervisorBackend) CheckMessage(identifier types.Identifier, payloadHash common.Hash) (types.SafetyLevel, error) {
	chainID := identifier.ChainID
	blockNum := identifier.BlockNumber
	logIdx := identifier.LogIndex
	ok, i, err := su.db.Check(chainID, blockNum, uint32(logIdx), backendTypes.TruncateHash(payloadHash))
	if err != nil {
		return types.Invalid, fmt.Errorf("failed to check log: %w", err)
	}
	if !ok {
		return types.Invalid, nil
	}
	safest := types.CrossUnsafe
	// at this point we have the log entry, and we can check if it is safe by various criteria
	for _, checker := range []db.SafetyChecker{
		db.NewSafetyChecker(types.Unsafe, su.db),
		db.NewSafetyChecker(types.Safe, su.db),
		db.NewSafetyChecker(types.Finalized, su.db),
	} {
		if i <= checker.CrossHeadForChain(chainID) {
			safest = checker.SafetyLevel()
		}
	}
	return safest, nil
}

func (su *SupervisorBackend) CheckMessages(
	messages []types.Message,
	minSafety types.SafetyLevel) error {
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

// CheckBlock checks if the block is safe according to the safety level
// The block is considered safe if all logs in the block are safe
// this is decided by finding the last log in the block and
func (su *SupervisorBackend) CheckBlock(chainID *hexutil.U256, blockHash common.Hash, blockNumber hexutil.Uint64) (types.SafetyLevel, error) {
	// TODO(#11612): this function ignores blockHash and assumes that the block in the db is the one we are looking for
	// In order to check block hash, the database must *always* insert a block hash checkpoint, which is not currently done
	safest := types.CrossUnsafe
	// find the last log index in the block
	i, err := su.db.LastLogInBlock(types.ChainID(*chainID), uint64(blockNumber))
	// TODO(#11836) checking for EOF as a non-error case is a bit of a code smell
	// and could potentially be incorrect if the given block number is intentionally far in the future
	if err != nil && !errors.Is(err, io.EOF) {
		su.logger.Error("failed to scan block", "err", err)
		return types.Invalid, fmt.Errorf("failed to scan block: %w", err)
	}
	// at this point we have the extent of the block, and we can check if it is safe by various criteria
	for _, checker := range []db.SafetyChecker{
		db.NewSafetyChecker(types.Unsafe, su.db),
		db.NewSafetyChecker(types.Safe, su.db),
		db.NewSafetyChecker(types.Finalized, su.db),
	} {
		if i <= checker.CrossHeadForChain(types.ChainID(*chainID)) {
			safest = checker.SafetyLevel()
		}
	}
	return safest, nil
}
