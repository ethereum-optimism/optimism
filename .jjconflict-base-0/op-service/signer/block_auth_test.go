package signer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestBlockAuth(t *testing.T) {
	good := common.Address{0: 123}
	chainID := eth.ChainIDFromUInt64(42)
	v1Auth := OPStackP2PBlockAuthV1{
		Allowed: good,
		Chain:   chainID,
	}
	require.Equal(t, chainID, v1Auth.ChainID())
	t.Run("checks", func(t *testing.T) {
		require.NoError(t, v1Auth.Check(good))
		require.ErrorContains(t, v1Auth.Check(common.Address{}), "unrecognized")
		require.ErrorContains(t, v1Auth.Check(common.Address{0: 124}), "unrecognized")
	})

	t.Run("missing config addr", func(t *testing.T) {
		missingAddr := OPStackP2PBlockAuthV1{
			Allowed: common.Address{},
			Chain:   chainID,
		}
		require.ErrorContains(t, missingAddr.Check(good), "missing")
	})

	payload := []byte("test")
	payloadHash := crypto.Keccak256Hash(payload)
	t.Run("valid case", func(t *testing.T) {
		require.ErrorContains(t, v1Auth.VerifyP2PBlockSignature(payloadHash, eth.Bytes65{}), "fail")
	})

	keyA, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := NewLocalSigner(keyA)
	sigA, err := signer.SignBlockV1(context.Background(), chainID, payloadHash)
	require.NoError(t, err)
	t.Run("empty sig", func(t *testing.T) {
		require.ErrorContains(t, v1Auth.VerifyP2PBlockSignature(payloadHash, eth.Bytes65{}), "fail")
	})
	t.Run("unrecognized key", func(t *testing.T) {
		require.ErrorContains(t, v1Auth.VerifyP2PBlockSignature(payloadHash, sigA), "unrecognized")
	})
	t.Run("malformed", func(t *testing.T) {
		sigB := sigA
		sigB[64] = 0xe0
		require.ErrorContains(t, v1Auth.VerifyP2PBlockSignature(payloadHash, sigB), "fail")
	})
	t.Run("success", func(t *testing.T) {
		v1Auth.Allowed = crypto.PubkeyToAddress(keyA.PublicKey)
		require.NoError(t, v1Auth.VerifyP2PBlockSignature(payloadHash, sigA))
	})
}
