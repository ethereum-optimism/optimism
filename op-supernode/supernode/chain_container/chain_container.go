package chain_container

import (
	"context"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"time"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const virtualNodeVersion = "0.0.0"

type ChainContainer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Pause(ctx context.Context) error
	Resume(ctx context.Context) error
}

type virtualNodeFactory func(cfg *opnodecfg.Config, log gethlog.Logger, initOverrides *rollupNode.InitializationOverrides, appVersion string) virtual_node.VirtualNode

type simpleChainContainer struct {
	vn                 virtual_node.VirtualNode
	vncfg              *opnodecfg.Config
	cfg                config.CLIConfig
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
