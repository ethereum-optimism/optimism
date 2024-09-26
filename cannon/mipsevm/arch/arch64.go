//go:build cannon64
// +build cannon64

package arch

import "encoding/binary"

type (
	// Word differs from the tradditional meaning in MIPS. The type represents the *maximum* architecture specific access length and value sizes
	Word = uint64
	// SignedInteger specifies the maximum signed integer type used for arithmetic.
	SignedInteger = int64
)

const (
	IsMips32      = false
	WordSize      = 64
	WordSizeBytes = WordSize >> 3
	PageAddrSize  = 12
	PageKeySize   = WordSize - PageAddrSize

	MemProofLeafCount = 60
	MemProofSize      = MemProofLeafCount * 32

	AddressMask = 0xFFFFFFFFFFFFFFF8
	ExtMask     = 0x7

	HeapStart       = 0x10_00_00_00_00_00_00_00
	HeapEnd         = 0x60_00_00_00_00_00_00_00
	ProgramBreak    = 0x40_00_00_00_00_00_00_00
	HighMemoryStart = 0x7F_FF_FF_FF_D0_00_00_00
)

var ByteOrderWord = byteOrder64{}

type byteOrder64 struct{}

func (bo byteOrder64) Word(b []byte) Word {
	return binary.BigEndian.Uint64(b)
}

func (bo byteOrder64) AppendWord(b []byte, v uint64) []byte {
	return binary.BigEndian.AppendUint64(b, v)
}

func (bo byteOrder64) PutWord(b []byte, v uint64) {
	binary.BigEndian.PutUint64(b, v)
}
