package mipsevm

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
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

type CachedPage struct {
	Data *Page
	// intermediate nodes only
	Cache [pageSize / 32][32]byte
	// true if the intermediate node is valid
	Ok [pageSize / 32]bool
}

func (p *CachedPage) Invalidate(pageAddr uint32) {
	if pageAddr >= pageSize {
		panic("invalid page addr")
	}
	k := (1 << pageAddrSize) | pageAddr
	// first cache layer caches nodes that has two 32 byte leaf nodes.
	k >>= 5 + 1
	for k > 0 {
		p.Ok[k] = false
		k >>= 1
	}
}

func (p *CachedPage) InvalidateFull() {
	p.Ok = [pageSize / 32]bool{} // reset everything to false
}

func (p *CachedPage) MerkleRoot() [32]byte {
	// hash the bottom layer
	for i := uint64(0); i < pageSize; i += 64 {
		j := pageSize/32/2 + i/64
		if p.Ok[j] {
			continue
		}
		p.Cache[j] = crypto.Keccak256Hash(p.Data[i : i+64])
		p.Ok[j] = true
	}

	// hash the cache layers
	for i := pageSize/32 - 2; i > 0; i++ {
		j := i >> 1
		if p.Ok[j] {
			continue
		}
		p.Cache[j] = HashPair(p.Cache[i], p.Cache[i+1])
		p.Ok[j] = true
	}

	return p.Cache[1]
}

func (p *CachedPage) MerkleizeSubtree(gindex uint64) [32]byte {
	_ = p.MerkleRoot() // fill cache
	if gindex >= pageSize/32 {
		if gindex >= pageSize/32*2 {
			panic("gindex too deep")
		}
		// it's pointing to a bottom node
		nodeIndex := gindex & pageAddrMask
		return *(*[32]byte)(p.Data[nodeIndex*32 : nodeIndex*32+32])
	}
	return p.Cache[gindex]
}
