package types

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func bi(i int) *big.Int {
	return big.NewInt(int64(i))
}

func TestBigMSB(t *testing.T) {
	large, ok := new(big.Int).SetString("18446744073709551615", 10)
	require.True(t, ok)
	tests := []struct {
		input    *big.Int
		expected int
	}{
		{bi(0), 0},
		{bi(1), 0},
		{bi(2), 1},
		{bi(4), 2},
		{bi(8), 3},
		{bi(16), 4},
		{bi(255), 7},
		{bi(1024), 10},
		{large, 63},
	}

	for _, test := range tests {
		result := bigMSB(test.input)
		if result != test.expected {
			t.Errorf("MSBIndex(%d) expected %d, but got %d", test.input, test.expected, result)
		}
	}
}

type testNodeInfo struct {
	GIndex       *big.Int
	Depth        int
	MaxDepth     int
	IndexAtDepth *big.Int
	TraceIndex   *big.Int
	AttackGIndex *big.Int // 0 indicates attack is not possible from this node
	DefendGIndex *big.Int // 0 indicates defend is not possible from this node
}

var treeNodes = []testNodeInfo{
	{GIndex: bi(1), Depth: 0, MaxDepth: 4, IndexAtDepth: bi(0), TraceIndex: bi(15), AttackGIndex: bi(2)},

	{GIndex: bi(2), Depth: 1, MaxDepth: 4, IndexAtDepth: bi(0), TraceIndex: bi(7), AttackGIndex: bi(4), DefendGIndex: bi(6)},
	{GIndex: bi(3), Depth: 1, MaxDepth: 4, IndexAtDepth: bi(1), TraceIndex: bi(15), AttackGIndex: bi(6)},

	{GIndex: bi(4), Depth: 2, MaxDepth: 4, IndexAtDepth: bi(0), TraceIndex: bi(3), AttackGIndex: bi(8), DefendGIndex: bi(10)},
	{GIndex: bi(5), Depth: 2, MaxDepth: 4, IndexAtDepth: bi(1), TraceIndex: bi(7), AttackGIndex: bi(10)},
	{GIndex: bi(6), Depth: 2, MaxDepth: 4, IndexAtDepth: bi(2), TraceIndex: bi(11), AttackGIndex: bi(12), DefendGIndex: bi(14)},
	{GIndex: bi(7), Depth: 2, MaxDepth: 4, IndexAtDepth: bi(3), TraceIndex: bi(15), AttackGIndex: bi(14)},

	{GIndex: bi(8), Depth: 3, MaxDepth: 4, IndexAtDepth: bi(0), TraceIndex: bi(1), AttackGIndex: bi(16), DefendGIndex: bi(18)},
	{GIndex: bi(9), Depth: 3, MaxDepth: 4, IndexAtDepth: bi(1), TraceIndex: bi(3), AttackGIndex: bi(18)},
	{GIndex: bi(10), Depth: 3, MaxDepth: 4, IndexAtDepth: bi(2), TraceIndex: bi(5), AttackGIndex: bi(20), DefendGIndex: bi(22)},
	{GIndex: bi(11), Depth: 3, MaxDepth: 4, IndexAtDepth: bi(3), TraceIndex: bi(7), AttackGIndex: bi(22)},
	{GIndex: bi(12), Depth: 3, MaxDepth: 4, IndexAtDepth: bi(4), TraceIndex: bi(9), AttackGIndex: bi(24), DefendGIndex: bi(26)},
	{GIndex: bi(13), Depth: 3, MaxDepth: 4, IndexAtDepth: bi(5), TraceIndex: bi(11), AttackGIndex: bi(26)},
	{GIndex: bi(14), Depth: 3, MaxDepth: 4, IndexAtDepth: bi(6), TraceIndex: bi(13), AttackGIndex: bi(28), DefendGIndex: bi(30)},
	{GIndex: bi(15), Depth: 3, MaxDepth: 4, IndexAtDepth: bi(7), TraceIndex: bi(15), AttackGIndex: bi(30)},

	{GIndex: bi(16), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(0), TraceIndex: bi(0)},
	{GIndex: bi(17), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(1), TraceIndex: bi(1)},
	{GIndex: bi(18), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(2), TraceIndex: bi(2)},
	{GIndex: bi(19), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(3), TraceIndex: bi(3)},
	{GIndex: bi(20), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(4), TraceIndex: bi(4)},
	{GIndex: bi(21), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(5), TraceIndex: bi(5)},
	{GIndex: bi(22), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(6), TraceIndex: bi(6)},
	{GIndex: bi(23), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(7), TraceIndex: bi(7)},
	{GIndex: bi(24), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(8), TraceIndex: bi(8)},
	{GIndex: bi(25), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(9), TraceIndex: bi(9)},
	{GIndex: bi(26), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(10), TraceIndex: bi(10)},
	{GIndex: bi(27), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(11), TraceIndex: bi(11)},
	{GIndex: bi(28), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(12), TraceIndex: bi(12)},
	{GIndex: bi(29), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(13), TraceIndex: bi(13)},
	{GIndex: bi(30), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(14), TraceIndex: bi(14)},
	{GIndex: bi(31), Depth: 4, MaxDepth: 4, IndexAtDepth: bi(15), TraceIndex: bi(15)},

	{GIndex: bi(0).Mul(bi(math.MaxInt64), bi(2)), Depth: 63, MaxDepth: 64, IndexAtDepth: bi(9223372036854775806), TraceIndex: bi(0).Sub(bi(0).Mul(bi(math.MaxInt64), bi(2)), bi(1))},
}

// TestGINConversions does To & From the generalized index on the treeNodesMaxDepth4 data
func TestGINConversions(t *testing.T) {
	for _, test := range treeNodes {
		from := NewPositionFromGIndex(test.GIndex)
		pos := NewPosition(test.Depth, test.IndexAtDepth)
		require.EqualValuesf(t, pos.Depth(), from.Depth(), "From GIndex %v vs pos %v", from.Depth(), pos.Depth())
		require.Zerof(t, pos.IndexAtDepth().Cmp(from.IndexAtDepth()), "From GIndex %v vs pos %v", from.IndexAtDepth(), pos.IndexAtDepth())
		to := pos.ToGIndex()
		require.Equal(t, test.GIndex, to)
	}
}

func TestTraceIndexOfRootWithLargeDepth(t *testing.T) {
	traceIdx := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 100), big.NewInt(1))
	pos := NewPositionFromGIndex(big.NewInt(1))
	actual := pos.TraceIndex(100)
	require.Equal(t, traceIdx, actual)
}

// TestTraceIndex creates the position & then tests the trace index function on the treeNodesMaxDepth4 data
func TestTraceIndex(t *testing.T) {
	for _, test := range treeNodes {
		pos := NewPosition(test.Depth, test.IndexAtDepth)
		result := pos.TraceIndex(test.MaxDepth)
		require.Equal(t, test.TraceIndex, result)
	}
}

func TestAttack(t *testing.T) {
	for _, test := range treeNodes {
		if test.AttackGIndex == nil || test.AttackGIndex.Cmp(big.NewInt(0)) == 0 {
			continue
		}
		pos := NewPosition(test.Depth, test.IndexAtDepth)
		result := pos.Attack()
		require.Equalf(t, test.AttackGIndex, result.ToGIndex(), "Attack from GIndex %v", pos.ToGIndex())
	}
}

func TestDefend(t *testing.T) {
	for _, test := range treeNodes {
		if test.DefendGIndex == nil || test.DefendGIndex.Cmp(big.NewInt(0)) == 0 {
			continue
		}
		pos := NewPosition(test.Depth, test.IndexAtDepth)
		result := pos.Defend()
		require.Equalf(t, test.DefendGIndex, result.ToGIndex(), "Defend from GIndex %v", pos.ToGIndex())
	}
}

func TestRelativeToAncestorAtDepth(t *testing.T) {
	t.Run("ErrorsForDeepAncestor", func(t *testing.T) {
		pos := NewPosition(1, big.NewInt(1))
		_, err := pos.RelativeToAncestorAtDepth(2)
		require.ErrorIs(t, err, ErrPositionDepthTooSmall)
	})

	t.Run("Success", func(t *testing.T) {
		pos := NewPosition(2, big.NewInt(1))
		expectedRelativePosition := NewPosition(1, big.NewInt(1))
		relativePosition, err := pos.RelativeToAncestorAtDepth(1)
		require.NoError(t, err)
		require.Equal(t, expectedRelativePosition, relativePosition)
	})
}
