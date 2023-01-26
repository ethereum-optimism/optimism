package node

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
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
	rpcSync   *sources.SyncClient   // Alt-sync RPC client, optional (may be nil)
	server    *rpcServer            // RPC server hosting the rollup-node API
	p2pNode   *p2p.NodeP2P          // P2P node functionality
	p2pSigner p2p.Signer            // p2p gogssip application messages will be signed with this signer
	tracer    Tracer                // tracer to get events for testing/debugging
	runCfg    *RuntimeConfig        // runtime configurables

	rollupHalt string // when to halt the rollup, disabled if empty

	beacon *sources.L1BeaconClient

	// some resources cannot be stopped directly, like the p2p gossipsub router (not our design),
	// and depend on this ctx to be closed.
	resourcesCtx   context.Context
	resourcesClose context.CancelFunc

	// Indicates when it's safe to close data sources used by the runtimeConfig bg loader
	runtimeConfigReloaderDone chan struct{}

	closed atomic.Bool
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
		rollupHalt: cfg.RollupHalt,
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
		return fmt.Errorf("failed to init the trace: %w", err)
	}
	if err := n.initL1(ctx, cfg); err != nil {
		return fmt.Errorf("failed to init L1: %w", err)
	}
	if err := n.initL1BeaconAPI(ctx, cfg); err != nil {
		return err
	}
	if err := n.initL2(ctx, cfg, snapshotLog); err != nil {
		return fmt.Errorf("failed to init L2: %w", err)
	}
	if err := n.initRuntimeConfig(ctx, cfg); err != nil { // depends on L2, to signal initial runtime values to
		return fmt.Errorf("failed to init the runtime config: %w", err)
	}
	if err := n.initRPCSync(ctx, cfg); err != nil {
		return fmt.Errorf("failed to init RPC sync: %w", err)
	}
	if err := n.initP2PSigner(ctx, cfg); err != nil {
		return fmt.Errorf("failed to init the P2P signer: %w", err)
	}
	if err := n.initP2P(ctx, cfg); err != nil {
		return fmt.Errorf("failed to init the P2P stack: %w", err)
	}
	// Only expose the server at the end, ensuring all RPC backend components are initialized.
	if err := n.initRPCServer(ctx, cfg); err != nil {
		return fmt.Errorf("failed to init the RPC server: %w", err)
	}
	if err := n.initMetricsServer(ctx, cfg); err != nil {
		return fmt.Errorf("failed to init the metrics server: %w", err)
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
	l1Node, rpcCfg, err := cfg.L1.Setup(ctx, n.log, &cfg.Rollup)
	if err != nil {
		return fmt.Errorf("failed to get L1 RPC client: %w", err)
	}

	n.l1Source, err = sources.NewL1Client(
		client.NewInstrumentedRPC(l1Node, n.metrics), n.log, n.metrics.L1SourceCache, rpcCfg)
	if err != nil {
		return fmt.Errorf("failed to create L1 source: %w", err)
	}

	if err := cfg.Rollup.ValidateL1Config(ctx, n.l1Source); err != nil {
		return fmt.Errorf("failed to validate the L1 config: %w", err)
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

	confDepth := cfg.Driver.VerifierConfDepth
	reload := func(ctx context.Context) (eth.L1BlockRef, error) {
		fetchCtx, fetchCancel := context.WithTimeout(ctx, time.Second*10)
		l1Head, err := n.l1Source.L1BlockRefByLabel(fetchCtx, eth.Unsafe)
		fetchCancel()
		if err != nil {
			n.log.Error("failed to fetch L1 head for runtime config initialization", "err", err)
			return eth.L1BlockRef{}, err
		}

		// Apply confirmation-distance
		blNum := l1Head.Number
		if blNum >= confDepth {
			blNum -= confDepth
		}
		fetchCtx, fetchCancel = context.WithTimeout(ctx, time.Second*10)
		confirmed, err := n.l1Source.L1BlockRefByNumber(fetchCtx, blNum)
		fetchCancel()
		if err != nil {
			n.log.Error("failed to fetch confirmed L1 block for runtime config loading", "err", err, "number", blNum)
			return eth.L1BlockRef{}, err
		}

		fetchCtx, fetchCancel = context.WithTimeout(ctx, time.Second*10)
		err = n.runCfg.Load(fetchCtx, confirmed)
		fetchCancel()
		if err != nil {
			n.log.Error("failed to fetch runtime config data", "err", err)
			return l1Head, err
		}

		err = n.handleProtocolVersionsUpdate(ctx)
		return l1Head, err
	}

	// initialize the runtime config before unblocking
	if _, err := retry.Do(ctx, 5, retry.Fixed(time.Second*10), func() (eth.L1BlockRef, error) {
		return reload(ctx)
	}); err != nil {
		return fmt.Errorf("failed to load runtime configuration repeatedly, last error: %w", err)
	}

	// start a background loop, to keep reloading it at the configured reload interval
	reloader := func(ctx context.Context, reloadInterval time.Duration) bool {
		if reloadInterval <= 0 {
			n.log.Debug("not running runtime-config reloading background loop")
			return false
		}
		ticker := time.NewTicker(reloadInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// If the reload fails, we will try again the next interval.
				// Missing a runtime-config update is not critical, and we do not want to overwhelm the L1 RPC.
				l1Head, err := reload(ctx)
				switch err {
				case errNodeHalt, nil:
					n.log.Debug("reloaded runtime config", "l1_head", l1Head)
					if err == errNodeHalt {
						return true
					}
				default:
					n.log.Warn("failed to reload runtime config", "err", err)
				}
			case <-ctx.Done():
				return false
			}
		}
	}

	n.runtimeConfigReloaderDone = make(chan struct{})
	// Manages the lifetime of reloader. In order to safely Close the OpNode
	go func(ctx context.Context, reloadInterval time.Duration) {
		halt := reloader(ctx, reloadInterval)
		close(n.runtimeConfigReloaderDone)
		if halt {
			if err := n.Close(); err != nil {
				n.log.Error("Failed to halt rollup", "err", err)
			}
		}
	}(n.resourcesCtx, cfg.RuntimeConfigReloadInterval) // this keeps running after initialization
	return nil
}

func (n *OpNode) initL1BeaconAPI(ctx context.Context, cfg *Config) error {
	if cfg.Beacon == nil {
		return nil
	}
	httpClient, err := cfg.Beacon.Setup(ctx, n.log)
	if err != nil {
		return fmt.Errorf("failed to setup L2 execution-engine RPC client: %w", err)
	}

	// TODO: wrap http client with metrics
	cl := sources.NewL1BeaconClient(httpClient)
	n.beacon = cl

	return nil
}

func (n *OpNode) initL2(ctx context.Context, cfg *Config, snapshotLog log.Logger) error {
	rpcClient, rpcCfg, err := cfg.L2.Setup(ctx, n.log, &cfg.Rollup)
	if err != nil {
		return fmt.Errorf("failed to setup L2 execution-engine RPC client: %w", err)
	}

	n.l2Source, err = sources.NewEngineClient(
		client.NewInstrumentedRPC(rpcClient, n.metrics), n.log, n.metrics.L2SourceCache, rpcCfg,
	)
	if err != nil {
		return fmt.Errorf("failed to create Engine client: %w", err)
	}

	if err := cfg.Rollup.ValidateL2Config(ctx, n.l2Source); err != nil {
		return err
	}

	n.l2Driver = driver.NewDriver(&cfg.Driver, &cfg.Rollup, n.l2Source, n.l1Source, n.beacon, n, n, n.log, snapshotLog, n.metrics, cfg.ConfigPersistence, &cfg.Sync)

	return nil
}

func (n *OpNode) initRPCSync(ctx context.Context, cfg *Config) error {
	rpcSyncClient, rpcCfg, err := cfg.L2Sync.Setup(ctx, n.log, &cfg.Rollup)
	if err != nil {
		return fmt.Errorf("failed to setup L2 execution-engine RPC client for backup sync: %w", err)
	}
	if rpcSyncClient == nil { // if no RPC client is configured to sync from, then don't add the RPC sync client
		return nil
	}
	syncClient, err := sources.NewSyncClient(n.OnUnsafeL2Payload, rpcSyncClient, n.log, n.metrics.L2SourceCache, rpcCfg)
	if err != nil {
		return fmt.Errorf("failed to create sync client: %w", err)
	}
	n.rpcSync = syncClient
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
		server.EnableAdminAPI(NewAdminAPI(n.l2Driver, n.metrics, n.log))
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
		p2pNode, err := p2p.NewNodeP2P(n.resourcesCtx, &cfg.Rollup, n.log, cfg.P2P, n, n.l2Source, n.runCfg, n.metrics)
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
	if n.rpcSync != nil {
		if err := n.rpcSync.Start(); err != nil {
			n.log.Error("Could not start the backup sync client", "err", err)
			return err
		}
		n.log.Info("Started L2-RPC sync service")
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

func (n *OpNode) RequestL2Range(ctx context.Context, start, end eth.L2BlockRef) error {
	if n.rpcSync != nil {
		return n.rpcSync.RequestL2Range(ctx, start, end)
	}
	if n.p2pNode != nil && n.p2pNode.AltSyncEnabled() {
		if unixTimeStale(start.Time, 12*time.Hour) {
			n.log.Debug("ignoring request to sync L2 range, timestamp is too old for p2p", "start", start, "end", end, "start_time", start.Time)
			return nil
		}
		return n.p2pNode.RequestL2Range(ctx, start, end)
	}
	n.log.Debug("ignoring request to sync L2 range, no sync method available", "start", start, "end", end)
	return nil
}

// unixTimeStale returns true if the unix timestamp is before the current time minus the supplied duration.
func unixTimeStale(timestamp uint64, duration time.Duration) bool {
	return time.Unix(int64(timestamp), 0).Before(time.Now().Add(-1 * duration))
}

func (n *OpNode) P2P() p2p.Node {
	return n.p2pNode
}

func (n *OpNode) RuntimeConfig() ReadonlyRuntimeConfig {
	return n.runCfg
}

// Close closes all resources.
func (n *OpNode) Close() error {
	if n.closed.Load() {
		return errors.New("node is already closed")
	}

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
		if n.rpcSync != nil {
			if err := n.rpcSync.Close(); err != nil {
				result = multierror.Append(result, fmt.Errorf("failed to close L2 engine backup sync client cleanly: %w", err))
			}
		}
	}

	// Wait for the runtime config loader to be done using the data sources before closing them
	if n.runtimeConfigReloaderDone != nil {
		<-n.runtimeConfigReloaderDone
	}

	// close L2 engine RPC client
	if n.l2Source != nil {
		n.l2Source.Close()
	}

	// close L1 data source
	if n.l1Source != nil {
		n.l1Source.Close()
	}

	if result == nil { // mark as closed if we successfully fully closed
		n.closed.Store(true)
	}

	return result.ErrorOrNil()
}

func (n *OpNode) Closed() bool {
	return n.closed.Load()
}

func (n *OpNode) ListenAddr() string {
	return n.server.listenAddr.String()
}

func (n *OpNode) HTTPEndpoint() string {
	return fmt.Sprintf("http://%s", n.ListenAddr())
}
