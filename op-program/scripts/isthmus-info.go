package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-program/chainconfig"
)

func main() {
	isthmusTime, err := chainconfig.GetSepoliaIsthmusTime()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading Sepolia Isthmus time: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sepolia Isthmus Hardfork Time: %d\n", isthmusTime)

	// Converting Unix timestamp to a human-readable date
	date := time.Unix(int64(isthmusTime), 0).UTC()
	fmt.Printf("Date (UTC): %s\n", date.Format(time.RFC3339))
}
