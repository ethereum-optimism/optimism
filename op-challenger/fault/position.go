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
