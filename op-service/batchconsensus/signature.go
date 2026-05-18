package batchconsensus

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var signedProofPrefix = []byte("BCSIG1")
var commonwareSimplexProofPrefix = []byte("CWSIMPLX1")

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

// BuildCommonwareSimplexProofCalldata creates verifier calldata that carries a
// Commonware Simplex finalization certificate after the fixed POC envelope used
// by the DevStack verifier.
func BuildCommonwareSimplexProofCalldata(req ProofRequest, certificate []byte, signers []*ecdsa.PrivateKey, valid bool) ([]byte, error) {
	if len(certificate) == 0 {
		return nil, fmt.Errorf("missing Commonware certificate")
	}
	if len(signers) == 0 {
		return nil, fmt.Errorf("missing signers")
	}
	digest := Digest(req)
	out := make([]byte, 0, len(commonwareSimplexProofPrefix)+len(digest)+1+(65*len(signers))+len(certificate))
	out = append(out, commonwareSimplexProofPrefix...)
	out = append(out, digest[:]...)
	if valid {
		out = append(out, 0x01)
	} else {
		out = append(out, 0x00)
	}
	signingDigest := commonwareSimplexSigningDigest(digest, certificate)
	for i, signer := range signers {
		if signer == nil {
			return nil, fmt.Errorf("missing signer %d", i)
		}
		sig, err := crypto.Sign(signingDigest[:], signer)
		if err != nil {
			return nil, fmt.Errorf("sign proof digest %d: %w", i, err)
		}
		if !valid && i == 0 {
			sig[0] ^= 0x01
		}
		out = append(out, sig...)
	}
	out = append(out, certificate...)
	return out, nil
}

func commonwareSimplexSigningDigest(digest common.Hash, certificate []byte) common.Hash {
	payload := make([]byte, 0, len(commonwareSimplexProofPrefix)+len(digest)+1+len(certificate))
	payload = append(payload, commonwareSimplexProofPrefix...)
	payload = append(payload, digest[:]...)
	payload = append(payload, 0x01)
	payload = append(payload, certificate...)
	return crypto.Keccak256Hash(payload)
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
