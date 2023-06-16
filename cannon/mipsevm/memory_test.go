package mipsevm

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryMerkleProof(t *testing.T) {
	t.Run("nearly empty tree", func(t *testing.T) {
		m := NewMemory()
		m.SetMemory(0x10000, 0xaabbccdd)
		proof := m.MerkleProof(0x10000)
		require.Equal(t, uint32(0xaabbccdd), binary.BigEndian.Uint32(proof[:4]))
		for i := 0; i < 32-5; i++ {
			require.Equal(t, zeroHashes[i][:], proof[32+i*32:32+i*32+32], "empty siblings")
		}
	})
	t.Run("fuller tree", func(t *testing.T) {
		m := NewMemory()
		m.SetMemory(0x10000, 0xaabbccdd)
		m.SetMemory(0x80004, 42)
		m.SetMemory(0x13370000, 123)
		root := m.MerkleRoot()
		proof := m.MerkleProof(0x80004)
		require.Equal(t, uint32(42), binary.BigEndian.Uint32(proof[4:8]))
		node := *(*[32]byte)(proof[:32])
		path := uint32(0x80004) >> 5
		for i := 32; i < len(proof); i += 32 {
			sib := *(*[32]byte)(proof[i : i+32])
			if path&1 != 0 {
				node = HashPair(sib, node)
			} else {
				node = HashPair(node, sib)
			}
			path >>= 1
		}
		require.Equal(t, root, node, "proof must verify")
	})
}

func TestMemoryMerkleRoot(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m := NewMemory()
		root := m.MerkleRoot()
		require.Equal(t, zeroHashes[32-5], root, "fully zeroed memory should have expected zero hash")
	})
	t.Run("empty page", func(t *testing.T) {
		m := NewMemory()
		m.SetMemory(0xF000, 0)
		root := m.MerkleRoot()
		require.Equal(t, zeroHashes[32-5], root, "fully zeroed memory should have expected zero hash")
	})
	t.Run("single page", func(t *testing.T) {
		m := NewMemory()
		m.SetMemory(0xF000, 1)
		root := m.MerkleRoot()
		require.NotEqual(t, zeroHashes[32-5], root, "non-zero memory")
	})
	t.Run("repeat zero", func(t *testing.T) {
		m := NewMemory()
		m.SetMemory(0xF000, 0)
		m.SetMemory(0xF004, 0)
		root := m.MerkleRoot()
		require.Equal(t, zeroHashes[32-5], root, "zero still")
	})
	t.Run("two empty pages", func(t *testing.T) {
		m := NewMemory()
		m.SetMemory(PageSize*3, 0)
		m.SetMemory(PageSize*10, 0)
		root := m.MerkleRoot()
		require.Equal(t, zeroHashes[32-5], root, "zero still")
	})
	t.Run("random few pages", func(t *testing.T) {
		m := NewMemory()
		m.SetMemory(PageSize*3, 1)
		m.SetMemory(PageSize*5, 42)
		m.SetMemory(PageSize*6, 123)
		p3 := m.MerkleizeSubtree((1 << PageKeySize) | 3)
		p5 := m.MerkleizeSubtree((1 << PageKeySize) | 5)
		p6 := m.MerkleizeSubtree((1 << PageKeySize) | 6)
		z := zeroHashes[PageAddrSize-5]
		r1 := HashPair(
			HashPair(
				HashPair(z, z),  // 0,1
				HashPair(z, p3), // 2,3
			),
			HashPair(
				HashPair(z, p5), // 4,5
				HashPair(p6, z), // 6,7
			),
		)
		r2 := m.MerkleizeSubtree(1 << (PageKeySize - 3))
		require.Equal(t, r1, r2, "expecting manual page combination to match subtree merkle func")
	})
	t.Run("invalidate page", func(t *testing.T) {
		m := NewMemory()
		m.SetMemory(0xF000, 0)
		require.Equal(t, zeroHashes[32-5], m.MerkleRoot(), "zero at first")
		m.SetMemory(0xF004, 1)
		require.NotEqual(t, zeroHashes[32-5], m.MerkleRoot(), "non-zero")
		m.SetMemory(0xF004, 0)
		require.Equal(t, zeroHashes[32-5], m.MerkleRoot(), "zero again")
	})
}

func TestMemoryReadWrite(t *testing.T) {

	t.Run("large random", func(t *testing.T) {
		m := NewMemory()
		data := make([]byte, 20_000)
		_, err := rand.Read(data[:])
		require.NoError(t, err)
		require.NoError(t, m.SetMemoryRange(0, bytes.NewReader(data)))
		for _, i := range []uint32{0, 4, 1000, 20_000 - 4} {
			v := m.GetMemory(i)
			expected := binary.BigEndian.Uint32(data[i : i+4])
			require.Equalf(t, expected, v, "read at %d", i)
		}
	})

	t.Run("repeat range", func(t *testing.T) {
		m := NewMemory()
		data := []byte(strings.Repeat("under the big bright yellow sun ", 40))
		require.NoError(t, m.SetMemoryRange(0x1337, bytes.NewReader(data)))
		res, err := io.ReadAll(m.ReadMemoryRange(0x1337-10, uint32(len(data)+20)))
		require.NoError(t, err)
		require.Equal(t, make([]byte, 10), res[:10], "empty start")
		require.Equal(t, data, res[10:len(res)-10], "result")
		require.Equal(t, make([]byte, 10), res[len(res)-10:], "empty end")
	})

	t.Run("read-write", func(t *testing.T) {
		m := NewMemory()
		m.SetMemory(12, 0xAABBCCDD)
		require.Equal(t, uint32(0xAABBCCDD), m.GetMemory(12))
		m.SetMemory(12, 0xAABB1CDD)
		require.Equal(t, uint32(0xAABB1CDD), m.GetMemory(12))
	})

	t.Run("unaligned read", func(t *testing.T) {
		m := NewMemory()
		m.SetMemory(12, 0xAABBCCDD)
		m.SetMemory(16, 0x11223344)
		require.Panics(t, func() {
			m.GetMemory(13)
		})
		require.Panics(t, func() {
			m.GetMemory(14)
		})
		require.Panics(t, func() {
			m.GetMemory(15)
		})
		require.Equal(t, uint32(0x11223344), m.GetMemory(16))
		require.Equal(t, uint32(0), m.GetMemory(20))
		require.Equal(t, uint32(0xAABBCCDD), m.GetMemory(12))
	})

	t.Run("unaligned write", func(t *testing.T) {
		m := NewMemory()
		m.SetMemory(12, 0xAABBCCDD)
		require.Panics(t, func() {
			m.SetMemory(13, 0x11223344)
		})
		require.Panics(t, func() {
			m.SetMemory(14, 0x11223344)
		})
		require.Panics(t, func() {
			m.SetMemory(15, 0x11223344)
		})
		require.Equal(t, uint32(0xAABBCCDD), m.GetMemory(12))
	})
}

func TestMemoryJSON(t *testing.T) {
	m := NewMemory()
	m.SetMemory(8, 123)
	dat, err := json.Marshal(m)
	require.NoError(t, err)
	var res Memory
	require.NoError(t, json.Unmarshal(dat, &res))
	require.Equal(t, uint32(123), res.GetMemory(8))
}
