package dsl

import (
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/interop/contracts/bindings/inbox"
	"github.com/stretchr/testify/require"
)

type Message struct {
	t       helpers.Testing
	user    *DSLUser
	chain   *Chain
	message string
	emitter *EmitterContract
	inbox   *InboxContract

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
		message: message,
	}
}

func (m *Message) Emit() *Message {
	emitAction := m.emitter.EmitMessage(m.user, m.message)
	m.initTx = emitAction(m.chain)
	m.initTx.IncludeOK()
	return m
}

func (m *Message) ExecuteOn(target *Chain, execOpts ...func(*ExecuteOpts)) *Message {
	require.NotNil(m.t, m.initTx, "message must be emitted before it can be executed")
	execAction := m.inbox.Execute(m.user, m.initTx, execOpts...)
	m.execTx = execAction(target)
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
