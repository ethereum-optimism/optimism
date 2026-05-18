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
		if cfg.BatchConsensusMockProofSidecar || cfg.BatchConsensusCommonwareSidecar {
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
	var code []byte
	emit := func(op ...byte) {
		code = append(code, op...)
	}
	push1 := func(v byte) {
		emit(0x60, v)
	}

	// Calldata length must be 104 bytes: "BCSIG1" || digest || 65-byte signature || valid marker.
	push1(0x68)
	emit(0x36, 0x14, 0x15) // CALLDATASIZE EQ ISZERO
	push1(0x00)
	falseJump := len(code) - 1
	emit(0x57) // JUMPI

	// Valid marker must be 0x01.
	push1(0x01)
	push1(0x67)
	emit(0x35) // CALLDATALOAD
	push1(0xf8)
	emit(0x1c, 0x14, 0x15) // SHR EQ ISZERO
	push1(0x00)
	falseJump2 := len(code) - 1
	emit(0x57) // JUMPI

	// ecrecover input: digest, v, r, s.
	push1(0x06)
	emit(0x35)
	push1(0x00)
	emit(0x52)

	push1(0x1b)
	push1(0x66)
	emit(0x35)
	push1(0xf8)
	emit(0x1c, 0x01) // SHR ADD
	push1(0x20)
	emit(0x52)

	push1(0x26)
	emit(0x35)
	push1(0x40)
	emit(0x52)

	push1(0x46)
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

	emit(0x73)
	code = append(code, signer.Bytes()...)
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

	code[falseJump] = byte(falseLabel)
	code[falseJump2] = byte(falseLabel)
	code[falseJump3] = byte(falseLabel)
	return code
}
