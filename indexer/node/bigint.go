package node

import "math/big"

var bigZero = big.NewInt(0)
var bigOne = big.NewInt(1)

// returns a new big.Int for `end` to which `end - start` <= size.
// @note (start, end) is an inclusive range
func clampBigInt(start, end *big.Int, size uint64) *big.Int {
	temp := new(big.Int)

	count := temp.Sub(end, start).Uint64() + 1
	if count <= size {
		return end
	}

	// we result the allocated temp as the new end
	temp.Add(start, big.NewInt(int64(size-1)))
	return temp
}
