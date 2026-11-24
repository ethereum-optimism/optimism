package bindings

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// UndeliveredMessageExporter binds the UndeliveredMessageExporter predeploy.
type UndeliveredMessageExporter struct {
	// Write functions
	ExportUndeliveredMessage func(sourceMessenger common.Address, source eth.ChainID, nonce *big.Int, sender common.Address, target common.Address, message []byte, minGasLimit uint32) TypedCall[eth.Bytes32] `sol:"exportUndeliveredMessage"`
}
