package signer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestLocalSigner(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := NewLocalSigner(key)
	chainID := eth.ChainIDFromUInt64(123)
	payloadHash := crypto.Keccak256Hash([]byte("test"))
	sig, err := signer.SignBlockV1(context.Background(), chainID, payloadHash)
	require.NoError(t, err)
	require.NotEqual(t, eth.Bytes65{}, sig)

	authCtx := &OPStackP2PBlockAuthV1{
		Allowed: crypto.PubkeyToAddress(key.PublicKey),
		Chain:   chainID,
	}
	require.NoError(t, authCtx.VerifyP2PBlockSignature(payloadHash, sig))

	require.NoError(t, signer.Close())
	require.Nil(t, signer.priv)
}
