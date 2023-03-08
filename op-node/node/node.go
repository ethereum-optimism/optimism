package node

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/sources"
)

type OpNode struct {
	log        log.Logger
	appVersion string
	metrics    *metrics.Metrics

	l1HeadsSub     ethereum.Subscription // Subscription to get L1 heads (automatically re-subscribes on error)
	l1SafeSub      ethereum.Subscription // Subscription to get L1 safe blocks, a.k.a. justified data (polling)
	l1FinalizedSub ethereum.Subscription // Subscription to get L1 safe blocks, a.k.a. justified data (polling)

	l1Source  *sources.L1Client     // L1 Client to fetch data from
	l2Driver  *driver.Driver        // L2 Engine to Sync
	l2Source  *sources.EngineClient // L2 Execution Engine RPC bindings
	server    *rpcServer            // RPC server hosting the rollup-node API
	p2pNode   *p2p.NodeP2P          // P2P node functionality
	p2pSigner p2p.Signer            // p2p gogssip application messages will be signed with this signer
	tracer    Tracer                // tracer to get events for testing/debugging
	runCfg    *RuntimeConfig        // runtime configurables

	// some resources cannot be stopped directly, like the p2p gossipsub router (not our design),
	// and depend on this ctx to be closed.
	resourcesCtx   context.Context
	resourcesClose context.CancelFunc
}

// The OpNode handles incoming gossip
var _ p2p.GossipIn = (*OpNode)(nil)

func New(ctx context.Context, cfg *Config, log log.Logger, snapshotLog log.Logger, appVersion string, m *metrics.Metrics) (*OpNode, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}

	n := &OpNode{
		log:        log,
		appVersion: appVersion,
		metrics:    m,
	}
	// not a context leak, gossipsub is closed with a context.
	n.resourcesCtx, n.resourcesClose = context.WithCancel(context.Background())

	err := n.init(ctx, cfg, snapshotLog)
	if err != nil {
		log.Error("Error initializing the rollup node", "err", err)
		// ensure we always close the node resources if we fail to initialize the node.
		if closeErr := n.Close(); closeErr != nil {
			return nil, multierror.Append(err, closeErr)
		}
		return nil, err
	}
	return n, nil
}

func (n *OpNode) init(ctx context.Context, cfg *Config, snapshotLog log.Logger) error {
	if err := n.initTracer(ctx, cfg); err != nil {
		return err
	}
	if err := n.initL1(ctx, cfg); err != nil {
		return err
	}
	if err := n.initRuntimeConfig(ctx, cfg); err != nil {
		return err
	}
	if err := n.initL2(ctx, cfg, snapshotLog); err != nil {
		return err
	}
	if err := n.initP2PSigner(ctx, cfg); err != nil {
		return err
	}
	if err := n.initP2P(ctx, cfg); err != nil {
		return err
	}
	// Only expose the server at the end, ensuring all RPC backend components are initialized.
	if err := n.initRPCServer(ctx, cfg); err != nil {
		return err
	}
	if err := n.initMetricsServer(ctx, cfg); err != nil {
		return err
	}
	return nil
}

func (n *OpNode) initTracer(ctx context.Context, cfg *Config) error {
	if cfg.Tracer != nil {
		n.tracer = cfg.Tracer
	} else {
		n.tracer = new(noOpTracer)
	}
	return nil
}

func (n *OpNode) initL1(ctx context.Context, cfg *Config) error {
	l1Node, trustRPC, rpcProvKind, err := cfg.L1.Setup(ctx, n.log)
	if err != nil {
		return fmt.Errorf("failed to get L1 RPC client: %w", err)
	}

	n.l1Source, err = sources.NewL1Client(
		client.NewInstrumentedRPC(l1Node, n.metrics), n.log, n.metrics.L1SourceCache,
		sources.L1ClientDefaultConfig(&cfg.Rollup, trustRPC, rpcProvKind))
	if err != nil {
		return fmt.Errorf("failed to create L1 source: %w", err)
	}

	if err := cfg.Rollup.ValidateL1Config(ctx, n.l1Source); err != nil {
		return err
	}

	// Keep subscribed to the L1 heads, which keeps the L1 maintainer pointing to the best headers to sync
	n.l1HeadsSub = event.ResubscribeErr(time.Second*10, func(ctx context.Context, err error) (event.Subscription, error) {
		if err != nil {
			n.log.Warn("resubscribing after failed L1 subscription", "err", err)
		}
		return eth.WatchHeadChanges(n.resourcesCtx, n.l1Source, n.OnNewL1Head)
	})
	go func() {
		err, ok := <-n.l1HeadsSub.Err()
		if !ok {
			return
		}
		n.log.Error("l1 heads subscription error", "err", err)
	}()

	// Poll for the safe L1 block and finalized block,
	// which only change once per epoch at most and may be delayed.
	n.l1SafeSub = eth.PollBlockChanges(n.resourcesCtx, n.log, n.l1Source, n.OnNewL1Safe, eth.Safe,
		cfg.L1EpochPollInterval, time.Second*10)
	n.l1FinalizedSub = eth.PollBlockChanges(n.resourcesCtx, n.log, n.l1Source, n.OnNewL1Finalized, eth.Finalized,
		cfg.L1EpochPollInterval, time.Second*10)
	return nil
}

func (n *OpNode) initRuntimeConfig(ctx context.Context, cfg *Config) error {
	// attempt to load runtime config, repeat N times
	n.runCfg = NewRuntimeConfig(n.log, n.l1Source, &cfg.Rollup)

	for i := 0; i < 5; i++ {
		fetchCtx, fetchCancel := context.WithTimeout(ctx, time.Second*10)
		l1Head, err := n.l1Source.L1BlockRefByLabel(fetchCtx, eth.Unsafe)
		fetchCancel()
		if err != nil {
			n.log.Error("failed to fetch L1 head for runtime config initialization", "err", err)
			continue
		}

		fetchCtx, fetchCancel = context.WithTimeout(ctx, time.Second*10)
		err = n.runCfg.Load(fetchCtx, l1Head)
		fetchCancel()
		if err != nil {
			n.log.Error("failed to fetch runtime config data", "err", err)
			continue
		}

		return nil
	}

	return errors.New("failed to load runtime configuration repeatedly")
}

func (n *OpNode) initL2(ctx context.Context, cfg *Config, snapshotLog log.Logger) error {
	rpcClient, err := cfg.L2.Setup(ctx, n.log)
	if err != nil {
		return fmt.Errorf("failed to setup L2 execution-engine RPC client: %w", err)
	}

	n.l2Source, err = sources.NewEngineClient(
		client.NewInstrumentedRPC(rpcClient, n.metrics), n.log, n.metrics.L2SourceCache,
		sources.EngineClientDefaultConfig(&cfg.Rollup),
	)
	if err != nil {
		return fmt.Errorf("failed to create Engine client: %w", err)
	}

	if err := cfg.Rollup.ValidateL2Config(ctx, n.l2Source); err != nil {
		return err
	}

	var syncClient *sources.SyncClient
	// If the L2 sync config is present, use it to create a sync client
	if cfg.L2Sync != nil {
		if err := cfg.L2Sync.Check(); err != nil {
			log.Info("L2 sync config is not present, skipping L2 sync client setup", "err", err)
		} else {
			rpcSyncClient, err := cfg.L2Sync.Setup(ctx, n.log)
			if err != nil {
				return fmt.Errorf("failed to setup L2 execution-engine RPC client for backup sync: %w", err)
			}

			// The sync client's RPC is always trusted
			config := sources.SyncClientDefaultConfig(&cfg.Rollup, true)

			syncClient, err = sources.NewSyncClient(n.OnUnsafeL2Payload, rpcSyncClient, n.log, n.metrics.L2SourceCache, config)
			if err != nil {
				return fmt.Errorf("failed to create sync client: %w", err)
			}
		}
	}

	n.l2Driver = driver.NewDriver(&cfg.Driver, &cfg.Rollup, n.l2Source, n.l1Source, syncClient, n, n.log, snapshotLog, n.metrics)

	return nil
}

func (n *OpNode) initRPCServer(ctx context.Context, cfg *Config) error {
	server, err := newRPCServer(ctx, &cfg.RPC, &cfg.Rollup, n.l2Source.L2Client, n.l2Driver, n.log, n.appVersion, n.metrics)
	if err != nil {
		return err
	}
	if n.p2pNode != nil {
		server.EnableP2P(p2p.NewP2PAPIBackend(n.p2pNode, n.log, n.metrics))
	}
	if cfg.RPC.EnableAdmin {
		server.EnableAdminAPI(NewAdminAPI(n.l2Driver, n.metrics))
		n.log.Info("Admin RPC enabled")
	}
	n.log.Info("Starting JSON-RPC server")
	if err := server.Start(); err != nil {
		return fmt.Errorf("unable to start RPC server: %w", err)
	}
	n.server = server
	return nil
}

func (n *OpNode) initMetricsServer(ctx context.Context, cfg *Config) error {
	if !cfg.Metrics.Enabled {
		n.log.Info("metrics disabled")
		return nil
	}
	n.log.Info("starting metrics server", "addr", cfg.Metrics.ListenAddr, "port", cfg.Metrics.ListenPort)
	go func() {
		if err := n.metrics.Serve(ctx, cfg.Metrics.ListenAddr, cfg.Metrics.ListenPort); err != nil {
			log.Crit("error starting metrics server", "err", err)
		}
	}()
	return nil
}

func (n *OpNode) initP2P(ctx context.Context, cfg *Config) error {
	if cfg.P2P != nil {
		p2pNode, err := p2p.NewNodeP2P(n.resourcesCtx, &cfg.Rollup, n.log, cfg.P2P, n, n.runCfg, n.metrics)
		if err != nil || p2pNode == nil {
			return err
		}
		n.p2pNode = p2pNode
		if n.p2pNode.Dv5Udp() != nil {
			go n.p2pNode.DiscoveryProcess(n.resourcesCtx, n.log, &cfg.Rollup, cfg.P2P.TargetPeers())
		}
	}
	return nil
}

func (n *OpNode) initP2PSigner(ctx context.Context, cfg *Config) error {
	// the p2p signer setup is optional
	if cfg.P2PSigner == nil {
		return nil
	}
	// p2pSigner may still be nil, the signer setup may not create any signer, the signer is optional
	var err error
	n.p2pSigner, err = cfg.P2PSigner.SetupSigner(ctx)
	return err
}

func (n *OpNode) Start(ctx context.Context) error {
	n.log.Info("Starting execution engine driver")

	// start driving engine: sync blocks by deriving them from L1 and driving them into the engine
	if err := n.l2Driver.Start(); err != nil {
		n.log.Error("Could not start a rollup node", "err", err)
		return err
	}

	// If the backup unsafe sync client is enabled, start its event loop
	if n.l2Driver.L2SyncCl != nil {
		if err := n.l2Driver.L2SyncCl.Start(); err != nil {
			n.log.Error("Could not start the backup sync client", "err", err)
			return err
		}
	}

	return nil
}

func (n *OpNode) OnNewL1Head(ctx context.Context, sig eth.L1BlockRef) {
	n.tracer.OnNewL1Head(ctx, sig)

	if n.l2Driver == nil {
		return
	}
	// Pass on the event to the L2 Engine
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	if err := n.l2Driver.OnL1Head(ctx, sig); err != nil {
		n.log.Warn("failed to notify engine driver of L1 head change", "err", err)
	}
}

func (n *OpNode) OnNewL1Safe(ctx context.Context, sig eth.L1BlockRef) {
	if n.l2Driver == nil {
		return
	}
	// Pass on the event to the L2 Engine
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	if err := n.l2Driver.OnL1Safe(ctx, sig); err != nil {
		n.log.Warn("failed to notify engine driver of L1 safe block change", "err", err)
	}
}

func (n *OpNode) OnNewL1Finalized(ctx context.Context, sig eth.L1BlockRef) {
	if n.l2Driver == nil {
		return
	}
	// Pass on the event to the L2 Engine
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	if err := n.l2Driver.OnL1Finalized(ctx, sig); err != nil {
		n.log.Warn("failed to notify engine driver of L1 finalized block change", "err", err)
	}
}

func (n *OpNode) PublishL2Payload(ctx context.Context, payload *eth.ExecutionPayload) error {
	n.tracer.OnPublishL2Payload(ctx, payload)

	// publish to p2p, if we are running p2p at all
	if n.p2pNode != nil {
		if n.p2pSigner == nil {
			return fmt.Errorf("node has no p2p signer, payload %s cannot be published", payload.ID())
		}
		n.log.Info("Publishing signed execution payload on p2p", "id", payload.ID())
		return n.p2pNode.GossipOut().PublishL2Payload(ctx, payload, n.p2pSigner)
	}
	// if p2p is not enabled then we just don't publish the payload
	return nil
}

func (n *OpNode) OnUnsafeL2Payload(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) error {
	// ignore if it's from ourselves
	if n.p2pNode != nil && from == n.p2pNode.Host().ID() {
		return nil
	}

	n.tracer.OnUnsafeL2Payload(ctx, from, payload)

	n.log.Info("Received signed execution payload from p2p", "id", payload.ID(), "peer", from)

	// Pass on the event to the L2 Engine
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	if err := n.l2Driver.OnUnsafeL2Payload(ctx, payload); err != nil {
		n.log.Warn("failed to notify engine driver of new L2 payload", "err", err, "id", payload.ID())
	}

	return nil
}

func (n *OpNode) P2P() p2p.Node {
	return n.p2pNode
}

// Close closes all resources.
func (n *OpNode) Close() error {
	var result *multierror.Error

	if n.server != nil {
		n.server.Stop()
	}
	if n.p2pNode != nil {
		if err := n.p2pNode.Close(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close p2p node: %w", err))
		}
	}
	if n.p2pSigner != nil {
		if err := n.p2pSigner.Close(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close p2p signer: %w", err))
		}
	}

	if n.resourcesClose != nil {
		n.resourcesClose()
	}

	// stop L1 heads feed
	if n.l1HeadsSub != nil {
		n.l1HeadsSub.Unsubscribe()
	}

	// close L2 driver
	if n.l2Driver != nil {
		if err := n.l2Driver.Close(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close L2 engine driver cleanly: %w", err))
		}

		// If the L2 sync client is present & running, close it.
		if n.l2Driver.L2SyncCl != nil {
			if err := n.l2Driver.L2SyncCl.Close(); err != nil {
				result = multierror.Append(result, fmt.Errorf("failed to close L2 engine backup sync client cleanly: %w", err))
			}
		}
	}

	// close L2 engine RPC client
	if n.l2Source != nil {
		n.l2Source.Close()
	}

	// close L1 data source
	if n.l1Source != nil {
		n.l1Source.Close()
	}
	return result.ErrorOrNil()
}

func (n *OpNode) ListenAddr() string {
	return n.server.listenAddr.String()
}

func (n *OpNode) HTTPEndpoint() string {
	return fmt.Sprintf("http://%s", n.ListenAddr())
}
