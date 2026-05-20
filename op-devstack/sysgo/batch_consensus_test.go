package sysgo

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/batchconsensus"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestBatchConsensusMockVerifierProofCode(t *testing.T) {
	signer, err := batchConsensusMockProofSigner()
	require.NoError(t, err)
	req, err := batchconsensus.NewProofRequest(
		big.NewInt(1),
		big.NewInt(2),
		common.HexToAddress("0x1111"),
		common.HexToAddress("0x2222"),
		[]common.Hash{common.HexToHash("0x3333")},
	)
	require.NoError(t, err)
	code := batchConsensusMockVerifierProofCode(batchConsensusMockProofSignerAddress())

	valid, err := batchconsensus.BuildSignedProofCalldata(req, signer, true)
	require.NoError(t, err)
	out, _, err := runtime.Execute(code, valid, nil)
	require.NoError(t, err)
	ok, err := batchconsensus.DecodeVerifyResult(out)
	require.NoError(t, err)
	require.True(t, ok)

	invalid, err := batchconsensus.BuildSignedProofCalldata(req, signer, false)
	require.NoError(t, err)
	out, _, err = runtime.Execute(code, invalid, nil)
	require.NoError(t, err)
	ok, err = batchconsensus.DecodeVerifyResult(out)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestBatchConsensusMockVerifierCommonwareProofCode(t *testing.T) {
	req, err := batchconsensus.NewProofRequest(
		big.NewInt(1),
		big.NewInt(2),
		common.HexToAddress("0x1111"),
		common.HexToAddress("0x2222"),
		[]common.Hash{common.HexToHash("0x3333")},
	)
	require.NoError(t, err)
	signers, err := batchConsensusCommonwareProofSigners()
	require.NoError(t, err)
	code := batchConsensusMockVerifierCommonwareProofCode(batchConsensusCommonwareProofSignerAddresses())
	certificate := common.FromHex("0x435753494d504c5831000200a9dc0572235267fb0049b4c044903f4ec7c23784811f24d2238248045975ad4000000000000000040e0393cef7cc3de40e20cf4e211a10dfd51277bfc8c7d039909bc1d925798efd31dd527be67cc5b9fe797cdde5523d750d4d90bf12f763ab5d0b1d2ed530515ddd0ad38f5287bd39c36d22a44843a9f9f0af2aa459c3399cc43a319edd5280ba630b442e2a59c8e718674a4dbc06cf0a4a2baa2f38a8da4d79e19e7cdf67c8e869b3fe40e383b25fc686507b49f81640ef09d25672a01226e48ccc533321831ebfdf39bab8e450680daad689faf453a10cc0c5eb8d46a40803a85b325a67e606b6ac")
	requireCommonwareP256Fixture(t, certificate)

	valid, err := batchconsensus.BuildCommonwareSimplexProofCalldata(req, certificate, signers, true)
	require.NoError(t, err)
	out := executeBatchConsensusCommonwareVerifier(t, code, valid)
	ok, err := batchconsensus.DecodeVerifyResult(out)
	require.NoError(t, err)
	require.Truef(t, ok, "code_len=%d calldata_len=%d code=%x out=%x", len(code), len(valid), code, out)

	withoutCertificate := valid[:len(valid)-len(certificate)]
	out = executeBatchConsensusCommonwareVerifier(t, code, withoutCertificate)
	ok, err = batchconsensus.DecodeVerifyResult(out)
	require.NoError(t, err)
	require.False(t, ok)

	wrongCertificate := append([]byte(nil), valid...)
	wrongCertificate[len(wrongCertificate)-1] ^= 0x01
	out = executeBatchConsensusCommonwareVerifier(t, code, wrongCertificate)
	ok, err = batchconsensus.DecodeVerifyResult(out)
	require.NoError(t, err)
	require.False(t, ok)

	invalid, err := batchconsensus.BuildCommonwareSimplexProofCalldata(req, certificate, signers, false)
	require.NoError(t, err)
	out = executeBatchConsensusCommonwareVerifier(t, code, invalid)
	ok, err = batchconsensus.DecodeVerifyResult(out)
	require.NoError(t, err)
	require.False(t, ok)
}

func executeBatchConsensusCommonwareVerifier(t *testing.T, code []byte, input []byte) []byte {
	t.Helper()
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)
	addr := common.BytesToAddress([]byte("contract"))
	statedb.CreateAccount(addr)
	statedb.SetCode(addr, code, tracing.CodeChangeUnspecified)
	for slot, value := range batchConsensusCommonwareSolidityVerifierStorage() {
		statedb.SetState(addr, slot, value)
	}
	out, _, err := runtime.Call(addr, input, &runtime.Config{
		ChainConfig: params.MergedTestChainConfig,
		State:       statedb,
	})
	require.NoError(t, err)
	return out
}

func requireCommonwareP256Fixture(t *testing.T, certificate []byte) {
	t.Helper()
	require.Len(t, certificate, 246)
	require.Equal(t, []byte("CWSIMPLX1"), certificate[:9])
	require.Equal(t, byte(0x0e), certificate[52])
	require.Equal(t, byte(0x03), certificate[53])
	msg := append([]byte("\x29op-batcher-consensus-poc/simplex_FINALIZE"), certificate[9:44]...)
	digest := sha256.Sum256(msg)
	for i, signer := range batchConsensusCommonwareP256ProofPublicKeys() {
		sig := certificate[54+(i*64) : 54+((i+1)*64)]
		r := new(big.Int).SetBytes(sig[:32])
		s := new(big.Int).SetBytes(sig[32:])
		pub := ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(signer.x[:]),
			Y:     new(big.Int).SetBytes(signer.y[:]),
		}
		require.Truef(t, ecdsa.Verify(&pub, digest[:], r, s), "signature %d", i)
	}
}

func TestBatchConsensusMockVerifierTrueCode(t *testing.T) {
	out, _, err := runtime.Execute(batchConsensusMockVerifierTrueCode, []byte("anything"), nil)
	require.NoError(t, err)
	ok, err := batchconsensus.DecodeVerifyResult(out)
	require.NoError(t, err)
	require.True(t, ok)
}
