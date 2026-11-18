package i

import "example.com/geth/types"

func Used(start, fee uint64) uint64 {
	left := types.Call(start)
	return start - fee - left
}

func Gas(start uint64) uint64 { return start - types.Gas2(start) }

func SameRoot(h *types.Header, want types.Hash) bool { return h.Root == want }

func take(int, error) {}

func Apply() { take(types.Apply()) }
