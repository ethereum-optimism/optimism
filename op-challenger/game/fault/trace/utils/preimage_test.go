package utils

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestPreimageLoader_NoPreimage(t *testing.T) {
	kv := kvstore.NewMemKV()
	loader := NewPreimageLoader(func() (PreimageSource, error) {
		return kv, nil
	})
	actual, err := loader.LoadPreimage(&ProofData{})
	require.NoError(t, err)
	require.Nil(t, actual)
}

func TestPreimageLoader_LocalPreimage(t *testing.T) {
	kv := kvstore.NewMemKV()
	loader := NewPreimageLoader(func() (PreimageSource, error) {
		return kv, nil
	})
	proof := &ProofData{
		OracleKey:    common.Hash{byte(preimage.LocalKeyType), 0xaa, 0xbb}.Bytes(),
		OracleValue:  nil,
		OracleOffset: 4,
	}
	actual, err := loader.LoadPreimage(proof)
	require.NoError(t, err)
	expected := types.NewPreimageOracleData(proof.OracleKey, nil, proof.OracleOffset)
	require.Equal(t, expected, actual)
	require.True(t, actual.IsLocal)
}

func TestPreimageLoader_SimpleTypes(t *testing.T) {
	tests := []preimage.KeyType{
		preimage.Keccak256KeyType,
		preimage.Sha256KeyType,
	}
	for _, keyType := range tests {
		keyType := keyType
		t.Run(fmt.Sprintf("type-%v", keyType), func(t *testing.T) {
			kv := kvstore.NewMemKV()
			loader := NewPreimageLoader(func() (PreimageSource, error) {
				return kv, nil
			})
			proof := &ProofData{
				OracleKey:    common.Hash{byte(keyType), 0xaa, 0xbb}.Bytes(),
				OracleValue:  []byte{1, 2, 3, 4, 5, 6},
				OracleOffset: 3,
			}
			actual, err := loader.LoadPreimage(proof)
			require.NoError(t, err)
			expected := types.NewPreimageOracleData(proof.OracleKey, proof.OracleValue, proof.OracleOffset)
			require.Equal(t, expected, actual)
		})
	}
}

func TestPreimageLoader_BlobPreimage(t *testing.T) {
	blob := kzg4844.Blob(testBlob())
	commitment, err := kzg4844.BlobToCommitment(&blob)
	require.NoError(t, err)

	fieldIndex := uint64(24)
	elementData := blob[fieldIndex<<5 : (fieldIndex+1)<<5]
	var point kzg4844.Point
	new(big.Int).SetUint64(fieldIndex).FillBytes(point[:])
	kzgProof, claim, err := kzg4844.ComputeProof(&blob, point)
	require.NoError(t, err)
	elementDataWithLengthPrefix := make([]byte, len(elementData)+lengthPrefixSize)
	binary.BigEndian.PutUint64(elementDataWithLengthPrefix[:lengthPrefixSize], uint64(len(elementData)))
	copy(elementDataWithLengthPrefix[lengthPrefixSize:], elementData)

	keyBuf := make([]byte, 80)
	copy(keyBuf[:48], commitment[:])
	binary.BigEndian.PutUint64(keyBuf[72:], fieldIndex)
	key := preimage.BlobKey(crypto.Keccak256Hash(keyBuf)).PreimageKey()

	proof := &ProofData{
		OracleKey:    key[:],
		OracleValue:  elementDataWithLengthPrefix,
		OracleOffset: 4,
	}

	t.Run("NoKeyPreimage", func(t *testing.T) {
		kv := kvstore.NewMemKV()
		loader := NewPreimageLoader(func() (PreimageSource, error) {
			return kv, nil
		})
		proof := &ProofData{
			OracleKey:    common.Hash{byte(preimage.BlobKeyType), 0xaf}.Bytes(),
			OracleValue:  proof.OracleValue,
			OracleOffset: proof.OracleOffset,
		}
		_, err := loader.LoadPreimage(proof)
		require.ErrorIs(t, err, kvstore.ErrNotFound)
	})

	t.Run("InvalidKeyPreimage", func(t *testing.T) {
		kv := kvstore.NewMemKV()
		loader := NewPreimageLoader(func() (PreimageSource, error) {
			return kv, nil
		})
		proof := &ProofData{
			OracleKey:    common.Hash{byte(preimage.BlobKeyType), 0xad}.Bytes(),
			OracleValue:  proof.OracleValue,
			OracleOffset: proof.OracleOffset,
		}
		require.NoError(t, kv.Put(preimage.Keccak256Key(proof.OracleKey).PreimageKey(), []byte{1, 2}))
		_, err := loader.LoadPreimage(proof)
		require.ErrorIs(t, err, ErrInvalidBlobKeyPreimage)
	})

	t.Run("MissingBlobs", func(t *testing.T) {
		kv := kvstore.NewMemKV()
		loader := NewPreimageLoader(func() (PreimageSource, error) {
			return kv, nil
		})
		proof := &ProofData{
			OracleKey:    common.Hash{byte(preimage.BlobKeyType), 0xae}.Bytes(),
			OracleValue:  proof.OracleValue,
			OracleOffset: proof.OracleOffset,
		}
		require.NoError(t, kv.Put(preimage.Keccak256Key(proof.OracleKey).PreimageKey(), keyBuf))
		_, err := loader.LoadPreimage(proof)
		require.ErrorIs(t, err, kvstore.ErrNotFound)
	})

	t.Run("Valid", func(t *testing.T) {
		kv := kvstore.NewMemKV()
		loader := NewPreimageLoader(func() (PreimageSource, error) {
			return kv, nil
		})
		storeBlob(t, kv, gokzg4844.KZGCommitment(commitment), gokzg4844.Blob(blob))
		actual, err := loader.LoadPreimage(proof)
		require.NoError(t, err)

		claimWithLength := make([]byte, len(claim)+lengthPrefixSize)
		binary.BigEndian.PutUint64(claimWithLength[:lengthPrefixSize], uint64(len(claim)))
		copy(claimWithLength[lengthPrefixSize:], claim[:])

		expected := types.NewPreimageOracleBlobData(proof.OracleKey, claimWithLength, proof.OracleOffset, fieldIndex, commitment[:], kzgProof[:])
		require.Equal(t, expected, actual)
		require.False(t, actual.IsLocal)

		// Check the KZG proof is valid
		var actualPoint kzg4844.Point
		new(big.Int).SetUint64(actual.BlobFieldIndex).FillBytes(actualPoint[:])
		actualClaim := kzg4844.Claim(actual.GetPreimageWithoutSize())
		actualCommitment := kzg4844.Commitment(actual.BlobCommitment)
		actualProof := kzg4844.Proof(actual.BlobProof)
		err = kzg4844.VerifyProof(actualCommitment, actualPoint, actualClaim, actualProof)
		require.NoError(t, err)
	})
}

func TestPreimageLoader_PrecompilePreimage(t *testing.T) {
	input := []byte("test input")
	key := preimage.PrecompileKey(crypto.Keccak256Hash(input)).PreimageKey()
	proof := &ProofData{
		OracleKey: key[:],
	}

	t.Run("NoInputPreimage", func(t *testing.T) {
		kv := kvstore.NewMemKV()
		loader := NewPreimageLoader(func() (PreimageSource, error) {
			return kv, nil
		})
		_, err := loader.LoadPreimage(proof)
		require.ErrorIs(t, err, kvstore.ErrNotFound)
	})
	t.Run("Valid", func(t *testing.T) {
		kv := kvstore.NewMemKV()
		loader := NewPreimageLoader(func() (PreimageSource, error) {
			return kv, nil
		})
		require.NoError(t, kv.Put(preimage.Keccak256Key(proof.OracleKey).PreimageKey(), input))
		actual, err := loader.LoadPreimage(proof)
		require.NoError(t, err)
		inputWithLength := lengthPrefixed(input)
		expected := types.NewPreimageOracleData(proof.OracleKey, inputWithLength, proof.OracleOffset)
		require.Equal(t, expected, actual)
	})
}

// Returns a serialized random field element in big-endian
func fieldElement(val uint64) [32]byte {
	r := fr.NewElement(val)
	return gokzg4844.SerializeScalar(r)
}

func testBlob() gokzg4844.Blob {
	var blob gokzg4844.Blob
	bytesPerBlob := gokzg4844.ScalarsPerBlob * gokzg4844.SerializedScalarSize
	for i := 0; i < bytesPerBlob; i += gokzg4844.SerializedScalarSize {
		fieldElementBytes := fieldElement(uint64(i))
		copy(blob[i:i+gokzg4844.SerializedScalarSize], fieldElementBytes[:])
	}
	return blob
}

func storeBlob(t *testing.T, kv kvstore.KV, commitment gokzg4844.KZGCommitment, blob gokzg4844.Blob) {
	// Pre-store versioned hash preimage (commitment)
	key := preimage.Sha256Key(sha256.Sum256(commitment[:]))
	err := kv.Put(key.PreimageKey(), commitment[:])
	require.NoError(t, err, "Failed to store versioned hash preimage in kvstore")

	// Pre-store blob field elements
	blobKeyBuf := make([]byte, 80)
	copy(blobKeyBuf[:48], commitment[:])
	for i := 0; i < params.BlobTxFieldElementsPerBlob; i++ {
		binary.BigEndian.PutUint64(blobKeyBuf[72:], uint64(i))
		feKey := crypto.Keccak256Hash(blobKeyBuf)
		err := kv.Put(preimage.Keccak256Key(feKey).PreimageKey(), blobKeyBuf)
		require.NoError(t, err)

		err = kv.Put(preimage.BlobKey(feKey).PreimageKey(), blob[i<<5:(i+1)<<5])
		require.NoError(t, err, "Failed to store field element preimage in kvstore")
	}
}
