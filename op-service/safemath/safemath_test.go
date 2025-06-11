package safemath

import (
	"math/big"
	"testing"

	"golang.org/x/exp/constraints"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// TestAdd tests add / safe-add functions. Should be part of the Go std-lib, but is not.
func TestAdd(t *testing.T) {
	t.Run("typed uint64", testAdd[hexutil.Uint64])
	t.Run("uint64", testAdd[uint64])
	t.Run("uint32", testAdd[uint32])
	t.Run("uint16", testAdd[uint16])
	t.Run("uint8", testAdd[uint8])
	t.Run("uint", testAdd[uint])
}

func testAdd[V constraints.Unsigned](t *testing.T) {
	m := ^V(0)
	require.Less(t, m+1, m, "sanity check max value does overflow")
	vals := []V{
		0, 1, 2, 3, m, m - 1, m - 2, m - 100, m / 2, (m / 2) - 1, (m / 2) + 1, (m / 2) + 2,
	}
	mBig := new(big.Int).SetUint64(uint64(m))
	// Try every value with every other value. (so this checks (a, b) but also (b, a) calls)
	for _, a := range vals {
		for _, b := range vals {
			expectedSum := new(big.Int).Add(
				new(big.Int).SetUint64(uint64(a)),
				new(big.Int).SetUint64(uint64(b)))
			expectedOverflow := expectedSum.Cmp(mBig) > 0
			{
				got, overflowed := SafeAdd(a, b)
				require.Equal(t, expectedOverflow, overflowed)
				// masked expected outcome to int size, since it may have overflowed
				require.Equal(t, expectedSum.Uint64()&uint64(m), uint64(got))
			}
			{
				got := SaturatingAdd(a, b)
				if expectedOverflow {
					require.Equal(t, uint64(m), uint64(got))
				} else {
					require.Equal(t, expectedSum.Uint64(), uint64(got))
				}
			}
		}
	}
}
