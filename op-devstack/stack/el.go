package stack

import (
	"time"

	"github.com/HashKeyChain/verse/op-service/apis"
	"github.com/HashKeyChain/verse/op-service/eth"
)

type ELNode interface {
	Common
	ChainID() eth.ChainID
	EthClient() apis.EthClient
	TransactionTimeout() time.Duration
}
