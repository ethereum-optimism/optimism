package bindings

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	supTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
)

type CrossL2InboxFactory struct {
	BaseCallFactory
}

func NewCrossL2InboxCallFactory(opts ...CallFactoryOption) *CrossL2InboxFactory {
	return &CrossL2InboxFactory{BaseCallFactory: *NewBaseCallFactory(opts...)}
}

func (f *CrossL2InboxFactory) WithDefaultAddr() {
	f.ApplyFactoryOptions(WithTo(common.HexToAddress(predeploys.CrossL2Inbox)))
}

type CrossL2Inbox struct {
	CrossL2InboxFactory

	ValidateMessage func(identifier supTypes.Identifier, msgHash eth.Bytes32) TypedCall[any] `sol:"validateMessage"`
}

func NewCrossL2Inbox(f *CrossL2InboxFactory) *CrossL2Inbox {
	crossL2Inbox := CrossL2Inbox{CrossL2InboxFactory: *f}
	InitImpl(&crossL2Inbox)
	return &crossL2Inbox
}
