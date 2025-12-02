package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/sources/depset"
)

func main() {
	// Parse command line flags
	enclave := flag.String("enclave", "", "Name of the Kurtosis enclave")
	flag.Parse()

	if *enclave == "" {
		fmt.Fprintln(os.Stderr, "Error: --enclave flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Get depset data
	e := depset.NewExtractor(*enclave)
	ctx := context.Background()
	data, err := e.ExtractData(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing deployer data: %v\n", err)
		os.Exit(1)
	}

	for name, depset := range data {
		fmt.Println(name)
		// Encode as JSON and write to stdout
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(depset); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	}
}
