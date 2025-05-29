package loadtest

import (
	"math/rand"
	"sync"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
)

type Initiator interface {
	Initiate(t devtest.T) []types.Message
}

type ManyMsgsInitiator struct {
	el          *dsl.L2ELNode
	eoa         *dsl.EOA
	eventLogger common.Address
	counter     *nonceCounter
	rngMu       sync.Mutex
	rng         *rand.Rand
}

func NewManyMsgsInitiator(funder *dsl.Funder, el *dsl.L2ELNode, eventLogger common.Address) *ManyMsgsInitiator {
	return &ManyMsgsInitiator{
		eoa:         funder.NewFundedEOA(eth.MillionEther),
		el:          el,
		eventLogger: eventLogger,
		counter:     new(nonceCounter),
		rng:         rand.New(rand.NewSource(1234)),
	}
}

func (in *ManyMsgsInitiator) randomInitTriggers() []txintent.Call {
	const numMsgs = 275 // About the max number of msgs we can create before hitting tx size limits.
	initCalls := make([]txintent.Call, 0, numMsgs)
	in.rngMu.Lock()
	defer in.rngMu.Unlock()
	for range numMsgs {
		initCalls = append(initCalls, interop.RandomInitTrigger(in.rng, in.eventLogger, in.rng.Intn(5), in.rng.Intn(10)))
	}
	return initCalls
}

func (in *ManyMsgsInitiator) Initiate(t devtest.T) []types.Message {
	return buildAndSendInitTx(t, in.eoa, in.el, &txintent.MultiTrigger{
		Emitter: constants.MultiCall3,
		Calls:   in.randomInitTriggers(),
	}, txplan.WithStaticNonce(in.counter.Next()))
}

type LargeMsgInitiator struct {
	eoa         *dsl.EOA
	el          *dsl.L2ELNode
	eventLogger common.Address
	counter     *nonceCounter
	rngMu       sync.Mutex
	rng         *rand.Rand
}

func NewLargeMsgInitiator(funder *dsl.Funder, el *dsl.L2ELNode, eventLogger common.Address) *LargeMsgInitiator {
	return &LargeMsgInitiator{
		eoa:         funder.NewFundedEOA(eth.MillionEther),
		el:          el,
		eventLogger: eventLogger,
		counter:     new(nonceCounter),
		rng:         rand.New(rand.NewSource(1234)),
	}
}

func (lin *LargeMsgInitiator) randomInitTrigger() *txintent.InitTrigger {
	lin.rngMu.Lock()
	defer lin.rngMu.Unlock()
	return interop.RandomInitTrigger(lin.rng, lin.eventLogger, 4, 75_000)
}

func (lin *LargeMsgInitiator) Initiate(t devtest.T) []types.Message {
	// TODO(#16039): can we create an even larger event without the event logger?
	return buildAndSendInitTx(t, lin.eoa, lin.el, lin.randomInitTrigger(), txplan.WithStaticNonce(lin.counter.Next()))
}

func buildAndSendInitTx(t devtest.T, eoa *dsl.EOA, el *dsl.L2ELNode, initCall txintent.Call, opts ...txplan.Option) []types.Message {
	initMsgsTx := txintent.NewIntent[txintent.Call, *txintent.InteropOutput](
		eoa.Plan(),
		retrySubmissionForever(el.Escape().EthClient()),
		retryInclusionForever(el.Escape().EthClient()),
		txplan.Combine(opts...),
	)
	initMsgsTx.Content.Set(initCall)
	_, err := initMsgsTx.PlannedTx.Success.Eval(t.Ctx())
	t.Require().NoError(err)
	out, err := initMsgsTx.Result.Eval(t.Ctx())
	t.Require().NoError(err)
	return out.Entries
}
