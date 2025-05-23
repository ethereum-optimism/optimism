package loadtest

import (
	"math/rand"

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

var rng = rand.New(rand.NewSource(1234))

type Initiator interface {
	Initiate(t devtest.T) []types.Message
}

type ManyMsgsInitiator struct {
	el          *dsl.L2ELNode
	eoa         *dsl.EOA
	eventLogger common.Address
	counter     *nonceCounter
}

func NewManyMsgsInitiator(funder *dsl.Funder, el *dsl.L2ELNode, eventLogger common.Address) *ManyMsgsInitiator {
	return &ManyMsgsInitiator{
		eoa:         funder.NewFundedEOA(eth.MillionEther),
		el:          el,
		eventLogger: eventLogger,
		counter:     new(nonceCounter),
	}
}

func (in *ManyMsgsInitiator) Initiate(t devtest.T) []types.Message {
	const numMsgs = 275 // About the max number of msgs we can create before hitting tx size limits.
	initCalls := make([]txintent.Call, 0, numMsgs)
	for range numMsgs {
		initCalls = append(initCalls, interop.RandomInitTrigger(rng, in.eventLogger, rng.Intn(5), rng.Intn(10)))
	}
	return buildAndSendInitTx(t, in.eoa, in.el, &txintent.MultiTrigger{
		Emitter: constants.MultiCall3,
		Calls:   initCalls,
	}, txplan.WithStaticNonce(in.counter.Next()))
}

type LargeMsgInitiator struct {
	eoa         *dsl.EOA
	el          *dsl.L2ELNode
	eventLogger common.Address
	counter     *nonceCounter
}

func NewLargeMsgInitiator(funder *dsl.Funder, el *dsl.L2ELNode, eventLogger common.Address) *LargeMsgInitiator {
	return &LargeMsgInitiator{
		eoa:         funder.NewFundedEOA(eth.MillionEther),
		el:          el,
		eventLogger: eventLogger,
		counter:     new(nonceCounter),
	}
}

func (lin *LargeMsgInitiator) Initiate(t devtest.T) []types.Message {
	// TODO(#16039): can we create an even larger event without the event logger?
	return buildAndSendInitTx(t, lin.eoa, lin.el, interop.RandomInitTrigger(rng, lin.eventLogger, 4, 75_000), txplan.WithStaticNonce(lin.counter.Next()))
}

func buildAndSendInitTx(t devtest.T, eoa *dsl.EOA, el *dsl.L2ELNode, initCall txintent.Call, opts ...txplan.Option) []types.Message {
	initMsgsTx := txintent.NewIntent[txintent.Call, *txintent.InteropOutput](eoa.Plan(), retryForever(el.Escape().EthClient()), txplan.Combine(opts...))
	initMsgsTx.Content.Set(initCall)
	_, err := initMsgsTx.PlannedTx.Success.Eval(t.Ctx())
	t.Require().NoError(err)
	out, err := initMsgsTx.Result.Eval(t.Ctx())
	t.Require().NoError(err)
	return out.Entries
}
