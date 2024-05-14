package types

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrPositionDepthTooSmall = errors.New("position depth is too small")

	RootPosition = NewPositionFromGIndex(big.NewInt(1))
)

// Depth is the depth of a position in a game tree where the root level has
// depth 0, the root's children have depth 1, their children have depth 2, and
// so on.
type Depth uint64

// Position is a golang wrapper around the dispute game Position type.
type Position struct {
	depth        Depth
	indexAtDepth *big.Int
}

func NewPosition(depth Depth, indexAtDepth *big.Int) Position {
	return Position{
		depth:        depth,
		indexAtDepth: indexAtDepth,
	}
}

// NewPositionFromGIndex creates a new Position given a generalized index.
func NewPositionFromGIndex(x *big.Int) Position {
	depth := bigMSB(x)
	withoutMSB := new(big.Int).Not(new(big.Int).Lsh(big.NewInt(1), uint(depth)))
	indexAtDepth := new(big.Int).And(x, withoutMSB)
	return NewPosition(depth, indexAtDepth)
}

func (p Position) String() string {
	return fmt.Sprintf("Position(depth: %v, indexAtDepth: %v)", p.depth, p.IndexAtDepth())
}

func (p Position) MoveRight() Position {
	return Position{
		depth:        p.depth,
		indexAtDepth: new(big.Int).Add(p.IndexAtDepth(), big.NewInt(1)),
	}
}

// RelativeToAncestorAtDepth returns a new position for a subtree.
// [ancestor] is the depth of the subtree root node.
func (p Position) RelativeToAncestorAtDepth(ancestor Depth) (Position, error) {
	if ancestor > p.depth {
		return Position{}, ErrPositionDepthTooSmall
	}
	newPosDepth := p.depth - ancestor
	nodesAtDepth := 1 << newPosDepth
	newIndexAtDepth := new(big.Int).Mod(p.IndexAtDepth(), big.NewInt(int64(nodesAtDepth)))
	return NewPosition(newPosDepth, newIndexAtDepth), nil
}

func (p Position) Depth() Depth {
	return p.depth
}

func (p Position) IndexAtDepth() *big.Int {
	if p.indexAtDepth == nil {
		return common.Big0
	}
	return p.indexAtDepth
}

func (p Position) IsRootPosition() bool {
	return p.depth == 0 && common.Big0.Cmp(p.IndexAtDepth()) == 0
}

func (p Position) lshIndex(amount Depth) *big.Int {
	return new(big.Int).Lsh(p.IndexAtDepth(), uint(amount))
}

// TraceIndex calculates the what the index of the claim value would be inside the trace.
// It is equivalent to going right until the final depth has been reached.
// Note: this method will panic if maxDepth < p.depth
func (p Position) TraceIndex(maxDepth Depth) *big.Int {
	// When we go right, we do a shift left and set the bottom bit to be 1.
	// To do this in a single step, do all the shifts at once & or in all 1s for the bottom bits.
	if maxDepth < p.depth {
		panic(fmt.Sprintf("maxDepth(%d) < p.depth(%d)", maxDepth, p.depth))
	}
	rd := maxDepth - p.depth
	rhs := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(rd)), big.NewInt(1))
	ti := new(big.Int).Or(p.lshIndex(rd), rhs)
	return ti
}

// move returns a new position at the left or right child.
func (p Position) move(right bool) Position {
	return Position{
		depth:        p.depth + 1,
		indexAtDepth: new(big.Int).Or(p.lshIndex(1), big.NewInt(int64(boolToInt(right)))),
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
	return new(big.Int).Div(p.IndexAtDepth(), big.NewInt(2))
}

func (p Position) RightOf(parent Position) bool {
	return p.parentIndexAtDepth().Cmp(parent.IndexAtDepth()) != 0
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

func (p Position) Print(maxDepth Depth) {
	fmt.Printf("GIN: %4b\tTrace Position is %4b\tTrace Depth is: %d\tTrace Index is: %d\n", p.ToGIndex(), p.IndexAtDepth(), p.depth, p.TraceIndex(maxDepth))
}

func (p Position) ToGIndex() *big.Int {
	return new(big.Int).Or(new(big.Int).Lsh(big.NewInt(1), uint(p.depth)), p.IndexAtDepth())
}

// bigMSB returns the index of the most significant bit
func bigMSB(x *big.Int) Depth {
	if x.Cmp(big.NewInt(0)) == 0 {
		return 0
	}
	return Depth(x.BitLen() - 1)
}
