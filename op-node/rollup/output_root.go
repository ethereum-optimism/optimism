package rollup

import (
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ComputeL2OutputRoot computes the L2 output root
func ComputeL2OutputRoot(l2OutputRootVersion eth.Bytes32, blockHash common.Hash, blockRoot common.Hash, storageRoot common.Hash) eth.Bytes32 {
	digest := crypto.Keccak256Hash(
		l2OutputRootVersion[:],
		blockRoot.Bytes(),
		storageRoot[:],
		blockHash.Bytes(),
	)
	return eth.Bytes32(digest)
}

// HashOutputRootProof computes the hash of the output root proof
func HashOutputRootProof(proof *bindings.TypesOutputRootProof) eth.Bytes32 {
	return ComputeL2OutputRoot(
		proof.Version,
		proof.StateRoot,
		proof.MessagePasserStorageRoot,
		proof.LatestBlockhash,
	)
}
