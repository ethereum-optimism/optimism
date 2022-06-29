package node

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/hashicorp/go-multierror"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/l1"
	"github.com/ethereum-optimism/optimism/op-node/l2"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

type OpNode struct {
	log        log.Logger
	appVersion string
	metrics    *metrics.Metrics
	l1HeadsSub ethereum.Subscription // Subscription to get L1 heads (automatically re-subscribes on error)
	l1Source   *l1.Source            // Source to fetch data from (also implements the Downloader interface)
	l2Engine   *driver.Driver        // L2 Engine to Sync
	l2Node     client.RPC            // L2 Execution Engine RPC connections to close at shutdown
	l2Client   client.Client         // L2 client wrapper around eth namespace
	server     *rpcServer            // RPC server hosting the rollup-node API
	p2pNode    *p2p.NodeP2P          // P2P node functionality
	p2pSigner  p2p.Signer            // p2p gogssip application messages will be signed with this signer
	tracer     Tracer                // tracer to get events for testing/debugging

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
	l1Node, trustRPC, err := cfg.L1.Setup(ctx, n.log)
	if err != nil {
		return fmt.Errorf("failed to get L1 RPC client: %w", err)
	}

	n.l1Source, err = l1.NewSource(client.NewInstrumentedRPC(l1Node, n.metrics), n.log, l1.DefaultConfig(&cfg.Rollup, trustRPC))
	if err != nil {
		return fmt.Errorf("failed to create L1 source: %v", err)
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
	return nil
}

func (n *OpNode) initL2(ctx context.Context, cfg *Config, snapshotLog log.Logger) error {
	rpcClient, err := cfg.L2.Setup(ctx, n.log)
	if err != nil {
		return fmt.Errorf("failed to setup L2 execution-engine RPC client: %w", err)
	}
	n.l2Node = client.NewInstrumentedRPC(rpcClient, n.metrics)
	n.l2Client = client.NewInstrumentedClient(rpcClient, n.metrics)
	source, err := l2.NewSource(n.l2Node, n.l2Client, &cfg.Rollup.Genesis, n.log)
	if err != nil {
		return err
	}

	n.l2Engine = driver.NewDriver(&cfg.Driver, &cfg.Rollup, source, n.l1Source, n, n.log, snapshotLog)

	return nil
}

func (n *OpNode) initRPCServer(ctx context.Context, cfg *Config) error {
	// TODO: attach the p2p node ID to the snapshot logger
	client, err := l2.NewReadOnlySource(n.l2Node, n.l2Client, &cfg.Rollup.Genesis, n.log)
	if err != nil {
		return err
	}
	n.server, err = newRPCServer(ctx, &cfg.RPC, &cfg.Rollup, client, n.l2Engine, n.log, n.appVersion, n.metrics)
	if err != nil {
		return err
	}
	if n.p2pNode != nil {
		n.server.EnableP2P(p2p.NewP2PAPIBackend(n.p2pNode, n.log, n.metrics))
	}
	n.log.Info("Starting JSON-RPC server")
	if err := n.server.Start(); err != nil {
		return fmt.Errorf("unable to start RPC server: %w", err)
	}
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
		p2pNode, err := p2p.NewNodeP2P(n.resourcesCtx, &cfg.Rollup, n.log, cfg.P2P, n)
		if err != nil {
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
	// Request initial head update, default to genesis otherwise
	reqCtx, reqCancel := context.WithTimeout(ctx, time.Second*10)
	// start driving engine: sync blocks by deriving them from L1 and driving them into the engine
	err := n.l2Engine.Start(reqCtx)
	reqCancel()
	if err != nil {
		n.log.Error("Could not start a rollup node", "err", err)
		return err
	}

	return nil
}

func (n *OpNode) OnNewL1Head(ctx context.Context, sig eth.L1BlockRef) {
	n.tracer.OnNewL1Head(ctx, sig)

	// Pass on the event to the L2 Engine
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	if err := n.l2Engine.OnL1Head(ctx, sig); err != nil {
		n.log.Warn("failed to notify engine driver of L1 head change", "err", err)
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
	if err := n.l2Engine.OnUnsafeL2Payload(ctx, payload); err != nil {
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

	// close L2 engine
	if n.l2Engine != nil {
		if err := n.l2Engine.Close(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close L2 engine driver cleanly: %w", err))
		}
	}

	// close L2 node
	if n.l2Node != nil {
		n.l2Node.Close()
	}

	// close L1 data source
	if n.l1Source != nil {
		n.l1Source.Close()
	}
	return result.ErrorOrNil()
}
