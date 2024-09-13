package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
)

const (
	fieldElemKeyLength = 80
	commitmentLength   = 48
	lengthPrefixSize   = 8
)

var (
	ErrInvalidBlobKeyPreimage = errors.New("invalid blob key preimage")
)

type PreimageSource interface {
	Get(key common.Hash) ([]byte, error)
	Close() error
}

type PreimageSourceCreator func() (PreimageSource, error)

type PreimageLoader struct {
	makeSource PreimageSourceCreator
}

func NewPreimageLoader(makeSource PreimageSourceCreator) *PreimageLoader {
	return &PreimageLoader{
		makeSource: makeSource,
	}
}

func (l *PreimageLoader) LoadPreimage(proof *ProofData) (*types.PreimageOracleData, error) {
	if len(proof.OracleKey) == 0 {
		return nil, nil
	}
	switch preimage.KeyType(proof.OracleKey[0]) {
	case preimage.BlobKeyType:
		return l.loadBlobPreimage(proof)
	case preimage.PrecompileKeyType:
		return l.loadPrecompilePreimage(proof)
	default:
		return types.NewPreimageOracleData(proof.OracleKey, proof.OracleValue, proof.OracleOffset), nil
	}
}

func (l *PreimageLoader) loadBlobPreimage(proof *ProofData) (*types.PreimageOracleData, error) {
	// The key for a blob field element is a keccak hash of commitment++fieldElementIndex.
	// First retrieve the preimage of the key as a keccak hash so we have the commitment and required field element
	inputsKey := preimage.Keccak256Key(proof.OracleKey).PreimageKey()
	source, err := l.makeSource()
	if err != nil {
		return nil, fmt.Errorf("failed to open preimage store: %w", err)
	}
	defer source.Close()
	inputs, err := source.Get(inputsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get key preimage: %w", err)
	}
	if len(inputs) != fieldElemKeyLength {
		return nil, fmt.Errorf("%w, expected length %v but was %v", ErrInvalidBlobKeyPreimage, fieldElemKeyLength, len(inputs))
	}
	commitment := inputs[:commitmentLength]
	requiredFieldElement := binary.BigEndian.Uint64(inputs[72:])

	// Now, reconstruct the full blob by loading the 4096 field elements.
	blob := eth.Blob{}
	fieldElemKey := make([]byte, fieldElemKeyLength)
	copy(fieldElemKey[:commitmentLength], commitment)
	for i := 0; i < params.BlobTxFieldElementsPerBlob; i++ {
		binary.BigEndian.PutUint64(fieldElemKey[72:], uint64(i))
		key := preimage.BlobKey(crypto.Keccak256(fieldElemKey)).PreimageKey()
		fieldElement, err := source.Get(key)
		if err != nil {
			return nil, fmt.Errorf("failed to load field element %v with key %v:  %w", i, common.Hash(key), err)
		}
		copy(blob[i<<5:(i+1)<<5], fieldElement[:])
	}

	// Sanity check the blob data matches the commitment
	blobCommitment, err := blob.ComputeKZGCommitment()
	if err != nil || !bytes.Equal(blobCommitment[:], commitment[:]) {
		return nil, fmt.Errorf("invalid blob commitment: %w", err)
	}
	// Compute the KZG proof for the required field element
	var point kzg4844.Point
	new(big.Int).SetUint64(requiredFieldElement).FillBytes(point[:])
	data := kzg4844.Blob(blob)
	kzgProof, claim, err := kzg4844.ComputeProof(&data, point)
	if err != nil {
		return nil, fmt.Errorf("failed to compute kzg proof: %w", err)
	}
	err = kzg4844.VerifyProof(kzg4844.Commitment(commitment), point, claim, kzgProof)
	if err != nil {
		return nil, fmt.Errorf("failed to verify proof: %w", err)
	}

	claimWithLength := lengthPrefixed(claim[:])
	return types.NewPreimageOracleBlobData(proof.OracleKey, claimWithLength, proof.OracleOffset, requiredFieldElement, commitment, kzgProof[:]), nil
}

func (l *PreimageLoader) loadPrecompilePreimage(proof *ProofData) (*types.PreimageOracleData, error) {
	inputKey := preimage.Keccak256Key(proof.OracleKey).PreimageKey()
	source, err := l.makeSource()
	if err != nil {
		return nil, fmt.Errorf("failed to open preimage store: %w", err)
	}
	defer source.Close()
	input, err := source.Get(inputKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get key preimage: %w", err)
	}
	inputWithLength := lengthPrefixed(input)
	return types.NewPreimageOracleData(proof.OracleKey, inputWithLength, proof.OracleOffset), nil
}

func lengthPrefixed(data []byte) []byte {
	dataWithLength := make([]byte, len(data)+lengthPrefixSize)
	binary.BigEndian.PutUint64(dataWithLength[:lengthPrefixSize], uint64(len(data)))
	copy(dataWithLength[lengthPrefixSize:], data)
	return dataWithLength
}
