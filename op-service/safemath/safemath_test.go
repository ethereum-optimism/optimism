package safemath

import (
	"fmt"
	"math/big"
	"testing"

	"golang.org/x/exp/constraints"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// TestAdd tests saturating-add / safe-add functions.
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

// TestSub tests saturating-sub / safe-sub functions.
func TestSub(t *testing.T) {
	t.Run("typed uint64", testSub[hexutil.Uint64])
	t.Run("uint64", testSub[uint64])
	t.Run("uint32", testSub[uint32])
	t.Run("uint16", testSub[uint16])
	t.Run("uint8", testSub[uint8])
	t.Run("uint", testSub[uint])
}

func testSub[V constraints.Unsigned](t *testing.T) {
	m := ^V(0)
	require.Less(t, m+1, m, "sanity check min value does underflow")
	vals := []V{
		0, 1, 2, 3, m, m - 1, m - 2, m - 100, m / 2, (m / 2) - 1, (m / 2) + 1, (m / 2) + 2,
	}
	// Try every value with every other value. (so this checks (a, b) but also (b, a) calls)
	for _, a := range vals {
		for _, b := range vals {
			t.Run(fmt.Sprintf("%d - %d", a, b), func(t *testing.T) {
				// masked expected outcome to int size, since it may have underflowed
				expectedOut := (uint64(a) - uint64(b)) & uint64(m)
				expectedUnderflow := b > a
				{
					got, underflowed := SafeSub(a, b)
					require.Equal(t, expectedUnderflow, underflowed)
					require.Equal(t, expectedOut, uint64(got))
				}
				{
					got := SaturatingSub(a, b)
					if expectedUnderflow {
						require.Equal(t, uint64(0), uint64(got))
					} else {
						require.Equal(t, expectedOut, uint64(got))
					}
				}
			})
		}
	}
}
