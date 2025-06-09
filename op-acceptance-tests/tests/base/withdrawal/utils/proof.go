package utils

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum-optimism/optimism/op-service/eth"
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

func GenerateProofDB(proof []hexutil.Bytes) *proofDB {
	p := proofDB{m: make(map[string][]byte)}
	for _, s := range proof {
		key := crypto.Keccak256(s)
		p.m[string(key)] = s
	}
	return &p
}

func VerifyAccountProof(root common.Hash, address common.Address, account types.StateAccount, proof []hexutil.Bytes) error {
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

func VerifyStorageProof(root common.Hash, proof eth.StorageProofEntry) error {
	secureKey := crypto.Keccak256(proof.Key)
	db := GenerateProofDB(proof.Proof)
	value, err := trie.VerifyProof(root, secureKey, db)
	if err != nil {
		return fmt.Errorf("failed to verify proof: %w", err)
	}

	expected := proof.Value.ToInt().Bytes()
	if bytes.Equal(value, expected) {
		return nil
	} else {
		return errors.New("proved value is not the same as the expected value")
	}
}

func VerifyProof(stateRoot common.Hash, proof *eth.AccountResult) error {
	balance, overflow := uint256.FromBig(proof.Balance.ToInt())
	if overflow {
		return fmt.Errorf("proof balance overflows uint256: %d", proof.Balance.ToInt())
	}
	err := VerifyAccountProof(
		stateRoot,
		proof.Address,
		types.StateAccount{
			Nonce:    uint64(proof.Nonce),
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
