//go:build !cannon64
// +build !cannon64

package arch

import "encoding/binary"

type (
	// Word differs from the tradditional meaning in MIPS. The type represents the *maximum* architecture specific access length and value sizes.
	Word = uint32
	// SignedInteger specifies the maximum signed integer type used for arithmetic.
	SignedInteger = int32
)

const (
	IsMips32      = true
	WordSize      = 32
	WordSizeBytes = WordSize >> 3
	PageAddrSize  = 12
	PageKeySize   = WordSize - PageAddrSize

	MemProofLeafCount = 28
	MemProofSize      = MemProofLeafCount * 32

	AddressMask = 0xFFffFFfc
	ExtMask     = 0x3

	HeapStart       = 0x05_00_00_00
	HeapEnd         = 0x60_00_00_00
	ProgramBreak    = 0x40_00_00_00
	HighMemoryStart = 0x7f_ff_d0_00
)

var ByteOrderWord = byteOrder32{}

type byteOrder32 struct{}

func (bo byteOrder32) Word(b []byte) Word {
	return binary.BigEndian.Uint32(b)
}

func (bo byteOrder32) AppendWord(b []byte, v uint32) []byte {
	return binary.BigEndian.AppendUint32(b, v)
}

func (bo byteOrder32) PutWord(b []byte, v uint32) {
	binary.BigEndian.PutUint32(b, v)
}
