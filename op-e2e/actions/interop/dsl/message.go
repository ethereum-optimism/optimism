package dsl

import (
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/interop/contracts/bindings/inbox"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type Message struct {
	t       helpers.Testing
	user    *DSLUser
	chain   *Chain
	message string
	emitter *EmitterContract
	inbox   *InboxContract
	l1Miner *helpers.L1Miner

	initTx *GeneratedTransaction
	execTx *GeneratedTransaction
}

func NewMessage(dsl *InteropDSL, chain *Chain, emitter *EmitterContract, message string) *Message {
	return &Message{
		t:       dsl.t,
		user:    dsl.CreateUser(),
		chain:   chain,
		emitter: emitter,
		inbox:   dsl.InboxContract,
		l1Miner: dsl.Actors.L1Miner,
		message: message,
	}
}

func (m *Message) Emit() *Message {
	emitAction := m.emitter.EmitMessage(m.user, m.message)
	m.initTx = emitAction(m.chain)
	m.initTx.IncludeOK()
	return m
}

// EmitDeposit emits a message via a user deposit transaction.
func (m *Message) EmitDeposit(l1User *DSLUser) *Message {
	emitAction := m.emitter.EmitMessage(m.user, m.message)
	m.initTx = emitAction(m.chain)
	opts, _ := m.user.TransactOpts(m.chain.ChainID.ToBig())
	m.initTx.IncludeDepositOK(l1User, opts, m.l1Miner)
	return m
}

// ActEmitDeposit returns an action that emits a message via a user deposit transaction.
func (m *Message) ActEmitDeposit(l1User *DSLUser) helpers.Action {
	return func(t helpers.Testing) {
		m.EmitDeposit(l1User)
	}
}

func sanityCheckAccessList(t helpers.Testing, li types.AccessList) {
	for _, e := range li {
		if e.Address == predeploys.CrossL2InboxAddr {
			if len(e.StorageKeys) > 0 {
				return
			}
		}
	}
	t.Fatal("expected executing-message entries in access-list")
}

func (m *Message) ExecuteOn(target *Chain, execOpts ...func(*ExecuteOpts)) *Message {
	require.NotNil(m.t, m.initTx, "message must be emitted before it can be executed")
	execAction := m.inbox.Execute(m.user, m.initTx, execOpts...)
	m.execTx = execAction(target)
	sanityCheckAccessList(m.t, m.execTx.tx.AccessList())
	m.execTx.IncludeOK()
	return m
}

// ExecutePendingOn executes a message that may not have been emitted yet.
func (m *Message) ExecutePendingOn(target *Chain, pendingMessageBlockNumber uint64, execOpts ...func(*ExecuteOpts)) *Message {
	var opts []func(*ExecuteOpts)
	opts = append(opts, WithPendingMessage(m.emitter, m.chain, pendingMessageBlockNumber, 0, m.message))
	opts = append(opts, execOpts...)
	execAction := m.inbox.Execute(m.user, nil, opts...)
	m.execTx = execAction(target)
	sanityCheckAccessList(m.t, m.execTx.tx.AccessList())
	m.execTx.IncludeOK()
	return m
}

func (m *Message) CheckEmitted() {
	require.NotNil(m.t, m.initTx, "message must be emitted before it can be checked")
	m.initTx.CheckIncluded()
}

func (m *Message) CheckNotEmitted() {
	require.NotNil(m.t, m.initTx, "message must be emitted before it can be checked")
	m.initTx.CheckNotIncluded()
}

func (m *Message) CheckNotExecuted() {
	require.NotNil(m.t, m.execTx, "message must be executed before it can be checked")
	m.execTx.CheckNotIncluded()
}

func (m *Message) CheckExecuted() {
	require.NotNil(m.t, m.execTx, "message must be executed before it can be checked")
	m.execTx.CheckIncluded()
}

func (m *Message) ExecutePayload() []byte {
	return m.execTx.MessagePayload()
}

func (m *Message) ExecuteIdentifier() inbox.Identifier {
	return m.execTx.Identifier()
}
