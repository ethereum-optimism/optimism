package main

type AccessList struct {
	memReads  []uint32
	memWrites []uint32
}

func (al *AccessList) Reset() {
	al.memReads = al.memReads[:0]
	al.memWrites = al.memWrites[:0]
}

func (al *AccessList) OnRead(addr uint32) {
	// if it matches the last, it's a duplicate; this happens because of multiple callbacks for the same effective addr.
	if len(al.memReads) > 0 && al.memReads[len(al.memReads)-1] == addr {
		return
	}
	al.memReads = append(al.memReads, addr)
}

func (al *AccessList) OnWrite(addr uint32) {
	// if it matches the last, it's a duplicate; this happens because of multiple callbacks for the same effective addr.
	if len(al.memWrites) > 0 && al.memWrites[len(al.memWrites)-1] == addr {
		return
	}
	al.memWrites = append(al.memWrites, addr)
}

var _ Tracer = (*AccessList)(nil)

type Tracer interface {
	// OnRead remembers reads from the given addr.
	// Warning: the addr is an effective-addr, i.e. always aligned.
	// But unicorn will fire it multiple times, for each byte that was changed within the effective addr boundaries.
	OnRead(addr uint32)
	// OnWrite remembers writes to the given addr.
	// Warning: the addr is an effective-addr, i.e. always aligned.
	// But unicorn will fire it multiple times, for each byte that was changed within the effective addr boundaries.
	OnWrite(addr uint32)
}

type NoOpTracer struct{}

func (n NoOpTracer) OnRead(addr uint32) {}

func (n NoOpTracer) OnWrite(addr uint32) {}

var _ Tracer = NoOpTracer{}
