package memory

import (
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
)

// byte32Pool is a sync.Pool for [32]byte slices
var byte32Pool = sync.Pool{
	New: func() interface{} {
		var b [32]byte
		return &b // Return a pointer to avoid extra allocations
	},
}

// GetByte32 retrieves a *[32]byte from the pool
func GetByte32() *[32]byte {
	return byte32Pool.Get().(*[32]byte)
}

// ReleaseByte32 returns a *[32]byte to the pool
func ReleaseByte32(b *[32]byte) {
	// Optional: Zero the array before putting it back
	*b = [32]byte{}
	byte32Pool.Put(b)
}

var hashPool = sync.Pool{
	New: func() interface{} {
		return crypto.NewKeccakState()
	},
}

func GetHasher() crypto.KeccakState {
	return hashPool.Get().(crypto.KeccakState)
}

func PutHasher(h crypto.KeccakState) {
	h.Reset()
	hashPool.Put(h)
}

func HashPairNodes(out *[32]byte, left, right *[32]byte) {
	h := GetHasher()
	h.Write(left[:])
	h.Write(right[:])
	_, _ = h.Read(out[:])
	PutHasher(h)
}

func HashData(out *[32]byte, data ...[]byte) {
	h := GetHasher()
	for _, b := range data {
		h.Write(b)
	}
	_, _ = h.Read(out[:])
	PutHasher(h)
}

func HashPair(left, right [32]byte) (out [32]byte) {
	HashPairNodes(&out, &left, &right)
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
