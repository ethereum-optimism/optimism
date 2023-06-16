package mipsevm

import (
	"debug/elf"
	"fmt"
	"sort"
)

type Symbol struct {
	Name  string `json:"name"`
	Start uint32 `json:"start"`
	Size  uint32 `json:"size"`
}

type Metadata struct {
	Symbols []Symbol `json:"symbols"`
}

func MakeMetadata(elfProgram *elf.File) (*Metadata, error) {
	syms, err := elfProgram.Symbols()
	if err != nil {
		return nil, fmt.Errorf("failed to load symbols table: %w", err)
	}
	// Make sure the table is sorted, Go outputs mostly sorted data, except some internal functions
	sort.Slice(syms, func(i, j int) bool {
		return syms[i].Value < syms[j].Value
	})
	out := &Metadata{Symbols: make([]Symbol, len(syms))}
	for i, s := range syms {
		out.Symbols[i] = Symbol{Name: s.Name, Start: uint32(s.Value), Size: uint32(s.Size)}
	}
	return out, nil
}

func (m *Metadata) LookupSymbol(addr uint32) string {
	if len(m.Symbols) == 0 {
		return "!unknown"
	}
	// find first symbol with higher start. Or n if no such symbol exists
	i := sort.Search(len(m.Symbols), func(i int) bool {
		return m.Symbols[i].Start > addr
	})
	if i == 0 {
		return "!start"
	}
	out := &m.Symbols[i-1]
	if out.Start+out.Size < addr { // addr may be pointing to a gap between symbols
		return "!gap"
	}
	return out.Name
}

func (m *Metadata) SymbolMatcher(name string) func(addr uint32) bool {
	for _, s := range m.Symbols {
		if s.Name == name {
			start := s.Start
			end := s.Start + s.Size
			return func(addr uint32) bool {
				return addr >= start && addr < end
			}
		}
	}
	return func(addr uint32) bool {
		return false
	}
}

// HexU32 to lazy-format integer attributes for logging
type HexU32 uint32

func (v HexU32) String() string {
	return fmt.Sprintf("%08x", uint32(v))
}

func (v HexU32) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}
