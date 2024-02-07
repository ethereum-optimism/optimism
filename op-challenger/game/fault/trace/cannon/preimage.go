package cannon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"

	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

const (
	fieldElemKeyLength = 80
	commitmentLength   = 48
)

var (
	ErrInvalidScalarValue = errors.New("invalid scalar value")
)

var kzg4844Ctx atomic.Pointer[gokzg4844.Context]

func kzg() *gokzg4844.Context {
	if kzg4844Ctx.Load() == nil {
		ctx, err := gokzg4844.NewContext4096Secure()
		if err != nil {
			panic("unable to load kzg trusted setup")
		}
		kzg4844Ctx.Store(ctx)
	}
	return kzg4844Ctx.Load()
}

type preimageSource func(key common.Hash) ([]byte, error)

type preimageLoader struct {
	getPreimage preimageSource
}

func newPreimageLoader(getPreimage preimageSource) *preimageLoader {
	return &preimageLoader{
		getPreimage: getPreimage,
	}
}

func (l *preimageLoader) LoadPreimage(proof *proofData) (*types.PreimageOracleData, error) {
	if len(proof.OracleKey) == 0 {
		return nil, nil
	}
	switch preimage.KeyType(proof.OracleKey[0]) {
	case preimage.BlobKeyType:
		return l.loadBlobPreimage(proof)
	default:
		return types.NewPreimageOracleData(proof.OracleKey, proof.OracleValue, proof.OracleOffset), nil
	}
}

func (l *preimageLoader) loadBlobPreimage(proof *proofData) (*types.PreimageOracleData, error) {
	if len(proof.OracleValue) != gokzg4844.SerializedScalarSize {
		return nil, fmt.Errorf("%w, expected length %v but was %v", ErrInvalidScalarValue, gokzg4844.SerializedScalarSize, len(proof.OracleValue))
	}

	// The key for a blob field element is a keccak hash of commitment++fieldElementIndex.
	// First retrieve the preimage of the key as a keccak hash so we have the commitment and required field element
	inputsKey := preimage.Keccak256Key(proof.OracleKey).PreimageKey()
	inputs, err := l.getPreimage(inputsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get key preimage: %w", err)
	}
	if len(inputs) != fieldElemKeyLength {
		return nil, fmt.Errorf("invalid key preimage, expected length %v but was %v", fieldElemKeyLength, len(inputs))
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
		fieldElement, err := l.getPreimage(key)
		if err != nil {
			return nil, fmt.Errorf("failed to load field element %v with key %v", i, common.Hash(key))
		}
		copy(blob[i<<5:(i+1)<<5], fieldElement[:])
	}

	// Sanity check the blob data matches the commitment
	blobCommitment, err := blob.ComputeKZGCommitment()
	if err != nil || !bytes.Equal(blobCommitment[:], commitment[:]) {
		return nil, fmt.Errorf("invalid blob commitment: %w", err)
	}

	// Compute the KZG proof for the required field element
	kzgProof, _, err := kzg().ComputeKZGProof(gokzg4844.Blob(blob), gokzg4844.Scalar(proof.OracleValue), 0)
	if err != nil {
		return nil, fmt.Errorf("failed to compute kzg proof: %w", err)
	}

	return types.NewPreimageOracleBlobData(proof.OracleKey, proof.OracleValue, proof.OracleOffset, requiredFieldElement, commitment, kzgProof[:]), nil
}
