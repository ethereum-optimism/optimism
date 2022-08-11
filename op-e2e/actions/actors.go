package actions

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type ActionStatus uint

const (
	ActionOK ActionStatus = iota
	ActionInvalid
)

type TestingBase interface {
	Cleanup(func())
	Error(args ...any)
	Errorf(format string, args ...any)
	Fail()
	FailNow()
	Failed() bool
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Helper()
	Log(args ...any)
	Logf(format string, args ...any)
	Name() string
	Setenv(key, value string)
	Skip(args ...any)
	SkipNow()
	Skipf(format string, args ...any)
	Skipped() bool
	TempDir() string
}

type StandardTesting struct {
	TestingBase
	ctx   context.Context
	state ActionStatus
}

func (st *StandardTesting) Ctx() context.Context {
	return st.ctx
}

func (st *StandardTesting) InvalidAction(format string, args ...any) {
	st.TestingBase.Helper()
	st.Errorf("invalid action err: "+format, args...)
	st.state = ActionInvalid
	return
}

func (st *StandardTesting) Reset(actionCtx context.Context) {
	st.state = ActionOK
	st.ctx = actionCtx
}

func (st *StandardTesting) State() ActionStatus {
	return st.state
}

type Testing interface {
	TestingBase
	Ctx() context.Context
	InvalidAction(format string, args ...any) // indicates the failure is due to action incompatibility, does not stop the test
}

var _ Testing = (*StandardTesting)(nil)

type OutputRootAPI interface {
	OutputAtBlock(ctx context.Context, number rpc.BlockNumber) ([]eth.Bytes32, error)
}

type SyncStatusAPI interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
}

type BlocksAPI interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

type L1TXAPI interface {
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
}

type Action func(t Testing)

type ActorL1Replica interface {
	actL1Sync(t Testing)
	actL1RewindToParent(t Testing)
	actL1RPCFail(t Testing)
}

type ActorL1Miner interface {
	actL1StartBlock(t Testing)
	actL1IncludeTx(from common.Address) Action
	actL1EndBlock(t Testing)
	actL1FinalizeNext(t Testing)
	actL1SafeNext(t Testing)
}

type ActorL2Batcher interface {
	actL2BatchBuffer(t Testing)
	actL2BatchSubmit(t Testing)
}

type ActorL2Proposer interface {
	actProposeOutputRoot(t Testing)
}

type ActorL2Engine interface {
	actL2IncludeTx(t Testing)
	actL2RPCFail(t Testing)
	// TODO snap syncing action things
}

type ActorL2Verifier interface {
	actL2PipelineStep(t Testing)
	actL2UnsafeGossipReceive(t Testing)
}

type ActorL2Sequencer interface {
	ActorL2Verifier
	actL2StartBlock(t Testing)
	actL2EndBlock(t Testing)
	actL2TryKeepL1Origin(t Testing)
	actL2UnsafeGossipFail(t Testing)
}

type ActorUser interface {
	actL1Deposit(t Testing)
	actL1AddTx(t Testing)
	actL2AddTx(t Testing)
	// TODO withdrawal tx
}

// TODO: action to sync/propagate tx pool on L1/L2 between replica and miner
