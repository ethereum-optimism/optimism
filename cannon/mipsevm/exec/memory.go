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
	// proof of first unique memory access
	memProof [memory.MEM_PROOF_SIZE]byte
	// proof of second unique memory access
	memProof2 [memory.MEM_PROOF_SIZE]byte
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

// TrackMemAccess2 creates a proof for a memory access following a call to TrackMemAccess
// This is used to generate proofs for contiguous memory accesses within the same step
func (m *MemoryTrackerImpl) TrackMemAccess2(effAddr uint32) {
	if m.memProofEnabled && m.lastMemAccess+4 != effAddr {
		panic(fmt.Errorf("unexpected disjointed mem access at %08x, last memory access is at %08x buffered", effAddr, m.lastMemAccess))
	}
	m.lastMemAccess = effAddr
	m.memProof2 = m.memory.MerkleProof(effAddr)
}

func (m *MemoryTrackerImpl) Reset(enableProof bool) {
	m.memProofEnabled = enableProof
	m.lastMemAccess = ^uint32(0)
}

func (m *MemoryTrackerImpl) MemProof() [memory.MEM_PROOF_SIZE]byte {
	return m.memProof
}

func (m *MemoryTrackerImpl) MemProof2() [memory.MEM_PROOF_SIZE]byte {
	return m.memProof2
}
