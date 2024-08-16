package exec

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
)

type MemTracker interface {
	TrackMemAccess(addr uint64)
}

type MemoryTrackerImpl struct {
	memory          *memory.Memory
	lastMemAccess   uint64
	memProofEnabled bool
	memProof        [memory.MEM_PROOF_SIZE]byte
}

func NewMemoryTracker(memory *memory.Memory) *MemoryTrackerImpl {
	return &MemoryTrackerImpl{memory: memory}
}

func (m *MemoryTrackerImpl) TrackMemAccess(effAddr uint64) {
	if m.memProofEnabled && m.lastMemAccess != effAddr {
		if m.lastMemAccess != ^uint64(0) {
			panic(fmt.Errorf("unexpected different mem access at %08x, already have access at %08x buffered", effAddr, m.lastMemAccess))
		}
		m.lastMemAccess = effAddr
		m.memProof = m.memory.MerkleProof(effAddr)
	}
}

func (m *MemoryTrackerImpl) Reset(enableProof bool) {
	m.memProofEnabled = enableProof
	m.lastMemAccess = ^uint64(0)
}

func (m *MemoryTrackerImpl) MemProof() [memory.MEM_PROOF_SIZE]byte {
	return m.memProof
}
