package exec

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
)

type MemTracker interface {
	TrackMemAccess(addr uint32)
}

type MemoryTrackerImpl struct {
	memory          *memory.Memory
	lastMemAccess   uint32
	memProofEnabled bool
	memProof        [memory.MEM_PROOF_SIZE]byte
}

func NewMemoryTracker(memory *memory.Memory) *MemoryTrackerImpl {
	return &MemoryTrackerImpl{memory: memory}
}

func (m *MemoryTrackerImpl) TrackMemAccess(effAddr uint32) {
	if m.memProofEnabled && m.lastMemAccess != effAddr {
		if m.lastMemAccess != ^uint32(0) {
			panic(fmt.Errorf("unexpected different mem access at %08x, already have access at %08x buffered", effAddr, m.lastMemAccess))
		}
		m.lastMemAccess = effAddr
		m.memProof = m.memory.MerkleProof(effAddr)
	}
}

func (m *MemoryTrackerImpl) Reset(enableProof bool) {
	m.memProofEnabled = enableProof
	m.lastMemAccess = ^uint32(0)
}

func (m *MemoryTrackerImpl) MemProof() [memory.MEM_PROOF_SIZE]byte {
	return m.memProof
}
