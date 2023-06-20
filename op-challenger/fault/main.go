package main

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// Position is a golang wrapper around the dispute game Position type.
// Depth refers to how many bisection steps have occurred.
// IndexAtDepth refers to the path that the bisection has taken
// where 1 = goes right & 0 = goes left.
type Position struct {
	Depth        int
	IndexAtDepth int
}

// TraceIndex calculates the what the index of the claim value
// would be inside the trace.
func (p *Position) TraceIndex(maxDepth int) int {
	lo := 0
	hi := 1 << maxDepth
	mid := hi
	path := p.IndexAtDepth
	for i := p.Depth - 1; i >= 0; i-- {
		mid = (lo + hi) / 2
		mask := 1 << i
		if path&mask == mask {
			lo = mid
		} else {
			hi = mid
		}
	}
	return mid
}

func (p *Position) move(right bool) {
	p.Depth++
	p.IndexAtDepth = (p.IndexAtDepth << 1) | boolToInt(right)
}

func boolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func (p *Position) parent() {
	p.Depth--
	p.IndexAtDepth = p.IndexAtDepth >> 1
}

func (p *Position) Attack() {
	p.move(false)
}

func (p *Position) Defend() {
	p.parent()
	p.move(true)
	p.move(false)
}

func (p *Position) Print(maxDepth int) {
	fmt.Printf("Trace Position is %04b\tTrace Depth is: %d\tTrace Index is: %d\n", p.IndexAtDepth, p.Depth, p.TraceIndex(maxDepth))
}

var (
	ErrNegativeIndex = errors.New("index cannot be negative")
	ErrIndexTooLarge = errors.New("index is larger than the maximum index")
)

// TraceProvider is a generic way to get a claim value at a specific
// step in the trace.
type TraceProvider interface {
	Get(i int) (common.Hash, error)
}

type Claim struct {
	Value common.Hash
	Position
}

type Response struct {
	Attack bool // note: can we flip this to true == going right / defending??
	Value  common.Hash
}

func main() {
	// Example 1
	// abcdefgh
	// abcdexyz
	// go left to d, then right to f, then left to e
	p := Position{}
	p.Print(3)
	p.Attack()
	p.Print(3)
	p.Defend()
	p.Print(3)
	p.Attack()
	p.Print(3)

	// Trace Position is 0000	Trace Depth is: 0	Trace Index is: 8
	// Trace Position is 0000	Trace Depth is: 1	Trace Index is: 4
	// Trace Position is 0010	Trace Depth is: 2	Trace Index is: 6
	// Trace Position is 0100	Trace Depth is: 3	Trace Index is: 5

	// Example 2
	// abcdefgh
	// abqrstuv
	// go left r, then left to b, then right to q
	p = Position{}
	p.Print(3)
	p.Attack()
	p.Print(3)
	p.Attack()
	p.Print(3)
	p.Defend()
	p.Print(3)

	// Trace Position is 0000	Trace Depth is: 0	Trace Index is: 8
	// Trace Position is 0000	Trace Depth is: 1	Trace Index is: 4
	// Trace Position is 0000	Trace Depth is: 2	Trace Index is: 2
	// Trace Position is 0010	Trace Depth is: 3	Trace Index is: 3
}
