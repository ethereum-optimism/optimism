package bigint

import "math/big"

var (
	Zero = big.NewInt(0)
	One  = big.NewInt(1)
)

// Clamp returns a new big.Int for `end` to which `end - start` <= size.
// @note (start, end) is an inclusive range
func Clamp(start, end *big.Int, size uint64) *big.Int {
	temp := new(big.Int)
	count := temp.Sub(end, start).Uint64() + 1
	if count <= size {
		return end
	}

	// we re-use the allocated temp as the new end
	temp.Add(start, big.NewInt(int64(size-1)))
	return temp
}

// Matcher returns an inner comparison function result for a big.Int
func Matcher(num int64) func(*big.Int) bool {
	return func(bi *big.Int) bool { return bi.Int64() == num }
}

type Range struct {
	Start *big.Int
	End   *big.Int
}

// Grouped will return a slice of inclusive ranges from (start, end),
// capped to the supplied size from `(start, end)`.
func Grouped(start, end *big.Int, size uint64) []Range {
	if end.Cmp(start) < 0 || size == 0 {
		return nil
	}
	bigMaxDiff := big.NewInt(int64(size - 1))

	groups := []Range{}
	for start.Cmp(end) <= 0 {
		diff := new(big.Int).Sub(end, start)
		switch {
		case diff.Uint64()+1 <= size:
			// re-use allocated diff as the next start
			groups = append(groups, Range{start, end})
			start = diff.Add(end, One)
		default:
			// re-use allocated diff as the next start
			end := new(big.Int).Add(start, bigMaxDiff)
			groups = append(groups, Range{start, end})
			start = diff.Add(end, One)
		}
	}

	return groups
}
