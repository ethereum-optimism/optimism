package batchconsensus

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var signedProofPrefix = []byte("BCSIG1")

func Digest(req ProofRequest) common.Hash {
	var buf bytes.Buffer
	buf.WriteString(req.L1ChainID)
	buf.WriteByte('|')
	buf.WriteString(req.L2ChainID)
	buf.WriteByte('|')
	buf.Write(req.BatchInbox.Bytes())
	buf.Write(req.Batcher.Bytes())
	for _, h := range req.BlobVersionedHashes {
		buf.Write(h[:])
	}
	return crypto.Keccak256Hash(buf.Bytes())
}

// BuildSignedProofCalldata creates signature-shaped verifier calldata for the POC sidecar path.
// The trailing byte is intentionally easy for the tiny devstack verifier bytecode to inspect.
func BuildSignedProofCalldata(req ProofRequest, signer *ecdsa.PrivateKey, valid bool) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("missing signer")
	}
	digest := Digest(req)
	sig, err := crypto.Sign(digest[:], signer)
	if err != nil {
		return nil, fmt.Errorf("sign proof digest: %w", err)
	}
	if !valid {
		sig[0] ^= 0x01
	}
	out := make([]byte, 0, len(signedProofPrefix)+len(digest)+len(sig)+1)
	out = append(out, signedProofPrefix...)
	out = append(out, digest[:]...)
	out = append(out, sig...)
	if valid {
		out = append(out, 0x01)
	} else {
		out = append(out, 0x00)
	}
	return out, nil
}

func BuildSignedProofResponse(req ProofRequest, signer *ecdsa.PrivateKey, valid bool) (ProofResponse, error) {
	calldata, err := BuildSignedProofCalldata(req, signer, valid)
	if err != nil {
		return ProofResponse{}, err
	}
	return ProofResponse{
		Provider:    ProviderCommonwarePOC,
		Certificate: calldata,
		Calldata:    calldata,
	}, nil
}

func RecoverSigner(req ProofRequest, calldata []byte) (common.Address, bool) {
	if len(calldata) != len(signedProofPrefix)+32+65+1 {
		return common.Address{}, false
	}
	if !bytes.Equal(calldata[:len(signedProofPrefix)], signedProofPrefix) {
		return common.Address{}, false
	}
	if calldata[len(calldata)-1] != 0x01 {
		return common.Address{}, false
	}
	digest := Digest(req)
	if !bytes.Equal(calldata[len(signedProofPrefix):len(signedProofPrefix)+32], digest[:]) {
		return common.Address{}, false
	}
	sig := calldata[len(signedProofPrefix)+32 : len(signedProofPrefix)+32+65]
	pub, err := crypto.SigToPub(digest[:], sig)
	if err != nil {
		return common.Address{}, false
	}
	return crypto.PubkeyToAddress(*pub), true
}
