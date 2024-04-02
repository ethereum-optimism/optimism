package asterisc

import (
	"encoding/json"
	"fmt"
	"math/bits"
	"sort"

	"github.com/ethereum/go-ethereum/crypto"
)

type Memory struct {
	// generalized index -> merkle root or nil if invalidated
	nodes map[uint64]*[32]byte

	// pageIndex -> cached page
	pages map[uint64]*CachedPage

	// Note: since we don't de-alloc pages, we don't do ref-counting.
	// Once a page exists, it doesn't leave memory

	// two caches: we often read instructions from one page, and do memory things with another page.
	// this prevents map lookups each instruction
	lastPageKeys [2]uint64
	lastPage     [2]*CachedPage
}

type pageEntry struct {
	Index uint64 `json:"index"`
	Data  *Page  `json:"data"`
}

func NewMemory() *Memory {
	return &Memory{
		nodes:        make(map[uint64]*[32]byte),
		pages:        make(map[uint64]*CachedPage),
		lastPageKeys: [2]uint64{^uint64(0), ^uint64(0)}, // default to invalid keys, to not match any pages
	}
}

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

func (m *Memory) MarshalJSON() ([]byte, error) {
	pages := make([]pageEntry, 0, len(m.pages))
	for k, p := range m.pages {
		pages = append(pages, pageEntry{
			Index: k,
			Data:  p.Data,
		})
	}
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Index < pages[j].Index
	})
	return json.Marshal(pages)
}

func (m *Memory) UnmarshalJSON(data []byte) error {
	var pages []pageEntry
	if err := json.Unmarshal(data, &pages); err != nil {
		return err
	}
	m.nodes = make(map[uint64]*[32]byte)
	m.pages = make(map[uint64]*CachedPage)
	m.lastPageKeys = [2]uint64{^uint64(0), ^uint64(0)}
	m.lastPage = [2]*CachedPage{nil, nil}
	for i, p := range pages {
		if _, ok := m.pages[p.Index]; ok {
			return fmt.Errorf("cannot load duplicate page, entry %d, page index %d", i, p.Index)
		}
		m.AllocPage(p.Index).Data = p.Data
	}
	return nil
}

func (m *Memory) AllocPage(pageIndex uint64) *CachedPage {
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

func (m *Memory) MerkleRoot() [32]byte {
	return m.MerkleizeSubtree(1)
}

func (m *Memory) MerkleizeSubtree(gindex uint64) [32]byte {
	l := uint64(bits.Len64(gindex))
	if l > ProofLen {
		panic("gindex too deep")
	}
	if l > PageKeySize {
		depthIntoPage := l - 1 - PageKeySize
		pageIndex := (gindex >> depthIntoPage) & PageKeyMask
		if p, ok := m.pages[uint64(pageIndex)]; ok {
			pageGindex := (1 << depthIntoPage) | (gindex & ((1 << depthIntoPage) - 1))
			return p.MerkleizeSubtree(pageGindex)
		} else {
			return zeroHashes[64-5+1-l] // page does not exist
		}
	}
	if l > PageKeySize+1 {
		panic("cannot jump into intermediate node of page")
	}
	n, ok := m.nodes[gindex]
	if !ok {
		// if the node doesn't exist, the whole sub-tree is zeroed
		return zeroHashes[64-5+1-l]
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
