package actions

import (
	"context"
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	gnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// L2Verifier is an actor that functions like a rollup node,
// without the full P2P/API/Node stack, but just the derivation state, and simplified driver.
type L2Verifier struct {
	log log.Logger

	eng interface {
		derive.Engine
		L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
	}

	// L2 rollup
	derivation *derive.DerivationPipeline

	l1      derive.L1Fetcher
	l1State *driver.L1State

	l2PipelineIdle bool
	l2Building     bool

	rollupCfg *rollup.Config

	rpc *rpc.Server

	failRPC error // mock error
}

type L2API interface {
	derive.Engine
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	// GetProof returns a proof of the account, it may return a nil result without error if the address was not found.
	GetProof(ctx context.Context, address common.Address, storage []common.Hash, blockTag string) (*eth.AccountResult, error)
	OutputV0AtBlock(ctx context.Context, blockHash common.Hash) (*eth.OutputV0, error)
}

func NewL2Verifier(t Testing, log log.Logger, l1 derive.L1Fetcher, eng L2API, cfg *rollup.Config, syncCfg *sync.Config) *L2Verifier {
	metrics := &testutils.TestDerivationMetrics{}
	pipeline := derive.NewDerivationPipeline(log, cfg, l1, eng, metrics, syncCfg)
	pipeline.Reset()

	rollupNode := &L2Verifier{
		log:            log,
		eng:            eng,
		derivation:     pipeline,
		l1:             l1,
		l1State:        driver.NewL1State(log, metrics),
		l2PipelineIdle: true,
		l2Building:     false,
		rollupCfg:      cfg,
		rpc:            rpc.NewServer(),
	}
	t.Cleanup(rollupNode.rpc.Stop)

	// setup RPC server for rollup node, hooked to the actor as backend
	m := &testutils.TestRPCMetrics{}
	backend := &l2VerifierBackend{verifier: rollupNode}
	apis := []rpc.API{
		{
			Namespace:     "optimism",
			Service:       node.NewNodeAPI(cfg, eng, backend, log, m),
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

func (s *L2Verifier) L2Finalized() eth.L2BlockRef {
	return s.derivation.Finalized()
}

func (s *L2Verifier) L2Safe() eth.L2BlockRef {
	return s.derivation.SafeL2Head()
}

func (s *L2Verifier) L2PendingSafe() eth.L2BlockRef {
	return s.derivation.PendingSafeL2Head()
}

func (s *L2Verifier) L2Unsafe() eth.L2BlockRef {
	return s.derivation.UnsafeL2Head()
}

func (s *L2Verifier) EngineSyncTarget() eth.L2BlockRef {
	return s.derivation.EngineSyncTarget()
}

func (s *L2Verifier) SyncStatus() *eth.SyncStatus {
	return &eth.SyncStatus{
		CurrentL1:          s.derivation.Origin(),
		CurrentL1Finalized: s.derivation.FinalizedL1(),
		HeadL1:             s.l1State.L1Head(),
		SafeL1:             s.l1State.L1Safe(),
		FinalizedL1:        s.l1State.L1Finalized(),
		UnsafeL2:           s.L2Unsafe(),
		SafeL2:             s.L2Safe(),
		FinalizedL2:        s.L2Finalized(),
		PendingSafeL2:      s.L2PendingSafe(),
		UnsafeL2SyncTarget: s.derivation.UnsafeL2SyncTarget(),
		EngineSyncTarget:   s.EngineSyncTarget(),
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
	s.derivation.Finalize(finalized)
}

// ActL2PipelineStep runs one iteration of the L2 derivation pipeline
func (s *L2Verifier) ActL2PipelineStep(t Testing) {
	if s.l2Building {
		t.InvalidAction("cannot derive new data while building L2 block")
		return
	}

	s.l2PipelineIdle = false
	err := s.derivation.Step(t.Ctx())
	if err == io.EOF || (err != nil && errors.Is(err, derive.EngineELSyncing)) {
		s.l2PipelineIdle = true
		return
	} else if err != nil && errors.Is(err, derive.NotEnoughData) {
		return
	} else if err != nil && errors.Is(err, derive.ErrReset) {
		s.log.Warn("Derivation pipeline is reset", "err", err)
		s.derivation.Reset()
		return
	} else if err != nil && errors.Is(err, derive.ErrTemporary) {
		s.log.Warn("Derivation process temporary error", "err", err)
		if errors.Is(err, sync.WrongChainErr) { // action-tests don't back off on temporary errors. Avoid a bad genesis setup from looping.
			t.Fatalf("genesis setup issue: %v", err)
		}
		return
	} else if err != nil && errors.Is(err, derive.ErrCritical) {
		t.Fatalf("derivation failed critically: %v", err)
	} else {
		return
	}
}

func (s *L2Verifier) ActL2PipelineFull(t Testing) {
	s.l2PipelineIdle = false
	for !s.l2PipelineIdle {
		s.ActL2PipelineStep(t)
	}
}

// ActL2UnsafeGossipReceive creates an action that can receive an unsafe execution payload, like gossipsub
func (s *L2Verifier) ActL2UnsafeGossipReceive(payload *eth.ExecutionPayload) Action {
	return func(t Testing) {
		s.derivation.AddUnsafePayload(payload)
	}
}
