package testutils

import (
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// TestID represents an eth.BlockID as string, and can be shortened for convenience in test definitions.
//
// Format: <hash-characters>:<number> where the <hash-characters> are
// copied over (i.e. not hex) and <number> is in decimal.
//
// Examples: "foobar:123", or "B:2"
type TestID string

func (id TestID) ID() eth.BlockID {
	parts := strings.Split(string(id), ":")
	if len(parts) != 2 {
		panic("bad id")
	}
	if len(parts[0]) > 32 {
		panic("test ID hash too long")
	}
	var h common.Hash
	copy(h[:], parts[0])
	v, err := strconv.ParseUint(parts[1], 0, 64)
	if err != nil {
		panic(err)
	}
	return eth.BlockID{
		Hash:   h,
		Number: v,
	}
}
