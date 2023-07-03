package examples

import (
	"github.com/ethereum-optimism/optimism/op-challenger/fault"
)

func PositionExampleOne() {
	// Example 1
	// abcdefgh
	// abcdexyz
	// go left to d, then right to f, then left to e
	p := fault.Position{}
	p.Print(3)
	p = p.Attack()
	p.Print(3)
	p = p.Defend()
	p.Print(3)
	p = p.Attack()
	p.Print(3)
}

func PositionExampleTwo() {
	// Example 2
	// abcdefgh
	// abqrstuv
	// go left r, then left to b, then right to q
	p := fault.Position{}
	p.Print(3)
	p = p.Attack()
	p.Print(3)
	p = p.Attack()
	p.Print(3)
	p = p.Defend()
	p.Print(3)
}
