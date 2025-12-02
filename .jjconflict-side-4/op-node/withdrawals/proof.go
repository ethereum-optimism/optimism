package withdrawals

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type proofDB struct {
	m map[string][]byte
}

func (p *proofDB) Has(key []byte) (bool, error) {
	_, ok := p.m[string(key)]
	return ok, nil
}

func (p *proofDB) Get(key []byte) ([]byte, error) {
	v, ok := p.m[string(key)]
	if !ok {
		return nil, errors.New("not found")
	}
	return v, nil
}

func GenerateProofDB(proof []string) *proofDB {
	p := proofDB{m: make(map[string][]byte)}
	for _, s := range proof {
		value := common.FromHex(s)
		key := crypto.Keccak256(value)
		p.m[string(key)] = value
	}
	return &p
}

func VerifyAccountProof(root common.Hash, address common.Address, account types.StateAccount, proof []string) error {
	expected, err := rlp.EncodeToBytes(&account)
	if err != nil {
		return fmt.Errorf("failed to encode rlp: %w", err)
	}
	secureKey := crypto.Keccak256(address[:])
	db := GenerateProofDB(proof)
	value, err := trie.VerifyProof(root, secureKey, db)
	if err != nil {
		return fmt.Errorf("failed to verify proof: %w", err)
	}

	if bytes.Equal(value, expected) {
		return nil
	} else {
		return errors.New("proved value is not the same as the expected value")
	}
}

func VerifyStorageProof(root common.Hash, proof gethclient.StorageResult) error {
	secureKey := crypto.Keccak256(common.FromHex(proof.Key))
	db := GenerateProofDB(proof.Proof)
	value, err := trie.VerifyProof(root, secureKey, db)
	if err != nil {
		return fmt.Errorf("failed to verify proof: %w", err)
	}

	expected := proof.Value.Bytes()
	if bytes.Equal(value, expected) {
		return nil
	} else {
		return errors.New("proved value is not the same as the expected value")
	}
}

func VerifyProof(stateRoot common.Hash, proof *gethclient.AccountResult) error {
	balance, overflow := uint256.FromBig(proof.Balance)
	if overflow {
		return fmt.Errorf("proof balance overflows uint256: %d", proof.Balance)
	}
	err := VerifyAccountProof(
		stateRoot,
		proof.Address,
		types.StateAccount{
			Nonce:    proof.Nonce,
			Balance:  balance,
			Root:     proof.StorageHash,
			CodeHash: proof.CodeHash[:],
		},
		proof.AccountProof,
	)
	if err != nil {
		return fmt.Errorf("failed to validate account: %w", err)
	}
	for i, storageProof := range proof.StorageProof {
		err = VerifyStorageProof(proof.StorageHash, storageProof)
		if err != nil {
			return fmt.Errorf("failed to validate storage proof %d: %w", i, err)
		}
	}
	return nil
}
