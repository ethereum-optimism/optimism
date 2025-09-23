package chain_container

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"time"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/types"
	gethlog "github.com/ethereum/go-ethereum/log"
)

type ChainContainer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Pause(ctx context.Context) error
	Resume(ctx context.Context) error
}

type simpleChainContainer struct {
	vn      virtual_node.VirtualNode
	vncfg   *opnodecfg.Config
	cfg     config.CLIConfig
	pause   atomic.Bool
	stop    atomic.Bool
	stopped chan struct{}
	log     gethlog.Logger
	chainID types.ChainID
}

func NewChainContainer(
	chainID types.ChainID,
	vncfg *opnodecfg.Config,
	log gethlog.Logger,
	cfg config.CLIConfig) ChainContainer {
	c := &simpleChainContainer{
		vncfg:   vncfg,
		cfg:     cfg,
		chainID: chainID,
		log:     log,
		stopped: make(chan struct{}, 1),
	}
	// TODO: Enable P2P for Virtual Nodes
	// (can be delayed assuming lite-node operates unsafe)
	vncfg.P2P = &p2p.Config{
		DisableP2P: true,
	}
	vncfg.SafeDBPath = c.subPath("safe_db")
	return c
}

func (c *simpleChainContainer) subPath(path string) string {
	return filepath.Join(c.cfg.DataDir, c.chainID.String(), path)
}

func (c *simpleChainContainer) Start(ctx context.Context) error {
	// when Start exits, signal that the chain container is stopped
	defer func() { c.stopped <- struct{}{} }()
	for {
		// initialize the virtual node
		c.vn = virtual_node.NewVirtualNode(c.vncfg, c.log)
		if c.pause.Load() {
			c.log.Info("chain container paused")
			time.Sleep(1 * time.Second)
			continue
		}
		if c.stop.Load() {
			break
		}
		err := c.vn.Start(ctx)
		if err != nil {
			c.log.Warn("virtual node exited", "error", err)
		}
	}
	c.log.Info("chain container Start function exiting")
	return nil
}

func (c *simpleChainContainer) Stop(ctx context.Context) error {
	c.stop.Store(true)
	c.log.Info("chain container stopping")
	err := c.vn.Stop(ctx)
	if err != nil {
		c.log.Error("error stopping virtual node", "error", err)
	}
	// wait for the Start loop to truly exit
	<-c.stopped
	return nil
}

func (c *simpleChainContainer) Pause(ctx context.Context) error {
	c.pause.Store(true)
	return nil
}

func (c *simpleChainContainer) Resume(ctx context.Context) error {
	c.pause.Store(false)
	return nil
}
