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

func WeiToETH(wei *big.Int) *big.Float {
	f := new(big.Float)
	f.SetString(wei.String())
	return f.Quo(f, big.NewFloat(1e18))
}
