package mipsevm

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"

	"github.com/ethereum/go-ethereum/crypto"
)

// Note: 2**12 = 4 KiB, the min phys page size in the Go runtime.
const (
	PageAddrSize = 12
	PageKeySize  = 32 - PageAddrSize
	PageSize     = 1 << PageAddrSize
	PageAddrMask = PageSize - 1
	MaxPageCount = 1 << PageKeySize
	PageKeyMask  = MaxPageCount - 1
)

func HashPair(left, right [32]byte) [32]byte {
	out := crypto.Keccak256Hash(left[:], right[:])
	//fmt.Printf("0x%x 0x%x -> 0x%x\n", left, right, out)
	return out
}

var zeroHashes = func() [256][32]byte {
	// empty parts of the tree are all zero. Precompute the hash of each full-zero range sub-tree level.
	var out [256][32]byte
	for i := 1; i < 256; i++ {
		out[i] = HashPair(out[i-1], out[i-1])
	}
	return out
}()

type Memory struct {
	// generalized index -> merkle root or nil if invalidated
	nodes map[uint64]*[32]byte

	// pageIndex -> cached page
	pages map[uint32]*CachedPage

	// Note: since we don't de-alloc pages, we don't do ref-counting.
	// Once a page exists, it doesn't leave memory

	// two caches: we often read instructions from one page, and do memory things with another page.
	// this prevents map lookups each instruction
	lastPageKeys [2]uint32
	lastPage     [2]*CachedPage
}

func NewMemory() *Memory {
	return &Memory{
		nodes:        make(map[uint64]*[32]byte),
		pages:        make(map[uint32]*CachedPage),
		lastPageKeys: [2]uint32{^uint32(0), ^uint32(0)}, // default to invalid keys, to not match any pages
	}
}

func (m *Memory) PageCount() int {
	return len(m.pages)
}

func (m *Memory) ForEachPage(fn func(pageIndex uint32, page *Page) error) error {
	for pageIndex, cachedPage := range m.pages {
		if err := fn(pageIndex, cachedPage.Data); err != nil {
			return err
		}
	}
	return nil
}

func (m *Memory) Invalidate(addr uint32) {
	// addr must be aligned to 4 bytes
	if addr&0x3 != 0 {
		panic(fmt.Errorf("unaligned memory access: %x", addr))
	}

	// find page, and invalidate addr within it
	if p, ok := m.pageLookup(addr >> PageAddrSize); ok {
		prevValid := p.Ok[1]
		p.Invalidate(addr & PageAddrMask)
		if !prevValid { // if the page was already invalid before, then nodes to mem-root will also still be.
			return
		}
	} else { // no page? nothing to invalidate
		return
	}

	// find the gindex of the first page covering the address
	gindex := ((uint64(1) << 32) | uint64(addr)) >> PageAddrSize

	for gindex > 0 {
		m.nodes[gindex] = nil
		gindex >>= 1
	}
}

func (m *Memory) MerkleizeSubtree(gindex uint64) [32]byte {
	l := uint64(bits.Len64(gindex))
	if l > 28 {
		panic("gindex too deep")
	}
	if l > PageKeySize {
		depthIntoPage := l - 1 - PageKeySize
		pageIndex := (gindex >> depthIntoPage) & PageKeyMask
		if p, ok := m.pages[uint32(pageIndex)]; ok {
			pageGindex := (1 << depthIntoPage) | (gindex & ((1 << depthIntoPage) - 1))
			return p.MerkleizeSubtree(pageGindex)
		} else {
			return zeroHashes[28-l] // page does not exist
		}
	}
	if l > PageKeySize+1 {
		panic("cannot jump into intermediate node of page")
	}
	n, ok := m.nodes[gindex]
	if !ok {
		// if the node doesn't exist, the whole sub-tree is zeroed
		return zeroHashes[28-l]
	}
	if n != nil {
		return *n
	}
	left := m.MerkleizeSubtree(gindex << 1)
	right := m.MerkleizeSubtree((gindex << 1) | 1)
	r := HashPair(left, right)
	m.nodes[gindex] = &r
	return r
}

func (m *Memory) MerkleProof(addr uint32) (out [28 * 32]byte) {
	proof := m.traverseBranch(1, addr, 0)
	// encode the proof
	for i := 0; i < 28; i++ {
		copy(out[i*32:(i+1)*32], proof[i][:])
	}
	return out
}

func (m *Memory) traverseBranch(parent uint64, addr uint32, depth uint8) (proof [][32]byte) {
	if depth == 32-5 {
		proof = make([][32]byte, 0, 32-5+1)
		proof = append(proof, m.MerkleizeSubtree(parent))
		return
	}
	if depth > 32-5 {
		panic("traversed too deep")
	}
	self := parent << 1
	sibling := self | 1
	if addr&(1<<(31-depth)) != 0 {
		self, sibling = sibling, self
	}
	proof = m.traverseBranch(self, addr, depth+1)
	siblingNode := m.MerkleizeSubtree(sibling)
	proof = append(proof, siblingNode)
	return
}

func (m *Memory) MerkleRoot() [32]byte {
	return m.MerkleizeSubtree(1)
}

func (m *Memory) pageLookup(pageIndex uint32) (*CachedPage, bool) {
	// hit caches
	if pageIndex == m.lastPageKeys[0] {
		return m.lastPage[0], true
	}
	if pageIndex == m.lastPageKeys[1] {
		return m.lastPage[1], true
	}
	p, ok := m.pages[pageIndex]

	// only cache existing pages.
	if ok {
		m.lastPageKeys[1] = m.lastPageKeys[0]
		m.lastPage[1] = m.lastPage[0]
		m.lastPageKeys[0] = pageIndex
		m.lastPage[0] = p
	}

	return p, ok
}

func (m *Memory) SetMemory(addr uint32, v uint32) {
	// addr must be aligned to 4 bytes
	if addr&0x3 != 0 {
		panic(fmt.Errorf("unaligned memory access: %x", addr))
	}

	pageIndex := addr >> PageAddrSize
	pageAddr := addr & PageAddrMask
	p, ok := m.pageLookup(pageIndex)
	if !ok {
		// allocate the page if we have not already.
		// Go may mmap relatively large ranges, but we only allocate the pages just in time.
		p = m.AllocPage(pageIndex)
	} else {
		m.Invalidate(addr) // invalidate this branch of memory, now that the value changed
	}
	binary.BigEndian.PutUint32(p.Data[pageAddr:pageAddr+4], v)
}

func (m *Memory) GetMemory(addr uint32) uint32 {
	// addr must be aligned to 4 bytes
	if addr&0x3 != 0 {
		panic(fmt.Errorf("unaligned memory access: %x", addr))
	}
	p, ok := m.pageLookup(addr >> PageAddrSize)
	if !ok {
		return 0
	}
	pageAddr := addr & PageAddrMask
	return binary.BigEndian.Uint32(p.Data[pageAddr : pageAddr+4])
}

func (m *Memory) AllocPage(pageIndex uint32) *CachedPage {
	p := &CachedPage{Data: new(Page)}
	m.pages[pageIndex] = p
	// make nodes to root
	k := (1 << PageKeySize) | uint64(pageIndex)
	for k > 0 {
		m.nodes[k] = nil
		k >>= 1
	}
	return p
}

// Serialize serializes a `Memory` struct to a byte slice.
func (m *Memory) Serialize(out io.Writer) error {
	// Write the version byte to the output
	if err := binary.Write(out, binary.BigEndian, uint8(0)); err != nil {
		return err
	}
	for k, p := range m.pages {
		// Write the page index as a big endian uint32
		if err := binary.Write(out, binary.BigEndian, k); err != nil {
			return err
		}
		// Write the length of the page data as a big endian uint32
		if err := binary.Write(out, binary.BigEndian, uint32(len(p.Data))); err != nil {
			return err
		}
		// Write the page data
		n, err := out.Write(p.Data[:])
		if err != nil {
			return err
		}
		if n != len(p.Data) {
			return fmt.Errorf("failed to write full page data")
		}
	}

	return nil
}

// Deserialize deserializes a `Memory` struct from a byte slice.
func (m *Memory) Deserialize(in io.Reader) error {
	// Read the version byte from the input
	var version uint8
	if err := binary.Read(in, binary.BigEndian, &version); err != nil {
		return err
	}
	if version != 0 {
		return fmt.Errorf("incorrect memory encoding version %d", version)
	}
	for {
		// Read the page index as a big endian uint32
		var pageIndex uint32
		err := binary.Read(in, binary.BigEndian, &pageIndex)
		if err == io.EOF {
			break // Exit the loop on EOF
		} else if err != nil {
			return err
		}

		// Check if there was already a page with this index
		if _, ok := m.pages[pageIndex]; ok {
			return fmt.Errorf("cannot load duplicate page, page index %d", pageIndex)
		}

		// Read the length of the page data as a big endian uint32
		var pageDataLen uint32
		err = binary.Read(in, binary.BigEndian, &pageDataLen)
		if err != nil {
			return err
		}

		if pageDataLen > PageSize {
			return fmt.Errorf("page data length exceeds PageSize")
		}

		// Read the page data into the pageBuffer
		var page Page
		n, err := in.Read(page[:pageDataLen]) // Read directly into pageBuffer
		if err != nil {
			return err
		}
		if uint32(n) != pageDataLen {
			return fmt.Errorf("failed to read full page data")
		}

		// Allocate the page and assign the data
		m.AllocPage(pageIndex).Data = &page
	}

	return nil
}

func (m *Memory) SetMemoryRange(addr uint32, r io.Reader) error {
	for {
		pageIndex := addr >> PageAddrSize
		pageAddr := addr & PageAddrMask
		p, ok := m.pageLookup(pageIndex)
		if !ok {
			p = m.AllocPage(pageIndex)
		}
		p.InvalidateFull()
		n, err := r.Read(p.Data[pageAddr:])
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		addr += uint32(n)
	}
}

type memReader struct {
	m     *Memory
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

	pageIndex := r.addr >> PageAddrSize
	start := r.addr & PageAddrMask
	end := uint32(PageSize)

	if pageIndex == (endAddr >> PageAddrSize) {
		end = endAddr & PageAddrMask
	}
	p, ok := r.m.pageLookup(pageIndex)
	if ok {
		n = copy(dest, p.Data[start:end])
	} else {
		n = copy(dest, make([]byte, end-start)) // default to zeroes
	}
	r.addr += uint32(n)
	r.count -= uint32(n)
	return n, nil
}

func (m *Memory) ReadMemoryRange(addr uint32, count uint32) io.Reader {
	return &memReader{m: m, addr: addr, count: count}
}

func (m *Memory) Usage() string {
	total := uint64(len(m.pages)) * PageSize
	const unit = 1024
	if total < unit {
		return fmt.Sprintf("%d B", total)
	}
	div, exp := uint64(unit), 0
	for n := total / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	// KiB, MiB, GiB, TiB, ...
	return fmt.Sprintf("%.1f %ciB", float64(total)/float64(div), "KMGTPE"[exp])
}
