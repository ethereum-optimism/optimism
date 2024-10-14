package testutil

import "github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"

type Word = arch.Word

// PlaceUint32InWord places a uint32 value into a Word based on the architecture and address.
// For example, in 64-bit architectures, a 32-bit value written to address 0x04 is
// written to the rightmost 32-bits of the word at address 0x00, while a 32-bit value written to address 0x08
// is written to the leftmost 32-bits of the 64-bit Word at address 0x08
func PlaceUint32InWord(addr, value Word) Word {
	offsetMask := Word(arch.ExtMask) & 0x4
	wordOffset := addr & offsetMask
	maxWordByteOffset := Word(arch.WordSizeBytes - 4)
	memBitOffset := (maxWordByteOffset - wordOffset) * 8

	return (value & 0xFFFF_FFFF) << memBitOffset
}

// SignExtendImmediate takes a 16-bit value and sign- or zero- extends it up to the arch.Word size
func SignExtendImmediate(imm Word) (Word, bool) {
	signExtended := false
	immediateMask := Word(0xFFFF)
	imm = imm & immediateMask

	if imm>>15 == 0x01 {
		// Sign extend
		imm = imm | ^immediateMask
		signExtended = true
	}

	return imm, signExtended
}
