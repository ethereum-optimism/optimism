package utils

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"

	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
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
	rng := rand.New(rand.NewSource(999))
	blob, commitment, err := testutils.RandomBlob(rng)
	require.NoError(t, err)

	indices := []uint64{0, 1, 24, 2222, 4095}
	for _, fieldIndex := range indices {
		elementData := blob[fieldIndex<<5 : (fieldIndex+1)<<5]
		zPoint := l1.RootsOfUnity[fieldIndex].Bytes()
		kzgProof, claim, err := kzg4844.ComputeProof(&blob, zPoint)
		require.NoError(t, err)
		elementDataWithLengthPrefix := make([]byte, len(elementData)+lengthPrefixSize)
		binary.BigEndian.PutUint64(elementDataWithLengthPrefix[:lengthPrefixSize], uint64(len(elementData)))
		copy(elementDataWithLengthPrefix[lengthPrefixSize:], elementData)

		keyBuf := make([]byte, 80)
		copy(keyBuf[:48], commitment[:])
		copy(keyBuf[48:], zPoint[:])
		key := preimage.BlobKey(crypto.Keccak256Hash(keyBuf)).PreimageKey()

		proof := &ProofData{
			OracleKey:    key[:],
			OracleValue:  elementDataWithLengthPrefix,
			OracleOffset: 4,
		}

		testName := func(str string) string {
			return fmt.Sprintf("%v (index %v)", str, fieldIndex)
		}

		t.Run(testName("NoKeyPreimage"), func(t *testing.T) {
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

		t.Run(testName("InvalidKeyPreimage"), func(t *testing.T) {
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

		t.Run(testName("MissingBlobs"), func(t *testing.T) {
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

		t.Run(testName("Valid"), func(t *testing.T) {
			kv := kvstore.NewMemKV()
			loader := NewPreimageLoader(func() (PreimageSource, error) {
				return kv, nil
			})
			storeBlob(t, kv, gokzg4844.KZGCommitment(commitment), gokzg4844.Blob(blob))
			actual, err := loader.LoadPreimage(proof)
			require.NoError(t, err)

			// Check the computed claim matches our expectation
			claimWithLength := make([]byte, len(claim)+lengthPrefixSize)
			binary.BigEndian.PutUint64(claimWithLength[:lengthPrefixSize], uint64(len(claim)))
			copy(claimWithLength[lengthPrefixSize:], claim[:])
			require.Equal(t, claimWithLength[:], elementDataWithLengthPrefix[:])

			expected := types.NewPreimageOracleBlobData(proof.OracleKey, proof.OracleValue, proof.OracleOffset, zPoint, commitment[:], kzgProof[:])
			require.Equal(t, expected, actual)
			require.False(t, actual.IsLocal)

			// Check the KZG proof is valid
			actualPoint := actual.ZPoint
			actualClaim := kzg4844.Claim(actual.GetPreimageWithoutSize())
			actualCommitment := kzg4844.Commitment(actual.BlobCommitment)
			actualProof := kzg4844.Proof(actual.BlobProof)
			err = kzg4844.VerifyProof(actualCommitment, actualPoint, actualClaim, actualProof)
			require.NoError(t, err)
		})
	}
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

func storeBlob(t *testing.T, kv kvstore.KV, commitment gokzg4844.KZGCommitment, blob gokzg4844.Blob) {
	// Pre-store versioned hash preimage (commitment)
	key := preimage.Sha256Key(sha256.Sum256(commitment[:]))
	err := kv.Put(key.PreimageKey(), commitment[:])
	require.NoError(t, err, "Failed to store versioned hash preimage in kvstore")

	// Pre-store blob field elements
	blobKeyBuf := make([]byte, 80)
	copy(blobKeyBuf[:48], commitment[:])
	for i := 0; i < params.BlobTxFieldElementsPerBlob; i++ {
		root := l1.RootsOfUnity[i].Bytes()
		copy(blobKeyBuf[48:], root[:])
		feKey := crypto.Keccak256Hash(blobKeyBuf)
		err := kv.Put(preimage.Keccak256Key(feKey).PreimageKey(), blobKeyBuf)
		require.NoError(t, err)

		err = kv.Put(preimage.BlobKey(feKey).PreimageKey(), blob[i<<5:(i+1)<<5])
		require.NoError(t, err, "Failed to store field element preimage in kvstore")
	}
}
