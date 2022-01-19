package node

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"

	"github.com/ethereum-optimism/optimistic-specs/opnode/l1"
	"github.com/ethereum-optimism/optimistic-specs/opnode/l2"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum/go-ethereum/rpc"
)

type GenesisConf struct {
	L2Hash common.Hash `ask:"--l2-hash" help:"Genesis block hash of L2"`
	L1Hash common.Hash `ask:"--l1-hash" help:"Block hash of L1 after (not incl.) which L1 starts deriving blocks"`
	L1Num  uint64      `ask:"--l1-num" help:"Block number of L1 matching the l1-hash"`
}

func (conf *GenesisConf) GetGenesis() l2.Genesis {
	return l2.Genesis{
		L1: eth.BlockID{Hash: conf.L1Hash, Number: conf.L1Num},
		// TODO: if we start from a squashed snapshot we might have a non-zero L2 genesis number
		L2: eth.BlockID{Hash: conf.L2Hash, Number: 0},
	}
}

type OpNodeCmd struct {
	L1NodeAddrs   []string `ask:"--l1" help:"Addresses of L1 User JSON-RPC endpoints to use (eth namespace required)"`
	L2EngineAddrs []string `ask:"--l2" help:"Addresses of L2 Engine JSON-RPC endpoints to use (engine and eth namespace required)"`

	LogCmd `ask:".log" help:"Log configuration"`

	Genesis GenesisConf `ask:".genesis" help:"Genesis anchor point"`

	// during later sequencer rollup implementation:
	// TODO: multi-addrs option (static peers)
	// TODO: bootnodes option (bootstrap discovery of more peers)

	log log.Logger

	// (combined) source to fetch data from
	l1Source eth.L1Source

	// engines to keep synced
	l2Engines []*l2.EngineDriver

	l1Downloader l1.Downloader

	ctx   context.Context
	close chan chan error
}

func (c *OpNodeCmd) Default() {
	c.L1NodeAddrs = []string{"http://127.0.0.1:8545"}
	c.L2EngineAddrs = []string{"http://127.0.0.1:8551"}
}

func (c *OpNodeCmd) Help() string {
	return "Run optimism node"
}

func (c *OpNodeCmd) Run(ctx context.Context, args ...string) error {
	logger := c.LogCmd.Create()
	c.log = logger
	c.ctx = ctx

	if c.Genesis == (GenesisConf{}) {
		return errors.New("genesis configuration required")
	}

	l1Sources := make([]eth.L1Source, 0, len(c.L1NodeAddrs))
	for i, addr := range c.L1NodeAddrs {
		// L1 exec engine: read-only, to update L2 consensus with
		l1Node, err := rpc.DialContext(ctx, addr)
		if err != nil {
			// HTTP or WS RPC may create a disconnected client, RPC over IPC may fail directly
			if l1Node == nil {
				return fmt.Errorf("failed to dial L1 address %d (%s): %v", i, addr, err)
			}
			c.log.Warn("failed to dial L1 address, but may connect later", "i", i, "addr", addr, "err", err)
		}
		// TODO: we may need to authenticate the connection with L1
		// l1Node.SetHeader()
		cl := ethclient.NewClient(l1Node)
		l1Sources = append(l1Sources, cl)
	}
	if len(l1Sources) == 0 {
		return fmt.Errorf("need at least one L1 source endpoint, see --l1")
	}

	// Combine L1 sources, so work can be balanced between them
	c.l1Source = eth.NewCombinedL1Source(l1Sources)
	l1CanonicalChain := eth.CanonicalChain(c.l1Source)

	c.l1Downloader = l1.NewDownloader(c.l1Source)

	for i, addr := range c.L2EngineAddrs {
		// L2 exec engine: updated by this OpNode (L2 consensus layer node)
		backend, err := rpc.DialContext(ctx, addr)
		if err != nil {
			if backend == nil {
				return fmt.Errorf("failed to dial L2 address %d (%s): %v", i, addr, err)
			}
			c.log.Warn("failed to dial L2 address, but may connect later", "i", i, "addr", addr, "err", err)
		}
		// TODO: we may need to authenticate the connection with L2
		// backend.SetHeader()
		client := &l2.EngineClient{
			RPCBackend: backend,
			EthBackend: ethclient.NewClient(backend),
			Log:        c.log.New("engine_client", i),
		}
		engine := &l2.EngineDriver{
			Log: c.log.New("engine", i),
			RPC: client,
			SyncRef: l2.SyncSource{
				L1: l1CanonicalChain,
				L2: client,
			},
		}
		c.l2Engines = append(c.l2Engines, engine)
	}

	// TODO: maybe spin up an API server
	//  (to get debug data, change runtime settings like logging, serve pprof, get peering info, node health, etc.)

	c.close = make(chan chan error)

	go c.RunNode()

	return nil
}

func (c *OpNodeCmd) RunNode() {
	c.log.Info("Starting OpNode")

	var unsub []func()
	mergeSub := func(sub ethereum.Subscription, errMsg string) {
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
	c.l1Downloader.AddReceiptWorkers(4)

	// Feed of eth.HeadSignal
	var l1HeadsFeed event.Feed

	c.log.Info("Attaching execution engine(s)")
	for _, eng := range c.l2Engines {
		// Update genesis info, to anchor sync at
		eng.Genesis = c.Genesis.GetGenesis()
		// Request initial head update, default to genesis otherwise
		reqCtx, reqCancel := context.WithTimeout(c.ctx, time.Second*10)
		if err := eng.RequestHeadUpdate(reqCtx); err != nil {
			eng.Log.Error("failed to fetch engine head, defaulting to genesis", "err", err)
			eng.UpdateHead(eng.Genesis.L1, eng.Genesis.L2)
		}
		reqCancel()

		// driver subscribes to L1 head changes
		l1SubCh := make(chan eth.HeadSignal, 10)
		l1HeadsFeed.Subscribe(l1SubCh)
		// start driving engine: sync blocks by deriving them from L1 and driving them into the engine
		engDriveSub := eng.Drive(c.ctx, c.l1Downloader, l1SubCh)
		mergeSub(engDriveSub, "engine driver unexpectedly failed")
	}

	// Keep subscribed to the L1 heads, which keeps the L1 maintainer pointing to the best headers to sync
	l1HeadsSub := event.ResubscribeErr(time.Second*10, func(ctx context.Context, err error) (event.Subscription, error) {
		if err != nil {
			c.log.Warn("resubscribing after failed L1 subscription", "err", err)
		}
		return eth.WatchHeadChanges(c.ctx, c.l1Source, func(sig eth.HeadSignal) {
			l1HeadsFeed.Send(sig)
		})
	})
	mergeSub(l1HeadsSub, "l1 heads subscription failed")

	// subscribe to L1 heads for info
	l1Heads := make(chan eth.HeadSignal, 10)
	l1HeadsFeed.Subscribe(l1Heads)

	c.log.Info("Start-up complete!")

	for {
		select {
		case l1Head := <-l1Heads:
			c.log.Info("New L1 head", "head", l1Head.Self, "parent", l1Head.Parent)
		// TODO: maybe log other info on interval or other chain events (individual engines also log things)
		case done := <-c.close:
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
			// signal back everything closed without error
			done <- nil
			return
		}
	}
}

func (c *OpNodeCmd) Close() error {
	if c.close != nil {
		done := make(chan error)
		c.close <- done
		err := <-done
		return err
	}
	return nil
}
