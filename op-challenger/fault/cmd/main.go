package main

import (
	"github.com/ethereum-optimism/optimism/op-challenger/fault"
)

func main() {
	// Example 1
	// abcdefgh
	// abcdexyz
	// go left to d, then right to f, then left to e
	p := fault.Position{}
	p.Print(3)
	p.Attack()
	p.Print(3)
	p.Defend()
	p.Print(3)
	p.Attack()
	p.Print(3)

	// GIN:    1	Trace Position is    0	Trace Depth is: 0	Trace Index is: 8
	// GIN:   10	Trace Position is    0	Trace Depth is: 1	Trace Index is: 4
	// GIN:  110	Trace Position is   10	Trace Depth is: 2	Trace Index is: 6
	// GIN: 1100	Trace Position is  100	Trace Depth is: 3	Trace Index is: 5

	// Example 2
	// abcdefgh
	// abqrstuv
	// go left r, then left to b, then right to q
	p = fault.Position{}
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
