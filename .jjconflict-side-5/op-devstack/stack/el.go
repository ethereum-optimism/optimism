package stack

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type ELNode interface {
	Common
	ID() ComponentID
	ChainID() eth.ChainID
	EthClient() apis.EthClient
	TransactionTimeout() time.Duration
}
