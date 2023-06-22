package examples

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/fault"
)

// SolverExampleOne uses the [fault.Solver] with a [fault.AlphabetProvider]
// to print out fault game traces for the "abcdexyz" counter-state.
func SolverExampleOne() {
	fmt.Println()
	fmt.Println("Solver: Example 1")
	fmt.Println()

	// Construct the fault position.
	canonical := "abcdefgh"
	disputed := "abcdexyz"
	maxDepth := 3
	parent := fault.Claim{
		Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
		Position: fault.NewPosition(0, 0),
	}
	canonicalProvider := fault.NewAlphabetProvider(canonical, uint64(maxDepth))
	disputedProvider := fault.NewAlphabetProvider(disputed, uint64(maxDepth))

	// Create a solver with the canonical provider.
	solver := fault.NewSolver(maxDepth, canonicalProvider)

	// Print the initial state.
	fmt.Println("Canonical state: ", canonical)
	fmt.Println("Disputed state: ", disputed)
	fmt.Println()
	fmt.Println("Proceeding with the following moves:")
	fmt.Println("go left to d, then right to f, then left to e")
	fmt.Println()

	// Get the claim from the disputed provider.
	claim, err := disputedProvider.Get(3)
	if err != nil {
		fmt.Printf("error getting claim from disputed provider: %v", err)
	}
	firstDisputedClaim := fault.Claim{
		Value:    claim,
		Position: fault.NewPosition(1, 0),
	}
	res, err := solver.NextMove(firstDisputedClaim, parent)
	if err != nil {
		fmt.Printf("error getting next move: %v", err)
	}
	fmt.Printf("Disputed claim: %s\n", claim)
	fmt.Printf("Expected claim: %s\n", parent.Value)
	fmt.Printf("Response: [Attack: %v, Value: %s]\n", res.Attack, res.Value)
	fmt.Println()

	// Get the next claim from the disputed provider.
	claim, err = disputedProvider.Get(5)
	if err != nil {
		fmt.Printf("error getting claim from disputed provider: %v", err)
	}
	firstDisputedClaim = fault.Claim{
		Value:    claim,
		Position: fault.NewPosition(2, 2),
	}
	res, err = solver.NextMove(firstDisputedClaim, parent)
	if err != nil {
		fmt.Printf("error getting next move: %v", err)
	}
	fmt.Printf("Disputed claim: %s\n", claim)
	fmt.Printf("Expected claim: %s\n", parent.Value)
	fmt.Printf("Response: [Attack: %v, Value: %s]\n", res.Attack, res.Value)
	fmt.Println()

	// This marks the end of the game!
	if res.Attack {
		fmt.Println("Game successfully completed!")
	} else {
		fmt.Println("Game failed!")
	}
	fmt.Println()
}
