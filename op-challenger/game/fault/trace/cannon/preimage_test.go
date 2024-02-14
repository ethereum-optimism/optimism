package cannon

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
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
	loader := newPreimageLoader(kvstore.NewMemKV().Get)
	actual, err := loader.LoadPreimage(&proofData{})
	require.NoError(t, err)
	require.Nil(t, actual)
}

func TestPreimageLoader_LocalPreimage(t *testing.T) {
	loader := newPreimageLoader(kvstore.NewMemKV().Get)
	proof := &proofData{
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
			loader := newPreimageLoader(kvstore.NewMemKV().Get)
			proof := &proofData{
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
	blob := testBlob()
	commitment, err := kzg4844.BlobToCommitment(kzg4844.Blob(blob))
	require.NoError(t, err)

	fieldIndex := uint64(24)
	elementData := blob[fieldIndex<<5 : (fieldIndex+1)<<5]
	kzgProof, _, err := kzg4844.ComputeProof(kzg4844.Blob(blob), kzg4844.Point(elementData))
	require.NoError(t, err)

	keyBuf := make([]byte, 80)
	copy(keyBuf[:48], commitment[:])
	binary.BigEndian.PutUint64(keyBuf[72:], fieldIndex)
	key := preimage.BlobKey(crypto.Keccak256Hash(keyBuf)).PreimageKey()

	proof := &proofData{
		OracleKey:    key[:],
		OracleValue:  elementData,
		OracleOffset: 4,
	}

	t.Run("RejectInvalidValueLength", func(t *testing.T) {
		kv := kvstore.NewMemKV()
		loader := newPreimageLoader(kv.Get)
		proof := &proofData{
			OracleKey:    proof.OracleKey,
			OracleValue:  []byte{1, 2, 3},
			OracleOffset: proof.OracleOffset,
		}
		_, err := loader.LoadPreimage(proof)
		require.ErrorIs(t, err, ErrInvalidScalarValue)
	})

	t.Run("NoKeyPreimage", func(t *testing.T) {
		kv := kvstore.NewMemKV()
		loader := newPreimageLoader(kv.Get)
		proof := &proofData{
			OracleKey:    common.Hash{byte(preimage.BlobKeyType), 0xaf}.Bytes(),
			OracleValue:  proof.OracleValue,
			OracleOffset: proof.OracleOffset,
		}
		_, err := loader.LoadPreimage(proof)
		require.ErrorIs(t, err, kvstore.ErrNotFound)
	})

	t.Run("InvalidKeyPreimage", func(t *testing.T) {
		kv := kvstore.NewMemKV()
		loader := newPreimageLoader(kv.Get)
		proof := &proofData{
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
		loader := newPreimageLoader(kv.Get)
		proof := &proofData{
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
		loader := newPreimageLoader(kv.Get)
		storeBlob(t, kv, gokzg4844.KZGCommitment(commitment), blob)
		actual, err := loader.LoadPreimage(proof)
		require.NoError(t, err)
		expected := types.NewPreimageOracleBlobData(proof.OracleKey, proof.OracleValue, proof.OracleOffset, fieldIndex, commitment[:], kzgProof[:])
		require.Equal(t, expected, actual)
		require.False(t, actual.IsLocal)
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
