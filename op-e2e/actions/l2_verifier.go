package actions

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/attributes"
	"github.com/ethereum-optimism/optimism/op-node/rollup/clsync"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/finality"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/safego"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	gnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// L2Verifier is an actor that functions like a rollup node,
// without the full P2P/API/Node stack, but just the derivation state, and simplified driver.
type L2Verifier struct {
	log log.Logger

	eng interface {
		engine.Engine
		L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
	}

	syncDeriver *driver.SyncDeriver

	// L2 rollup
	engine     *engine.EngineController
	derivation *derive.DerivationPipeline
	clSync     *clsync.CLSync

	attributesHandler driver.AttributesHandler
	safeHeadListener  rollup.SafeHeadListener
	finalizer         driver.Finalizer
	syncCfg           *sync.Config

	l1      derive.L1Fetcher
	l1State *driver.L1State

	l2PipelineIdle bool
	l2Building     bool

	rollupCfg *rollup.Config

	rpc *rpc.Server

	failRPC error // mock error

	// The L2Verifier actor is embedded in the L2Sequencer actor,
	// but must not be copied for the deriver-functionality to modify the same state.
	_ safego.NoCopy
}

type L2API interface {
	engine.Engine
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	// GetProof returns a proof of the account, it may return a nil result without error if the address was not found.
	GetProof(ctx context.Context, address common.Address, storage []common.Hash, blockTag string) (*eth.AccountResult, error)
	OutputV0AtBlock(ctx context.Context, blockHash common.Hash) (*eth.OutputV0, error)
}

type safeDB interface {
	rollup.SafeHeadListener
	node.SafeDBReader
}

func NewL2Verifier(t Testing, log log.Logger, l1 derive.L1Fetcher, blobsSrc derive.L1BlobsFetcher, plasmaSrc driver.PlasmaIface, eng L2API, cfg *rollup.Config, syncCfg *sync.Config, safeHeadListener safeDB) *L2Verifier {
	metrics := &testutils.TestDerivationMetrics{}
	ec := engine.NewEngineController(eng, log, metrics, cfg, syncCfg.SyncMode)

	clSync := clsync.NewCLSync(log, cfg, metrics, ec)

	var finalizer driver.Finalizer
	if cfg.PlasmaEnabled() {
		finalizer = finality.NewPlasmaFinalizer(log, cfg, l1, ec, plasmaSrc)
	} else {
		finalizer = finality.NewFinalizer(log, cfg, l1, ec)
	}

	attributesHandler := attributes.NewAttributesHandler(log, cfg, ec, eng)

	pipeline := derive.NewDerivationPipeline(log, cfg, l1, blobsSrc, plasmaSrc, eng, metrics)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	rootDeriver := &rollup.SynchronousDerivers{}
	synchronousEvents := driver.NewSynchronousEvents(log, ctx, rootDeriver)

	syncDeriver := &driver.SyncDeriver{
		Derivation:        pipeline,
		Finalizer:         finalizer,
		AttributesHandler: attributesHandler,
		SafeHeadNotifs:    safeHeadListener,
		CLSync:            clSync,
		Engine:            ec,
		SyncCfg:           syncCfg,
		Config:            cfg,
		L1:                l1,
		L2:                eng,
		Emitter:           synchronousEvents,
		Log:               log,
		Ctx:               ctx,
		Drain:             synchronousEvents.Drain,
	}

	engDeriv := engine.NewEngDeriver(log, ctx, cfg, ec, synchronousEvents)

	rollupNode := &L2Verifier{
		log:               log,
		eng:               eng,
		engine:            ec,
		clSync:            clSync,
		derivation:        pipeline,
		finalizer:         finalizer,
		attributesHandler: attributesHandler,
		safeHeadListener:  safeHeadListener,
		syncCfg:           syncCfg,
		syncDeriver:       syncDeriver,
		l1:                l1,
		l1State:           driver.NewL1State(log, metrics),
		l2PipelineIdle:    true,
		l2Building:        false,
		rollupCfg:         cfg,
		rpc:               rpc.NewServer(),
	}

	*rootDeriver = rollup.SynchronousDerivers{
		syncDeriver,
		engDeriv,
		rollupNode,
	}

	t.Cleanup(rollupNode.rpc.Stop)

	// setup RPC server for rollup node, hooked to the actor as backend
	m := &testutils.TestRPCMetrics{}
	backend := &l2VerifierBackend{verifier: rollupNode}
	apis := []rpc.API{
		{
			Namespace:     "optimism",
			Service:       node.NewNodeAPI(cfg, eng, backend, safeHeadListener, log, m),
			Public:        true,
			Authenticated: false,
		},
		{
			Namespace:     "admin",
			Version:       "",
			Service:       node.NewAdminAPI(backend, m, log),
			Public:        true, // TODO: this field is deprecated. Do we even need this anymore?
			Authenticated: false,
		},
	}
	require.NoError(t, gnode.RegisterApis(apis, nil, rollupNode.rpc), "failed to set up APIs")
	return rollupNode
}

type l2VerifierBackend struct {
	verifier *L2Verifier
}

func (s *l2VerifierBackend) BlockRefWithStatus(ctx context.Context, num uint64) (eth.L2BlockRef, *eth.SyncStatus, error) {
	ref, err := s.verifier.eng.L2BlockRefByNumber(ctx, num)
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

func (s *l2VerifierBackend) OnUnsafeL2Payload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
	return nil
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
	return &eth.SyncStatus{
		CurrentL1:          s.derivation.Origin(),
		CurrentL1Finalized: s.finalizer.FinalizedL1(),
		HeadL1:             s.l1State.L1Head(),
		SafeL1:             s.l1State.L1Safe(),
		FinalizedL1:        s.l1State.L1Finalized(),
		UnsafeL2:           s.L2Unsafe(),
		SafeL2:             s.L2Safe(),
		FinalizedL2:        s.L2Finalized(),
		PendingSafeL2:      s.L2PendingSafe(),
	}
}

func (s *L2Verifier) RollupClient() *sources.RollupClient {
	return sources.NewRollupClient(s.RPCClient())
}

func (s *L2Verifier) RPCClient() client.RPC {
	cl := rpc.DialInProc(s.rpc)
	return testutils.RPCErrFaker{
		RPC: client.NewBaseRPCClient(cl),
		ErrFn: func() error {
			err := s.failRPC
			s.failRPC = nil // reset back, only error once.
			return err
		},
	}
}

// ActRPCFail makes the next L2 RPC request fail
func (s *L2Verifier) ActRPCFail(t Testing) {
	if s.failRPC != nil { // already set to fail?
		t.InvalidAction("already set a mock rpc error")
		return
	}
	s.failRPC = errors.New("mock RPC error")
}

func (s *L2Verifier) ActL1HeadSignal(t Testing) {
	head, err := s.l1.L1BlockRefByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	s.l1State.HandleNewL1HeadBlock(head)
}

func (s *L2Verifier) ActL1SafeSignal(t Testing) {
	safe, err := s.l1.L1BlockRefByLabel(t.Ctx(), eth.Safe)
	require.NoError(t, err)
	s.l1State.HandleNewL1SafeBlock(safe)
}

func (s *L2Verifier) ActL1FinalizedSignal(t Testing) {
	finalized, err := s.l1.L1BlockRefByLabel(t.Ctx(), eth.Finalized)
	require.NoError(t, err)
	s.l1State.HandleNewL1FinalizedBlock(finalized)
	s.finalizer.Finalize(t.Ctx(), finalized)
}

func (s *L2Verifier) OnEvent(ev rollup.Event) {
	switch x := ev.(type) {
	case rollup.EngineTemporaryErrorEvent:
		s.log.Warn("Derivation process temporary error", "err", x.Err)
		if errors.Is(x.Err, sync.WrongChainErr) { // action-tests don't back off on temporary errors. Avoid a bad genesis setup from looping.
			panic(fmt.Errorf("genesis setup issue: %v", x.Err))
		}
	case rollup.ResetEvent:
		s.log.Warn("Derivation pipeline is being reset", "err", x.Err)
	case rollup.CriticalErrorEvent:
		panic(fmt.Errorf("derivation failed critically: %v", x.Err))
	case driver.DeriverIdleEvent:
		s.l2PipelineIdle = true
	}
}

// ActL2PipelineStep runs one iteration of the L2 derivation pipeline
func (s *L2Verifier) ActL2PipelineStep(t Testing) {
	if s.l2Building {
		t.InvalidAction("cannot derive new data while building L2 block")
		return
	}
	s.syncDeriver.Emitter.Emit(driver.StepEvent{})
	require.NoError(t, s.syncDeriver.Drain(), "complete all event processing triggered by deriver step")
}

func (s *L2Verifier) ActL2PipelineFull(t Testing) {
	s.l2PipelineIdle = false
	i := 0
	for !s.l2PipelineIdle {
		i += 1
		// Some tests do generate a lot of derivation steps
		// (e.g. thousand blocks span-batch, or deep reorgs).
		// Hence we set the sanity limit to something really high.
		if i > 10_000 {
			t.Fatalf("ActL2PipelineFull running for too long. Is a deriver looping?")
		}
		s.ActL2PipelineStep(t)
	}
}

// ActL2UnsafeGossipReceive creates an action that can receive an unsafe execution payload, like gossipsub
func (s *L2Verifier) ActL2UnsafeGossipReceive(payload *eth.ExecutionPayloadEnvelope) Action {
	return func(t Testing) {
		s.clSync.AddUnsafePayload(payload)
	}
}

// ActL2InsertUnsafePayload creates an action that can insert an unsafe execution payload
func (s *L2Verifier) ActL2InsertUnsafePayload(payload *eth.ExecutionPayloadEnvelope) Action {
	return func(t Testing) {
		ref, err := derive.PayloadToBlockRef(s.rollupCfg, payload.ExecutionPayload)
		require.NoError(t, err)
		err = s.engine.InsertUnsafePayload(t.Ctx(), payload, ref)
		require.NoError(t, err)
	}
}
