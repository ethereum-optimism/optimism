package sysgo

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/batchconsensus"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
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
	certificate := []byte("CWSIMPLX1-finalization")

	valid, err := batchconsensus.BuildCommonwareSimplexProofCalldata(req, certificate, signers, true)
	require.NoError(t, err)
	out, _, err := runtime.Execute(code, valid, nil)
	require.NoError(t, err)
	ok, err := batchconsensus.DecodeVerifyResult(out)
	require.NoError(t, err)
	require.Truef(t, ok, "code_len=%d calldata_len=%d code=%x out=%x", len(code), len(valid), code, out)

	withoutCertificate := valid[:len(valid)-len(certificate)]
	out, _, err = runtime.Execute(code, withoutCertificate, nil)
	require.NoError(t, err)
	ok, err = batchconsensus.DecodeVerifyResult(out)
	require.NoError(t, err)
	require.False(t, ok)

	invalid, err := batchconsensus.BuildCommonwareSimplexProofCalldata(req, certificate, signers, false)
	require.NoError(t, err)
	out, _, err = runtime.Execute(code, invalid, nil)
	require.NoError(t, err)
	ok, err = batchconsensus.DecodeVerifyResult(out)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestBatchConsensusMockVerifierTrueCode(t *testing.T) {
	out, _, err := runtime.Execute(batchConsensusMockVerifierTrueCode, []byte("anything"), nil)
	require.NoError(t, err)
	ok, err := batchconsensus.DecodeVerifyResult(out)
	require.NoError(t, err)
	require.True(t, ok)
}
