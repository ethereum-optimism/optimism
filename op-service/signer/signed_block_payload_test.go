package signer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestSignedP2PBlock(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey)
	chainID := eth.ChainIDFromUInt64(42)
	v1Auth := &OPStackP2PBlockAuthV1{
		Allowed: addr,
		Chain:   chainID,
	}
	payload := []byte("hello")

	sigA, err := NewLocalSigner(key).SignBlockV1(context.Background(), chainID, PayloadHash(payload))
	require.NoError(t, err)

	t.Run("valid", func(t *testing.T) {
		bl := SignedP2PBlock{
			Raw:       payload,
			Signature: sigA,
		}
		require.NoError(t, bl.VerifySignature(v1Auth))
	})

	t.Run("invalid msg", func(t *testing.T) {
		bl := SignedP2PBlock{
			Raw:       []byte("different"),
			Signature: sigA,
		}
		require.Error(t, bl.VerifySignature(v1Auth))
	})

	keyB, err := crypto.GenerateKey()
	require.NoError(t, err)
	sigB, err := NewLocalSigner(keyB).SignBlockV1(context.Background(), chainID, PayloadHash(payload))
	require.NoError(t, err)
	t.Run("different key", func(t *testing.T) {
		bl := SignedP2PBlock{
			Raw:       payload,
			Signature: sigB,
		}
		require.Error(t, bl.VerifySignature(v1Auth))
	})
}
