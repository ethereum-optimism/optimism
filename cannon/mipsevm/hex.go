package mipsevm

import "fmt"

// HexU32 to lazy-format integer attributes for logging
type HexU32 uint32

func (v HexU32) String() string {
	return fmt.Sprintf("%08x", uint32(v))
}

func (v HexU32) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}
