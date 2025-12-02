package transactions

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/holiman/uint256"
)

var (
	emptyBlob       *kzg4844.Blob
	emptyBlobCommit kzg4844.Commitment
	emptyBlobProof  kzg4844.Proof
)

func init() {
	var err error
	emptyBlob = &kzg4844.Blob{}
	emptyBlobCommit, err = kzg4844.BlobToCommitment(emptyBlob)
	if err != nil {
		panic("failed to create empty blob commitment: " + err.Error())
	}
	emptyBlobProof, err = kzg4844.ComputeBlobProof(emptyBlob, emptyBlobCommit)
	if err != nil {
		panic("failed to create empty blob proof: " + err.Error())
	}
}

// with thanks to fjl
// https://github.com/ethereum/go-ethereum/commit/2a6beb6a39d7cb3c5906dd4465d65da6efcc73cd
func CreateEmptyBlobTx(withSidecar bool, chainID uint64) *types.BlobTx {
	sidecar := &types.BlobTxSidecar{
		Blobs:       []kzg4844.Blob{*emptyBlob},
		Commitments: []kzg4844.Commitment{emptyBlobCommit},
		Proofs:      []kzg4844.Proof{emptyBlobProof},
	}
	blobTx := &types.BlobTx{
		ChainID:    uint256.NewInt(chainID),
		Nonce:      0,
		GasTipCap:  uint256.NewInt(2200000000000),
		GasFeeCap:  uint256.NewInt(5000000000000),
		Gas:        25000,
		To:         common.Address{0x03, 0x04, 0x05},
		Value:      uint256.NewInt(99),
		Data:       make([]byte, 50),
		BlobFeeCap: uint256.NewInt(150000000000),
		BlobHashes: sidecar.BlobHashes(),
	}
	if withSidecar {
		blobTx.Sidecar = sidecar
	}
	return blobTx
}
