package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
)

type Faucet interface {
	Common
	ID() ComponentID
	API() apis.Faucet
}
