package txinclude

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNonceManagerNext(t *testing.T) {
	t.Run("sequential nonces with no gaps", func(t *testing.T) {
		nm := newNonceManager(5)

		require.Equal(t, uint64(5), nm.Next())
		require.Equal(t, uint64(6), nm.Next())
		require.Equal(t, uint64(7), nm.Next())
		require.Equal(t, uint64(8), nm.Next())
	})

	t.Run("handles single gap", func(t *testing.T) {
		nm := newNonceManager(20)

		nm.InsertGap(15)

		require.Equal(t, uint64(15), nm.Next())
		require.Equal(t, uint64(20), nm.Next())
		require.Equal(t, uint64(21), nm.Next())
	})

	t.Run("gap first before incrementing", func(t *testing.T) {
		nm := newNonceManager(10)

		// Add some gaps
		nm.InsertGap(3)
		nm.InsertGap(5)
		nm.InsertGap(7)

		// Should return gaps in order first
		require.Equal(t, uint64(3), nm.Next())
		require.Equal(t, uint64(5), nm.Next())
		require.Equal(t, uint64(7), nm.Next())

		// Then continue with normal sequence
		require.Equal(t, uint64(10), nm.Next())
		require.Equal(t, uint64(11), nm.Next())
	})
}

func TestNonceManagerInsertGap(t *testing.T) {
	t.Run("inserts gaps in sorted order", func(t *testing.T) {
		nm := newNonceManager(100)

		// Insert gaps in random order
		nm.InsertGap(50)
		nm.InsertGap(30)
		nm.InsertGap(70)
		nm.InsertGap(40)
		nm.InsertGap(60)

		// Verify they are sorted
		require.Equal(t, []uint64{30, 40, 50, 60, 70}, nm.gaps)
	})

	t.Run("duplicate gap is a no-op", func(t *testing.T) {
		nm := newNonceManager(100)

		nm.InsertGap(50)
		nm.InsertGap(50)

		// Verify gaps unchanged
		require.Equal(t, uint64(50), nm.Next())
		require.Equal(t, uint64(100), nm.Next())
	})

	t.Run("handles multiple duplicates", func(t *testing.T) {
		nm := newNonceManager(100)

		nm.InsertGap(10)
		nm.InsertGap(20)
		nm.InsertGap(30)

		nm.InsertGap(10)
		nm.InsertGap(20)
		nm.InsertGap(30)

		require.Equal(t, uint64(10), nm.Next())
		require.Equal(t, uint64(20), nm.Next())
		require.Equal(t, uint64(30), nm.Next())
		require.Equal(t, uint64(100), nm.Next())
	})
}
