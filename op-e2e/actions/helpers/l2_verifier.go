package helpers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	gnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	opnodemetrics "github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/attributes"
	"github.com/ethereum-optimism/optimism/op-node/rollup/clsync"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/finality"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop/indexing"
	"github.com/ethereum-optimism/optimism/op-node/rollup/status"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/safego"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/syncnode"
)

var interopJWTSecret = [32]byte{4}

type InteropControl interface {
	PullEvents(ctx context.Context) (pulledAny bool, err error)
}

// L2Verifier is an actor that functions like a rollup node,
// without the full P2P/API/Node stack, but just the derivation state, and simplified driver.
type L2Verifier struct {
	eventSys event.System

	log log.Logger

	Eng L2API

	syncStatus driver.SyncStatusTracker

	synchronousEvents event.Emitter

	drainer event.Drainer

	// L2 rollup
	engine            *engine.EngineController
	derivationMetrics *testutils.TestDerivationMetrics
	derivation        *derive.DerivationPipeline
	syncDeriver       *driver.SyncDeriver
	finalizer         driver.Finalizer

	safeHeadListener rollup.SafeHeadListener
	syncCfg          *sync.Config

	l1 derive.L1Fetcher

	L2PipelineIdle bool
	l2Building     bool

	RollupCfg *rollup.Config

	rpc *rpc.Server

	interopSys interop.SubSystem // may be nil if interop is not active

	InteropControl InteropControl // if managed by an op-supervisor

	failRPC func(call []rpc.BatchElem) error // mock error

	// The L2Verifier actor is embedded in the L2Sequencer actor,
	// but must not be copied for the deriver-functionality to modify the same state.
	_ safego.NoCopy
}

type L2API interface {
	engine.Engine
	indexing.L2Source
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	// GetProof returns a proof of the account, it may return a nil result without error if the address was not found.
	GetProof(ctx context.Context, address common.Address, storage []common.Hash, blockTag string) (*eth.AccountResult, error)
	OutputV0AtBlock(ctx context.Context, blockHash common.Hash) (*eth.OutputV0, error)

	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
	BlockRefByNumber(ctx context.Context, num uint64) (eth.BlockRef, error)
	ChainID(ctx context.Context) (*big.Int, error)
}

type safeDB interface {
	rollup.SafeHeadListener
	node.SafeDBReader
}

func NewL2Verifier(t Testing, log log.Logger, l1 derive.L1Fetcher,
	blobsSrc derive.L1BlobsFetcher, altDASrc driver.AltDAIface,
	eng L2API, cfg *rollup.Config, depSet depset.DependencySet, syncCfg *sync.Config, safeHeadListener safeDB,
) *L2Verifier {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	executor := event.NewGlobalSynchronous(ctx)
	sys := event.NewSystem(log, executor)
	t.Cleanup(sys.Stop)
	opts := event.WithEmitLimiter(
		// TestSyncBatchType/DerivationWithFlakyL1RPC does *a lot* of quick retries
		// TestL2BatcherBatchType/ExtendedTimeWithoutL1Batches as well.
		rate.Limit(100_000),
		100_000,
		func() {
			log.Warn("Hitting events rate-limit. An events code-path may be hot-looping.")
			t.Fatal("Tests must not hot-loop events")
		},
	)

	var interopSys interop.SubSystem
	if cfg.InteropTime != nil {
		mm := indexing.NewIndexingMode(log, cfg, "127.0.0.1", 0, interopJWTSecret, l1, eng, &opmetrics.NoopRPCMetrics{})
		mm.TestDisableEventDeduplication()
		interopSys = mm
		sys.Register("interop", interopSys, opts)
		require.NoError(t, interopSys.Start(context.Background()))
		t.Cleanup(func() {
			_ = interopSys.Stop(context.Background())
		})
	}

	metrics := &testutils.TestDerivationMetrics{}
	ec := engine.NewEngineController(ctx, eng, log, opnodemetrics.NoopMetrics, cfg, syncCfg, sys.Register("engine-controller", nil, opts))

	sys.Register("engine-reset",
		engine.NewEngineResetDeriver(ctx, log, cfg, l1, eng, syncCfg), opts)

	clSync := clsync.NewCLSync(log, cfg, metrics, ec)
	sys.Register("cl-sync", clSync, opts)

	var finalizer driver.Finalizer
	if cfg.AltDAEnabled() {
		finalizer = finality.NewAltDAFinalizer(ctx, log, cfg, l1, altDASrc)
	} else {
		finalizer = finality.NewFinalizer(ctx, log, cfg, l1)
	}
	sys.Register("finalizer", finalizer, opts)

	attrHandler := attributes.NewAttributesHandler(log, cfg, ctx, eng, ec)
	sys.Register("attributes-handler", attrHandler, opts)

	indexingMode := interopSys != nil
	pipeline := derive.NewDerivationPipeline(log, cfg, depSet, l1, blobsSrc, altDASrc, eng, metrics, indexingMode)
	sys.Register("pipeline", derive.NewPipelineDeriver(ctx, pipeline), opts)

	testActionEmitter := sys.Register("test-action", nil, opts)

	syncStatusTracker := status.NewStatusTracker(log, metrics)
	sys.Register("status", syncStatusTracker, opts)

	syncDeriver := &driver.SyncDeriver{
		Derivation:     pipeline,
		SafeHeadNotifs: safeHeadListener,
		CLSync:         clSync,
		Engine:         ec,
		SyncCfg:        syncCfg,
		Config:         cfg,
		L1:             l1,
		// No need to initialize L1Tracker because no L1 block cache is used for testing
		L2:                  eng,
		Log:                 log,
		Ctx:                 ctx,
		ManagedBySupervisor: indexingMode,
	}
	// TODO(#16917) Remove Event System Refactor Comments
	//  Couple SyncDeriver and EngineController for event refactoring
	//  Couple EngDeriver and NewAttributesHandler for event refactoring
	ec.SyncDeriver = syncDeriver
	sys.Register("sync", syncDeriver, opts)
	sys.Register("engine", ec, opts)

	rollupNode := &L2Verifier{
		eventSys:          sys,
		log:               log,
		Eng:               eng,
		engine:            ec,
		derivationMetrics: metrics,
		derivation:        pipeline,
		syncDeriver:       syncDeriver,
		finalizer:         finalizer,
		safeHeadListener:  safeHeadListener,
		syncCfg:           syncCfg,
		drainer:           executor,
		l1:                l1,
		syncStatus:        syncStatusTracker,
		L2PipelineIdle:    true,
		l2Building:        false,
		RollupCfg:         cfg,
		rpc:               rpc.NewServer(),
		synchronousEvents: testActionEmitter,
		interopSys:        interopSys,
	}
	sys.Register("verifier", rollupNode, opts)

	t.Cleanup(rollupNode.rpc.Stop)

	// setup RPC server for rollup node, hooked to the actor as backend
	backend := &l2VerifierBackend{verifier: rollupNode}
	apis := []rpc.API{
		{
			Namespace:     "optimism",
			Service:       node.NewNodeAPI(cfg, depSet, eng, backend, safeHeadListener, log),
			Public:        true,
			Authenticated: false,
		},
		{
			Namespace:     "admin",
			Version:       "",
			Service:       node.NewAdminAPI(backend, log),
			Public:        true, // TODO: this field is deprecated. Do we even need this anymore?
			Authenticated: false,
		},
		{
			Namespace: "opstack",
			Service:   node.NewOpstackAPI(ec, &testutils.FakePublishAPI{Log: log}),
		},
	}
	require.NoError(t, gnode.RegisterApis(apis, nil, rollupNode.rpc), "failed to set up APIs")
	return rollupNode
}

func (v *L2Verifier) InteropSyncNode(t Testing) syncnode.SyncNode {
	require.NotNil(t, v.interopSys, "interop sub-system must be running")
	m, ok := v.interopSys.(*indexing.IndexingMode)
	require.True(t, ok, "Interop sub-system must be in managed-mode if used as sync-node")
	auth := rpc.WithHTTPAuth(gnode.NewJWTAuth(m.JWTSecret()))
	opts := []client.RPCOption{client.WithGethRPCOptions(auth)}
	cl, err := client.CheckAndDial(t.Ctx(), v.log, m.WSEndpoint(), 5*time.Second, auth)
	require.NoError(t, err)
	t.Cleanup(cl.Close)
	bCl := client.NewBaseRPCClient(cl)
	dialSetup := &syncnode.RPCDialSetup{JWTSecret: m.JWTSecret(), Endpoint: m.WSEndpoint()}
	return syncnode.NewRPCSyncNode("action-tests-l2-verifier", bCl, opts, v.log, dialSetup)
}

type l2VerifierBackend struct {
	verifier *L2Verifier
}

func (s *l2VerifierBackend) BlockRefWithStatus(ctx context.Context, num uint64) (eth.L2BlockRef, *eth.SyncStatus, error) {
	ref, err := s.verifier.Eng.L2BlockRefByNumber(ctx, num)
	return ref, s.verifier.SyncStatus(), err
}

func (s *l2VerifierBackend) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return s.verifier.SyncStatus(), nil
}

func (s *l2VerifierBackend) ResetDerivationPipeline(ctx context.Context) error {
	s.verifier.derivation.Reset()
	return nil
}

func (s *l2VerifierBackend) StartSequencer(ctx context.Context, blockHash common.Hash) error {
	return nil
}

func (s *l2VerifierBackend) StopSequencer(ctx context.Context) (common.Hash, error) {
	return common.Hash{}, errors.New("stopping the L2Verifier sequencer is not supported")
}

func (s *l2VerifierBackend) SequencerActive(ctx context.Context) (bool, error) {
	return false, nil
}

func (s *l2VerifierBackend) OverrideLeader(ctx context.Context) error {
	return nil
}

func (s *l2VerifierBackend) OnUnsafeL2Payload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) {
}

func (s *l2VerifierBackend) ConductorEnabled(ctx context.Context) (bool, error) {
	return false, nil
}

func (s *l2VerifierBackend) SetRecoverMode(ctx context.Context, mode bool) error {
	return errors.New("recover mode unsupported")
}

func (s *L2Verifier) DerivationMetricsTracer() *testutils.TestDerivationMetrics {
	return s.derivationMetrics
}

func (s *L2Verifier) L2Finalized() eth.L2BlockRef {
	return s.engine.Finalized()
}

func (s *L2Verifier) L2Safe() eth.L2BlockRef {
	return s.engine.SafeL2Head()
}

func (s *L2Verifier) L2PendingSafe() eth.L2BlockRef {
	return s.engine.PendingSafeL2Head()
}

func (s *L2Verifier) L2Unsafe() eth.L2BlockRef {
	return s.engine.UnsafeL2Head()
}

func (s *L2Verifier) L2BackupUnsafe() eth.L2BlockRef {
	return s.engine.BackupUnsafeL2Head()
}

func (s *L2Verifier) SyncStatus() *eth.SyncStatus {
	return s.syncStatus.SyncStatus()
}

func (s *L2Verifier) RollupClient() *sources.RollupClient {
	return sources.NewRollupClient(s.RPCClient())
}

func (s *L2Verifier) RPCClient() client.RPC {
	cl := rpc.DialInProc(s.rpc)
	return testutils.RPCErrFaker{
		RPC: client.NewBaseRPCClient(cl),
		ErrFn: func(call []rpc.BatchElem) error {
			if s.failRPC == nil {
				return nil
			}
			return s.failRPC(call)
		},
	}
}

// ActRPCFail makes the next L2 RPC request fail
func (s *L2Verifier) ActRPCFail(t Testing) {
	if s.failRPC != nil { // already set to fail?
		t.InvalidAction("already set a mock rpc error")
		return
	}
	s.failRPC = func(call []rpc.BatchElem) error {
		s.failRPC = nil
		return errors.New("mock RPC error")
	}
}

func (s *L2Verifier) ActL1HeadSignal(t Testing) {
	head, err := s.l1.L1BlockRefByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	s.syncStatus.OnL1Unsafe(head)
	s.syncDeriver.OnL1Unsafe(t.Ctx())
	require.Equal(t, head, s.syncStatus.SyncStatus().HeadL1)
}

func (s *L2Verifier) ActL1SafeSignal(t Testing) {
	safe, err := s.l1.L1BlockRefByLabel(t.Ctx(), eth.Safe)
	require.NoError(t, err)
	s.syncStatus.OnL1Safe(safe)
	require.Equal(t, safe, s.syncStatus.SyncStatus().SafeL1)
}

func (s *L2Verifier) ActL1FinalizedSignal(t Testing) {
	finalized, err := s.l1.L1BlockRefByLabel(t.Ctx(), eth.Finalized)
	require.NoError(t, err)
	s.syncStatus.OnL1Finalized(finalized)
	s.finalizer.OnL1Finalized(finalized)
	s.syncDeriver.OnL1Finalized(t.Ctx())
	require.Equal(t, finalized, s.syncStatus.SyncStatus().FinalizedL1)
}

func (s *L2Verifier) OnEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case rollup.L1TemporaryErrorEvent:
		s.log.Warn("L1 temporary error", "err", x.Err)
	case rollup.EngineTemporaryErrorEvent:
		s.log.Warn("Engine temporary error", "err", x.Err)
		if errors.Is(x.Err, sync.WrongChainErr) { // action-tests don't back off on temporary errors. Avoid a bad genesis setup from looping.
			panic(fmt.Errorf("genesis setup issue: %w", x.Err))
		}
	case rollup.ResetEvent:
		s.log.Warn("Derivation pipeline is being reset", "err", x.Err)
	case rollup.CriticalErrorEvent:
		panic(fmt.Errorf("derivation failed critically: %w", x.Err))
	case derive.DeriverIdleEvent:
		s.L2PipelineIdle = true
	case derive.PipelineStepEvent:
		s.L2PipelineIdle = false
	case driver.StepReqEvent:
		s.synchronousEvents.Emit(ctx, driver.StepEvent{})
	default:
		return false
	}
	return true
}

func (s *L2Verifier) ActL2EventsUntilPending(t Testing, num uint64) {
	s.ActL2EventsUntil(t, func(ev event.Event) bool {
		x, ok := ev.(engine.PendingSafeUpdateEvent)
		return ok && x.PendingSafe.Number == num
	}, 1000, false)
}

func (s *L2Verifier) ActL2EventsUntil(t Testing, fn func(ev event.Event) bool, max int, excl bool) {
	t.Helper()
	if s.l2Building {
		t.InvalidAction("cannot derive new data while building L2 block")
		return
	}
	for i := 0; i < max; i++ {
		err := s.drainer.DrainUntil(fn, excl)
		if err == nil {
			return
		}
		if err == io.EOF {
			s.synchronousEvents.Emit(t.Ctx(), driver.StepEvent{})
		}
	}
	t.Fatalf("event condition did not hit, ran maximum number of steps: %d", max)
}

func (s *L2Verifier) ActL2PipelineFull(t Testing) {
	s.synchronousEvents.Emit(t.Ctx(), driver.StepEvent{})
	require.NoError(t, s.drainer.Drain(), "complete all event processing triggered by deriver step")
}

// ActL2UnsafeGossipReceive creates an action that can receive an unsafe execution payload, like gossipsub
func (s *L2Verifier) ActL2UnsafeGossipReceive(payload *eth.ExecutionPayloadEnvelope) Action {
	return func(t Testing) {
		s.synchronousEvents.Emit(t.Ctx(), clsync.ReceivedUnsafePayloadEvent{Envelope: payload})
	}
}

// ActL2InsertUnsafePayload creates an action that can insert an unsafe execution payload
func (s *L2Verifier) ActL2InsertUnsafePayload(payload *eth.ExecutionPayloadEnvelope) Action {
	return func(t Testing) {
		ref, err := derive.PayloadToBlockRef(s.RollupCfg, payload.ExecutionPayload)
		require.NoError(t, err)
		err = s.engine.InsertUnsafePayload(t.Ctx(), payload, ref)
		require.NoError(t, err)
	}
}

func (s *L2Verifier) SyncSupervisor(t Testing) {
	require.NotNil(t, s.InteropControl, "must be managed by op-supervisor")
	_, err := s.InteropControl.PullEvents(t.Ctx())
	require.NoError(t, err)
}
