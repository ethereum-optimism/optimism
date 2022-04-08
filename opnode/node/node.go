package node

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/backoff"

	"github.com/ethereum-optimism/optimistic-specs/opnode/bss"
	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/l1"
	"github.com/ethereum-optimism/optimistic-specs/opnode/l2"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/driver"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type OpNode struct {
	log       log.Logger
	l1Source  *l1.Source       // Source to fetch data from (also implements the Downloader interface)
	l2Engines []*driver.Driver // engines to keep synced
	l2Nodes   []*rpc.Client    // L2 Execution Engines to close at shutdown
	server    *rpcServer
	done      chan struct{}
	wg        sync.WaitGroup
}

func dialRPCClientWithBackoff(ctx context.Context, log log.Logger, addr string) (*rpc.Client, error) {
	bOff := backoff.Exponential()
	var ret *rpc.Client
	err := backoff.Do(10, bOff, func() error {
		client, err := rpc.DialContext(ctx, addr)
		if err != nil {
			if client == nil {
				return fmt.Errorf("failed to dial address (%s): %w", addr, err)
			}
			log.Warn("failed to dial address, but may connect later", "addr", addr, "err", err)
		}
		ret = client
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func New(ctx context.Context, cfg *Config, log log.Logger, appVersion string) (*OpNode, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}

	l1Node, err := dialRPCClientWithBackoff(ctx, log, cfg.L1NodeAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial L1 address (%s): %w", cfg.L1NodeAddr, err)
	}

	// TODO: we may need to authenticate the connection with L1
	// l1Node.SetHeader()
	l1Source, err := l1.NewSource(l1Node, log, l1.DefaultConfig(&cfg.Rollup, cfg.L1TrustRPC))
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 source: %v", err)
	}
	var l2Engines []*driver.Driver
	genesis := cfg.Rollup.Genesis

	var l2Nodes []*rpc.Client
	closeNodes := func() {
		for _, n := range l2Nodes {
			n.Close()
		}
	}

	for i, addr := range cfg.L2EngineAddrs {
		l2Node, err := dialRPCClientWithBackoff(ctx, log, addr)
		if err != nil {
			closeNodes()
			return nil, err
		}
		l2Nodes = append(l2Nodes, l2Node)

		// TODO: we may need to authenticate the connection with L2
		// backend.SetHeader()
		client, err := l2.NewSource(l2Node, &genesis, log.New("engine_client", i))
		if err != nil {
			closeNodes()
			return nil, err
		}

		var submitter *bss.BatchSubmitter
		if cfg.Sequencer {
			submitter = &bss.BatchSubmitter{
				Client:    ethclient.NewClient(l1Node),
				ToAddress: cfg.Rollup.BatchInboxAddress,
				ChainID:   cfg.Rollup.L1ChainID,
				PrivKey:   cfg.SubmitterPrivKey,
			}
		}
		engine := driver.NewDriver(cfg.Rollup, client, l1Source, log.New("engine", i, "Sequencer", cfg.Sequencer), submitter, cfg.Sequencer)
		l2Engines = append(l2Engines, engine)
	}

	l2Node, err := dialRPCClientWithBackoff(ctx, log, cfg.L2NodeAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2 address (%s): %w", cfg.L2NodeAddr, err)
	}
	server, err := newRPCServer(ctx, cfg.RPCListenAddr, cfg.RPCListenPort, &l2EthClientImpl{l2Node}, cfg.WithdrawalContractAddr, log, appVersion)
	if err != nil {
		return nil, err
	}

	n := &OpNode{
		log:       log,
		l1Source:  l1Source,
		l2Engines: l2Engines,
		server:    server,
		done:      make(chan struct{}),
		l2Nodes:   l2Nodes,
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

	// Feed of eth.L1BlockRef
	var l1HeadsFeed event.Feed

	c.log.Info("Attaching execution engine(s)")
	for _, eng := range c.l2Engines {
		// Request initial head update, default to genesis otherwise
		reqCtx, reqCancel := context.WithTimeout(ctx, time.Second*10)

		// driver subscribes to L1 head changes
		l1SubCh := make(chan eth.L1BlockRef, 10)
		l1HeadsFeed.Subscribe(l1SubCh)
		// start driving engine: sync blocks by deriving them from L1 and driving them into the engine
		err := eng.Start(reqCtx, l1SubCh)
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
		return eth.WatchHeadChanges(context.Background(), c.l1Source, func(sig eth.L1BlockRef) {
			l1HeadsFeed.Send(sig)
		})
	})
	handleUnsubscribe(l1HeadsSub, "l1 heads subscription failed")

	// subscribe to L1 heads for info
	l1Heads := make(chan eth.L1BlockRef, 10)
	l1HeadsFeed.Subscribe(l1Heads)

	c.log.Info("Starting JSON-RPC server")
	if err := c.server.Start(); err != nil {
		return fmt.Errorf("unable to start RPC server: %w", err)
	}

	c.log.Info("Start-up complete!")
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case l1Head := <-l1Heads:
				c.log.Info("New L1 head", "head", l1Head, "parent", l1Head.ParentHash)
			// TODO: maybe log other info on interval or other chain events (individual engines also log things)
			case <-c.done:
				c.log.Info("Closing OpNode")
				// close all tasks
				for _, f := range unsub {
					f()
				}
				// close L2 engines
				for _, eng := range c.l2Engines {
					eng.Close()
				}
				// close L2 nodes
				for _, n := range c.l2Nodes {
					n.Close()
				}
				// close L1 data source
				c.l1Source.Close()

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
	c.wg.Wait()
	c.server.Stop()
}
