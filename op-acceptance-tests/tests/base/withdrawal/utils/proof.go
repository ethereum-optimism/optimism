package utils

import (
	"encoding/hex"
	"fmt"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Ported from op-node/withdrawals/proof.go to fit in the op-devstack, using op-service proof types
func VerifyProof(stateRoot common.Hash, proof *eth.AccountResult) error {
	balance, overflow := uint256.FromBig(proof.Balance.ToInt())
	if overflow {
		return fmt.Errorf("proof balance overflows uint256: %d", proof.Balance.ToInt())
	}
	proofHex := []string{}
	for _, p := range proof.AccountProof {
		proofHex = append(proofHex, hex.EncodeToString(p))
	}
	err := withdrawals.VerifyAccountProof(
		stateRoot,
		proof.Address,
		types.StateAccount{
			Nonce:    uint64(proof.Nonce),
			Balance:  balance,
			Root:     proof.StorageHash,
			CodeHash: proof.CodeHash[:],
		},
		proofHex,
	)
	if err != nil {
		return fmt.Errorf("failed to validate account: %w", err)
	}
	for i, storageProof := range proof.StorageProof {
		proofHex := []string{}
		for _, p := range storageProof.Proof {
			proofHex = append(proofHex, hex.EncodeToString(p))
		}
		convertedProof := gethclient.StorageResult{
			Key:   storageProof.Key.String(),
			Value: storageProof.Value.ToInt(),
			Proof: proofHex,
		}
		err = withdrawals.VerifyStorageProof(proof.StorageHash, convertedProof)
		if err != nil {
			return fmt.Errorf("failed to validate storage proof %d: %w", i, err)
		}
	}
	return nil
}
