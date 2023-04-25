package mipsevm

import "fmt"

type MemEntry struct {
	EffAddr  uint32
	PreValue uint32
}

type AccessList struct {
	mem *Memory

	memAccessAddr uint32

	proofData []byte
}

func (al *AccessList) Reset() {
	al.memAccessAddr = ^uint32(0)
	al.proofData = nil
}

func (al *AccessList) OnRead(effAddr uint32) {
	if al.memAccessAddr == effAddr {
		return
	}
	if al.memAccessAddr != ^uint32(0) {
		panic(fmt.Errorf("bad read of %08x, already have %08x", effAddr, al.memAccessAddr))
	}
	al.memAccessAddr = effAddr
	proof := al.mem.MerkleProof(effAddr)
	al.proofData = append(al.proofData, proof[:]...)
}

func (al *AccessList) OnWrite(effAddr uint32) {
	if al.memAccessAddr == effAddr {
		return
	}
	if al.memAccessAddr != ^uint32(0) {
		panic(fmt.Errorf("bad write of %08x, already have %08x", effAddr, al.memAccessAddr))
	}
	proof := al.mem.MerkleProof(effAddr)
	al.proofData = append(al.proofData, proof[:]...)
}

func (al *AccessList) PreInstruction(pc uint32) {
	proof := al.mem.MerkleProof(pc)
	al.proofData = append(al.proofData, proof[:]...)
}

var _ Tracer = (*AccessList)(nil)

type Tracer interface {
	// OnRead remembers reads from the given effAddr.
	// Warning: the addr is an effective-addr, i.e. always aligned.
	// But unicorn may fire it multiple times, for each byte that was changed within the effective addr boundaries.
	OnRead(effAddr uint32)
	// OnWrite remembers writes to the given effAddr.
	// Warning: the addr is an effective-addr, i.e. always aligned.
	// But unicorn may fire it multiple times, for each byte that was changed within the effective addr boundaries.
	OnWrite(effAddr uint32)

	PreInstruction(pc uint32)
}

type NoOpTracer struct{}

func (n NoOpTracer) OnRead(effAddr uint32) {}

func (n NoOpTracer) OnWrite(effAddr uint32) {}

func (n NoOpTracer) PreInstruction(pc uint32) {}

var _ Tracer = NoOpTracer{}
