package sysgo

import (
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

func batchConsensusMockVerifierCommonwareProofCode(signers []common.Address) []byte {
	return batchConsensusMockVerifierSignatureProofCode(signatureProofLayout{
		signers:      signers,
		prefix:       []byte("CWSIMPLX1"),
		minSize:      0xee,
		digestOffset: 0x09,
		markerOffset: 0x29,
		sigOffset:    0x2a,
	})
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
		push1(layout.digestOffset)
		emit(0x35)
		push1(0x00)
		emit(0x52)

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
