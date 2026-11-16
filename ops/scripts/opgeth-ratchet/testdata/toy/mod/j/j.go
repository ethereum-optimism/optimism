package j

import "example.com/geth/types"

func FieldFirst(b *types.Block) uint64 { return b.GasLimit - types.Call(1) }

func AssignToField(b *types.Block) { b.GasUsed = types.Call(1) }

func VarFirst() uint64 {
	limit := types.Limit()
	left := types.Call(1)
	return limit - left
}

func SameAsEmpty(h *types.Header) bool { return h.Root == types.EmptyRoot() }
