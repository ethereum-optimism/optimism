package node

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/l1"
	"github.com/ethereum-optimism/optimistic-specs/opnode/l2"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/driver"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type Config struct {
	// L1 and L2 nodes
	L1NodeAddr    string   // Address of L1 User JSON-RPC endpoint to use (eth namespace required)
	L2EngineAddrs []string // Addresses of L2 Engine JSON-RPC endpoints to use (engine and eth namespace required)

	// Genesis Information
	L2Hash common.Hash // Genesis block hash of L2
	L1Hash common.Hash // Block hash of L1 after (not incl.) which L1 starts deriving blocks
	L1Num  uint64      // Block number of L1 matching the l1-hash
}

// Check verifies that the given configuration makes sense
func (cfg *Config) Check() error {
	return nil
}

type OpNode struct {
	log       log.Logger
	l1Source  l1.Source        // Source to fetch data from (also implements the Downloader interface)
	l2Engines []*driver.Driver // engines to keep synced
	done      chan struct{}
}

func (conf *Config) GetGenesis() rollup.Genesis {
	return rollup.Genesis{
		L1: eth.BlockID{Hash: conf.L1Hash, Number: conf.L1Num},
		// TODO: if we start from a squashed snapshot we might have a non-zero L2 genesis number
		L2: eth.BlockID{Hash: conf.L2Hash, Number: 0},
	}
}

func New(ctx context.Context, cfg *Config, log log.Logger) (*OpNode, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}

	// L1 exec engine: read-only, to update L2 consensus with
	l1Node, err := rpc.DialContext(ctx, cfg.L1NodeAddr)
	if err != nil {
		// HTTP or WS RPC may create a disconnected client, RPC over IPC may fail directly
		if l1Node == nil {
			return nil, fmt.Errorf("failed to dial L1 addres (%s): %v", cfg.L1NodeAddr, err)
		}
		log.Warn("failed to dial L1 address, but may connect later", "addr", cfg.L1NodeAddr, "err", err)
	}

	// TODO: we may need to authenticate the connection with L1
	// l1Node.SetHeader()
	l1Source := l1.NewSource(ethclient.NewClient(l1Node))
	genesis := cfg.GetGenesis()
	var l2Engines []*driver.Driver
	for i, addr := range cfg.L2EngineAddrs {
		// L2 exec engine: updated by this OpNode (L2 consensus layer node)
		backend, err := rpc.DialContext(ctx, addr)
		if err != nil {
			if backend == nil {
				return nil, fmt.Errorf("failed to dial L2 address %d (%s): %v", i, addr, err)
			}
			log.Warn("failed to dial L2 address, but may connect later", "i", i, "addr", addr, "err", err)
		}
		// TODO: we may need to authenticate the connection with L2
		// backend.SetHeader()
		client := &l2.EngineClient{
			RPCBackend: backend,
			EthBackend: ethclient.NewClient(backend),
			Log:        log.New("engine_client", i),
		}
		engine := driver.NewDriver(client, l1Source, log.New("engine", i), genesis)
		l2Engines = append(l2Engines, engine)
	}

	n := &OpNode{
		log:       log,
		l1Source:  l1Source,
		l2Engines: l2Engines,
		done:      make(chan struct{}),
	}

	return n, nil
}

func (c *OpNode) Start(ctx context.Context) error {
	c.log.Info("Starting OpNode")

	var unsub []func()
	handleUnsubscribe := func(sub ethereum.Subscription, errMsg string) {
		unsub = append(unsub, sub.Unsubscribe)
		go func() {
			err, ok := <-sub.Err()
			if !ok {
				return
			}
			c.log.Error(errMsg, "err", err)
		}()
	}

	c.log.Info("Fetching rollup starting point")

	// We download receipts in parallel
	c.l1Source.AddReceiptWorkers(4)

	// Feed of eth.HeadSignal
	var l1HeadsFeed event.Feed

	c.log.Info("Attaching execution engine(s)")
	for _, eng := range c.l2Engines {
		// Request initial head update, default to genesis otherwise
		reqCtx, reqCancel := context.WithTimeout(ctx, time.Second*10)

		// driver subscribes to L1 head changes
		l1SubCh := make(chan eth.HeadSignal, 10)
		l1HeadsFeed.Subscribe(l1SubCh)
		// start driving engine: sync blocks by deriving them from L1 and driving them into the engine
		err := eng.Start(reqCtx, l1SubCh)
		// engDriveSub := eng.Drive(c.ctx, l1SubCh)
		// handleUnsubscribe(engDriveSub, "engine driver unexpectedly failed")
		reqCancel()
		if err != nil {
			c.log.Error("Could not start a rollup node", "err", err)
			return err
		}
	}

	// Keep subscribed to the L1 heads, which keeps the L1 maintainer pointing to the best headers to sync
	l1HeadsSub := event.ResubscribeErr(time.Second*10, func(ctx context.Context, err error) (event.Subscription, error) {
		if err != nil {
			c.log.Warn("resubscribing after failed L1 subscription", "err", err)
		}
		return eth.WatchHeadChanges(ctx, c.l1Source, func(sig eth.HeadSignal) {
			l1HeadsFeed.Send(sig)
		})
	})
	handleUnsubscribe(l1HeadsSub, "l1 heads subscription failed")

	// subscribe to L1 heads for info
	l1Heads := make(chan eth.HeadSignal, 10)
	l1HeadsFeed.Subscribe(l1Heads)

	c.log.Info("Start-up complete!")
	go func() {

		for {
			select {
			case l1Head := <-l1Heads:
				c.log.Info("New L1 head", "head", l1Head.Self, "parent", l1Head.Parent)
			// TODO: maybe log other info on interval or other chain events (individual engines also log things)
			case <-c.done:
				c.log.Info("Closing OpNode")
				// close all tasks
				for _, f := range unsub {
					f()
				}
				// close L1 data source
				c.l1Source.Close()
				// close L2 engines
				for _, eng := range c.l2Engines {
					eng.Close()
				}
				return
			}
		}
	}()
	return nil
}

func (c *OpNode) Stop() {
	if c.done != nil {
		close(c.done)
	}
}
