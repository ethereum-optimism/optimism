package main

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	IstanbulExtraVanity = 32 // Fixed number of extra-data bytes reserved for validator vanity
)

type IstanbulAggregatedSeal struct {
	// Bitmap is a bitmap having an active bit for each validator that signed this block
	Bitmap *big.Int
	// Signature is an aggregated BLS signature resulting from signatures by each validator that signed this block
	Signature []byte
	// Round is the round in which the signature was created.
	Round *big.Int
}

type IstanbulExtra struct {
	// AddedValidators are the validators that have been added in the block
	AddedValidators []common.Address
	// AddedValidatorsPublicKeys are the BLS public keys for the validators added in the block
	AddedValidatorsPublicKeys [][96]byte
	// RemovedValidators is a bitmap having an active bit for each removed validator in the block
	RemovedValidators *big.Int
	// Seal is an ECDSA signature by the proposer
	Seal []byte
	// AggregatedSeal contains the aggregated BLS signature created via IBFT consensus.
	AggregatedSeal IstanbulAggregatedSeal
	// ParentAggregatedSeal contains and aggregated BLS signature for the previous block.
	ParentAggregatedSeal IstanbulAggregatedSeal
}

// transformHeader removes the aggregated seal from the header
func transformHeader(header []byte) ([]byte, error) {
	newHeader := new(types.Header) // TODO double check on decoding type
	err := rlp.DecodeBytes(header, newHeader)
	if err != nil {
		return nil, err
	}

	if len(newHeader.Extra) < IstanbulExtraVanity {
		return nil, errors.New("invalid istanbul header extra-data")
	}

	istanbulExtra := IstanbulExtra{}
	err = rlp.DecodeBytes(newHeader.Extra[IstanbulExtraVanity:], &istanbulExtra)
	if err != nil {
		return nil, err
	}

	istanbulExtra.AggregatedSeal = IstanbulAggregatedSeal{}

	payload, err := rlp.EncodeToBytes(&istanbulExtra)
	if err != nil {
		return nil, err
	}

	newHeader.Extra = append(newHeader.Extra[:IstanbulExtraVanity], payload...)

	return rlp.EncodeToBytes(newHeader)
}

func hasSameHash(newHeader, oldHash []byte) (bool, common.Hash) {
	newHash := crypto.Keccak256Hash(newHeader)
	return bytes.Equal(oldHash, newHash.Bytes()), newHash
}

// transformBlockBody migrates the block body from the old format to the new format (works with []byte input output)
func transformBlockBody(oldBodyData []byte) ([]byte, error) {
	// decode body into celo-blockchain Body structure
	// remove epochSnarkData and randomness data
	var celoBody struct {
		Transactions   rlp.RawValue // TODO use types.Transactions to make sure all tx are deserializable
		Randomness     rlp.RawValue
		EpochSnarkData rlp.RawValue
	}
	if err := rlp.DecodeBytes(oldBodyData, &celoBody); err != nil {
		return nil, fmt.Errorf("failed to RLP decode body: %w", err)
	}

	// TODO create a types.BlockBody structure and encode it back to []byte

	// transform into op-geth types.Body structure
	// since Body is a slice of types.Transactions, we can just remove the randomness and epochSnarkData and add empty array for UnclesHashes
	newBodyData, err := rlp.EncodeToBytes([]interface{}{celoBody.Transactions, nil})
	if err != nil {
		return nil, fmt.Errorf("failed to RLP encode body: %w", err)
	}

	return newBodyData, nil
}
