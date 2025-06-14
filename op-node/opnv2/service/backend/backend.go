package backend

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	gosync "sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/config"
	metrics2 "github.com/ethereum-optimism/optimism/op-node/opnv2/metrics"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/controller"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/derive2"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/l1access"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/l1rewind"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/l2rewind"
	sp2p "github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/p2p"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/payloads"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rel"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/runcfg2"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/cross"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/status"
)

type Backend struct {
	started atomic.Bool
	logger  log.Logger
	m       metrics2.Metricer
	dataDir string

	eventSys  event.System
	eventExec event.Executor // may implement event.Drainer if synchronous

	// synchronousProcessors disables background-workers,
	// requiring manual triggers for the backend to process l2 data.
	synchronousProcessors bool

	sysContext context.Context
	sysCancel  context.CancelFunc

	// cfgSet is the full config set that the backend uses to know about the chains it is indexing
	cfgSet        depset.FullConfigSet
	rollupConfigs depset.RollupConfigSetV2

	l1ChainID eth.ChainID

	// linker checks if the configuration constraints of a message (check chain ID + timestamp)
	linker depset.LinkChecker

	// chainDBs is the primary interface to the databases, including logs, derived-from information and L1 finalization
	chainDBs *db.ChainsDB

	l1RPC  *sources.L1Client
	beacon *sources.L1BeaconClient
	// l1Accessor provides access to the L1 chain for the L1 processor and subscribes to new block events
	l1Accessor *l1access.L1Accessor

	// L2 read-only RPC clients, mapped by their preserved ID.
	// Until we have identified which chain they correspond to we do not recognize them as REL component.
	// RPC connections persist here, and are closed on shutdown.
	readEndpoints locks.RWMap[rel.ID, client.RPC]

	// L2 execution-engine RPC clients, mapped by their preserved ID.
	// Until we have identified which chain they correspond to we do not recognize them as RWEL component.
	// RPC connections persist here, and are closed on shutdown.
	engineEndpoints locks.RWMap[rwel.ID, client.RPC]

	// rels are read-only execution-layer nodes, we use them as fallback for execution verification.
	rels locks.RWMap[rel.ID, *rel.REL]
	// rwel are read-write execution-layer nodes, we use them to sync execution layer nodes and verify execution fully.
	rwels locks.RWMap[rwel.ID, *rwel.RWEL]

	pipelines locks.RWMap[derive2.ID, *derive2.PipelineV2]

	// TODO set these, once known
	readL2Clients locks.RWMap[rel.ID, apis.L2EthExtendedClient]
	rwL2Clients   locks.RWMap[rwel.ID, apis.L2EthExtendedClient]

	// chainProcessors are notified of new unsafe blocks, and add the unsafe log events data into the events DB
	chainProcessors locks.RWMap[eth.ChainID, *processors.ChainProcessor]
	// L1 rewinders are notified of potential L1 divergence,
	// and resolve inconsistency by rewinding local-safe/cross-safe DBs
	l1Rewinders locks.RWMap[eth.ChainID, *l1rewind.L1Rewinder]
	// L2 rewinders are notified of potential log-index divergence from the local-safe DB,
	// and resolve inconsistency by rewinding the local-unsafe log index.
	l2Rewinders locks.RWMap[eth.ChainID, *l2rewind.L2Rewinder]
	// Cross-unsafe workers will check cross-unsafe safety and promote when possible
	crossUnsafeWorkers locks.RWMap[eth.ChainID, *cross.CrossUnsafeWorker]
	// Cross-safe workers will check cross-safety and promote when possible
	crossSafeWorkers locks.RWMap[eth.ChainID, *cross.CrossSafeWorker]

	payloads locks.RWMap[eth.ChainID, *payloads.Payloads]

	// statusTracker tracks the sync status of the supervisor
	statusTracker *status.StatusTracker

	// controller makes all the syncing / state-change decisions but does none of the long-running computation.
	controller *controller.Controller

	// chainMetrics are used to track metrics for each chain
	// they are reused for processors and databases of the same chain
	chainMetrics locks.RWMap[eth.ChainID, *metrics2.ChainMetrics]

	runCfg *runcfg2.RuntimeConfig

	p2pNode *sp2p.NodeP2P // P2P node functionality
	p2pMu   gosync.Mutex  // protects p2pNode
}

var _ apis.SupervisorQueryAPI = (*Backend)(nil)

var (
	errAlreadyStopped = errors.New("already stopped")
	errAlreadyStarted = errors.New("already started")

	ErrUnexpectedMinSafetyLevel = errors.New("unexpected min-safety level")
)

func NewBackend(ctx context.Context, logger log.Logger,
	m metrics2.Metricer, cfg *config.Config,
) (*Backend, error) {
	// attempt to prepare the data directory
	if err := db.PrepDataDir(cfg.Datadir); err != nil {
		return nil, err
	}

	// In the future we may introduce other executors.
	// For now, we just use a synchronous executor, and poll the drain function of it.
	eventExec := event.NewGlobalSynchronous(ctx)
	// TODO: loop to await next event, then execute

	eventSys := event.NewSystem(logger, eventExec)
	eventSys.AddTracer(event.NewMetricsTracer(m))

	sysCtx, sysCancel := context.WithCancel(ctx)

	super := &Backend{
		logger:     logger,
		m:          m,
		dataDir:    cfg.Datadir,
		eventSys:   eventSys,
		eventExec:  eventExec,
		sysCancel:  sysCancel,
		sysContext: sysCtx,
		// For testing we can avoid running the processors.
		synchronousProcessors: cfg.SynchronousProcessors,
	}

	// Initialize the resources of the backend.
	// Stop if any of the resources fails to be initialized.
	if err := super.initResources(ctx, cfg); err != nil {
		err = fmt.Errorf("failed to init resources: %w", err)
		return nil, errors.Join(err, super.Stop(ctx))
	}

	return super, nil
}

// initResources initializes all the resources, such as DBs and processors for chains.
// An error may returned, without closing the thus-far initialized resources.
// Upon error the caller should call Stop() on the supervisor backend to clean up and release resources.
func (b *Backend) initResources(ctx context.Context, cfg *config.Config) error {
	// Load the chain configs
	if err := b.initChainConfigs(ctx, cfg); err != nil {
		return fmt.Errorf("failed to init chain config: %w", err)
	}

	chains := b.cfgSet.Chains()

	b.controller = controller.NewController(b.sysContext, b.logger, b.rollupConfigs, b.DependencySet())
	b.eventSys.Register("controller", b.controller)

	// Create chains DB
	b.chainDBs = db.NewChainsDB(b.logger, b.cfgSet, b.m)
	b.eventSys.Register("chainsDBs", b.chainDBs)

	// Setup metrics adapters for each chain
	for _, chainID := range chains {
		cm := metrics2.NewChainMetrics(chainID, b.m)
		b.chainMetrics.Set(chainID, cm)
	}

	// TODO setup monitor

	// create status tracker
	b.statusTracker = status.NewStatusTracker(b.cfgSet.Chains())
	b.eventSys.Register("status", b.statusTracker)

	// for each chain known to the dependency set, create the necessary DB resources
	for _, chainID := range chains {
		if err := b.openChainDBs(chainID); err != nil {
			return fmt.Errorf("failed to open chain %s: %w", chainID, err)
		}
	}

	// setup L1 connections
	if err := b.initL1(ctx, cfg); err != nil {
		return fmt.Errorf("failed to set up L1 connection: %w", err)
	}

	// Init the runtime config loading (depends on L1).
	b.initRunCfg()

	// setup L2 connections
	if err := b.initL2ELs(ctx, cfg); err != nil {
		return fmt.Errorf("failed to setup L2 execution layer connections: %w", err)
	}

	// setup rewinders
	for _, chainID := range chains {
		l1Rewinder := l1rewind.NewL1Rewinder(b.logger, chainID, b.chainDBs, b.l1Accessor)
		b.eventSys.Register(fmt.Sprintf("l1-rewinder-%s", chainID), l1Rewinder)
		b.l1Rewinders.Set(chainID, l1Rewinder)
		l2Rewinder := l2rewind.NewL2Rewinder(b.logger, chainID, b.chainDBs)
		b.eventSys.Register(fmt.Sprintf("l2-rewinder-%s", chainID), l2Rewinder)
		b.l2Rewinders.Set(chainID, l2Rewinder)
	}

	// initialize all cross-unsafe processors
	for _, chainID := range chains {
		worker := cross.NewCrossUnsafeWorker(b.logger, chainID, b.chainDBs, b.linker)
		b.eventSys.Register(fmt.Sprintf("cross-unsafe-%s", chainID), worker)
		b.crossUnsafeWorkers.Set(chainID, worker)
	}
	// initialize all cross-safe processors
	for _, chainID := range chains {
		worker := cross.NewCrossSafeWorker(b.logger, chainID, b.chainDBs, b.linker)
		b.eventSys.Register(fmt.Sprintf("cross-safe-%s", chainID), worker)
		b.crossSafeWorkers.Set(chainID, worker)
	}
	// For each chain initialize a chain processor service,
	// after cross-unsafe workers are ready to receive updates
	for _, chainID := range chains {
		logProcessor := processors.NewLogProcessor(chainID, b.chainDBs)
		chainProcessor := processors.NewChainProcessor(b.sysContext, b.logger, chainID, logProcessor, b.chainDBs)
		b.eventSys.Register(fmt.Sprintf("events-%s", chainID), chainProcessor)
		b.chainProcessors.Set(chainID, chainProcessor)
	}

	// For each chain initialize a buffer for incoming payloads
	for _, chainID := range chains {
		payloadsBuf := payloads.NewPayloads(b.sysContext, b.logger, chainID)
		b.eventSys.Register(fmt.Sprintf("payloads-%s", chainID), payloadsBuf)
		b.payloads.Set(chainID, payloadsBuf)
	}

	if err := b.initP2P(cfg); err != nil {
		return fmt.Errorf("failed to init P2P stack: %w", err)
	}

	return nil
}

func (b *Backend) initChainConfigs(ctx context.Context, cfg *config.Config) error {
	depSet, err := cfg.DependencySetSource.LoadDependencySet(ctx)
	if err != nil {
		return fmt.Errorf("failed to load dependency set: %w", err)
	}
	rollupCfgSet, err := cfg.RollupConfigSetSource.LoadRollupConfigSetV2(ctx)
	if err != nil {
		return fmt.Errorf("failed to load rollup config set: %w", err)
	}
	b.rollupConfigs = rollupCfgSet

	cfgSet := depset.FullConfigSetMerged{
		RollupConfigSet: rollupCfgSet,
		DependencySet:   depSet,
	}
	if err := cfgSet.CheckChains(); err != nil {
		return err
	}
	b.cfgSet = cfgSet

	var l1ChainID eth.ChainID
	// Sanity-check that all configured L2s share the same L1
	for _, chainID := range rollupCfgSet.Chains() {
		rollupCfg := rollupCfgSet.RollupConfig(chainID)
		got := eth.ChainIDFromBig(rollupCfg.L1ChainID)
		if l1ChainID == (eth.ChainID{}) {
			l1ChainID = got
		} else if l1ChainID != got {
			return fmt.Errorf("rollup configs refer to different L1 (%s vs %s)", l1ChainID, got)
		}
	}
	b.l1ChainID = l1ChainID

	b.linker = depset.LinkerFromConfig(cfgSet)
	return nil
}

// openChainDBs initializes all the DB resources of a specific chain.
// It is a sub-task of initResources.
func (b *Backend) openChainDBs(chainID eth.ChainID) error {
	cm, ok := b.chainMetrics.Get(chainID)
	if !ok {
		return fmt.Errorf("missing chain metrics for %s", chainID)
	}

	logDB, err := db.OpenLogDB(b.logger, chainID, b.dataDir, cm)
	if err != nil {
		return fmt.Errorf("failed to open logDB of chain %s: %w", chainID, err)
	}
	b.chainDBs.AddLogDB(chainID, logDB)

	localDB, err := db.OpenLocalDerivationDB(b.logger.New("db-kind", "local-db", "chainID", chainID), chainID, b.dataDir, cm)
	if err != nil {
		return fmt.Errorf("failed to open local derived-from DB of chain %s: %w", chainID, err)
	}
	b.chainDBs.AddLocalDerivationDB(chainID, localDB)

	crossDB, err := db.OpenCrossDerivationDB(b.logger.New("db-kind", "cross-db", "chainID", chainID), chainID, b.dataDir, cm)
	if err != nil {
		return fmt.Errorf("failed to open cross derived-from DB of chain %s: %w", chainID, err)
	}
	b.chainDBs.AddCrossDerivationDB(chainID, crossDB)

	b.chainDBs.AddCrossUnsafeTracker(chainID)

	return nil
}

func (b *Backend) initL1(ctx context.Context, cfg *config.Config) error {
	// Set the default L1 cache size to 40 minutes of L1 data.
	// Unlike op-node v1, which sets the cache to 12 hours worth of L1 data
	// (because of the sequencing window walk-back legacy constraint there).
	defaultCacheSize := 40 * 60 * 60 / 12
	l1RPC, l1RPCCfg, err := cfg.L1.Setup(ctx, b.logger, defaultCacheSize, b.m)
	if err != nil {
		return fmt.Errorf("failed to get L1 RPC client: %w", err)
	}
	// TODO: hook up L1 cache metrics
	l1Source, err := sources.NewL1Client(l1RPC, b.logger, nil, l1RPCCfg)
	if err != nil {
		return fmt.Errorf("failed to create L1 source: %w", err)
	}
	b.l1RPC = l1Source

	b.logger.Info("Checking L1 chain ID")
	// Retry, we don't want start-up to abort too eagerly
	// on a L1 connection that's only just coming online at the same time.
	l1ChainIDBig, err := retry.Do(ctx, 10, retry.Exponential(), func() (*big.Int, error) {
		result, err := l1Source.ChainID(ctx)
		if err != nil {
			b.logger.Warn("Failed to check L1 chain ID", "err", err)
		}
		return result, err
	})
	if err != nil {
		return fmt.Errorf("failed to check L1 chain ID: %w", err)
	}
	if got := eth.ChainIDFromBig(l1ChainIDBig); b.l1ChainID != got {
		return fmt.Errorf("expected L1 RPC to be connected to chain %s, but got %s", b.l1ChainID, got)
	}

	// Now that we have a connected L1 RPC on the right chain, let's try fix the rollup configs where needed.
	if err := b.fixRollupConfigs(ctx); err != nil {
		return err
	}

	beaconClient, fallbacks, err := cfg.Beacon.Setup(ctx, b.logger)
	if err != nil {
		return fmt.Errorf("failed to setup L1 Beacon API client: %w", err)
	}
	beaconCfg := sources.L1BeaconClientConfig{
		FetchAllSidecars: cfg.Beacon.ShouldFetchAllSidecars(),
	}
	b.beacon = sources.NewL1BeaconClient(beaconClient, beaconCfg, fallbacks...)

	l1Cfg := &l1access.Config{
		ConfDepth: cfg.L1ConfDepth,
		Subscribe: cfg.SynchronousProcessors,
	}
	l1Accessor := l1access.NewL1Access(b.sysContext, b.logger, l1Source, l1Cfg)
	b.eventSys.Register("l1Accessor", l1Accessor)
	b.l1Accessor = l1Accessor

	// If we are not operating synchronously (action tests),
	// set up background subscribers to pick up on L1 data as things change externally.
	if !b.synchronousProcessors {
		b.l1Accessor.SubscribeLatestHandler()
		b.l1Accessor.SubscribeFinalityHandler()
	}
	return nil
}

func (b *Backend) fixRollupConfigs(ctx context.Context) error {
	for _, chainID := range b.rollupConfigs.Chains() {
		rollupCfg := b.rollupConfigs.RollupConfig(chainID)
		if rollupCfg.Genesis.L1Time == 0 {
			b.logger.Error("The rollup-configuration is missing the new '.genesis.l1_time' attribute! "+
				"Attempting to repair that now", "l2", rollupCfg.L2ChainID)
			if err := retry.Do0(ctx, 10, retry.Exponential(), func() error {
				anchor, err := b.l1RPC.L1BlockRefByHash(ctx, rollupCfg.Genesis.L1.Hash)
				if err != nil {
					b.logger.Error("Failed to retrieve L1 anchor block of L2 chain",
						"blockID", rollupCfg.Genesis.L1, "err", err, "l2", rollupCfg.L2ChainID)
					return err
				}
				rollupCfg.Genesis.L1Time = anchor.Time
				return nil
			}); err != nil {
				return fmt.Errorf("failed to fix missing L1 timestamp config attribute with L1 RPC: %w", err)
			}
		}
	}
	return nil
}

func (b *Backend) initRunCfg() {
	v := runcfg2.NewRuntimeConfig(b.logger, b.DependencySet(), b.rollupConfigs, b.l1RPC)
	b.eventSys.Register("runcfg", v)
	b.runCfg = v
}

func (b *Backend) initL2ELs(ctx context.Context, cfg *config.Config) error {
	readELs, err := cfg.L2.ReadELs(ctx, b.logger, b.m)
	if err != nil {
		return err
	}
	for _, relClient := range readELs {
		id := rel.NextID()
		b.readEndpoints.Set(id, relClient)
	}
	engineELs, err := cfg.L2.Engines(ctx, b.logger, b.m)
	if err != nil {
		return err
	}
	for _, engineClient := range engineELs {
		id := rwel.NextID()
		b.engineEndpoints.Set(id, engineClient)
	}
	// Try to update all ELs once initially
	b.updateL2ELs()

	// TODO background service that re-tries to dial the endpoints for chainID,
	// and do not block the setup. Once identified, then we can promote them to REL/RWEL.
	return nil
}

func (b *Backend) initP2P(cfg *config.Config) (err error) {
	b.p2pMu.Lock()
	defer b.p2pMu.Unlock()
	if b.p2pNode != nil {
		panic("p2p node already initialized")
	}
	if cfg.P2P != nil && !cfg.P2P.Disabled() {
		em := b.eventSys.Register("p2p-block-receiver", nil)
		rec := p2p.NewBlockReceiver(b.logger, em, b.m)
		// Instantiate customized P2P stack, to not include req-resp and handle multi-chain properly
		b.p2pNode, err = sp2p.NewNodeP2P(b.sysContext, b.logger, cfg.P2P, rec, b.runCfg, b.m)
		if err != nil {
			return
		}
		if b.p2pNode.Dv5Udp() != nil {
			allowedNodes := &sp2p.MultiChainFilter{
				Allowed: b.DependencySet().Chains(),
				Log:     b.logger,
			}
			go b.p2pNode.DiscoveryProcess(b.sysContext, b.logger, allowedNodes, cfg.P2P.TargetPeers())
		}
	}
	return nil
}

func (b *Backend) updateL2ELs() {
	for _, id := range b.readEndpoints.Keys() {
		b.updateREL(id)
	}
	for _, id := range b.engineEndpoints.Keys() {
		b.updateRWEL(id)
	}
}

// updateREL checks the given read only endpoint, and adds it to the read-only nodes, once we identified the chain.
func (b *Backend) updateREL(id rel.ID) {
	if b.rels.Has(id) {
		b.logger.Debug("Already have set up L2 read-only RPC", "id", id)
		return
	}
	cl, ok := b.readEndpoints.Get(id)
	if !ok {
		b.logger.Error("Cannot update L2 read-only RPC, unknown id", "id", id)
		return
	}
	ctx, cancel := context.WithTimeout(b.sysContext, time.Second*10)
	defer cancel()
	chainID, err := sources.ChainID(ctx, cl)
	if err != nil {
		b.logger.Error("Failed to identify chain L2 read-only EL RPC", "index", id, "err", err)
		return
	}
	if !b.DependencySet().HasChain(chainID) {
		b.logger.Error("Chain of L2 read-only EL RPC is not part of dependency set", "chainID", chainID)
		return
	}
	rollupCfg := b.rollupConfigs.RollupConfig(chainID)
	clCfg := sources.L2ClientDefaultConfig(rollupCfg, true)
	cm, ok := b.chainMetrics.Get(chainID)
	if !ok {
		panic(fmt.Errorf("expected every chain in dependency set to have chain metrics, but %s has none", chainID))
	}
	l2CL, err := sources.NewL2Client(cl, b.logger.New("chainID", chainID, "id", id), cm, clCfg)
	if err != nil {
		b.logger.Error("Cannot create L2 read-only EL RPC", "err", err)
		return
	}
	b.readL2Clients.Set(id, l2CL)
	relNode := rel.NewREL(b.sysContext, id, l2CL, rollupCfg)
	b.rels.Set(id, relNode)
	b.eventSys.Register(id.String(), relNode)
	b.logger.Info("Connected to L2 read-only RPC", "id", id, "chainID", chainID)
}

// updateRWEL checks the given engine endpoint, and adds it to the engine nodes, once we identified the chain.
func (b *Backend) updateRWEL(id rwel.ID) {
	if b.rwels.Has(id) {
		b.logger.Debug("Already have set up L2 engine RPC", "id", id)
		return
	}
	cl, ok := b.engineEndpoints.Get(id)
	if !ok {
		b.logger.Error("Cannot update L2 engine RPC, unknown id", "id", id)
		return
	}
	ctx, cancel := context.WithTimeout(b.sysContext, time.Second*10)
	defer cancel()
	chainID, err := sources.ChainID(ctx, cl)
	if err != nil {
		b.logger.Error("Failed to identify chain L2 engine EL RPC", "index", id, "err", err)
		return
	}
	if !b.DependencySet().HasChain(chainID) {
		b.logger.Error("Chain of L2 engine EL RPC is not part of dependency set", "chainID", chainID)
		return
	}
	cm, ok := b.chainMetrics.Get(chainID)
	if !ok {
		panic(fmt.Errorf("expected every chain in dependency set to have chain metrics, but %s has none", chainID))
	}
	rollupCfg := b.rollupConfigs.RollupConfig(chainID)
	clCfg := sources.EngineClientDefaultConfig(rollupCfg)
	logger := b.logger.New("chainID", chainID, "id", id)
	l2CL, err := sources.NewEngineClient(cl, logger, cm, clCfg)
	if err != nil {
		b.logger.Error("Cannot create L2 engine EL RPC", "err", err)
		return
	}
	b.rwL2Clients.Set(id, l2CL)
	rwelNode := rwel.NewRWEL(b.sysContext, id, logger, l2CL, cm, rollupCfg)
	b.rwels.Set(id, rwelNode)
	b.eventSys.Register(id.String(), rwelNode)
	b.logger.Info("Connected to L2 engine RPC", "id", id, "chainID", chainID)

	if p, ok := b.chainProcessors.Get(chainID); ok {
		p.AddSource(&IndexingAdapter{Source: l2CL})
	}

	pipeline := derive2.NewDerivationPipeline(
		b.sysContext, b.logger, rollupCfg, b.DependencySet(),
		b.l1RPC, b.beacon, nil, l2CL, cm)
	b.pipelines.Set(pipeline.ID(), pipeline)
	b.eventSys.Register("derivation-"+id.String(), pipeline)

	// make the controller aware of the node and pipeline, so we can use it for syncing
	b.controller.AddRWEL(id, chainID)
	b.controller.AddPipeline(pipeline.ID(), chainID)
}

func (b *Backend) Start(ctx context.Context) error {
	// ensure we only start once
	if !b.started.CompareAndSwap(false, true) {
		return errAlreadyStarted
	}

	// initiate "ResumeFromLastSealedBlock" on the chains db,
	// which rewinds the database to the last block that is guaranteed to have been fully recorded
	if err := b.chainDBs.ResumeFromLastSealedBlock(); err != nil {
		return fmt.Errorf("failed to resume chains db: %w", err)
	}

	// TODO start controller background process, if not synchronous

	return nil
}

func (b *Backend) Stop(ctx context.Context) error {
	if !b.started.CompareAndSwap(true, false) {
		return errAlreadyStopped
	}
	b.logger.Info("Closing supervisor backend")

	b.sysCancel()
	defer b.eventSys.Stop()

	var result error

	// TODO stop controller background service

	b.p2pMu.Lock()
	if b.p2pNode != nil {
		if err := b.p2pNode.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close p2p node: %w", err))
		}
		// Prevent further use of p2p.
		b.p2pNode = nil
	}
	b.p2pMu.Unlock()

	for _, l2RPC := range b.readEndpoints.Values() {
		l2RPC.Close()
	}
	for _, l2RPC := range b.engineEndpoints.Values() {
		l2RPC.Close()
	}

	if b.l1Accessor != nil {
		b.l1Accessor.UnsubscribeFinalityHandler()
		b.l1Accessor.UnsubscribeLatestHandler()
	}

	if b.l1RPC != nil {
		b.l1RPC.Close()
	}

	b.chainProcessors.Clear()
	b.l1Rewinders.Clear()
	b.l2Rewinders.Clear()
	b.crossUnsafeWorkers.Clear()
	b.crossSafeWorkers.Clear()
	b.payloads.Clear()

	// close the databases
	result = errors.Join(result, b.chainDBs.Close())

	return result
}

func (b *Backend) P2P() p2p.Node {
	return b.p2pNode
}

func (b *Backend) DependencySet() depset.DependencySet {
	return b.cfgSet
}

// PullLatestL1 makes the supervisor aware of the latest L1 block. Exposed for testing purposes.
func (b *Backend) PullLatestL1(ctx context.Context) error {
	return b.l1Accessor.PullLatest(ctx)
}

// PullFinalizedL1 makes the supervisor aware of the finalized L1 block. Exposed for testing purposes.
func (b *Backend) PullFinalizedL1(ctx context.Context) error {
	return b.l1Accessor.PullFinalized(ctx)
}

// SetConfDepthL1 changes the confirmation depth of the L1 chain that is accessible to the supervisor.
func (b *Backend) SetConfDepthL1(depth uint64) {
	b.l1Accessor.SetConfDepth(depth)
}

// Rewind rolls back the state of the supervisor for the given chain.
func (b *Backend) Rewind(ctx context.Context, chain eth.ChainID, block eth.BlockID) error {
	return b.chainDBs.Rewind(chain, block)
}

func (b *Backend) EventSystem() event.System {
	return b.eventSys
}

func (b *Backend) Executor() event.Executor {
	return b.eventExec
}
