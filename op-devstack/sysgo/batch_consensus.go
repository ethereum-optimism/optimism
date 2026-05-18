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
			code = batchConsensusMockVerifierCommonwareProofCode(batchConsensusMockProofSignerAddress())
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

func batchConsensusMockVerifierCommonwareProofCode(signer common.Address) []byte {
	return batchConsensusMockVerifierSignatureProofCode(signatureProofLayout{
		signer:       signer,
		prefix:       []byte("CWSIMPLX1"),
		minSize:      0x6b,
		digestOffset: 0x09,
		rOffset:      0x29,
		sOffset:      0x49,
		vOffset:      0x69,
		markerOffset: 0x6a,
	})
}

type signatureProofLayout struct {
	signer       common.Address
	prefix       []byte
	exactSize    byte
	minSize      byte
	digestOffset byte
	rOffset      byte
	sOffset      byte
	vOffset      byte
	markerOffset byte
}

func batchConsensusMockVerifierSignatureProofCode(layout signatureProofLayout) []byte {
	var code []byte
	emit := func(op ...byte) {
		code = append(code, op...)
	}
	push1 := func(v byte) {
		emit(0x60, v)
	}
	pushBytes := func(v []byte) {
		if len(v) == 0 || len(v) > 32 {
			panic("invalid push size")
		}
		emit(byte(0x5f + len(v)))
		code = append(code, v...)
	}
	var falseJumps []int

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
	push1(0x00)
	falseJump := len(code) - 1
	emit(0x57) // JUMPI
	falseJumps = append(falseJumps, falseJump)

	if len(layout.prefix) > 0 {
		pushBytes(layout.prefix)
		push1(0x00)
		emit(0x35) // CALLDATALOAD
		push1(byte(8 * (32 - len(layout.prefix))))
		emit(0x1c, 0x14, 0x15) // SHR EQ ISZERO
		push1(0x00)
		falseJumpPrefix := len(code) - 1
		emit(0x57) // JUMPI
		falseJumps = append(falseJumps, falseJumpPrefix)
	}

	// Valid marker must be 0x01.
	push1(0x01)
	push1(layout.markerOffset)
	emit(0x35) // CALLDATALOAD
	push1(0xf8)
	emit(0x1c, 0x14, 0x15) // SHR EQ ISZERO
	push1(0x00)
	falseJump2 := len(code) - 1
	emit(0x57) // JUMPI
	falseJumps = append(falseJumps, falseJump2)

	// ecrecover input: digest, v, r, s.
	push1(layout.digestOffset)
	emit(0x35)
	push1(0x00)
	emit(0x52)

	push1(0x1b)
	push1(layout.vOffset)
	emit(0x35)
	push1(0xf8)
	emit(0x1c, 0x01) // SHR ADD
	push1(0x20)
	emit(0x52)

	push1(layout.rOffset)
	emit(0x35)
	push1(0x40)
	emit(0x52)

	push1(layout.sOffset)
	emit(0x35)
	push1(0x60)
	emit(0x52)

	push1(0x20)
	push1(0x80)
	push1(0x80)
	push1(0x00)
	push1(0x01)
	emit(0x5a, 0xfa, 0x15) // GAS STATICCALL ISZERO
	push1(0x00)
	falseJump3 := len(code) - 1
	emit(0x57) // JUMPI
	falseJumps = append(falseJumps, falseJump3)

	emit(0x73)
	code = append(code, layout.signer.Bytes()...)
	push1(0x80)
	emit(0x51, 0x14) // MLOAD EQ
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
		code[jump] = byte(falseLabel)
	}
	return code
}
