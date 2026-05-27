package chain_container

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
)

const virtualNodeVersion = "0.1.0"

// ErrHistoryUnavailable is the permanent counterpart of ethereum.NotFound:
// SafeDB on this node cannot and will not contain the requested history
// (e.g. snap/CL-sync bootstrap gap). Interop halts on this rather than
// retrying; recovery requires operator intervention.
var ErrHistoryUnavailable = errors.New("safedb history unavailable on this node")

// ErrSafeDBNotReady is returned by FirstSafeHeadTimestamp when SafeDB does not
// yet have a stable first entry: either it has no entries at all, or the
// deriver has not advanced past the first entry's L1 (so SafeHeadUpdated may
// still overwrite the L2 value at that L1 key). This is a transient condition
// during cold start; callers should back off and retry.
var ErrSafeDBNotReady = errors.New("safedb not ready: no stable first entry yet")

type ChainContainer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Pause(ctx context.Context) error
	Resume(ctx context.Context) error

	ID() eth.ChainID
	ELFinalizedHead(ctx context.Context) (eth.L2BlockRef, error)
	LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error)
	// TimestampToBlockNumber maps an L2 unix timestamp to the L2 block number (rollup derivation).
	TimestampToBlockNumber(ctx context.Context, ts uint64) (uint64, error)
	BlockNumberToTimestamp(ctx context.Context, blocknum uint64) (uint64, error)
	// FirstSafeHeadTimestamp returns the L2 block timestamp of the first
	// entry in this chain's SafeDB. Returns ErrSafeDBNotReady when the chain
	// has not yet derived a stable first safe head.
	FirstSafeHeadTimestamp(ctx context.Context) (uint64, error)
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
	OptimisticAt(ctx context.Context, ts uint64) (l2, l1 eth.BlockID, err error)
	// OutputRootAtL2BlockHash returns the L2 output root for the canonical
	// block at the given hash. Post-Isthmus the root is derived from the
	// header alone; pre-Isthmus it falls back to eth_getProof on state at
	// that block. Returns ethereum.NotFound if the EL no longer has the
	// block at that hash on its canonical chain.
	OutputRootAtL2BlockHash(ctx context.Context, blockHash common.Hash) (eth.Bytes32, error)
	OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputV0, error)
	RegisterVerifier(v activity.VerificationActivity)
	// VerifierCurrentL1 returns the CurrentL1 of the registered verifier.
	// The bool is false when no verifier is registered.
	VerifierCurrentL1() (eth.BlockID, bool)
	// FetchReceipts fetches the receipts for a given block by hash.
	// Returns block info and receipts, or an error if the block or receipts cannot be fetched.
	FetchReceipts(ctx context.Context, blockHash eth.BlockID) (eth.BlockInfo, types.Receipts, error)
	// BlockTime returns the block time in seconds for this chain.
	BlockTime() uint64
	// PruneDeniedAtOrAfterTimestamp removes deny-list entries with DecisionTimestamp >= timestamp.
	// Returns map of removed hashes by height.
	PruneDeniedAtOrAfterTimestamp(timestamp uint64) (map[uint64][]common.Hash, error)
	// PauseAndStopVN pauses the chain container restart loop and stops the virtual node.
	// This is used to freeze a chain's VN before a multi-chain rewind begins, preventing
	// the VN from issuing forkchoice updates that race with the rewind of a peer chain.
	PauseAndStopVN(ctx context.Context) error
	// IsDenied checks if a block hash is on the deny list at the given height.
	IsDenied(height uint64, payloadHash common.Hash) (bool, error)
	// GetDeniedOutput returns the reconstructed OutputV0 for a denied block.
	// Returns nil if the block is not denied at that height.
	GetDeniedOutput(height uint64, payloadHash common.Hash) (*eth.OutputV0, error)
	// OutputV0AtBlockNumber returns the full OutputV0 for the block at the given number.
	OutputV0AtBlockNumber(ctx context.Context, l2BlockNum uint64) (*eth.OutputV0, error)
	// PayloadByHash returns the canonical execution payload envelope for the given block hash.
	// Used by interop build paths to capture canonical payloads for WAL'd rewind operations.
	PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error)
	// PayloadByNumber returns the canonical execution payload envelope for the given block number.
	// Used by interop build paths when the rewind target is below the verified frontier.
	PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error)
	// SetResetCallback sets a callback that is invoked when the chain resets.
	// The supernode uses this to notify activities about chain resets.
	SetResetCallback(cb ResetCallback)
}

// WARNING: InteropChain exposes the reorg-triggering operations (RewindEngine,
// InvalidateBlock) that bypass the interop WAL model when invoked outside of
// interop transition application. ONLY the interop activity should accept or
// hold a value of this interface. Every other caller must take the narrower
// ChainContainer above so the misuse is caught at compile time.
type InteropChain interface {
	ChainContainer
	// HasDeniedAtOrAfterTimestamp returns true if any deny-list entry has
	// DecisionTimestamp >= timestamp, without mutating the deny list.
	HasDeniedAtOrAfterTimestamp(timestamp uint64) (bool, error)
	// RewindEngine rewinds the engine to the supplied target block. target is the canonical
	// payload at the rewind destination, loaded from durable WAL storage by the caller —
	// not re-derived from the live engine. invalidatedBlock is the block that triggered the
	// rewind and is passed to reset callbacks (it is purely informational).
	RewindEngine(ctx context.Context, target *eth.ExecutionPayloadEnvelope, invalidatedBlock eth.BlockRef) error
	// InvalidateBlock adds a block to the deny list and triggers a rewind if the chain
	// currently uses that block at the specified height. parentPayload is the canonical
	// payload at height-1 (the rewind destination) loaded from durable WAL storage —
	// callers must capture it at build time, before the rewind starts. Returns true if
	// a rewind was triggered, false otherwise.
	InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash, decisionTimestamp uint64, stateRoot, messagePasserStorageRoot eth.Bytes32, parentPayload *eth.ExecutionPayloadEnvelope) (bool, error)
}

type virtualNodeFactory func(cfg *opnodecfg.Config, log gethlog.Logger, initOverrides *rollupNode.InitializationOverrides, appVersion string, superAuthority rollup.SuperAuthority) virtual_node.VirtualNode

// ResetCallback is called when the chain container resets due to an invalidated block.
// The supernode uses this to notify activities about the reset.
// invalidatedBlock is the block that was invalidated and triggered the reset.
type ResetCallback func(chainID eth.ChainID, timestamp uint64, invalidatedBlock eth.BlockRef)

type simpleChainContainer struct {
	// vnMu protects vn. The chain container restart loop writes vn concurrently
	// with activity goroutines (e.g. Interop.checkChainsReady) reading it, so an
	// unsynchronized access produces a torn interface read: non-nil type with
	// nil data pointer, which slips past a plain `vn == nil` check and panics
	// on the next method dispatch.
	vnMu               sync.RWMutex
	vn                 virtual_node.VirtualNode
	vncfg              *opnodecfg.Config
	cfg                config.CLIConfig
	engine             engine_controller.EngineController
	denyList           *DenyList
	pause              atomic.Bool
	stop               atomic.Bool
	resetting          atomic.Bool
	stopped            chan struct{}
	log                gethlog.Logger
	chainID            eth.ChainID
	initOverload       *rollupNode.InitializationOverrides     // Base shared resources for all virtual nodes
	rpcHandler         *oprpc.Handler                          // Current per-chain RPC handler instance
	setHandler         func(chainID string, h http.Handler)    // Set the RPC handler on the router for the chain
	addMetricsRegistry func(key string, g prometheus.Gatherer) // Set the metrics registry on the global metrics server
	appVersion         string
	virtualNodeFactory virtualNodeFactory    // Factory function to create virtual node (for testing)
	rollupClient       *sources.RollupClient // In-proc rollup RPC client bound to rpcHandler
	metrics            *resources.SupernodeMetrics

	// verifierMu guards writes and reads of the verifier.
	verifierMu sync.RWMutex
	verifier   activity.VerificationActivity
	onReset    ResetCallback // Called when chain resets to notify activities
}

// Interface conformance assertions
var _ ChainContainer = (*simpleChainContainer)(nil)
var _ InteropChain = (*simpleChainContainer)(nil)

// rollup.SuperAuthority conformance is asserted in super_authority.go alongside
// the method definitions.

func NewChainContainer(
	chainID eth.ChainID,
	vncfg *opnodecfg.Config,
	log gethlog.Logger,
	cfg config.CLIConfig,
	initOverload *rollupNode.InitializationOverrides,
	rpcHandler *oprpc.Handler,
	setHandler func(chainID string, h http.Handler),
	addMetricsRegistry func(key string, g prometheus.Gatherer),
	metrics *resources.SupernodeMetrics,
) InteropChain {
	if metrics == nil {
		metrics = resources.NewSupernodeMetrics()
	}
	c := &simpleChainContainer{
		vncfg:              vncfg,
		cfg:                cfg,
		chainID:            chainID,
		log:                log,
		stopped:            make(chan struct{}, 1),
		initOverload:       initOverload,
		rpcHandler:         rpcHandler,
		setHandler:         setHandler,
		addMetricsRegistry: addMetricsRegistry,
		appVersion:         virtualNodeVersion,
		virtualNodeFactory: defaultVirtualNodeFactory,
		metrics:            metrics,
	}
	vncfg.SafeDBPath = c.subPath("safe_db")
	vncfg.RPC = cfg.RPCConfig
	// Attach in-proc rollup client if an initial handler is provided
	if c.rpcHandler != nil {
		if err := c.attachInProcRollupClient(); err != nil {
			log.Warn("failed to attach in-proc rollup client (initial)", "err", err)
		}
	}
	// Initialize the deny list for block invalidation
	denyListPath := c.subPath("denylist")
	if denyList, err := OpenDenyList(denyListPath); err != nil {
		log.Error("failed to open deny list", "err", err)
	} else {
		c.denyList = denyList
	}
	return c
}

func (c *simpleChainContainer) ID() eth.ChainID {
	return c.chainID
}

// RegisterVerifier registers the verification activity for this chain container.
// Allows late binding when activities and chains have circular dependencies.
// The chain container supports a single verifier; a second call panics.
func (c *simpleChainContainer) RegisterVerifier(v activity.VerificationActivity) {
	c.verifierMu.Lock()
	defer c.verifierMu.Unlock()
	if c.verifier != nil {
		panic("chain container supports only one verifier")
	}
	c.verifier = v
}

func (c *simpleChainContainer) VerifierCurrentL1() (eth.BlockID, bool) {
	v := c.registeredVerifier()
	if v == nil {
		return eth.BlockID{}, false
	}
	return v.CurrentL1(), true
}

func (c *simpleChainContainer) registeredVerifier() activity.VerificationActivity {
	c.verifierMu.RLock()
	defer c.verifierMu.RUnlock()
	return c.verifier
}

// defaultVirtualNodeFactory is the default factory that creates a real VirtualNode
func defaultVirtualNodeFactory(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string, superAuthority rollup.SuperAuthority) virtual_node.VirtualNode {
	initOverload.SuperAuthority = superAuthority
	return virtual_node.NewVirtualNode(cfg, log, initOverload, appVersion)
}

// getVN returns a consistent snapshot of c.vn. Always read c.vn through this
// helper; a bare read races with the restart loop's assignment.
func (c *simpleChainContainer) getVN() virtual_node.VirtualNode {
	c.vnMu.RLock()
	defer c.vnMu.RUnlock()
	return c.vn
}

// setVN writes c.vn under the lock. Used by the restart loop when it installs
// a new virtual node on each iteration.
func (c *simpleChainContainer) setVN(vn virtual_node.VirtualNode) {
	c.vnMu.Lock()
	defer c.vnMu.Unlock()
	c.vn = vn
}

func (c *simpleChainContainer) subPath(path string) string {
	return filepath.Join(c.cfg.DataDir, c.chainID.String(), path)
}

func (c *simpleChainContainer) Start(ctx context.Context) error {
	defer func() { c.stopped <- struct{}{} }()
	for {
		// Refresh per-start derived fields
		c.vncfg.SafeDBPath = c.subPath("safe_db")
		c.vncfg.RPC = c.cfg.RPCConfig
		// create a fresh handler per (re)start, swap it into the router, and inject into overload
		h := oprpc.NewHandler("", oprpc.WithLogger(c.log.New("chain_id", c.chainID.String())))
		if c.setHandler != nil {
			c.setHandler(c.chainID.String(), h)
		}
		c.initOverload.RPCHandler = h
		c.rpcHandler = h
		// attach in-proc rollup client for this handler
		if err := c.attachInProcRollupClient(); err != nil {
			c.log.Warn("failed to attach in-proc rollup client", "err", err)
		}

		// Disable per-VN metrics server and provide metrics registry hook
		c.vncfg.Metrics.Enabled = false
		if c.initOverload != nil {
			c.initOverload.MetricsRegistry = func(reg *prometheus.Registry) {
				if c.addMetricsRegistry != nil {
					c.addMetricsRegistry(c.chainID.String(), reg)
				}
			}
			// Pass the chain container as SuperAuthority for payload denylist checks
			c.initOverload.SuperAuthority = c
		}
		// Pass in the chain container as a SuperAuthority
		vn := c.virtualNodeFactory(c.vncfg, c.log, c.initOverload, c.appVersion, c)
		if vn == nil {
			return virtual_node.ErrVirtualNodeNotRunning
		}
		c.setVN(vn)
		if c.pause.Load() {
			// Check for stop/cancellation even while paused, so teardown doesn't hang.
			// Without this, a stuck pause (e.g. from RewindEngine exiting before Resume)
			// causes this loop to spin forever, blocking wg.Wait() in Supernode.Stop().
			if c.stop.Load() || ctx.Err() != nil {
				c.log.Info("chain container stop requested while paused, stopping restart loop")
				break
			}
			c.log.Info("chain container paused")
			time.Sleep(1 * time.Second)
			continue
		}
		if c.stop.Load() {
			break
		}

		// Initialize engine controller if not yet connected (retries on each VN restart)
		if c.engine == nil && c.vncfg.L2 != nil {
			setupCtx, setupCancel := context.WithTimeout(ctx, 10*time.Second)
			engLog := c.log.New("chain_id", c.chainID.String(), "component", "engine_controller")
			if eng, engErr := engine_controller.NewEngineControllerFromConfig(setupCtx, engLog, c.vncfg); engErr != nil {
				c.log.Error("failed to setup engine controller", "err", engErr)
			} else {
				c.engine = eng
			}
			setupCancel()
		}

		// start the virtual node
		err := vn.Start(ctx)
		if err != nil {
			c.log.Warn("virtual node exited with error", "vn_id", vn, "error", err)
		} else {
			c.log.Info("virtual node exited", "vn_id", vn)
		}

		// always stop the virtual node after it exits
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if stopErr := vn.Stop(stopCtx); stopErr != nil {
			c.log.Error("error stopping virtual node", "error", stopErr)
		} else {
			c.log.Info("virtual node stopped", "vn_id", vn)
		}

		cancel()
		if ctx.Err() != nil {
			c.log.Info("chain container context cancelled, stopping restart loop", "ctx_err", ctx.Err())
			break
		}

		// check if the chain container was stopped
		if c.stop.Load() {
			c.log.Info("chain container stop requested, stopping restart loop")
			break
		}

		c.metrics.VNRestarts.WithLabelValues(c.chainID.String()).Inc()
	}
	c.log.Info("chain container exiting")
	return nil
}

func (c *simpleChainContainer) Stop(ctx context.Context) error {
	c.stop.Store(true)
	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Close in-proc rollup RPC resources
	if c.rollupClient != nil {
		c.rollupClient.Close()
	}

	if vn := c.getVN(); vn != nil {
		if err := vn.Stop(stopCtx); err != nil {
			c.log.Error("error stopping virtual node", "error", err)
		}
	}

	// Close engine controller RPC resources
	if c.engine != nil {
		_ = c.engine.Close()
	}

	// Close deny list database
	if c.denyList != nil {
		if err := c.denyList.Close(); err != nil {
			c.log.Error("error closing deny list", "error", err)
		}
	}

	select {
	case <-c.stopped:
		return nil
	case <-stopCtx.Done():
		return stopCtx.Err()
	}
}

func (c *simpleChainContainer) Pause(ctx context.Context) error {
	c.pause.Store(true)
	return nil
}

func (c *simpleChainContainer) Resume(ctx context.Context) error {
	c.pause.Store(false)
	return nil
}

func (c *simpleChainContainer) ELFinalizedHead(ctx context.Context) (eth.L2BlockRef, error) {
	if c.engine == nil {
		return eth.L2BlockRef{}, engine_controller.ErrNoEngineClient
	}
	return c.engine.L2BlockRefByLabel(ctx, eth.Finalized)
}

func (c *simpleChainContainer) TimestampToBlockNumber(ctx context.Context, ts uint64) (uint64, error) {
	if c.vncfg == nil {
		return 0, fmt.Errorf("rollup config not available")
	}
	return c.vncfg.Rollup.TargetBlockNumber(ts)
}

func (c *simpleChainContainer) BlockNumberToTimestamp(ctx context.Context, blocknum uint64) (uint64, error) {
	if c.vncfg == nil {
		return 0, fmt.Errorf("rollup config not available")
	}
	if blocknum < c.vncfg.Rollup.Genesis.L2.Number {
		return 0, fmt.Errorf("block number %d before genesis %d", blocknum, c.vncfg.Rollup.Genesis.L2.Number)
	}
	return c.vncfg.Rollup.TimestampForBlock(blocknum), nil
}

// FirstSafeHeadTimestamp returns the timestamp of SafeDB's first entry, but
// only once the deriver has moved past that entry's L1. SafeHeadUpdated
// overwrites entries at the same L1 key, so until then the L2 value is still
// in flight and snapshots can go stale. Sample SyncStatus before
// FirstSafeHeadEntry so firstEntry.L1 < capturedCurrentL1 implies the writes
// at that L1 had already completed.
func (c *simpleChainContainer) FirstSafeHeadTimestamp(ctx context.Context) (uint64, error) {
	vn := c.getVN()
	if vn == nil {
		return 0, virtual_node.ErrVirtualNodeNotRunning
	}
	status, err := vn.SyncStatus(ctx)
	if err != nil {
		return 0, fmt.Errorf("sync status: %w", err)
	}
	l1, l2, err := vn.FirstSafeHeadEntry(ctx)
	if err != nil {
		if errors.Is(err, safedb.ErrNotFound) {
			return 0, ErrSafeDBNotReady
		}
		return 0, fmt.Errorf("first safedb entry: %w", err)
	}
	if status.CurrentL1.Number <= l1.Number {
		c.log.Debug("first SafeDB entry not yet stable: deriver still on its L1",
			"first_l1", l1.Number, "current_l1", status.CurrentL1.Number)
		return 0, ErrSafeDBNotReady
	}
	return c.BlockNumberToTimestamp(ctx, l2.Number)
}

// LocalSafeBlockAtTimestamp returns the highest L2 block with timestamp <= ts using the L2 client,
// if the block at that timestamp is local safe.
func (c *simpleChainContainer) LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	if c.engine == nil {
		return eth.L2BlockRef{}, engine_controller.ErrNoEngineClient
	}

	// Compute the target block directly from rollup config
	num, err := c.vncfg.Rollup.TargetBlockNumber(ts)
	c.log.Debug("computed target block number from timestamp", "timestamp", ts, "targetBlockNumber", num)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	ss, err := c.SyncStatus(ctx)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	head := ss.LocalSafeL2
	if num > head.Number {
		c.log.Debug("target block number exceeds local safe head", "targetBlockNumber", num, "head", head.Number)
		return eth.L2BlockRef{}, ethereum.NotFound
	}

	return c.engine.L2BlockRefByNumber(ctx, num)
}

// SyncStatus returns the in-process op-node sync status for this chain.
func (c *simpleChainContainer) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	vn := c.getVN()
	if vn == nil {
		if c.log != nil {
			c.log.Warn("SyncStatus: virtual node not initialized")
		}
		return nil, virtual_node.ErrVirtualNodeNotRunning
	}
	st, err := vn.SyncStatus(ctx)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (c *simpleChainContainer) OutputRootAtL2BlockHash(ctx context.Context, blockHash common.Hash) (eth.Bytes32, error) {
	if c.engine == nil {
		return eth.Bytes32{}, engine_controller.ErrNoEngineClient
	}
	out, err := c.engine.OutputV0ByBlockHash(ctx, blockHash)
	if err != nil {
		return eth.Bytes32{}, err
	}
	return eth.OutputRoot(out), nil
}

// safeDBAtL2 delegates to the virtual node to resolve the earliest L1 at which the L2 became safe.
func (c *simpleChainContainer) safeDBAtL2(ctx context.Context, l2 eth.BlockID) (eth.BlockID, error) {
	vn := c.getVN()
	if vn == nil {
		return eth.BlockID{}, fmt.Errorf("virtual node not initialized")
	}
	status, err := c.SyncStatus(ctx)
	if err != nil {
		return eth.BlockID{}, err
	}
	currentL1 := status.CurrentL1
	c.log.Debug("safeDBAtL2", "l2", l2, "currentL1", currentL1, "err", err)
	l1, err := vn.L1AtSafeHead(ctx, l2)
	if err != nil {
		// Permanent history gap -> ErrHistoryUnavailable (interop halts).
		// Transient lag -> ethereum.NotFound (callers back off and retry).
		if errors.Is(err, virtual_node.ErrL1AtSafeHeadUnavailable) {
			return eth.BlockID{}, fmt.Errorf("L1 at safe head unavailable for L2 %s: %w", l2, ErrHistoryUnavailable)
		}
		if errors.Is(err, virtual_node.ErrL1AtSafeHeadNotFound) {
			return eth.BlockID{}, fmt.Errorf("L1 at safe head not available for L2 %s: %w", l2, ethereum.NotFound)
		}
		return eth.BlockID{}, err
	}
	return l1, nil
}

// OptimisticAt returns the optimistic (pre-verified) L2 and L1 blocks for the given L2 timestamp.
func (c *simpleChainContainer) OptimisticAt(ctx context.Context, ts uint64) (l2, l1 eth.BlockID, err error) {
	l2Block, err := c.LocalSafeBlockAtTimestamp(ctx, ts)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			c.log.Debug("l2 block at timestamp is not local safe yet", "timestamp", ts, "err", err)
		} else {
			c.log.Error("error determining l2 block at given timestamp", "timestamp", ts, "err", err)
		}
		return eth.BlockID{}, eth.BlockID{}, err
	}
	l1Block, err := c.safeDBAtL2(ctx, l2Block.ID())
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			c.log.Debug("l1 block at which l2 block became safe is not available yet", "l2", l2Block.ID(), "err", err)
		} else {
			c.log.Error("error determining l1 block number at which l2 block became safe", "l2", l2Block.ID(), "err", err)
		}
		return eth.BlockID{}, eth.BlockID{}, err
	}

	// VerifiedAt only constrains the result when registered verification
	// activities report that the timestamp is not yet verified. Otherwise the
	// current safe L2/L1 pair can be returned directly.
	return l2Block.ID(), l1Block, nil
}

// OptimisticOutputAtTimestamp returns the OutputV0 for the "optimistic" L2 block at the given timestamp.
// If the block at this height has been denied (invalidated and replaced), the optimistic output
// is the original (pre-replacement) block's output from the deny list — because optimistically
// the block would not have been replaced. Otherwise it returns the current local safe block's output.
func (c *simpleChainContainer) OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputV0, error) {
	blockNum, err := c.TimestampToBlockNumber(ctx, ts)
	if err != nil {
		return nil, fmt.Errorf("failed to convert timestamp to block number: %w", err)
	}

	if c.denyList != nil {
		outV0, err := c.denyList.LastDeniedOutputV0(blockNum)
		if err != nil {
			return nil, fmt.Errorf("failed to query deny list at height %d: %w", blockNum, err)
		}
		if outV0 != nil {
			return outV0, nil
		}
	}

	return c.OutputV0AtBlockNumber(ctx, blockNum)
}

// FetchReceipts fetches the receipts for a given block by hash.
func (c *simpleChainContainer) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error) {
	if c.engine == nil {
		return nil, engine_controller.ErrNoEngineClient
	}
	return c.engine.PayloadByHash(ctx, hash)
}

func (c *simpleChainContainer) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	if c.engine == nil {
		return nil, engine_controller.ErrNoEngineClient
	}
	return c.engine.PayloadByNumber(ctx, number)
}

func (c *simpleChainContainer) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, types.Receipts, error) {
	if c.engine == nil {
		return nil, nil, engine_controller.ErrNoEngineClient
	}
	return c.engine.FetchReceipts(ctx, blockID.Hash)
}

// BlockTime returns the block time in seconds for this chain from the rollup config.
func (c *simpleChainContainer) BlockTime() uint64 {
	if c.vncfg == nil {
		return 0
	}
	return c.vncfg.Rollup.BlockTime
}

// attachInProcRollupClient creates a new in-proc rollup RPC client bound to the current rpcHandler.
// It will close any existing client before replacing it.
func (c *simpleChainContainer) attachInProcRollupClient() error {
	if c.rpcHandler == nil {
		return fmt.Errorf("rpc handler not initialized")
	}
	inproc, err := c.rpcHandler.DialInProc()
	if err != nil {
		return err
	}
	// Close previous rollup client if present
	if c.rollupClient != nil {
		c.rollupClient.Close()
	}
	c.rollupClient = sources.NewRollupClient(client.NewBaseRPCClient(inproc))
	return nil
}

// isCriticalRewindError returns true if the error is a critical configuration error
// that should not be retried.
func isCriticalRewindError(err error) bool {
	return errors.Is(err, engine_controller.ErrNoEngineClient) ||
		errors.Is(err, engine_controller.ErrNoRollupConfig) ||
		errors.Is(err, engine_controller.ErrRewindNilTarget) ||
		errors.Is(err, engine_controller.ErrRewindTargetMismatch) ||
		errors.Is(err, engine_controller.ErrRewindComputeTargetsFailed) ||
		errors.Is(err, engine_controller.ErrRewindTimestampToBlockConversion) ||
		errors.Is(err, engine_controller.ErrRewindOverFinalizedHead)
}

// RewindEngine is part of the InteropChain interface — callers must hold that
// wider interface (only interop transition application does) to invoke it.
func (c *simpleChainContainer) RewindEngine(ctx context.Context, target *eth.ExecutionPayloadEnvelope, invalidatedBlock eth.BlockRef) error {
	if target == nil || target.ExecutionPayload == nil {
		return engine_controller.ErrRewindNilTarget
	}
	timestamp := uint64(target.ExecutionPayload.Timestamp)
	if !c.resetting.CompareAndSwap(false, true) {
		return fmt.Errorf("reset already in progress")
	}
	defer c.resetting.Store(false)

	vn := c.getVN()
	if vn == nil {
		return fmt.Errorf("virtual node not initialized")
	}
	if c.engine == nil {
		return fmt.Errorf("engine not initialized")
	}

	// Pause the container to stop it restarting the vn when we kill it
	err := c.Pause(ctx)
	if err != nil {
		return err
	}
	// Always resume the container on return, even if we exit early due to context cancellation
	// or an error mid-rewind. Without this, a cancelled ctx leaves pause=true permanently,
	// causing the Start() loop to spin forever and block Supernode.Stop()'s wg.Wait().
	defer c.Resume(context.Background()) //nolint:errcheck
	c.log.Info("chain_container/RewindEngine: paused container")

	// stop the vn
	err = vn.Stop(ctx)
	if err != nil {
		return err
	}
	c.log.Info("chain_container/RewindEngine: stopped vn")

retryLoop:
	for {
		err = c.engine.Rewind(ctx, target)
		switch {
		case isCriticalRewindError(err):
			c.log.Error("chain_container/RewindEngine: critical error", "err", err)
			return err
		case err == nil:
			c.log.Info("chain_container/RewindEngine: executed engine rewind")
			break retryLoop
		default:
			// context.DeadlineExceeded is treated as transient: the caller's ctx is
			// the long-lived service ctx (no deadline), so a DeadlineExceeded here
			// originates from a per-call RPC deadline inside the engine client — e.g.
			// a slow synthetic FCU against op-reth. Keep retrying: if the EL is truly
			// down a subsequent attempt will surface a transport error, and if it is
			// still chewing on the rewind, waiting is the right answer. The ctx.Done()
			// branch below stops the loop on service shutdown.
			c.log.Error("chain_container/RewindEngine: temporary error", "err", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
			}
		}
	}

	// Notify activities about the reset
	if c.onReset != nil {
		c.onReset(c.chainID, timestamp, invalidatedBlock)
	}

	// resume the chain container to trigger a new vn to be started
	err = c.Resume(ctx)
	if err != nil {
		return err
	}
	c.log.Info("chain_container/RewindEngine: resumed container")

	return nil
}

// PauseAndStopVN pauses the container restart loop and stops the running virtual node.
// This must be called before a multi-chain rewind to prevent a peer chain's VN from
// issuing forkchoice updates that race with the rewind operation.
// RewindEngine's own Pause+Stop calls are idempotent when called after this.
func (c *simpleChainContainer) PauseAndStopVN(ctx context.Context) error {
	if err := c.Pause(ctx); err != nil {
		return err
	}
	vn := c.getVN()
	if vn == nil {
		return nil
	}
	return vn.Stop(ctx)
}

// SetResetCallback sets a callback that is invoked when the chain resets.
// This must only be called during initialization, before the chain container starts processing.
// Calling this while InvalidateBlock may be running is unsafe.
func (c *simpleChainContainer) SetResetCallback(cb ResetCallback) {
	c.onReset = cb
}
