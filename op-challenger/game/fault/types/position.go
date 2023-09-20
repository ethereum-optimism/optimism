package types

import (
	"fmt"
	"math/big"
)

// Position is a golang wrapper around the dispute game Position type.
type Position struct {
	depth        int
	indexAtDepth *big.Int
}

func NewPosition(depth int, indexAtDepth int) Position {
	return Position{
		depth:        depth,
		indexAtDepth: big.NewInt(int64(indexAtDepth)),
	}
}

func NewPositionFromGIndex(x uint64) Position {
	depth := MSBIndex(x)
	indexAtDepth := ^(1 << depth) & x
	return NewPosition(depth, int(indexAtDepth))
}

func (p Position) Depth() int {
	return p.depth
}

func (p Position) IndexAtDepth() *big.Int {
	if p.indexAtDepth == nil {
		return big.NewInt(0)
	}
	return p.indexAtDepth
}

func (p Position) IsRootPosition() bool {
	return p.depth == 0 && big.NewInt(0).Cmp(p.indexAtDepth) == 0
}

func (p Position) lshIndex(amount int) *big.Int {
	return new(big.Int).Lsh(p.IndexAtDepth(), uint(amount))
}

// TraceIndex calculates the what the index of the claim value would be inside the trace.
// It is equivalent to going right until the final depth has been reached.
func (p Position) TraceIndex(maxDepth int) uint64 {
	// When we go right, we do a shift left and set the bottom bit to be 1.
	// To do this in a single step, do all the shifts at once & or in all 1s for the bottom bits.
	rd := maxDepth - p.depth
	rhs := ((1 << rd) - 1)
	ti := new(big.Int).Or(p.lshIndex(rd), big.NewInt(int64(rhs)))
	return ti.Uint64()
}

// move returns a new position at the left or right child.
func (p Position) move(right bool) Position {
	return Position{
		depth:        p.depth + 1,
		indexAtDepth: big.NewInt(0).Or(p.lshIndex(1), big.NewInt(int64(boolToInt(right)))),
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func (p Position) parentIndexAtDepth() *big.Int {
	return big.NewInt(0).Div(p.IndexAtDepth(), big.NewInt(2))
}

func (p Position) DefendsParent(parentIndex *big.Int) bool {
	return p.parentIndexAtDepth().Cmp(parentIndex) != 0
}

// parent return a new position that is the parent of this Position.
func (p Position) parent() Position {
	return Position{
		depth:        p.depth - 1,
		indexAtDepth: p.parentIndexAtDepth(),
	}
}

// Attack creates a new position which is the attack position of this one.
func (p Position) Attack() Position {
	return p.move(false)
}

// Defend creates a new position which is the defend position of this one.
func (p Position) Defend() Position {
	return p.parent().move(true).move(false)
}

func (p Position) Print(maxDepth int) {
	fmt.Printf("GIN: %4b\tTrace Position is %4b\tTrace Depth is: %d\tTrace Index is: %d\n", p.ToGIndex(), p.indexAtDepth.Uint64(), p.depth, p.TraceIndex(maxDepth))
}

func (p Position) ToGIndex() *big.Int {
	return big.NewInt(0).Or(big.NewInt(1<<p.depth), p.IndexAtDepth())
}

// MSBIndex returns the index of the most significant bit
func MSBIndex(x uint64) int {
	if x == 0 {
		return 0
	}
	out := 0
	for ; x != 0; out++ {
		x = x >> 1
	}
	return out - 1
}
