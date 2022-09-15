package rollup

import (
	"bytes"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func ComputeL2OutputRoot(l2OutputRootVersion eth.Bytes32, blockHash common.Hash, blockRoot common.Hash, storageRoot common.Hash) eth.Bytes32 {
	var buf bytes.Buffer
	buf.Write(l2OutputRootVersion[:])
	buf.Write(blockRoot.Bytes())
	buf.Write(storageRoot[:])
	buf.Write(blockHash.Bytes())
	return eth.Bytes32(crypto.Keccak256Hash(buf.Bytes()))
}
