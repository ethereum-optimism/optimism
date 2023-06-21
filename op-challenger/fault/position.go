package fault

import "fmt"

// Position is a golang wrapper around the dispute game Position type.
type Position struct {
	depth        int
	indexAtDepth int
}

func NewPosition(depth, indexAtDepth int) Position {
	return Position{depth, indexAtDepth}
}

func NewPositionFromGIndex(x uint64) Position {
	depth := MSBIndex(x)
	indexAtDepth := ^(1 << depth) & x
	return NewPosition(depth, int(indexAtDepth))
}

func (p *Position) Depth() int {
	return p.depth
}

func (p *Position) IndexAtDepth() int {
	return p.indexAtDepth
}

// TraceIndex calculates the what the index of the claim value would be inside the trace.
// It is equivalent to going right until the final depth has been reached.
func (p *Position) TraceIndex(maxDepth int) int {
	// When we go right, we do a shift left and set the bottom bit to be 1.
	// To do this in a single step, do all the shifts at once & or in all 1s for the bottom bits.
	rd := maxDepth - p.depth
	return p.indexAtDepth<<rd | ((1 << rd) - 1)
}

// move goes to the left or right child.
func (p *Position) move(right bool) {
	p.depth++
	p.indexAtDepth = (p.indexAtDepth << 1) | boolToInt(right)
}

func boolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

// parent moves up to the parent.
func (p *Position) parent() {
	p.depth--
	p.indexAtDepth = p.indexAtDepth >> 1
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
	fmt.Printf("GIN: %4b\tTrace Position is %4b\tTrace Depth is: %d\tTrace Index is: %d\n", p.ToGIndex(), p.indexAtDepth, p.depth, p.TraceIndex(maxDepth))
}

func (p *Position) ToGIndex() uint64 {
	return uint64(1<<p.depth | p.indexAtDepth)
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
