package main

import (
	"encoding/hex"
	"fmt"
	"io"
)

const (
	pageAddrSize = 10
	pageSize     = 1 << pageAddrSize
	pageAddrMask = pageSize - 1
)

type Page [pageSize]byte

func (p *Page) MarshalText() ([]byte, error) {
	dst := make([]byte, hex.EncodedLen(len(p)))
	hex.Encode(dst, p[:])
	return dst, nil
}

func (p *Page) UnmarshalText(dat []byte) error {
	if len(dat) != pageSize*2 {
		return fmt.Errorf("expected %d hex chars, but got %d", pageSize*2, len(dat))
	}
	_, err := hex.Decode(p[:], dat)
	return err
}

type State struct {
	PC   uint32 `json:"pc"`
	Hi   uint32 `json:"hi"`
	Lo   uint32 `json:"lo"`
	Heap uint32 `json:"heap"` // to handle mmap growth

	Registers [32]uint32 `json:"registers"`

	Memory map[uint32]*Page `json:"memory"`

	Exit   uint32 `json:"exit"`
	Exited bool   `json:"exited"`

	Step uint64 `json:"step"`
}

// TODO: VM state pre-image:
// PC, Hi, Lo, Heap = 4 * 32/8 = 16 bytes
// Registers = 32 * 32/8 = 256 bytes
// Memory tree root = 32 bytes
// Misc exit/step data = TBD
// + proof(s) for memory leaf nodes

func (s *State) ApplyRegisterDiff(regs [32]uint32, hi, lo uint32) {
	for i := 0; i < 32; i++ {
		s.Registers[i] = regs[i]
	}
	s.Hi = hi
	s.Lo = lo
}

func (s *State) SetMemory(addr uint32, v uint32, size uint32) {
	for i := size; i > 0; i-- {
		pageIndex := addr >> pageAddrSize
		pageAddr := addr & pageAddrMask
		p, ok := s.Memory[pageIndex]
		if !ok {
			panic(fmt.Errorf("missing page %x (addr write at %x)", pageIndex, addr))
		}
		b := uint8(v)
		p[pageAddr] = b
		v = v >> 8
		addr += 1
	}
}

type memReader struct {
	state *State
	addr  uint32
	count uint32
}

func (r *memReader) Read(dest []byte) (n int, err error) {
	if r.count == 0 {
		return 0, io.EOF
	}

	// Keep iterating over memory until we have all our data.
	// It may wrap around the address range, and may not be aligned
	endAddr := r.addr + r.count

	pageIndex := r.addr >> pageAddrSize
	start := r.addr & pageAddrMask
	end := uint32(pageSize)

	if pageIndex == (endAddr >> pageAddrSize) {
		end = endAddr & pageAddrMask
	}
	p, ok := r.state.Memory[pageIndex]
	if ok {
		n = copy(dest, p[start:end])
	} else {
		n = copy(dest, make([]byte, end-start)) // default to zeroes
	}
	r.addr += uint32(n)
	r.count -= uint32(n)
	return n, nil
}

func (s *State) ReadMemory(addr uint32, count uint32) io.Reader {
	return &memReader{state: s, addr: addr, count: count}
}

// TODO merkleization

// TODO convert access-list to calldata and state-sets for EVM
