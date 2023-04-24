package mipsevm

type MemEntry struct {
	EffAddr  uint32
	PreValue uint32
}

type AccessList struct {
	memReads  []MemEntry
	memWrites []MemEntry
}

func (al *AccessList) Reset() {
	al.memReads = al.memReads[:0]
	al.memWrites = al.memWrites[:0]
}

func (al *AccessList) OnRead(effAddr uint32, preValue uint32) {
	// if it matches the last, it's a duplicate; this happens because of multiple callbacks for the same effective addr.
	if len(al.memReads) > 0 && al.memReads[len(al.memReads)-1].EffAddr == effAddr {
		return
	}
	al.memReads = append(al.memReads, MemEntry{EffAddr: effAddr, PreValue: preValue})
}

func (al *AccessList) OnWrite(effAddr uint32, preValue uint32) {
	// if it matches the last, it's a duplicate; this happens because of multiple callbacks for the same effective addr.
	if len(al.memWrites) > 0 && al.memWrites[len(al.memWrites)-1].EffAddr == effAddr {
		return
	}
	al.memWrites = append(al.memWrites, MemEntry{EffAddr: effAddr, PreValue: preValue})
}

var _ Tracer = (*AccessList)(nil)

type Tracer interface {
	// OnRead remembers reads from the given effAddr.
	// Warning: the addr is an effective-addr, i.e. always aligned.
	// But unicorn will fire it multiple times, for each byte that was changed within the effective addr boundaries.
	OnRead(effAddr uint32, value uint32)
	// OnWrite remembers writes to the given effAddr.
	// Warning: the addr is an effective-addr, i.e. always aligned.
	// But unicorn will fire it multiple times, for each byte that was changed within the effective addr boundaries.
	OnWrite(effAddr uint32, value uint32)
}

type NoOpTracer struct{}

func (n NoOpTracer) OnRead(effAddr uint32, value uint32) {}

func (n NoOpTracer) OnWrite(effAddr uint32, value uint32) {}

var _ Tracer = NoOpTracer{}
