package dsl

import (
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// CrossL2Inbox returns a binding of this chain's CrossL2Inbox.
func (el *L2ELNode) CrossL2Inbox() bindings.CrossL2Inbox {
	return bindings.NewBindings[bindings.CrossL2Inbox](bindings.WithClient(el.EthClient()),
		bindings.WithTo(predeploys.CrossL2InboxAddr), bindings.WithTest(el.t))
}

// ValidateExecMessageRequiresAccessList requires that this chain's CrossL2Inbox.validateMessage
// reverts with NotInAccessList for msg when the access list does not declare it, and succeeds
// when the access list declares it.
func (el *L2ELNode) ValidateExecMessageRequiresAccessList(msg messages.Message) {
	call := el.CrossL2Inbox().ValidateMessage(msg.Identifier, eth.Bytes32(msg.PayloadHash))

	_, err := contractio.Read(call, el.ctx)
	el.require.ErrorContains(errutil.TryAddRevertReason(err), hexutil.Encode(ErrorSelector("NotInAccessList()")),
		"validateMessage must revert when the access list does not declare the message")

	accessList, err := (&txintent.ExecTrigger{Executor: predeploys.CrossL2InboxAddr, Msg: msg}).AccessList()
	el.require.NoError(err)
	_, err = contractio.Read(call, el.ctx, txplan.WithAccessList(accessList))
	el.require.NoError(err, "validateMessage must succeed when the access list declares the message")
}
