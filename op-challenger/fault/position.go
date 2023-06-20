package fault

import "fmt"

// Position is a golang wrapper around the dispute game Position type.
// Depth refers to how many bisection steps have occurred.
// IndexAtDepth refers to the path that the bisection has taken
// where 1 = goes right & 0 = goes left.
type Position struct {
	Depth        int
	IndexAtDepth int
}

// TraceIndex calculates the what the index of the claim value would be inside the trace.
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

// move goes to the left or right child in the generalized index tree.
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

// parent moves up to the parent in the generalized index tree.
func (p *Position) parent() {
	p.Depth--
	p.IndexAtDepth = p.IndexAtDepth >> 1
}

// Attack moves this position to a position to the left which disagrees with this position.
func (p *Position) Attack() {
	p.move(false)
}

// Defend moves this position to the right which agrees with this position. Note:
func (p *Position) Defend() {
	p.parent()
	p.move(true)
	p.move(false)
}

func (p *Position) Print(maxDepth int) {
	fmt.Printf("GIN: %4b\tTrace Position is %4b\tTrace Depth is: %d\tTrace Index is: %d\n", p.ToGIN(), p.IndexAtDepth, p.Depth, p.TraceIndex(maxDepth))
}

func (p *Position) ToGIN() uint64 {
	return uint64(1<<p.Depth | p.IndexAtDepth)
}

func FromGIN(x uint64) Position {
	depth := MSBIndex(x)
	indexAtDepth := ^(1 << depth) & x
	return Position{
		Depth:        depth,
		IndexAtDepth: int(indexAtDepth),
	}
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
