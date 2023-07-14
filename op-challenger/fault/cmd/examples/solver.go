package examples

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/fault"
)

func PrettyPrintAlphabetClaim(name string, claim fault.Claim) {
	value := claim.Value
	idx := value[30]
	letter := value[31]
	if claim.IsRoot() {
		fmt.Printf("%s\ttrace %v letter %c\n", name, idx, letter)
	} else {
		fmt.Printf("%s\ttrace %v letter %c is attack %v\n", name, idx, letter, !claim.DefendsParent())
	}

}

// SolverExampleOne uses the [fault.Solver] with a [fault.AlphabetProvider]
// to print out fault game traces for the "abcdexyz" counter-state.
func SolverExampleOne() {
	fmt.Println("Solver: Example 1")

	// Construct the fault position.
	canonical := "abcdefgh"
	disputed := "abcdexyz"
	maxDepth := 3
	// Root claim is z at trace index 7 from the disputed provider
	root := fault.Claim{
		ClaimData: fault.ClaimData{
			Value:    common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000077a"),
			Position: fault.NewPosition(0, 0),
		},
	}

	canonicalProvider := fault.NewAlphabetProvider(canonical, uint64(maxDepth))
	disputedProvider := fault.NewAlphabetProvider(disputed, uint64(maxDepth))

	// Create a solver with the canonical provider.
	cannonicalSolver := fault.NewSolver(maxDepth, canonicalProvider)
	disputedSolver := fault.NewSolver(maxDepth, disputedProvider)

	// Print the initial state.
	fmt.Println("Canonical state: ", canonical)
	fmt.Println("Disputed state:  ", disputed)
	fmt.Println()
	fmt.Println("Proceeding with the following moves:")
	fmt.Println("go left to d, then right to x (cannonical is f), then left to e")
	fmt.Println()
	PrettyPrintAlphabetClaim("Root claim", root)

	claim1, err := cannonicalSolver.NextMove(root, false)
	if err != nil {
		fmt.Printf("error getting claim from provider: %v", err)
	}
	PrettyPrintAlphabetClaim("Cannonical move", *claim1)

	claim2, err := disputedSolver.NextMove(*claim1, false)
	if err != nil {
		fmt.Printf("error getting claim from provider: %v", err)
	}
	PrettyPrintAlphabetClaim("Disputed moved", *claim2)

	claim3, err := cannonicalSolver.NextMove(*claim2, false)
	if err != nil {
		fmt.Printf("error getting claim from provider: %v", err)
	}
	PrettyPrintAlphabetClaim("Cannonical move", *claim3)
}
