package loadtest

import (
	"math/big"
	"slices"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

type Relayer interface {
	Relay(t devtest.T, msgs []types.Message)
}

type ValidRelayer struct {
	el  *dsl.L2ELNode
	eoa *dsl.EOA
	// supervisor is used to check if executing messages are cross-safe.
	supervisor *dsl.Supervisor

	counter *nonceCounter
}

func NewValidRelayer(funder *dsl.Funder, el *dsl.L2ELNode, supervisor *dsl.Supervisor) *ValidRelayer {
	return &ValidRelayer{
		el:         el,
		eoa:        funder.NewFundedEOA(eth.MillionEther),
		supervisor: supervisor,
		counter:    new(nonceCounter),
	}
}

func (e *ValidRelayer) Relay(t devtest.T, msgs []types.Message) {
	tx := buildRelayTx(e.eoa, e.el, msgs, retryForever(e.el.Escape().EthClient()), txplan.WithStaticNonce(e.counter.Next()))
	receipt, err := tx.Included.Eval(t.Ctx())
	t.Require().NoError(err)
	_, err = tx.Success.Eval(t.Ctx())
	t.Require().NoError(err)
	t.Require().Len(receipt.Logs, len(msgs))

	// Wait for the transaction to be cross-safe.
	includedBlock, err := tx.IncludedBlock.Eval(t.Ctx())
	t.Require().NoError(err)
	for {
		// NOTE: it may be desirable to query proxyd instead of the supervisor if/when the devstack supports it.
		crossSafeID, err := e.supervisor.Escape().QueryAPI().CrossSafe(t.Ctx(), e.el.ChainID())
		t.Require().NoError(err)
		if includedBlock.ID().Number <= crossSafeID.Derived.Number {
			break
		}
		e.el.WaitForBlock()
	}
	// Sanity check that includedBlock is still in the canonical chain.
	_, err = e.el.Escape().EthClient().BlockRefByHash(t.Ctx(), includedBlock.Hash)
	t.Require().NoError(err)
}

// DelayedRelayer executes messages after waiting for a specified period.
type DelayedRelayer struct {
	e     *ValidRelayer
	delay time.Duration
	wg    *sync.WaitGroup
}

func NewDelayedRelayer(e *ValidRelayer, wg *sync.WaitGroup, delay time.Duration) *DelayedRelayer {
	return &DelayedRelayer{
		e:     e,
		wg:    wg,
		delay: delay,
	}
}

func (de *DelayedRelayer) Relay(t devtest.T, msgs []types.Message) {
	de.wg.Add(1)
	go func() {
		defer de.wg.Done()
		select {
		case <-t.Ctx().Done():
		case <-time.After(de.delay):
			de.e.Relay(t, msgs)
		}
	}()
}

type ToInvalidMsgFn func(types.Message) types.Message

type InvalidRelayer struct {
	eoa         *dsl.EOA
	el          *dsl.L2ELNode
	makeInvalid ToInvalidMsgFn
}

func NewInvalidRelayer(funder *dsl.Funder, el *dsl.L2ELNode, makeInvalid ToInvalidMsgFn) *InvalidRelayer {
	return &InvalidRelayer{
		el:          el,
		eoa:         funder.NewFundedEOA(eth.MillionEther),
		makeInvalid: makeInvalid,
	}
}

func (ie *InvalidRelayer) Relay(t devtest.T, validMsgs []types.Message) {
	msgs := make([]types.Message, len(validMsgs))
	copy(msgs, validMsgs)
	// Replace the last message with the invalid message.
	// Merely appending can cause us to hit tx size limits if the original slice has a lot of messages.
	msgs = append(msgs[:len(msgs)-1], ie.makeInvalid(msgs[len(msgs)-1]))
	tx := buildRelayTx(ie.eoa, ie.el, msgs)
	_, err := tx.Submitted.Eval(t.Ctx())
	t.Require().ErrorContains(err, core.ErrTxFilteredOut.Error())
}

func makeInvalidChainID(msg types.Message) types.Message {
	bigChainID := msg.Identifier.ChainID.ToBig()
	bigChainID.Add(bigChainID, big.NewInt(1))
	msg.Identifier.ChainID = eth.ChainIDFromBig(bigChainID)
	return msg
}

func makeInvalidOrigin(msg types.Message) types.Message {
	bigOrigin := msg.Identifier.Origin.Big()
	bigOrigin.Add(bigOrigin, big.NewInt(1))
	msg.Identifier.Origin = common.BigToAddress(bigOrigin)
	return msg
}

func makeInvalidBlockNumber(msg types.Message) types.Message {
	msg.Identifier.BlockNumber++
	return msg
}

func makeInvalidLogIndex(msg types.Message) types.Message {
	msg.Identifier.LogIndex++
	return msg
}

func makeInvalidTimestamp(msg types.Message) types.Message {
	msg.Identifier.Timestamp++
	return msg
}

func makeInvalidPayloadHash(msg types.Message) types.Message {
	bigPayloadHash := msg.PayloadHash.Big()
	bigPayloadHash.Add(bigPayloadHash, big.NewInt(1))
	msg.PayloadHash = common.BigToHash(bigPayloadHash)
	return msg
}

func buildRelayTx(eoa *dsl.EOA, el *dsl.L2ELNode, msgs []types.Message, opts ...txplan.Option) *txplan.PlannedTx {
	execCalls := make([]txintent.Call, 0, len(msgs))
	for _, msg := range msgs {
		execCalls = append(execCalls, &txintent.ExecTrigger{
			Executor: constants.CrossL2Inbox,
			Msg:      msg,
		})
	}
	tx := txintent.NewIntent[*txintent.MultiTrigger, txintent.Result](eoa.Plan(), txplan.Combine(opts...))
	tx.Content.Set(&txintent.MultiTrigger{
		Emitter: constants.MultiCall3,
		Calls:   execCalls,
	})

	maxTimestamp := slices.MaxFunc(msgs, func(x, y types.Message) int {
		if x.Identifier.Timestamp > y.Identifier.Timestamp {
			return 1
		} else if x.Identifier.Timestamp < y.Identifier.Timestamp {
			return -1
		}
		return 0
	}).Identifier.Timestamp
	// The relay tx is invalid until we know it will be included at a higher timestamp than any of the initiating messages, modulo reorgs.
	// NOTE: this should be `<`, but the mempool filtering in op-geth currently uses the unsafe head's timestamp instead of
	// the pending timestamp. See https://github.com/ethereum-optimism/op-geth/issues/603.
	for el.BlockRefByLabel(eth.Unsafe).Time <= maxTimestamp {
		el.WaitForBlock()
	}
	return tx.PlannedTx
}
