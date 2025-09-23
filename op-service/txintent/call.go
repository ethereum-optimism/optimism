package txintent

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Call expresses minimal representation to plan transaction to write.
type Call interface {
	To() (*common.Address, error)
	AccessList() (types.AccessList, error)
	Input
}

type Input interface {
	EncodeInput() ([]byte, error)
}

// CallView expresses minimal representation to plan transaction to view, embedding Call interface.
// It is typed for interpreting the read result, and binds client for viewing.
type CallView[O any] interface {
	Call
	Output[O]
	Client() apis.EthClient
}

type Output[O any] interface {
	DecodeOutput(data []byte) (dest O, err error)
}
