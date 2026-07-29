package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Faucet interface {
	Common
	ChainID() eth.ChainID
	API() apis.Faucet
}
