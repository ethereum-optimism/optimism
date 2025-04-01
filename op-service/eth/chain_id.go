package eth

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/holiman/uint256"
)

type ChainID uint256.Int

func ChainIDFromBig(chainID *big.Int) ChainID {
	return ChainID(*uint256.MustFromBig(chainID))
}

func ChainIDFromUInt64(i uint64) ChainID {
	return ChainID(*uint256.NewInt(i))
}

func ChainIDFromBytes32(b [32]byte) ChainID {
	val := new(uint256.Int).SetBytes(b[:])
	return ChainID(*val)
}

func ParseDecimalChainID(chainID string) (ChainID, error) {
	v, err := uint256.FromDecimal(chainID)
	if err != nil {
		return ChainID{}, err
	}
	return ChainID(*v), nil
}

// String encodes the chainID in decimal form.
func (id ChainID) String() string {
	return ((*uint256.Int)(&id)).Dec()
}

func (id ChainID) Bytes32() [32]byte {
	return (*uint256.Int)(&id).Bytes32()
}

// IsUint64 reports if the chainID fits in 64 bits
func (id ChainID) IsUint64() bool {
	return (*uint256.Int)(&id).IsUint64()
}

// Uint64 is a convenience function, to turn the chainID into 64 bits.
// Not all chain IDs fit in 64 bits. This should be used only for compatibility with
// legacy tools / libs that do not use the full 32 byte chainID type.
func (id ChainID) Uint64() (v uint64, ok bool) {
	return (*uint256.Int)(&id).Uint64(), id.IsUint64()
}

// EvilChainIDToUInt64 converts a ChainID to a uint64 and panic's if the ChainID is too large for a UInt64
// It is "evil" because 32 byte ChainIDs should be universally supported which this method breaks. It is provided
// for legacy purposes to facilitate a transition to full 32 byte chain ID support and should not be used in new code.
// Existing calls should be replaced with full 32 byte support whenever possible.
func EvilChainIDToUInt64(id ChainID) uint64 {
	v := (*uint256.Int)(&id)
	if !v.IsUint64() {
		panic(fmt.Errorf("ChainID too large for uint64: %v", id))
	}
	return v.Uint64()
}

func (id *ChainID) ToBig() *big.Int {
	return (*uint256.Int)(id).ToBig()
}

// MarshalText marshals into the decimal representation of the chainID
func (id ChainID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

// UnmarshalText can unmarshal both a hexadecimal (must have 0x prefix) or decimal chainID
func (id *ChainID) UnmarshalText(data []byte) error {
	var x uint256.Int
	err := x.UnmarshalText(data)
	if err != nil {
		return err
	}
	*id = ChainID(x)
	return nil
}

func (id ChainID) Cmp(other ChainID) int {
	return (*uint256.Int)(&id).Cmp((*uint256.Int)(&other))
}

// SortChainID sorts chain IDs in ascending order, in-place.
func SortChainID(ids []ChainID) {
	sort.Slice(ids, func(i, j int) bool {
		return ids[i].Cmp(ids[j]) < 0
	})
}
