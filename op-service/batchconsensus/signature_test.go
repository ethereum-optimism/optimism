package batchconsensus

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestBuildSignedProofCalldataRecoverSigner(t *testing.T) {
	key, err := crypto.HexToECDSA("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	req, err := NewProofRequest(
		big.NewInt(1),
		big.NewInt(2),
		common.HexToAddress("0x1111"),
		common.HexToAddress("0x2222"),
		[]common.Hash{common.HexToHash("0x3333")},
	)
	require.NoError(t, err)

	calldata, err := BuildSignedProofCalldata(req, key, true)
	require.NoError(t, err)
	signer, ok := RecoverSigner(req, calldata)
	require.True(t, ok)
	require.Equal(t, crypto.PubkeyToAddress(key.PublicKey), signer)

	invalid, err := BuildSignedProofCalldata(req, key, false)
	require.NoError(t, err)
	_, ok = RecoverSigner(req, invalid)
	require.False(t, ok)

	otherReq := req
	otherReq.BlobVersionedHashes = []common.Hash{common.HexToHash("0x4444")}
	_, ok = RecoverSigner(otherReq, calldata)
	require.False(t, ok)

	resp, err := BuildSignedProofResponse(req, key, true)
	require.NoError(t, err)
	require.Equal(t, ProviderCommonwarePOC, resp.Provider)
	require.Equal(t, resp.Calldata, resp.Certificate)
}
