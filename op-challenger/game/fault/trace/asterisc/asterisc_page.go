package asterisc

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

// Duplicated from https://github.com/ethereum-optimism/asterisc/blob/4c4705809b4adbb854d265f76ace719b08c732e6/rvgo/fast/page.go

const (
	PageAddrSize = 12
	PageKeySize  = 64 - PageAddrSize
	PageSize     = 1 << PageAddrSize
	PageAddrMask = PageSize - 1
	MaxPageCount = 1 << PageKeySize
	PageKeyMask  = MaxPageCount - 1
	ProofLen     = 64 - 4
)

type Page [PageSize]byte

func (p *Page) MarshalText() ([]byte, error) {
	dst := make([]byte, hex.EncodedLen(len(p)))
	hex.Encode(dst, p[:])
	return dst, nil
}

func (p *Page) UnmarshalText(dat []byte) error {
	if len(dat) != PageSize*2 {
		return fmt.Errorf("expected %d hex chars, but got %d", PageSize*2, len(dat))
	}
	_, err := hex.Decode(p[:], dat)
	return err
}

type CachedPage struct {
	Data *Page
	// intermediate nodes only
	Cache [PageSize / 32][32]byte
	// true if the intermediate node is valid
	Ok [PageSize / 32]bool
}

func (p *CachedPage) MerkleRoot() [32]byte {
	// hash the bottom layer
	for i := uint64(0); i < PageSize; i += 64 {
		j := PageSize/32/2 + i/64
		if p.Ok[j] {
			continue
		}
		p.Cache[j] = crypto.Keccak256Hash(p.Data[i : i+64])
		//fmt.Printf("0x%x 0x%x -> 0x%x\n", p.Data[i:i+32], p.Data[i+32:i+64], p.Cache[j])
		p.Ok[j] = true
	}

	// hash the cache layers
	for i := PageSize/32 - 2; i > 0; i -= 2 {
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
	if gindex >= PageSize/32 {
		if gindex >= PageSize/32*2 {
			panic("gindex too deep")
		}
		// it's pointing to a bottom node
		nodeIndex := gindex & (PageAddrMask >> 5)
		return *(*[32]byte)(p.Data[nodeIndex*32 : nodeIndex*32+32])
	}
	return p.Cache[gindex]
}
