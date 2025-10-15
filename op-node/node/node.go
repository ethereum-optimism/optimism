package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	gosync "sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethevent "github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/node/runcfg"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/node/tracer"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop/indexing"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sequencing"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

var ErrAlreadyClosed = errors.New("node is already closed")

type closableSafeDB interface {
	rollup.SafeHeadListener
	SafeDBReader
	io.Closer
}

// L1Source provides the necessary L1 blockchain data for the node.
type L1Source interface {
	L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error)
	L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error)
	L1BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L1BlockRef, error)
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
	ReadStorageAt(ctx context.Context, address common.Address, storageSlot common.Hash, blockHash common.Hash) (common.Hash, error)
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
	Close()
}

// L1Beacon provides access to L1 beacon chain data, specifically for blob data retrieval.
type L1Beacon interface {
	GetBlobs(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash) ([]*eth.Blob, error)
}

type OpNode struct {
	// Retain the config to test for active features rather than test for runtime state.
	cfg        *config.Config
	log        log.Logger
	appVersion string
	metrics    *metrics.Metrics

	l1HeadsSub     ethereum.Subscription // Subscription to get L1 heads (automatically re-subscribes on error)
	l1SafeSub      ethereum.Subscription // Subscription to get L1 safe blocks, a.k.a. justified data (polling)
	l1FinalizedSub ethereum.Subscription // Subscription to get L1 safe blocks, a.k.a. justified data (polling)

	eventSys   event.System
	eventDrain driver.Drain

	l1Source  L1Source              // L1 Client to fetch data from
	l2Driver  *driver.Driver        // L2 Engine to Sync
	l2Source  *sources.EngineClient // L2 Execution Engine RPC bindings
	server    *oprpc.Server         // RPC server hosting the rollup-node API
	p2pNode   *p2p.NodeP2P          // P2P node functionality
	p2pMu     gosync.Mutex          // protects p2pNode
	p2pSigner p2p.Signer            // p2p gossip application messages will be signed with this signer
	runCfg    *runcfg.RuntimeConfig // runtime configurables

	safeDB closableSafeDB

	rollupHalt string // when to halt the rollup, disabled if empty

	pprofService *oppprof.Service
	metricsSrv   *httputil.HTTPServer

	beacon L1Beacon

	interopSys interop.SubSystem

	// some resources cannot be stopped directly, like the p2p gossipsub router (not our design),
	// and depend on this ctx to be closed.
	resourcesCtx   context.Context
	resourcesClose context.CancelFunc

	// Indicates when it's safe to close data sources used by the runtimeConfig bg loader
	runtimeConfigReloaderDone chan struct{}

	closed atomic.Bool

	// cancels execution prematurely, e.g. to halt. This may be nil.
	cancel context.CancelCauseFunc
	halted atomic.Bool

	tracer tracer.Tracer // used for testing PublishBlock and SignAndPublishL2Payload
}

// New creates a new OpNode instance.
// The provided ctx argument is for the span of initialization only;
// the node will immediately Stop(ctx) before finishing initialization if the context is canceled during initialization.
func New(ctx context.Context, cfg *config.Config, log log.Logger, appVersion string, m *metrics.Metrics) (*OpNode, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}

	n := &OpNode{
		cfg:        cfg,
		log:        log,
		appVersion: appVersion,
		metrics:    m,
		rollupHalt: cfg.RollupHalt,
		cancel:     cfg.Cancel,
		tracer:     cfg.Tracer,
	}
	// not a context leak, gossipsub is closed with a context.
	n.resourcesCtx, n.resourcesClose = context.WithCancel(context.Background())

	err := n.init(ctx, cfg, initializationOverrides{})
	if err != nil {
		log.Error("Error initializing the rollup node", "err", err)
		// ensure we always close the node resources if we fail to initialize the node.
		if closeErr := n.Stop(ctx); closeErr != nil {
			return nil, multierror.Append(err, closeErr)
		}
		return nil, err
	}
	return n, nil
}

type initializationOverrides struct {
	L1Source   L1Source
	Beacon     L1Beacon
	RPCHandler *oprpc.Handler
}

// init progressively creates and sets up all the components of the OpNode
// some later initialization steps depend on the node being partially initialized with other components,
// so order is important to ensure that all resources are available when needed.
func (n *OpNode) init(ctx context.Context, cfg *config.Config, overrides initializationOverrides) error {
	n.log.Info("Initializing rollup node", "version", n.appVersion)

	var err error

	n.eventSys, n.eventDrain, err = initEventSystem(n)
	if err != nil {
		return fmt.Errorf("failed to init event system: %w", err)
	}

	if overrides.Beacon == nil {
		beacon, err := initL1BeaconAPI(ctx, cfg, n)
		if err != nil {
			return err
		}
		n.beacon = beacon
	} else {
		n.beacon = overrides.Beacon
	}

	if overrides.L1Source == nil {
		l1Source, err := initL1Source(ctx, cfg, n)
		if err != nil {
			return fmt.Errorf("failed to init L1 Source: %w", err)
		}
		n.l1Source = l1Source
	} else {
		n.l1Source = overrides.L1Source
	}

	// initL2 may use side effects to register interop subsystem to the node.EventSystem
	n.l2Source, n.interopSys, n.l2Driver, n.safeDB, err = initL2(ctx, cfg, n)
	if err != nil {
		return fmt.Errorf("failed to init L2: %w", err)
	}

	n.l1HeadsSub, n.l1SafeSub, n.l1FinalizedSub, err = initL1Handlers(cfg, n)
	if err != nil {
		return fmt.Errorf("failed to init L1 Source: %w", err)
	}

	// initRuntimeConfig relies on side effects to set the runCfg, node.halted and call node.cancel if needed
	if err := initRuntimeConfig(ctx, cfg, n); err != nil {
		return fmt.Errorf("failed to init the runtime config: %w", err)
	}

	n.p2pSigner, err = initP2PSigner(ctx, cfg, n)
	if err != nil {
		return fmt.Errorf("failed to init the P2P signer: %w", err)
	}

	n.p2pNode, err = initP2P(cfg, n)
	if err != nil {
		return fmt.Errorf("failed to init the P2P stack: %w", err)
	}

	// Only expose the server at the end, ensuring all RPC backend components are initialized.
	if overrides.RPCHandler == nil {
		n.server, err = initRPCServer(cfg, n)
		if err != nil {
			return fmt.Errorf("failed to init the RPC server: %w", err)
		}
	} else {
		// the node registers to an existing RPC server's handler if provided
		// the node assumes the RPC server is already started
		n.server = nil
		err := registerAPIs(cfg, n, overrides.RPCHandler)
		if err != nil {
			// panic here is to match the behavior of oprcp.Server.AddAPI,
			// which wraps the Handler and panics if the API can't be added.
			panic(fmt.Errorf("invalid API: %w", err))
		}
	}

	n.metricsSrv, err = initMetricsServer(cfg, n)
	if err != nil {
		return fmt.Errorf("failed to init the metrics server: %w", err)
	}

	n.metrics.RecordInfo(n.appVersion)
	n.metrics.RecordUp()

	n.pprofService, err = initPProf(cfg, n)
	if err != nil {
		return fmt.Errorf("failed to init profiling: %w", err)
	}

	return nil
}

func initEventSystem(node *OpNode) (event.System, driver.Drain, error) {
	// This executor will be configurable in the future, for parallel event processing
	executor := event.NewGlobalSynchronous(node.resourcesCtx).WithMetrics(node.metrics)
	sys := event.NewSystem(node.log, executor)
	sys.AddTracer(event.NewMetricsTracer(node.metrics))
	sys.Register("node", event.DeriverFunc(node.onEvent))
	return sys, executor, nil
}

func initL1Source(ctx context.Context, cfg *config.Config, node *OpNode) (*sources.L1Client, error) {
	// Cache 3/2 worth of sequencing window of receipts and txs
	defaultCacheSize := int(cfg.Rollup.SeqWindowSize) * 3 / 2
	l1RPC, l1Cfg, err := cfg.L1.Setup(ctx, node.log, defaultCacheSize, node.metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to get L1 RPC client: %w", err)
	}

	l1Source, err := sources.NewL1Client(l1RPC, node.log, node.metrics.L1SourceCache, l1Cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 source: %w", err)
	}

	if err := cfg.Rollup.ValidateL1Config(ctx, node.log, l1Source); err != nil {
		return nil, fmt.Errorf("failed to validate the L1 config: %w", err)
	}

	return l1Source, nil
}

func initL1Handlers(cfg *config.Config, node *OpNode) (ethereum.Subscription, ethereum.Subscription, ethereum.Subscription, error) {
	if node.l2Driver == nil {
		return nil, nil, nil, errors.New("l2 driver must be initialized")
	}
	onL1Head := func(ctx context.Context, sig eth.L1BlockRef) {
		// TODO(#16917) Remove Event System Refactor Comments
		//  L1UnsafeEvent fan out is updated to procedural method calls
		if node.cfg.Tracer != nil {
			node.cfg.Tracer.OnNewL1Head(ctx, sig)
		}
		node.l2Driver.SyncDeriver.L1Tracker.OnL1Unsafe(sig)
		node.l2Driver.StatusTracker.OnL1Unsafe(sig)
		node.l2Driver.SyncDeriver.OnL1Unsafe(ctx)
	}
	onL1Safe := func(ctx context.Context, sig eth.L1BlockRef) {
		node.l2Driver.StatusTracker.OnL1Safe(sig)
	}
	onL1Finalized := func(ctx context.Context, sig eth.L1BlockRef) {
		// TODO(#16917) Remove Event System Refactor Comments
		//  FinalizeL1Event fan out is updated to procedural method calls
		node.l2Driver.StatusTracker.OnL1Finalized(sig)
		node.l2Driver.Finalizer.OnL1Finalized(sig)
		node.l2Driver.SyncDeriver.OnL1Finalized(ctx)
	}

	// Keep subscribed to the L1 heads, which keeps the L1 maintainer pointing to the best headers to sync
	l1HeadsSub := gethevent.ResubscribeErr(time.Second*10, func(ctx context.Context, err error) (gethevent.Subscription, error) {
		if err != nil {
			node.log.Warn("resubscribing after failed L1 subscription", "err", err)
		}
		return eth.WatchHeadChanges(ctx, node.l1Source, onL1Head)
	})
	go func() {
		err, ok := <-l1HeadsSub.Err()
		if !ok {
			return
		}
		node.log.Error("l1 heads subscription error", "err", err)
	}()

	// Poll for the safe L1 block and finalized block,
	// which only change once per epoch at most and may be delayed.
	l1SafeSub := eth.PollBlockChanges(node.log, node.l1Source, onL1Safe, eth.Safe,
		cfg.L1EpochPollInterval, time.Second*10)
	l1FinalizedSub := eth.PollBlockChanges(node.log, node.l1Source, onL1Finalized, eth.Finalized,
		cfg.L1EpochPollInterval, time.Second*10)

	return l1HeadsSub, l1SafeSub, l1FinalizedSub, nil
}

// initRuntimeConfig initializes the runtime config and starts a background loop to reload it at the configured interval.
// note: this function relies on side effects to set node.runCfg
func initRuntimeConfig(ctx context.Context, cfg *config.Config, node *OpNode) error {
	// attempt to load runtime config, repeat N times
	runCfg := runcfg.NewRuntimeConfig(node.log, node.l1Source, &cfg.Rollup)
	// Set node.runCfg early so handleProtocolVersionsUpdate can access it during initialization
	node.runCfg = runCfg

	confDepth := cfg.Driver.VerifierConfDepth
	reload := func(ctx context.Context) (eth.L1BlockRef, error) {
		fetchCtx, fetchCancel := context.WithTimeout(ctx, time.Second*10)
		l1Head, err := node.l1Source.L1BlockRefByLabel(fetchCtx, eth.Unsafe)
		fetchCancel()
		if err != nil {
			node.log.Error("failed to fetch L1 head for runtime config initialization", "err", err)
			return eth.L1BlockRef{}, err
		}

		// Apply confirmation-distance
		blNum := l1Head.Number
		if blNum >= confDepth {
			blNum -= confDepth
		}
		fetchCtx, fetchCancel = context.WithTimeout(ctx, time.Second*10)
		confirmed, err := node.l1Source.L1BlockRefByNumber(fetchCtx, blNum)
		fetchCancel()
		if err != nil {
			node.log.Error("failed to fetch confirmed L1 block for runtime config loading", "err", err, "number", blNum)
			return eth.L1BlockRef{}, err
		}

		fetchCtx, fetchCancel = context.WithTimeout(ctx, time.Second*10)
		err = runCfg.Load(fetchCtx, confirmed)
		fetchCancel()
		if err != nil {
			node.log.Error("failed to fetch runtime config data", "err", err)
			return l1Head, err
		}

		err = node.handleProtocolVersionsUpdate(ctx)
		return l1Head, err
	}

	// initialize the runtime config before unblocking
	if err := retry.Do0(ctx, 5, retry.Fixed(time.Second*10), func() error {
		_, err := reload(ctx)
		if errors.Is(err, errNodeHalt) { // don't retry on halt error
			err = nil
		}
		return err
	}); err != nil {
		return fmt.Errorf("failed to load runtime configuration repeatedly, last error: %w", err)
	}

	// start a background loop, to keep reloading it at the configured reload interval
	reloader := func(ctx context.Context, reloadInterval time.Duration) {
		if reloadInterval <= 0 {
			node.log.Debug("not running runtime-config reloading background loop")
			return
		}
		ticker := time.NewTicker(reloadInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// If the reload fails, we will try again the next interval.
				// Missing a runtime-config update is not critical, and we do not want to overwhelm the L1 RPC.
				l1Head, err := reload(ctx)
				if err != nil {
					if errors.Is(err, errNodeHalt) {
						node.halted.Store(true)
						if node.cancel != nil { // node cancellation is always available when started as CLI app
							node.cancel(errNodeHalt)
							return
						} else {
							node.log.Debug("opted to halt, but cannot halt node", "l1_head", l1Head)
						}
					} else {
						node.log.Warn("failed to reload runtime config", "err", err)
					}
				} else {
					node.log.Debug("reloaded runtime config", "l1_head", l1Head)
				}
			case <-ctx.Done():
				return
			}
		}
	}

	// Manages the lifetime of reloader. In order to safely Close the OpNode
	go func(ctx context.Context, reloadInterval time.Duration) {
		reloader(ctx, reloadInterval)
	}(node.resourcesCtx, cfg.RuntimeConfigReloadInterval) // this keeps running after initialization
	return nil
}

func initL1BeaconAPI(ctx context.Context, cfg *config.Config, node *OpNode) (*sources.L1BeaconClient, error) {
	// If Ecotone upgrade is not scheduled yet, then there is no need for a Beacon API.
	if cfg.Rollup.EcotoneTime == nil {
		return nil, nil
	}
	// Once the Ecotone upgrade is scheduled, we must have initialized the Beacon API settings.
	if cfg.Beacon == nil {
		return nil, fmt.Errorf("missing L1 Beacon Endpoint configuration: this API is mandatory for Ecotone upgrade at t=%d", *cfg.Rollup.EcotoneTime)
	}

	// We always initialize a client. We will get an error on requests if the client does not work.
	// This way the op-node can continue non-L1 functionality when the user chooses to ignore the Beacon API requirement.
	beaconClient, fallbacks, err := cfg.Beacon.Setup(ctx, node.log)
	if err != nil {
		return nil, fmt.Errorf("failed to setup L1 Beacon API client: %w", err)
	}
	beaconCfg := sources.L1BeaconClientConfig{
		FetchAllSidecars: cfg.Beacon.ShouldFetchAllSidecars(),
	}
	beacon := sources.NewL1BeaconClient(beaconClient, beaconCfg, fallbacks...)

	// Retry retrieval of the Beacon API version, to be more robust on startup against Beacon API connection issues.
	beaconVersion, missingEndpoint, err := retry.Do2[string, bool](ctx, 5, retry.Exponential(), func() (string, bool, error) {
		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		beaconVersion, err := beacon.GetVersion(ctx)
		if err != nil {
			if errors.Is(err, client.ErrNoEndpoint) {
				return "", true, nil // don't return an error, we do not have to retry when there is a config issue.
			}
			return "", false, err
		}
		return beaconVersion, false, nil
	})
	if missingEndpoint {
		// Allow the user to continue if they explicitly ignore the requirement of the endpoint.
		if cfg.Beacon.ShouldIgnoreBeaconCheck() {
			node.log.Warn("This endpoint is required for the Ecotone upgrade, but is missing, and configured to be ignored. " +
				"The node may be unable to retrieve EIP-4844 blobs data.")
			return beacon, nil
		} else {
			// If the client tells us the endpoint was not configured,
			// then explain why we need it, and what the user can do to ignore this.
			node.log.Error("The Ecotone upgrade requires a L1 Beacon API endpoint, to retrieve EIP-4844 blobs data. " +
				"This can be ignored with the --l1.beacon.ignore option, " +
				"but the node may be unable to sync from L1 without this endpoint.")
			return nil, errors.New("missing L1 Beacon API endpoint")
		}
	} else if err != nil {
		if cfg.Beacon.ShouldIgnoreBeaconCheck() {
			node.log.Warn("Failed to check L1 Beacon API version, but configuration ignores results. "+
				"The node may be unable to retrieve EIP-4844 blobs data.", "err", err)
			return beacon, nil
		} else {
			return nil, fmt.Errorf("failed to check L1 Beacon API version: %w", err)
		}
	} else {
		node.log.Info("Connected to L1 Beacon API, ready for EIP-4844 blobs retrieval.", "version", beaconVersion)
		return beacon, nil
	}
}

func initL2(ctx context.Context, cfg *config.Config, node *OpNode) (*sources.EngineClient, interop.SubSystem, *driver.Driver, closableSafeDB, error) {
	rpcClient, rpcCfg, err := cfg.L2.Setup(ctx, node.log, &cfg.Rollup, node.metrics)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to setup L2 execution-engine RPC client: %w", err)
	}

	rpcCfg.FetchWithdrawalRootFromState = cfg.FetchWithdrawalRootFromState

	l2Source, err := sources.NewEngineClient(rpcClient, node.log, node.metrics.L2SourceCache, rpcCfg)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to create Engine client: %w", err)
	}

	if err := cfg.Rollup.ValidateL2Config(ctx, l2Source, cfg.Sync.SyncMode == sync.ELSync); err != nil {
		return nil, nil, nil, nil, err
	}

	indexingMode := false
	sys, err := cfg.InteropConfig.Setup(ctx, node.log, &node.cfg.Rollup, node.l1Source, l2Source, node.metrics)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to setup interop: %w", err)
	} else if sys != nil { // we continue with legacy mode if no interop sub-system is set up.
		_, indexingMode = sys.(*indexing.IndexingMode)
		node.eventSys.Register("interop", sys)
	}

	var sequencerConductor conductor.SequencerConductor = &conductor.NoOpConductor{}
	if cfg.ConductorEnabled {
		sequencerConductor = NewConductorClient(cfg, node.log, node.metrics)
	}

	// if altDA is not explicitly activated in the node CLI, the config + any error will be ignored.
	rpCfg, err := cfg.Rollup.GetOPAltDAConfig()
	if cfg.AltDA.Enabled && err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to get altDA config: %w", err)
	}
	altDA := altda.NewAltDA(node.log, cfg.AltDA, rpCfg, node.metrics.AltDAMetrics)
	var safeDB closableSafeDB
	if cfg.SafeDBPath != "" {
		node.log.Info("Safe head database enabled", "path", cfg.SafeDBPath)
		safeDB, err = safedb.NewSafeDB(node.log, cfg.SafeDBPath)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to create safe head database at %v: %w", cfg.SafeDBPath, err)
		}
	} else {
		safeDB = safedb.Disabled
	}

	if cfg.Rollup.ChainOpConfig == nil {
		return nil, nil, nil, nil, fmt.Errorf("cfg.Rollup.ChainOpConfig is nil. Please see https://github.com/ethereum-optimism/optimism/releases/tag/op-node/v1.11.0: %w", err)
	}

	l2Driver := driver.NewDriver(node.eventSys, node.eventDrain, &cfg.Driver, &cfg.Rollup, cfg.L1ChainConfig, cfg.DependencySet, l2Source, node.l1Source,
		node.beacon, node, node, node.log, node.metrics, cfg.ConfigPersistence, safeDB, &cfg.Sync, sequencerConductor, altDA, indexingMode)

	// Wire up IndexingMode to engine controller for direct procedure call
	if sys != nil {
		if indexingMode, ok := sys.(*indexing.IndexingMode); ok {
			indexingMode.SetEngineController(l2Driver.SyncDeriver.Engine)
		}
	}

	return l2Source, sys, l2Driver, safeDB, nil
}

func initRPCServer(cfg *config.Config, node *OpNode) (*oprpc.Server, error) {
	server := newRPCServer(&cfg.RPC, &cfg.Rollup, cfg.DependencySet,
		node.l2Source.L2Client, node.l2Driver, node.safeDB,
		node.log, node.metrics, node.appVersion)
	if err := registerAPIs(cfg, node, server.Handler); err != nil {
		// panic here is to match the behavior of oprcp.Server.AddAPI,
		// which wraps the Handler and panics if the API can't be added.
		panic(fmt.Errorf("invalid API: %w", err))
	}
	node.log.Info("Starting JSON-RPC server")
	if err := server.Start(); err != nil {
		return nil, fmt.Errorf("unable to start RPC server: %w", err)
	}
	node.log.Info("Started JSON-RPC server", "addr", server.Endpoint())
	return server, nil
}

func registerAPIs(cfg *config.Config, node *OpNode, handler *oprpc.Handler) error {
	if p2pNode := node.getP2PNodeIfEnabled(); p2pNode != nil {
		if err := handler.AddAPI(rpc.API{
			Namespace: p2p.NamespaceRPC,
			Service:   p2p.NewP2PAPIBackend(p2pNode, node.log),
		}); err != nil {
			return fmt.Errorf("failed to add P2P API: %w", err)
		}
		node.log.Info("P2P RPC enabled")
	}
	if cfg.ExperimentalOPStackAPI {
		if err := handler.AddAPI(rpc.API{
			Namespace: "opstack",
			Service:   NewOpstackAPI(node.l2Driver.SyncDeriver.Engine, node),
		}); err != nil {
			return fmt.Errorf("failed to add Experimental OP stack API: %w", err)
		}
		node.log.Info("Experimental OP stack API enabled")
	}
	if cfg.RPC.EnableAdmin {
		if err := handler.AddAPI(rpc.API{
			Namespace: "admin",
			Service:   NewAdminAPI(node.l2Driver, node.log),
		}); err != nil {
			return fmt.Errorf("failed to add Admin API: %w", err)
		}
		node.log.Info("Admin RPC enabled")
	}
	return nil
}

func initMetricsServer(cfg *config.Config, node *OpNode) (*httputil.HTTPServer, error) {
	if !cfg.Metrics.Enabled {
		node.log.Info("metrics disabled")
		return nil, nil
	}
	node.log.Debug("starting metrics server", "addr", cfg.Metrics.ListenAddr, "port", cfg.Metrics.ListenPort)
	metricsSrv, err := node.metrics.StartServer(cfg.Metrics.ListenAddr, cfg.Metrics.ListenPort)
	if err != nil {
		return nil, fmt.Errorf("failed to start metrics server: %w", err)
	}
	node.log.Info("started metrics server", "addr", metricsSrv.Addr())
	return metricsSrv, nil
}

func initPProf(cfg *config.Config, node *OpNode) (*oppprof.Service, error) {
	pprofService := oppprof.New(
		cfg.Pprof.ListenEnabled,
		cfg.Pprof.ListenAddr,
		cfg.Pprof.ListenPort,
		cfg.Pprof.ProfileType,
		cfg.Pprof.ProfileDir,
		cfg.Pprof.ProfileFilename,
	)

	if err := pprofService.Start(); err != nil {
		return nil, fmt.Errorf("failed to start pprof service: %w", err)
	}

	return pprofService, nil
}

func (n *OpNode) p2pEnabled() bool {
	return n.cfg.P2PEnabled()
}

func initP2P(cfg *config.Config, node *OpNode) (*p2p.NodeP2P, error) {
	node.p2pMu.Lock()
	defer node.p2pMu.Unlock()
	if node.p2pNode != nil {
		panic("p2p node already initialized")
	}
	if node.p2pEnabled() {
		if node.l2Driver.SyncDeriver == nil {
			panic("SyncDeriver must be initialized")
		}
		// embed syncDeriver and tracer(optional) to the blockReceiver to handle unsafe payloads via p2p
		rec := p2p.NewBlockReceiver(node.log, node.metrics, node.l2Driver.SyncDeriver, node.cfg.Tracer)
		p2pNode, err := p2p.NewNodeP2P(node.resourcesCtx, &cfg.Rollup, node.log, cfg.P2P, rec, node.l2Source, node.runCfg, node.metrics)
		if err != nil {
			return nil, err
		}
		if p2pNode.Dv5Udp() != nil {
			go p2pNode.DiscoveryProcess(node.resourcesCtx, node.log, &cfg.Rollup, cfg.P2P.TargetPeers())
		}
		return p2pNode, nil
	}
	return nil, nil
}

func initP2PSigner(ctx context.Context, cfg *config.Config, node *OpNode) (p2p.Signer, error) {
	// the p2p signer setup is optional
	if cfg.P2PSigner == nil {
		return nil, nil
	}
	// p2pSigner may still be nil, the signer setup may not create any signer, the signer is optional
	p2pSigner, err := cfg.P2PSigner.SetupSigner(ctx)
	return p2pSigner, err
}

func (n *OpNode) Start(ctx context.Context) error {
	if n.interopSys != nil {
		if err := n.interopSys.Start(ctx); err != nil {
			n.log.Error("Could not start interop sub system", "err", err)
			return err
		}
	}
	n.log.Info("Starting execution engine driver")
	// start driving engine: sync blocks by deriving them from L1 and driving them into the engine
	if err := n.l2Driver.Start(); err != nil {
		n.log.Error("Could not start a rollup node", "err", err)
		return err
	}
	log.Info("Rollup node started")
	return nil
}

// onEvent handles broadcast events.
// The OpNode itself is a deriver to catch system-critical events.
// Other event-handling should be encapsulated into standalone derivers.
func (n *OpNode) onEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case rollup.CriticalErrorEvent:
		n.log.Error("Critical error", "err", x.Err)
		n.cancel(fmt.Errorf("critical error: %w", x.Err))
		return true
	default:
		return false
	}
}

func (n *OpNode) PublishBlock(ctx context.Context, signedEnvelope *opsigner.SignedExecutionPayloadEnvelope) error {
	if n.tracer != nil {
		n.tracer.OnPublishL2Payload(ctx, signedEnvelope.Envelope)
	}
	if p2pNode := n.getP2PNodeIfEnabled(); p2pNode != nil {
		n.log.Info("Publishing signed execution payload on p2p", "id", signedEnvelope.ID())
		return p2pNode.GossipOut().PublishSignedL2Payload(ctx, signedEnvelope)
	}
	return errors.New("P2P not enabled")
}

func (n *OpNode) SignAndPublishL2Payload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
	if n.tracer != nil {
		n.tracer.OnPublishL2Payload(ctx, envelope)
	}
	// publish to p2p, if we are running p2p at all
	if p2pNode := n.getP2PNodeIfEnabled(); p2pNode != nil {
		if n.p2pSigner == nil {
			return fmt.Errorf("node has no p2p signer, payload %s cannot be published", envelope.ID())
		}
		n.log.Info("Publishing signed execution payload on p2p", "id", envelope.ID())
		return p2pNode.GossipOut().SignAndPublishL2Payload(ctx, envelope, n.p2pSigner)
	}
	// if p2p is not enabled then we just don't publish the payload
	return nil
}

func (n *OpNode) RequestL2Range(ctx context.Context, start, end eth.L2BlockRef) error {
	if p2pNode := n.getP2PNodeIfEnabled(); p2pNode != nil && p2pNode.AltSyncEnabled() {
		if unixTimeStale(start.Time, 12*time.Hour) {
			n.log.Debug(
				"ignoring request to sync L2 range, timestamp is too old for p2p",
				"start", start,
				"end", end,
				"start_time", start.Time)
			return nil
		}
		return p2pNode.RequestL2Range(ctx, start, end)
	}
	n.log.Debug("ignoring request to sync L2 range, no sync method available", "start", start, "end", end)
	return nil
}

// unixTimeStale returns true if the unix timestamp is before the current time minus the supplied duration.
func unixTimeStale(timestamp uint64, duration time.Duration) bool {
	return time.Unix(int64(timestamp), 0).Before(time.Now().Add(-1 * duration))
}

func (n *OpNode) P2P() p2p.Node {
	return n.getP2PNodeIfEnabled()
}

func (n *OpNode) RuntimeConfig() runcfg.ReadonlyRuntimeConfig {
	return n.runCfg
}

// Stop stops the node and closes all resources.
// If the provided ctx is expired, the node will accelerate the stop where possible, but still fully close.
func (n *OpNode) Stop(ctx context.Context) error {
	if n.closed.Load() {
		return ErrAlreadyClosed
	}

	var result *multierror.Error

	if n.server != nil {
		if err := n.server.Stop(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close RPC server: %w", err))
		}
	}

	// Stop sequencer and report last hash. l2Driver can be nil if we're cleaning up a failed init.
	if n.l2Driver != nil {
		latestHead, err := n.l2Driver.StopSequencer(ctx)
		switch {
		case errors.Is(err, sequencing.ErrSequencerNotEnabled):
		case errors.Is(err, driver.ErrSequencerAlreadyStopped):
			n.log.Info("stopping node: sequencer already stopped", "latestHead", latestHead)
		case err == nil:
			n.log.Info("stopped sequencer", "latestHead", latestHead)
		default:
			result = multierror.Append(result, fmt.Errorf("error stopping sequencer: %w", err))
		}
	}

	n.p2pMu.Lock()
	if n.p2pNode != nil {
		if err := n.p2pNode.Close(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close p2p node: %w", err))
		}
		// Prevent further use of p2p.
		n.p2pNode = nil
	}
	n.p2pMu.Unlock()

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
	// stop polling for L1 safe-head changes
	if n.l1SafeSub != nil {
		n.l1SafeSub.Unsubscribe()
	}
	// stop polling for L1 finalized-head changes
	if n.l1FinalizedSub != nil {
		n.l1FinalizedSub.Unsubscribe()
	}

	// close L2 driver
	if n.l2Driver != nil {
		if err := n.l2Driver.Close(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close L2 engine driver cleanly: %w", err))
		}
	}

	// close the interop sub system
	if n.interopSys != nil {
		if err := n.interopSys.Stop(ctx); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close interop sub-system: %w", err))
		}
	}

	if n.eventSys != nil {
		n.eventSys.Stop()
	}

	if n.safeDB != nil {
		if err := n.safeDB.Close(); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close safe head db: %w", err))
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

	if n.halted.Load() {
		// if we had a halt upon initialization, idle for a while, with open metrics, to prevent a rapid restart-loop
		tim := time.NewTimer(time.Minute * 5)
		n.log.Warn("halted, idling to avoid immediate shutdown repeats")
		defer tim.Stop()
		select {
		case <-tim.C:
		case <-ctx.Done():
		}
	}

	// Close metrics and pprof only after we are done idling
	if n.pprofService != nil {
		if err := n.pprofService.Stop(ctx); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close pprof server: %w", err))
		}
	}
	if n.metricsSrv != nil {
		if err := n.metricsSrv.Stop(ctx); err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to close metrics server: %w", err))
		}
	}

	return result.ErrorOrNil()
}

func (n *OpNode) Stopped() bool {
	return n.closed.Load()
}

func (n *OpNode) HTTPEndpoint() string {
	if n.server == nil {
		return ""
	}
	return fmt.Sprintf("http://%s", n.server.Endpoint())
}

func (n *OpNode) HTTPPort() (int, error) {
	return n.server.Port()
}

func (n *OpNode) InteropRPC() (rpcEndpoint string, jwtSecret eth.Bytes32) {
	m, ok := n.interopSys.(*indexing.IndexingMode)
	if !ok {
		return "", [32]byte{}
	}
	return m.WSEndpoint(), m.JWTSecret()
}

func (n *OpNode) InteropRPCPort() (int, error) {
	m, ok := n.interopSys.(*indexing.IndexingMode)
	if !ok {
		return 0, fmt.Errorf("failed to fetch interop port for op-node")
	}
	return m.WSPort()
}

func (n *OpNode) getP2PNodeIfEnabled() *p2p.NodeP2P {
	if !n.p2pEnabled() {
		return nil
	}

	n.p2pMu.Lock()
	defer n.p2pMu.Unlock()
	return n.p2pNode
}
