package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-fjord/checks"
)

func main() {
	if err := checks.RunApp(os.Args); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
