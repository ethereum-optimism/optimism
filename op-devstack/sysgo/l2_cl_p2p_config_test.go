package sysgo

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestDevstackP2PSignerSurvivesRestart(t *testing.T) {
	dt := devtest.ParallelT(t)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	_, setup := newDevstackP2PConfig(dt, dt.Logger(), 2, true, false, hex.EncodeToString(crypto.FromECDSA(key)))
	first, err := setup.SetupSigner(t.Context())
	require.NoError(t, err)
	chain := eth.ChainIDFromUInt64(902)
	payload := crypto.Keccak256Hash([]byte("restart test"))
	auth := signer.OPStackP2PBlockAuthV1{Allowed: crypto.PubkeyToAddress(key.PublicKey), Chain: chain}
	sig, err := first.SignBlockV1(t.Context(), chain, payload)
	require.NoError(t, err)
	require.NoError(t, auth.VerifyP2PBlockSignature(payload, sig))
	require.NoError(t, first.Close())
	second, err := setup.SetupSigner(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, second.Close()) })
	require.NotSame(t, first, second)
	sig, err = second.SignBlockV1(t.Context(), chain, payload)
	require.NoError(t, err)
	require.NoError(t, auth.VerifyP2PBlockSignature(payload, sig))
	_, err = first.SignBlockV1(t.Context(), chain, payload)
	require.ErrorContains(t, err, "signer is closed", "restart must not revive an old instance")
}
