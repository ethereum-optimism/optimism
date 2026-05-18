package sysgo

import (
	"crypto/elliptic"
	"encoding/hex"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var DefaultBatchConsensusMockVerifierAddress = common.HexToAddress("0x00000000000000000000000000000000000bc0de")

var batchConsensusMockVerifierFalseCode = common.FromHex("0x600060005260206000f3")
var batchConsensusMockVerifierTrueCode = common.FromHex("0x600160005260206000f3")

func applyBatchConsensusMockVerifier(cfg PresetConfig, l1Net *L1Network, l2Nets ...*L2Network) {
	if cfg.BatchConsensusMockVerifierAddress == nil {
		return
	}
	addr := *cfg.BatchConsensusMockVerifierAddress
	if l1Net.genesis.Alloc == nil {
		l1Net.genesis.Alloc = make(types.GenesisAlloc)
	}
	if cfg.BatchConsensusCommonwareSidecar {
		osakaTime := uint64(0)
		if l1Net.genesis.Config.PragueTime != nil {
			osakaTime = *l1Net.genesis.Config.PragueTime
		}
		l1Net.genesis.Config.OsakaTime = &osakaTime
	}
	code := batchConsensusMockVerifierFalseCode
	if cfg.BatchConsensusMockVerifierAccept {
		code = batchConsensusMockVerifierTrueCode
		if cfg.BatchConsensusCommonwareSidecar {
			code = batchConsensusMockVerifierCommonwareProofCode(batchConsensusCommonwareProofSignerAddresses())
		} else if cfg.BatchConsensusMockProofSidecar {
			code = batchConsensusMockVerifierProofCode(batchConsensusMockProofSignerAddress())
		}
	}
	l1Net.genesis.Alloc[addr] = types.Account{
		Balance: big.NewInt(0),
		Code:    code,
	}
	l1GenesisID := eth.ToBlockID(l1Net.genesis.ToBlock())
	for _, l2Net := range l2Nets {
		l2Net.rollupCfg.Genesis.L1 = l1GenesisID
		l2Net.rollupCfg.BatchConsensusVerifierAddress = addr
	}
}

func batchConsensusMockVerifierProofCode(signer common.Address) []byte {
	return batchConsensusMockVerifierSignatureProofCode(signatureProofLayout{
		signer:       signer,
		exactSize:    0x68,
		digestOffset: 0x06,
		rOffset:      0x26,
		sOffset:      0x46,
		vOffset:      0x66,
		markerOffset: 0x67,
	})
}

func batchConsensusMockVerifierCommonwareProofCode(_ []common.Address) []byte {
	return batchConsensusMockVerifierCommonwareP256ProofCode(batchConsensusCommonwareP256ProofPublicKeys())
}

var batchConsensusCommonwareP256ProofSignerKeys = []string{
	"1123456789abcdef1123456789abcdef1123456789abcdef1123456789abcdef",
	"2123456789abcdef2123456789abcdef2123456789abcdef2123456789abcdef",
	"3123456789abcdef3123456789abcdef3123456789abcdef3123456789abcdef",
	"4123456789abcdef4123456789abcdef4123456789abcdef4123456789abcdef",
}

type batchConsensusP256PublicKey struct {
	x [32]byte
	y [32]byte
}

func batchConsensusCommonwareP256ProofPublicKeys() []batchConsensusP256PublicKey {
	return batchConsensusCommonwareP256ProofPublicKeysForAllValidators()[:3]
}

func batchConsensusCommonwareP256ProofPublicKeysForAllValidators() []batchConsensusP256PublicKey {
	curve := elliptic.P256()
	out := make([]batchConsensusP256PublicKey, 0, len(batchConsensusCommonwareP256ProofSignerKeys))
	for _, rawHex := range batchConsensusCommonwareP256ProofSignerKeys {
		raw, err := hex.DecodeString(rawHex)
		if err != nil {
			panic(err)
		}
		x, y := curve.ScalarBaseMult(raw)
		var key batchConsensusP256PublicKey
		x.FillBytes(key.x[:])
		y.FillBytes(key.y[:])
		out = append(out, key)
	}
	return out
}

func batchConsensusMockVerifierCommonwareP256ProofCode(signers []batchConsensusP256PublicKey) []byte {
	if len(signers) != 3 {
		panic("Commonware P256 verifier expects exactly three quorum signers")
	}

	const (
		calldataSize       = 0x01e3
		outerDigestOffset  = 0x0009
		outerMarkerOffset  = 0x0029
		certificateOffset  = 0x00ed
		proposalOffset     = certificateOffset + 0x0009
		payloadOffset      = proposalOffset + 0x0003
		signersLenOffset   = proposalOffset + 0x0023
		bitmapOffset       = signersLenOffset + 0x0008
		signatureLenOffset = bitmapOffset + 0x0001
		signatureOffset    = signatureLenOffset + 0x0001
		proposalLen        = 0x0023
		hashInputLen       = 0x004d
		p256InputOffset    = 0x0080
		p256OutputOffset   = 0x0120
	)

	var code []byte
	emit := func(op ...byte) {
		code = append(code, op...)
	}
	push1 := func(v byte) {
		emit(0x60, v)
	}
	push2 := func(v uint16) {
		emit(0x61, byte(v>>8), byte(v))
	}
	pushN := func(v []byte) {
		if len(v) == 0 || len(v) > 32 {
			panic("invalid push size")
		}
		emit(byte(0x5f + len(v)))
		code = append(code, v...)
	}
	push32 := func(v []byte) {
		if len(v) != 32 {
			panic("invalid push32 size")
		}
		emit(0x7f)
		code = append(code, v...)
	}
	pushOffset := func(v uint16) {
		if v <= 0xff {
			push1(byte(v))
			return
		}
		push2(v)
	}
	push2Placeholder := func() int {
		emit(0x61, 0x00, 0x00)
		return len(code) - 2
	}
	var falseJumps []int
	jumpFalseIfTopIsTrue := func() {
		jump := push2Placeholder()
		emit(0x57)
		falseJumps = append(falseJumps, jump)
	}
	requireTopTrue := func() {
		emit(0x15) // ISZERO
		jumpFalseIfTopIsTrue()
	}
	requireByteEq := func(offset uint16, want byte) {
		push1(want)
		pushOffset(offset)
		emit(0x35) // CALLDATALOAD
		push1(0xf8)
		emit(0x1c, 0x14) // SHR EQ
		requireTopTrue()
	}
	requireWordEq := func(leftOffset, rightOffset uint16) {
		pushOffset(leftOffset)
		emit(0x35)
		pushOffset(rightOffset)
		emit(0x35, 0x14) // CALLDATALOAD EQ
		requireTopTrue()
	}

	push2(calldataSize)
	emit(0x36, 0x14) // CALLDATASIZE EQ
	requireTopTrue()

	pushN([]byte("CWSIMPLX1"))
	push1(0x00)
	emit(0x35)
	push1(byte(8 * (32 - len("CWSIMPLX1"))))
	emit(0x1c, 0x14) // SHR EQ
	requireTopTrue()

	pushN([]byte("CWSIMPLX1"))
	push2(certificateOffset)
	emit(0x35)
	push1(byte(8 * (32 - len("CWSIMPLX1"))))
	emit(0x1c, 0x14) // SHR EQ
	requireTopTrue()

	requireByteEq(outerMarkerOffset, 0x01)
	requireByteEq(bitmapOffset, 0x0e)
	requireByteEq(signatureLenOffset, 0x03)
	requireWordEq(outerDigestOffset, payloadOffset)

	push1(0x04)
	push2(signersLenOffset)
	emit(0x35)
	push1(0xc0)
	emit(0x1c, 0x14) // SHR EQ
	requireTopTrue()

	// Commonware secp256r1 signs sha256(union_unique(finalize_namespace, proposal.encode())).
	namespacePrefix := []byte("\x29op-batcher-consensus-poc/simplex_FINALIZE")
	first := make([]byte, 32)
	copy(first, namespacePrefix[:32])
	second := make([]byte, 32)
	copy(second, namespacePrefix[32:])
	push32(first)
	push1(0x00)
	emit(0x52) // MSTORE
	push32(second)
	push1(0x20)
	emit(0x52) // MSTORE
	push1(proposalLen)
	push2(proposalOffset)
	push1(0x2a)
	emit(0x37) // CALLDATACOPY

	push1(0x20)
	push1(0x00)
	push1(hashInputLen)
	push1(0x00)
	push1(0x02)
	emit(0x5a, 0xfa) // GAS STATICCALL
	requireTopTrue()

	for i, signer := range signers {
		sigOffset := uint16(signatureOffset + (i * 64))
		push1(0x00)
		emit(0x51)
		push1(p256InputOffset)
		emit(0x52)
		push2(sigOffset)
		emit(0x35)
		push1(p256InputOffset + 0x20)
		emit(0x52)
		push2(sigOffset + 0x20)
		emit(0x35)
		push1(p256InputOffset + 0x40)
		emit(0x52)
		push32(signer.x[:])
		push1(p256InputOffset + 0x60)
		emit(0x52)
		push32(signer.y[:])
		push2(p256InputOffset + 0x80)
		emit(0x52)
		push1(0x00)
		push2(p256OutputOffset)
		emit(0x52)

		push1(0x20)
		push2(p256OutputOffset)
		push1(0xa0)
		push1(p256InputOffset)
		push2(0x0100)
		emit(0x5a, 0xfa) // GAS STATICCALL
		requireTopTrue()

		push1(0x01)
		push2(p256OutputOffset)
		emit(0x51, 0x14) // MLOAD EQ
		requireTopTrue()
	}

	push1(0x01)
	push1(0x00)
	emit(0x52)
	push1(0x20)
	push1(0x00)
	emit(0xf3)

	falseLabel := len(code)
	emit(0x5b)
	push1(0x00)
	push1(0x00)
	emit(0x52)
	push1(0x20)
	push1(0x00)
	emit(0xf3)

	for _, jump := range falseJumps {
		code[jump] = byte(falseLabel >> 8)
		code[jump+1] = byte(falseLabel)
	}
	return code
}

type signatureProofLayout struct {
	signer       common.Address
	signers      []common.Address
	prefix       []byte
	exactSize    byte
	minSize      byte
	digestOffset byte
	markerOffset byte
	rOffset      byte
	sOffset      byte
	vOffset      byte
	sigOffset    byte
	certOffset   byte
}

func batchConsensusMockVerifierSignatureProofCode(layout signatureProofLayout) []byte {
	var code []byte
	emit := func(op ...byte) {
		code = append(code, op...)
	}
	push1 := func(v byte) {
		emit(0x60, v)
	}
	push2Placeholder := func() int {
		emit(0x61, 0x00, 0x00)
		return len(code) - 2
	}
	pushBytes := func(v []byte) {
		if len(v) == 0 || len(v) > 32 {
			panic("invalid push size")
		}
		emit(byte(0x5f + len(v)))
		code = append(code, v...)
	}
	var falseJumps []int
	signers := layout.signers
	if len(signers) == 0 {
		signers = []common.Address{layout.signer}
	}

	// Calldata length must match the selected proof envelope. The Commonware
	// envelope must also include at least one byte of certificate payload after
	// the signature-shaped EVM bridge proof.
	if layout.exactSize != 0 {
		push1(layout.exactSize)
		emit(0x36, 0x14, 0x15) // CALLDATASIZE EQ ISZERO
	} else {
		push1(layout.minSize)
		emit(0x36, 0x11, 0x15) // CALLDATASIZE GT ISZERO
	}
	falseJump := push2Placeholder()
	emit(0x57) // JUMPI
	falseJumps = append(falseJumps, falseJump)

	if len(layout.prefix) > 0 {
		pushBytes(layout.prefix)
		push1(0x00)
		emit(0x35) // CALLDATALOAD
		push1(byte(8 * (32 - len(layout.prefix))))
		emit(0x1c, 0x14, 0x15) // SHR EQ ISZERO
		falseJumpPrefix := push2Placeholder()
		emit(0x57) // JUMPI
		falseJumps = append(falseJumps, falseJumpPrefix)
	}

	// Valid marker must be 0x01.
	push1(0x01)
	push1(layout.markerOffset)
	emit(0x35) // CALLDATALOAD
	push1(0xf8)
	emit(0x1c, 0x14, 0x15) // SHR EQ ISZERO
	falseJump2 := push2Placeholder()
	emit(0x57) // JUMPI
	falseJumps = append(falseJumps, falseJump2)

	if layout.certOffset != 0 {
		// The Commonware verifier signs keccak256(prefix || digest || marker || certificate),
		// binding the EVM-checkable quorum proof to the exact certificate payload carried in calldata.
		push1(layout.sigOffset)
		push1(0x00)
		push1(0x00)
		emit(0x37) // CALLDATACOPY

		push1(layout.certOffset)
		emit(0x36, 0x03) // CALLDATASIZE SUB
		push1(layout.certOffset)
		push1(layout.sigOffset)
		emit(0x37) // CALLDATACOPY

		push1(layout.certOffset)
		emit(0x36, 0x03) // CALLDATASIZE SUB
		push1(layout.sigOffset)
		emit(0x01) // ADD
		push1(0x00)
		emit(0x20) // SHA3
		push1(0x00)
		emit(0x52)
	}

	for i, signer := range signers {
		rOffset := layout.rOffset
		sOffset := layout.sOffset
		vOffset := layout.vOffset
		if layout.sigOffset != 0 {
			sigOffset := int(layout.sigOffset) + (i * 65)
			rOffset = byte(sigOffset)
			sOffset = byte(sigOffset + 32)
			vOffset = byte(sigOffset + 64)
		}

		// ecrecover input: digest, v, r, s.
		if layout.certOffset == 0 {
			push1(layout.digestOffset)
			emit(0x35)
			push1(0x00)
			emit(0x52)
		}

		push1(0x1b)
		push1(vOffset)
		emit(0x35)
		push1(0xf8)
		emit(0x1c, 0x01) // SHR ADD
		push1(0x20)
		emit(0x52)

		push1(rOffset)
		emit(0x35)
		push1(0x40)
		emit(0x52)

		push1(sOffset)
		emit(0x35)
		push1(0x60)
		emit(0x52)

		push1(0x20)
		push1(0x80)
		push1(0x80)
		push1(0x00)
		push1(0x01)
		emit(0x5a, 0xfa, 0x15) // GAS STATICCALL ISZERO
		falseJump3 := push2Placeholder()
		emit(0x57) // JUMPI
		falseJumps = append(falseJumps, falseJump3)

		emit(0x73)
		code = append(code, signer.Bytes()...)
		push1(0x80)
		emit(0x51, 0x14, 0x15) // MLOAD EQ ISZERO
		falseJump4 := push2Placeholder()
		emit(0x57) // JUMPI
		falseJumps = append(falseJumps, falseJump4)
	}

	push1(0x01)
	push1(0x00)
	emit(0x52)
	push1(0x20)
	push1(0x00)
	emit(0xf3)

	falseLabel := len(code)
	emit(0x5b)
	push1(0x00)
	push1(0x00)
	emit(0x52)
	push1(0x20)
	push1(0x00)
	emit(0xf3)

	for _, jump := range falseJumps {
		code[jump] = byte(falseLabel >> 8)
		code[jump+1] = byte(falseLabel)
	}
	return code
}
