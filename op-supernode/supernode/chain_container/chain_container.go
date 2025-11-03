package chain_container

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"time"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const virtualNodeVersion = "0.1.0"

type ChainContainer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Pause(ctx context.Context) error
	Resume(ctx context.Context) error

	SafeAtTimestamp(ctx context.Context, ts uint64) (bool, error)
	VerifiedAtTimestamp(ctx context.Context, ts uint64) (bool, error)
	BlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error)
	OutputRootAtL1(ctx context.Context, l1BlockNum uint64) (eth.Bytes32, error)
	SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (l1 eth.BlockID, l2 eth.BlockID, err error)
	CurrentL1(ctx context.Context) (eth.BlockRef, error)
	LastL1(ctx context.Context) (eth.BlockRef, error)
	VerifiedToTimestamp() (uint64, error)
	VerifiedToTimestampWithL1(ctx context.Context, l1BlockNum uint64) (uint64, error)
}

type virtualNodeFactory func(cfg *opnodecfg.Config, log gethlog.Logger, initOverrides *rollupNode.InitializationOverrides, appVersion string) virtual_node.VirtualNode

type simpleChainContainer struct {
	vn                 virtual_node.VirtualNode
	vncfg              *opnodecfg.Config
	cfg                config.CLIConfig
	engine             engine_controller.EngineController
	pause              atomic.Bool
	stop               atomic.Bool
	stopped            chan struct{}
	log                gethlog.Logger
	chainID            eth.ChainID
	initOverload       *rollupNode.InitializationOverrides  // Base shared resources for all virtual nodes
	rpcHandler         *oprpc.Handler                       // Current per-chain RPC handler instance
	setHandler         func(chainID string, h http.Handler) // Set the RPC handler on the router for the chain
	setMetricsHandler  func(chainID string, h http.Handler) // Set the metrics handler on the router for the chain
	appVersion         string
	virtualNodeFactory virtualNodeFactory // Factory function to create virtual node (for testing)
}

func NewChainContainer(
	chainID eth.ChainID,
	vncfg *opnodecfg.Config,
	log gethlog.Logger,
	cfg config.CLIConfig,
	initOverload *rollupNode.InitializationOverrides,
	rpcHandler *oprpc.Handler,
	setHandler func(chainID string, h http.Handler),
	setMetricsHandler func(chainID string, h http.Handler),
) ChainContainer {
	c := &simpleChainContainer{
		vncfg:              vncfg,
		cfg:                cfg,
		chainID:            chainID,
		log:                log,
		stopped:            make(chan struct{}, 1),
		initOverload:       initOverload,
		rpcHandler:         rpcHandler,
		setHandler:         setHandler,
		setMetricsHandler:  setMetricsHandler,
		appVersion:         virtualNodeVersion,
		virtualNodeFactory: defaultVirtualNodeFactory,
	}
	// Disable P2P and inherit paths from supernode base config
	vncfg.P2P = &p2p.Config{DisableP2P: true}
	vncfg.SafeDBPath = c.subPath("safe_db")
	vncfg.RPC = cfg.RPCConfig
	// Initialize engine controller (separate connection, not an op-node override) with a short setup timeout
	if vncfg.L2 != nil {
		setupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// Provide contextual logger to engine controller
		engLog := log.New("chain_id", chainID.String(), "component", "engine_controller")
		if eng, err := engine_controller.NewEngineControllerFromConfig(setupCtx, engLog, vncfg); err != nil {
			log.Error("failed to setup engine controller", "err", err)
		} else {
			c.engine = eng
		}
	}
	return c
}

// defaultVirtualNodeFactory is the default factory that creates a real VirtualNode
func defaultVirtualNodeFactory(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides, appVersion string) virtual_node.VirtualNode {
	return virtual_node.NewVirtualNode(cfg, log, initOverload, appVersion)
}

func (c *simpleChainContainer) subPath(path string) string {
	return filepath.Join(c.cfg.DataDir, c.chainID.String(), path)
}

func (c *simpleChainContainer) Start(ctx context.Context) error {
	defer func() { c.stopped <- struct{}{} }()
	for {
		// Refresh per-start derived fields
		c.vncfg.P2P = &p2p.Config{DisableP2P: true}
		c.vncfg.SafeDBPath = c.subPath("safe_db")
		c.vncfg.RPC = c.cfg.RPCConfig
		// create a fresh handler per (re)start, swap it into the router, and inject into overload
		h := oprpc.NewHandler("", oprpc.WithLogger(c.log.New("chain_id", c.chainID.String())))
		if c.setHandler != nil {
			c.setHandler(c.chainID.String(), h)
		}
		c.initOverload.RPCHandler = h
		c.rpcHandler = h

		// Disable per-VN metrics server and provide metrics registry hook
		c.vncfg.Metrics.Enabled = false
		if c.initOverload != nil {
			chainID := c.chainID.String()
			c.initOverload.MetricsRegistry = func(reg *prometheus.Registry) {
				if c.setMetricsHandler != nil {
					// Mount per-chain metrics handler at /{chain}/metrics via router
					handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
					c.setMetricsHandler(chainID, handler)
				}
			}
		}
		c.vn = c.virtualNodeFactory(c.vncfg, c.log, c.initOverload, c.appVersion)
		if c.pause.Load() {
			c.log.Info("chain container paused")
			time.Sleep(1 * time.Second)
			continue
		}
		if c.stop.Load() {
			break
		}

		// start the virtual node
		err := c.vn.Start(ctx)
		if err != nil {
			c.log.Warn("virtual node exited with error", "error", err)
		}

		// always stop the virtual node after it exits
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if stopErr := c.vn.Stop(stopCtx); stopErr != nil {
			c.log.Error("error stopping virtual node", "error", stopErr)
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

	}
	c.log.Info("chain container exiting")
	return nil
}

func (c *simpleChainContainer) Stop(ctx context.Context) error {
	c.stop.Store(true)
	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if c.vn != nil {
		if err := c.vn.Stop(stopCtx); err != nil {
			c.log.Error("error stopping virtual node", "error", err)
		}
	}

	// Close engine controller RPC resources
	if c.engine != nil {
		_ = c.engine.Close()
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

// SafeAtTimestamp checks if the block at the given timestamp is safe according to the virtual node.
func (c *simpleChainContainer) SafeAtTimestamp(ctx context.Context, ts uint64) (bool, error) {
	if c.vn == nil {
		return false, nil
	}
	safeTs, err := c.vn.SafeTimestamp(ctx)
	if err != nil {
		return false, err
	}
	return safeTs >= ts, nil
}

// VerifiedAtTimestamp checks if the block at the given timestamp is safe according to the virtual node.
// And also has been verified by all Verification Activities of the Supernode.
func (c *simpleChainContainer) VerifiedAtTimestamp(ctx context.Context, ts uint64) (bool, error) {
	// there are currently no verification activities, so we just check if the block is safe
	return c.SafeAtTimestamp(ctx, ts)
}

// BlockAtTimestamp returns the highest L2 block with timestamp <= ts using the L2 client.
func (c *simpleChainContainer) BlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	if c.engine == nil {
		return eth.L2BlockRef{}, engine_controller.ErrNoEngineClient
	}
	return c.engine.BlockAtTimestamp(ctx, ts)
}

// OutputRootAtTimestamp computes the output root for the L2 block at or before the given timestamp.
func (c *simpleChainContainer) OutputRootAtL1(ctx context.Context, l1BlockNum uint64) (eth.Bytes32, error) {
	if c.engine == nil {
		return eth.Bytes32{}, engine_controller.ErrNoEngineClient
	}
	// Use SafeDB mapping to find the L2 safe head corresponding to (or before) the given L1 block number
	if c.vn == nil {
		return eth.Bytes32{}, fmt.Errorf("virtual node not initialized")
	}
	if c.log != nil {
		c.log.Debug("OutputRootAtL1: query SafeHeadAtL1", "l1_block", l1BlockNum)
	}
	_, l2SafeHead, err := c.vn.SafeHeadAtL1(ctx, l1BlockNum)
	if err != nil {
		if c.log != nil {
			c.log.Warn("OutputRootAtL1: SafeHeadAtL1 failed", "l1_block", l1BlockNum, "err", err)
		}
		return eth.Bytes32{}, err
	}
	if c.log != nil {
		c.log.Debug("OutputRootAtL1: resolved L2 safe head", "l1_block", l1BlockNum, "l2_block", l2SafeHead.Number)
	}
	out, err := c.engine.OutputV0AtBlockNumber(ctx, l2SafeHead.Number)
	if err != nil {
		if c.log != nil {
			c.log.Warn("OutputRootAtL1: engine output fetch failed", "l2_block", l2SafeHead.Number, "err", err)
		}
		return eth.Bytes32{}, err
	}
	if c.log != nil {
		c.log.Debug("OutputRootAtL1: got output root", "l2_block", l2SafeHead.Number)
	}
	return eth.OutputRoot(out), nil
}

// Interface conformance assertions
var _ ChainContainer = (*simpleChainContainer)(nil)

// SafeHeadAtL1 queries the embedded op-node RPC handler for the SafeDB mapping at/preceding the given L1 block number.
func (c *simpleChainContainer) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error) {
	if c.vn == nil {
		return eth.BlockID{}, eth.BlockID{}, fmt.Errorf("virtual node not initialized")
	}
	return c.vn.SafeHeadAtL1(ctx, l1BlockNum)
}

// CurrentL1 returns the most recent processed L1 block reference.
// This uses the SafeDB mapping as a proxy for current processed L1.
func (c *simpleChainContainer) CurrentL1(ctx context.Context) (eth.BlockRef, error) {
	if c.vn == nil {
		if c.log != nil {
			c.log.Warn("CurrentL1: virtual node not initialized")
		}
		return eth.BlockRef{}, nil
	}
	// Avoid overflow in SafeDB implementation which does l1BlockNum+1; use MaxUint64-1 to get the latest entry
	l1, _, err := c.vn.SafeHeadAtL1(ctx, math.MaxUint64-1)
	if err != nil {
		// Gracefully handle SafeDB not being enabled or populated yet.
		if errors.Is(err, safedb.ErrNotEnabled) || errors.Is(err, safedb.ErrNotFound) {
			if c.log != nil {
				c.log.Debug("CurrentL1: SafeDB unavailable", "err", err)
			}
			return eth.BlockRef{}, nil
		}
		return eth.BlockRef{}, err
	}
	// ParentHash and Time may not be available from SafeDB mapping; only Number/Hash are set.
	return eth.BlockRef{Hash: l1.Hash, Number: l1.Number}, nil
}

// LastL1 returns the latest recorded L1 in the SafeDB mapping.
// This does not rely on sync status, just the SafeDB (if enabled and populated).
func (c *simpleChainContainer) LastL1(ctx context.Context) (eth.BlockRef, error) {
	if c.vn == nil {
		return eth.BlockRef{}, fmt.Errorf("virtual node not initialized")
	}
	l1, _, err := c.vn.SafeHeadAtL1(ctx, math.MaxUint64-1)
	if err != nil {
		return eth.BlockRef{}, err
	}
	return eth.BlockRef{Hash: l1.Hash, Number: l1.Number}, nil
}

// VerifiedToTimestamp returns the latest verified timestamp. Currently proxies SafeTimestamp.
func (c *simpleChainContainer) VerifiedToTimestamp() (uint64, error) {
	if c.vn == nil {
		return 0, nil
	}
	return c.vn.SafeTimestamp(context.Background())
}

// VerifiedToTimestampWithL1 returns the latest verified timestamp with L1 context. Currently proxies SafeTimestamp.
func (c *simpleChainContainer) VerifiedToTimestampWithL1(ctx context.Context, l1BlockNum uint64) (uint64, error) {
	if c.vn == nil {
		return 0, fmt.Errorf("virtual node not initialized")
	}
	// Resolve the L2 safe head that became safe at or before this L1 block number
	_, l2SafeHead, err := c.vn.SafeHeadAtL1(ctx, l1BlockNum)
	if err != nil {
		// If SafeDB is unavailable/unpopulated, surface 0 for now
		if errors.Is(err, safedb.ErrNotEnabled) || errors.Is(err, safedb.ErrNotFound) {
			if c.log != nil {
				c.log.Debug("VerifiedToTimestampWithL1: SafeDB unavailable", "l1_block", l1BlockNum, "err", err)
			}
			return 0, nil
		}
		return 0, err
	}
	if c.engine == nil {
		return 0, engine_controller.ErrNoEngineClient
	}
	// Fetch the L2 block ref to read its timestamp
	ref, err := c.engine.L2BlockRefByNumber(ctx, l2SafeHead.Number)
	if err != nil {
		return 0, err
	}
	return ref.Time, nil
}
